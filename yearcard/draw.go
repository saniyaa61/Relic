package yearcard

// Drawing helpers: text with the bundled fonts (and a symbols fallback),
// rounded rectangles and circles, all anti-aliased, on an *image.RGBA.
// Colours blend in sRGB, as a browser canvas does.

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"path"
	"strings"
	"sync"
	"unicode/utf8"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"

	"github.com/saniyaa61/relic/assets"
)

// Font files, by the prototype's canvas font names.
const (
	serifSemiBold = "PlayfairDisplay-SemiBold.ttf" // 600 "Playfair Display"
	loraItalic    = "Lora-Italic.ttf"              // italic 400 Lora
	sansMedium    = "DMSans-Medium.ttf"            // 500 "DM Sans"
	symbols       = "NotoSansSymbols2-Regular.ttf" // ★ (the browser borrows it from a system font)
)

var (
	fontMu    sync.Mutex
	fontFiles = map[string]*sfnt.Font{}
	faces     = map[string]font.Face{}
)

func loadFont(name string) (*sfnt.Font, error) {
	if f := fontFiles[name]; f != nil {
		return f, nil
	}
	data, err := assets.Fonts.ReadFile(path.Join("fonts", name))
	if err != nil {
		return nil, err
	}
	f, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	fontFiles[name] = f
	return f, nil
}

// face returns the named font at size px.
func face(name string, size float64) font.Face {
	fontMu.Lock()
	defer fontMu.Unlock()
	key := fmt.Sprintf("%s@%g", name, size)
	if fc := faces[key]; fc != nil {
		return fc
	}
	f, err := loadFont(name)
	if err != nil {
		panic(err) // bundled files: a failure is a build mistake
	}
	fc, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		panic(err)
	}
	faces[key] = fc
	return fc
}

// textStyle is a canvas font plus fill colour and letter-spacing (px).
type textStyle struct {
	file    string
	size    float64
	color   color.Color
	spacing float64
}

func (s textStyle) face() font.Face { return face(s.file, s.size) }

// faceFor picks the face that has r: the style's, else the symbols font.
func (s textStyle) faceFor(r rune) font.Face {
	f := s.face()
	if _, ok := f.GlyphAdvance(r); ok {
		return f
	}
	return face(symbols, s.size)
}

// width is how wide s is drawn, letter-spacing included.
func (s textStyle) width(txt string) float64 {
	var w fixed.Int26_6
	prev := rune(-1)
	var prevFace font.Face
	for _, r := range txt {
		f := s.faceFor(r)
		if prev >= 0 && f == prevFace {
			w += f.Kern(prev, r)
		}
		a, _ := f.GlyphAdvance(r)
		w += a + fixed.Int26_6(s.spacing*64)
		prev, prevFace = r, f
	}
	return float64(w) / 64
}

// draw writes txt with its baseline at y, starting at x (left aligned).
func (s textStyle) draw(dst *image.RGBA, txt string, x, y float64) {
	src := image.NewUniform(s.color)
	dot := fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6(y * 64)}
	prev := rune(-1)
	var prevFace font.Face
	for _, r := range txt {
		f := s.faceFor(r)
		if prev >= 0 && f == prevFace {
			dot.X += f.Kern(prev, r)
		}
		dr, mask, mp, adv, ok := f.Glyph(dot, r)
		if ok {
			draw.DrawMask(dst, dr, src, image.Point{}, mask, mp, draw.Over)
		}
		dot.X += adv + fixed.Int26_6(s.spacing*64)
		prev, prevFace = r, f
	}
}

func (s textStyle) drawCentered(dst *image.RGBA, txt string, cx, y float64) {
	s.draw(dst, txt, cx-s.width(txt)/2, y)
}

// fit shrinks the style by 2px at a time, down to minSize, until txt fits
// maxW (the prototype's ycFit). It returns the fitted style.
func (s textStyle) fit(txt string, maxW, minSize float64) textStyle {
	for s.size > minSize && s.width(txt) > maxW {
		s.size -= 2
	}
	return s
}

// wrap breaks txt into at most maxLines lines of maxW, ending the last
// with "…" if cut, and trimming any over-long word (the prototype's ycWrap).
func (s textStyle) wrap(txt string, maxW float64, maxLines int) []string {
	var lines []string
	cur := ""
	for _, w := range strings.Fields(txt) {
		t := w
		if cur != "" {
			t = cur + " " + w
		}
		if s.width(t) <= maxW {
			cur = t
		} else {
			if cur != "" {
				lines = append(lines, cur)
			}
			cur = w
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines[maxLines-1] += "…"
	}
	for i, l := range lines {
		for utf8.RuneCountInString(l) > 1 && s.width(l) > maxW {
			r := []rune(l)
			l = string(r[:max(len(r)-2, 0)]) + "…"
		}
		lines[i] = l
	}
	return lines
}

// ---------- shapes ----------

// k is the cubic Bézier handle length for a quarter circle.
const k = 0.5522847498

func rrectPath(z *vector.Rasterizer, x, y, w, h, r float64) {
	r = math.Min(r, math.Min(w, h)/2)
	f := func(v float64) float32 { return float32(v) }
	z.MoveTo(f(x+r), f(y))
	z.LineTo(f(x+w-r), f(y))
	z.CubeTo(f(x+w-r+k*r), f(y), f(x+w), f(y+r-k*r), f(x+w), f(y+r))
	z.LineTo(f(x+w), f(y+h-r))
	z.CubeTo(f(x+w), f(y+h-r+k*r), f(x+w-r+k*r), f(y+h), f(x+w-r), f(y+h))
	z.LineTo(f(x+r), f(y+h))
	z.CubeTo(f(x+r-k*r), f(y+h), f(x), f(y+h-r+k*r), f(x), f(y+h-r))
	z.LineTo(f(x), f(y+r))
	z.CubeTo(f(x), f(y+r-k*r), f(x+r-k*r), f(y), f(x+r), f(y))
	z.ClosePath()
}

// fillShape paints src through the shape built by path.
func fillShape(dst *image.RGBA, src image.Image, sp image.Point, build func(z *vector.Rasterizer)) {
	b := dst.Bounds()
	z := vector.NewRasterizer(b.Dx(), b.Dy())
	build(z)
	z.Draw(dst, b, src, sp)
}

func fillRRect(dst *image.RGBA, x, y, w, h, r float64, c color.Color) {
	fillShape(dst, image.NewUniform(c), image.Point{}, func(z *vector.Rasterizer) { rrectPath(z, x, y, w, h, r) })
}

func fillCircle(dst *image.RGBA, cx, cy, r float64, c color.Color) {
	fillRRect(dst, cx-r, cy-r, 2*r, 2*r, r, c)
}

func fillRect(dst *image.RGBA, x, y, w, h float64, c color.Color) {
	fillRRect(dst, x, y, w, h, 0, c)
}

// strokeRRect draws a rounded rectangle's outline of width lw centred on
// the edge, as canvas stroke() does.
func strokeRRect(dst *image.RGBA, x, y, w, h, r, lw float64, c color.Color) {
	fillShape(dst, image.NewUniform(c), image.Point{}, func(z *vector.Rasterizer) {
		rrectPath(z, x-lw/2, y-lw/2, w+lw, h+lw, r+lw/2)
		// The inner edge, wound the other way, cuts the hole.
		reverseRRectPath(z, x+lw/2, y+lw/2, w-lw, h-lw, r-lw/2)
	})
}

// reverseRRectPath is rrectPath wound anticlockwise.
func reverseRRectPath(z *vector.Rasterizer, x, y, w, h, r float64) {
	r = math.Max(0, math.Min(r, math.Min(w, h)/2))
	f := func(v float64) float32 { return float32(v) }
	z.MoveTo(f(x+r), f(y))
	z.CubeTo(f(x+r-k*r), f(y), f(x), f(y+r-k*r), f(x), f(y+r))
	z.LineTo(f(x), f(y+h-r))
	z.CubeTo(f(x), f(y+h-r+k*r), f(x+r-k*r), f(y+h), f(x+r), f(y+h))
	z.LineTo(f(x+w-r), f(y+h))
	z.CubeTo(f(x+w-r+k*r), f(y+h), f(x+w), f(y+h-r+k*r), f(x+w), f(y+h-r))
	z.LineTo(f(x+w), f(y+r))
	z.CubeTo(f(x+w), f(y+r-k*r), f(x+w-r+k*r), f(y), f(x+w-r), f(y))
	z.ClosePath()
}

// drawCover draws img scaled to cover w×h (cropping the overflow, centred)
// inside a rounded rectangle (the prototype's ycDrawCover).
func drawCover(dst *image.RGBA, img image.Image, x, y, w, h, r float64) {
	ib := img.Bounds()
	ir := float64(ib.Dx()) / float64(ib.Dy())
	br := w / h
	var src image.Rectangle
	if ir > br {
		sw := int(math.Round(float64(ib.Dy()) * br))
		sx := ib.Min.X + (ib.Dx()-sw)/2
		src = image.Rect(sx, ib.Min.Y, sx+sw, ib.Max.Y)
	} else {
		sh := int(math.Round(float64(ib.Dx()) / br))
		sy := ib.Min.Y + (ib.Dy()-sh)/2
		src = image.Rect(ib.Min.X, sy, ib.Max.X, sy+sh)
	}
	scaled := image.NewRGBA(image.Rect(0, 0, int(math.Round(w)), int(math.Round(h))))
	xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), img, src, xdraw.Src, nil)
	fillShape(dst, scaled, image.Pt(-int(math.Round(x)), -int(math.Round(y))), func(z *vector.Rasterizer) { rrectPath(z, x, y, w, h, r) })
}

// withAlpha is c at opacity a (canvas globalAlpha).
func withAlpha(c color.NRGBA, a float64) color.NRGBA {
	c.A = uint8(math.Round(float64(c.A) * a))
	return c
}

type rasterizer = vector.Rasterizer
