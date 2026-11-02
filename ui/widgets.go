package ui

import (
	"image"
	"image/color"
	"math"
	"sync"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
)

// pressable draws w and dims it while pressed: the touch version of the
// prototype's hover opacity (".add-btn:hover{opacity:.88}").
func pressable(gtx layout.Context, c *widget.Clickable, w layout.Widget) layout.Dimensions {
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if c.Pressed() || recentlyReleased(gtx, c) {
			defer paint.PushOpacity(gtx.Ops, 0.88).Pop()
		}
		return w(gtx)
	})
}

// pressFlash is how long a press still shows after the finger lifts. A
// quick tap on a phone often starts and ends within one frame, so without
// it the button never looked pressed and the tap seemed ignored.
const pressFlash = 120 * time.Millisecond

// showsPressed reports whether c should look pressed: held down, or
// released within pressFlash.
func showsPressed(gtx layout.Context, c *widget.Clickable) bool {
	return c.Pressed() || recentlyReleased(gtx, c)
}

func recentlyReleased(gtx layout.Context, c *widget.Clickable) bool {
	for _, h := range c.History() {
		if h.End.IsZero() || h.Cancelled || gtx.Now.IsZero() {
			continue
		}
		if until := h.End.Add(pressFlash); gtx.Now.Before(until) {
			gtx.Execute(op.InvalidateCmd{At: until})
			return true
		}
	}
	return false
}

// rrect fills a rounded rectangle of size sz at the origin.
func rrect(gtx layout.Context, sz image.Point, radius int, c color.NRGBA) {
	paint.FillShape(gtx.Ops, c, clip.UniformRRect(image.Rectangle{Max: sz}, clampRadius(radius, sz)).Op(gtx.Ops))
}

// boxShadow approximates CSS box-shadow: 0 offsetY blur rgba(0,0,0,alpha)
// behind a rounded rectangle of size sz, over a page of colour bg. Gio has
// no blur, so the shadow is worked out once on the CPU (the Gaussian
// fall-off a CSS blur has: full strength well inside the edge, half at the
// edge, fading to nothing a blur's width outside) and cached as an image,
// which the GPU then draws in one go. (Drawing it as 24 stacked rings each
// frame made pages with many shadows, like the Library, slow on phones.)
func boxShadow(gtx layout.Context, sz image.Point, radius int, offsetY, blur unit.Dp, alpha float32, bg color.NRGBA) {
	b := gtx.Dp(blur)
	if b <= 0 || sz.X <= 0 || sz.Y <= 0 {
		return
	}
	img, pad, full := shadowImage(shadowKey{sz, clampRadius(radius, sz), b, alpha, bg})
	defer op.Offset(image.Pt(-pad, -pad+gtx.Dp(offsetY))).Push(gtx.Ops).Pop()
	defer clip.Rect{Max: full}.Push(gtx.Ops).Pop()
	if small := img.Size(); small != full {
		img.Filter = paint.FilterLinear
		k := f32.Pt(float32(full.X)/float32(small.X), float32(full.Y)/float32(small.Y))
		defer op.Affine(f32.AffineId().Scale(f32.Point{}, k)).Push(gtx.Ops).Pop()
	}
	img.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

type shadowKey struct {
	size   image.Point
	radius int
	blur   int
	alpha  float32
	bg     color.NRGBA
}

var (
	shadowMu    sync.Mutex
	shadowCache = map[shadowKey]paint.ImageOp{}
)

// shadowImage is the shadow for k, with pad px of fall-off on every side,
// and the size it covers. A shadow is soft all over, so a wide one is
// worked out at a fraction of its size (sampling it at least every blur/8
// px) and stretched back smoothly by the GPU: the entry page's poster
// shadow took ~50 ms at full size on a desktop, several times that on a
// phone, as the page opened.
func shadowImage(k shadowKey) (paint.ImageOp, int, image.Point) {
	b := float64(k.blur)
	pad := int(math.Ceil(1.5 * b))
	w, h := k.size.X+2*pad, k.size.Y+2*pad
	full := image.Pt(w, h)
	shadowMu.Lock()
	defer shadowMu.Unlock()
	if op, ok := shadowCache[k]; ok {
		return op, pad, full
	}
	if len(shadowCache) > 300 {
		clear(shadowCache)
	}
	sigma := b / 2
	// The colour for each strength, mixed for bg as shade does.
	var levels [256]color.NRGBA
	for i := range levels {
		levels[i] = shade(k.alpha*float32(i)/255, k.bg)
	}
	step := max(1, k.blur/8)
	sw, sh := (w+step-1)/step, (h+step-1)/step
	sx, sy := float64(w)/float64(sw), float64(h)/float64(sh)
	img := image.NewNRGBA(image.Rect(0, 0, sw, sh))
	hx, hy := float64(k.size.X)/2, float64(k.size.Y)/2
	r := float64(k.radius)
	for y := 0; y < sh; y++ {
		for x := 0; x < sw; x++ {
			// Distance outside the rounded rectangle (negative inside),
			// at this pixel's centre in full-size pixels.
			qx := math.Abs((float64(x)+0.5)*sx-float64(pad)-hx) - (hx - r)
			qy := math.Abs((float64(y)+0.5)*sy-float64(pad)-hy) - (hy - r)
			d := math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
			cover := 0.5 * math.Erfc(d/(sigma*math.Sqrt2))
			c := levels[int(math.Round(cover*255))]
			i := img.PixOffset(x, y)
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
		}
	}
	op := paint.NewImageOp(img)
	shadowCache[k] = op
	return op, pad, full
}

// IconButton is the prototype's .ibtn: a 34dp round button with an 18dp
// accent line icon, shaded with the border colour while pressed.
type IconButton struct {
	Click widget.Clickable
}

// Layout draws the button. It looks 34dp but answers taps over 48dp around it (the size
// phones recommend for a fingertip; owner, 2026-11-02: the header's gear
// and back arrows lagged, as taps a little off the small circle missed),
// and shows pressed for a moment after a quick tap, like pressable.
func (b *IconButton) Layout(gtx layout.Context, th *Theme, ic *Icon) layout.Dimensions {
	sz, hit := gtx.Dp(34), gtx.Dp(48)
	pad := (hit - sz) / 2
	st := op.Offset(image.Pt(-pad, -pad)).Push(gtx.Ops)
	b.Click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		inner := op.Offset(image.Pt(pad, pad)).Push(gtx.Ops)
		if b.Click.Pressed() || recentlyReleased(gtx, &b.Click) {
			paint.FillShape(gtx.Ops, th.Border, clip.Ellipse{Max: image.Pt(sz, sz)}.Op(gtx.Ops))
		}
		off := (sz - gtx.Dp(18)) / 2
		icon := op.Offset(image.Pt(off, off)).Push(gtx.Ops)
		ic.Layout(gtx, 18, 1.6, th.Accent)
		icon.Pop()
		inner.Pop()
		return layout.Dimensions{Size: image.Pt(hit, hit)}
	})
	st.Pop()
	return layout.Dimensions{Size: image.Pt(sz, sz)}
}

// Button is a filled text button. Prototype styles: .rmodal-btn (accent,
// Playfair italic), .rmodal-btn.danger and .rmodal-cancel.
type Button struct {
	Click widget.Clickable
}

// ButtonStyle picks the look of a Button.
type ButtonStyle int

const (
	ButtonAccent ButtonStyle = iota // accent fill, Playfair italic
	ButtonDanger                    // red fill, Playfair italic
	ButtonQuiet                     // tag fill with border, DM Sans (Cancel)
)

// dangerRed is the prototype's .rmodal-btn.danger colour, the same in
// every theme.
var dangerRed = rgb(0xE05555)

func (b *Button) Layout(gtx layout.Context, th *Theme, style ButtonStyle, label string) layout.Dimensions {
	bg, fg, border := th.Accent, th.BtnText, color.NRGBA{}
	txt := Text{Font: font.Font{Typeface: Display, Style: font.Italic}, Size: 14}
	// The filled buttons have no border in the prototype, but card draws
	// a 1dp one (in the fill colour), so their padding is 1dp less than
	// the prototype's 12px.
	pad := layout.UniformInset(11)
	switch style {
	case ButtonDanger:
		bg, fg = dangerRed, rgb(0xFFFFFF)
	case ButtonQuiet:
		bg, fg, border = th.Tag, th.Muted, th.Border
		txt = Text{Font: font.Font{Typeface: Sans}, Size: 13}
		pad = layout.Inset{Top: 12, Bottom: 12, Left: 17, Right: 17}
	}
	txt.Color = fg
	return pressable(gtx, &b.Click, func(gtx layout.Context) layout.Dimensions {
		if border.A == 0 {
			border = bg
		}
		return card(gtx, bg, border, 12, pad, 0, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return txt.Layout(gtx, th, label)
			})
		})
	})
}

// Input is a text field: the prototype's .rmodal-input, .search-input,
// .finput and (multi-line) .ftarea — a filled box with a thin border that
// turns accent on focus.
type Input struct {
	Editor widget.Editor
}

// InputStyle sets an Input's look; the zero value is unusable.
type InputStyle struct {
	Bg          color.NRGBA
	Radius      unit.Dp
	Pad         layout.Inset
	Size        unit.Sp
	Font        font.Font // zero: DM Sans
	Placeholder string
	// PlaceholderAlpha is the placeholder's opacity (CSS ::placeholder).
	PlaceholderAlpha float32
	// Multiline makes a wrapping text area (.ftarea); MinHeight is its
	// smallest height and LineHeight its CSS line-height.
	Multiline  bool
	MinHeight  unit.Dp
	LineHeight float32
	// Center centres the text and placeholder (single line only).
	Center bool
}

func (in *Input) Layout(gtx layout.Context, th *Theme, s InputStyle) layout.Dimensions {
	in.Editor.SingleLine = !s.Multiline
	if s.Center {
		in.Editor.Alignment = textMiddle
	}
	f := s.Font
	if f.Typeface == "" {
		f.Typeface = Sans
	}
	if s.LineHeight > 0 {
		in.Editor.LineHeight = unit.Sp(float32(s.Size) * s.LineHeight)
		in.Editor.LineHeightScale = 1
	}
	border := th.Border
	if gtx.Focused(&in.Editor) {
		border = th.Accent
	}
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	if s.MinHeight > 0 {
		gtx.Constraints.Min.Y = max(gtx.Constraints.Min.Y, gtx.Dp(s.MinHeight))
	}
	return card(gtx, s.Bg, border, s.Radius, s.Pad, 0, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		if in.Editor.Len() == 0 && s.Placeholder != "" {
			rec := op.Record(gtx.Ops)
			pgtx := gtx
			pgtx.Constraints.Min = image.Point{}
			if s.Multiline {
				Paragraph{Font: f, Size: s.Size, LineHeight: s.LineHeight, Color: withAlpha(th.Muted, s.PlaceholderAlpha)}.Layout(pgtx, th, s.Placeholder)
			} else {
				d := Text{Font: f, Size: s.Size, MaxLines: 1, Color: withAlpha(th.Muted, s.PlaceholderAlpha)}.Layout(pgtx, th, s.Placeholder)
				if s.Center {
					ph := rec.Stop()
					rec = op.Record(gtx.Ops)
					st := op.Offset(image.Pt((gtx.Constraints.Max.X-d.Size.X)/2, 0)).Push(gtx.Ops)
					ph.Add(gtx.Ops)
					st.Pop()
				}
			}
			ph := rec.Stop()
			defer ph.Add(gtx.Ops)
		}
		return in.Editor.Layout(gtx, th.Shaper, f, s.Size, colorOp(gtx, th.Text), colorOp(gtx, withAlpha(th.Accent, 0.25)))
	})
}

// Focus moves keyboard focus (and the soft keyboard) to the input.
func (in *Input) Focus(gtx layout.Context) {
	gtx.Execute(key.FocusCmd{Tag: &in.Editor})
}
