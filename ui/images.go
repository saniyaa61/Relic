package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"os"
	"path/filepath"
	"sync"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"golang.org/x/image/draw"
)

// posterCache holds decoded posters, and copies resized to each box they
// are drawn in. Gio's GPU scaling has no mipmaps, so shrinking a 600px
// poster to a 76px thumbnail on the fly would look grainy; resizing once
// on the CPU keeps them crisp.
type posterCache struct {
	mu        sync.Mutex
	src       map[string]image.Image // nil value: failed to load
	resized   map[posterKey]paint.ImageOp
	backdrops map[backdropKey]paint.ImageOp
}

type posterKey struct {
	name string
	size image.Point
}

// poster returns the named poster cropped and scaled to fill size px (CSS
// object-fit: cover), or false if there is none.
func (a *App) poster(name string, size image.Point) (paint.ImageOp, bool) {
	if name == "" || a.PosterDir == "" || size.X <= 0 || size.Y <= 0 {
		return paint.ImageOp{}, false
	}
	c := &a.posters
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.src == nil {
		c.src = map[string]image.Image{}
		c.resized = map[posterKey]paint.ImageOp{}
	}
	k := posterKey{name, size}
	if op, ok := c.resized[k]; ok {
		return op, true
	}
	src, seen := c.src[name]
	if !seen {
		data, err := os.ReadFile(filepath.Join(a.PosterDir, filepath.Base(name)))
		if err == nil {
			src, err = jpeg.Decode(bytes.NewReader(data))
		}
		if err != nil {
			log.Printf("relic: poster %s: %v", name, err)
		}
		c.src[name] = src
	}
	if src == nil {
		return paint.ImageOp{}, false
	}
	op := paint.NewImageOp(coverScale(src, size))
	c.resized[k] = op
	return op, true
}

// forgetPosters drops cached images, after posters change on disk.
func (a *App) forgetPosters() {
	a.posters.mu.Lock()
	a.posters.src, a.posters.resized, a.posters.backdrops = nil, nil, nil
	a.posters.mu.Unlock()
}

// coverScale crops src to size's aspect ratio around its centre and
// scales it to size.
func coverScale(src image.Image, size image.Point) image.Image {
	b := src.Bounds()
	crop := b
	if b.Dx()*size.Y > b.Dy()*size.X { // wider than the box: trim the sides
		w := b.Dy() * size.X / size.Y
		crop.Min.X = b.Min.X + (b.Dx()-w)/2
		crop.Max.X = crop.Min.X + w
	} else {
		h := b.Dx() * size.Y / size.X
		crop.Min.Y = b.Min.Y + (b.Dy()-h)/2
		crop.Max.Y = crop.Min.Y + h
	}
	dst := image.NewNRGBA(image.Rectangle{Max: size})
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, crop, draw.Src, nil)
	return dst
}

// drawPoster fills a rounded box of size px with the poster, if there is
// one, and reports whether it drew anything.
func (a *App) drawPoster(gtx layout.Context, name string, size image.Point, radius int) bool {
	img, ok := a.poster(name, size)
	if !ok {
		return false
	}
	defer clip.UniformRRect(image.Rectangle{Max: size}, clampRadius(radius, size)).Push(gtx.Ops).Pop()
	img.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return true
}

// posterImage returns the decoded poster, for its size.
func (a *App) posterImage(name string) (image.Image, bool) {
	if _, ok := a.poster(name, image.Pt(1, 1)); !ok {
		return nil, false
	}
	a.posters.mu.Lock()
	defer a.posters.mu.Unlock()
	img := a.posters.src[name]
	return img, img != nil
}

// drawPosterContain draws the poster into a box of its own proportions.
func (a *App) drawPosterContain(gtx layout.Context, name string, size image.Point, radius int) bool {
	return a.drawPoster(gtx, name, size, radius)
}

// blurredBackdrop is the prototype hero's backdrop: the poster filling the
// area, blurred (CSS blur(18px), scale 1.1) at 55% opacity over bg. Gio has
// no blur, so we shrink the poster to a few pixels and let the GPU stretch
// it back smoothly, which looks the same at this strength. The 55% mix
// with bg is done here in sRGB, as a browser would (Gio would mix in
// linear light, which looks greyer).
func blurredBackdrop(gtx layout.Context, a *App, name string, size image.Point, bg color.NRGBA) {
	small := image.Pt(max(size.X/24, 2), max(size.Y/24, 2))
	img, ok := a.backdrop(name, small, bg)
	if !ok {
		return
	}
	sx := float32(size.X) * 1.1 / float32(small.X)
	sy := float32(size.Y) * 1.1 / float32(small.Y)
	off := f32.Pt(-float32(size.X)*0.05, -float32(size.Y)*0.05)
	defer op.Affine(f32.AffineId().Scale(f32.Point{}, f32.Pt(sx, sy)).Offset(off)).Push(gtx.Ops).Pop()
	img.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

type backdropKey struct {
	name string
	size image.Point
	bg   color.NRGBA
}

// backdrop returns the tiny poster mixed 55% over bg, cached.
func (a *App) backdrop(name string, size image.Point, bg color.NRGBA) (paint.ImageOp, bool) {
	if _, ok := a.poster(name, size); !ok {
		return paint.ImageOp{}, false
	}
	c := &a.posters
	c.mu.Lock()
	defer c.mu.Unlock()
	k := backdropKey{name, size, bg}
	if op, ok := c.backdrops[k]; ok {
		return op, true
	}
	src := coverScale(c.src[name], size).(*image.NRGBA)
	mix := func(p, q uint8) uint8 { return uint8(0.55*float32(p) + 0.45*float32(q) + 0.5) }
	for i := 0; i < len(src.Pix); i += 4 {
		src.Pix[i] = mix(src.Pix[i], bg.R)
		src.Pix[i+1] = mix(src.Pix[i+1], bg.G)
		src.Pix[i+2] = mix(src.Pix[i+2], bg.B)
		src.Pix[i+3] = 255
	}
	if c.backdrops == nil {
		c.backdrops = map[backdropKey]paint.ImageOp{}
	}
	op := paint.NewImageOp(src)
	c.backdrops[k] = op
	return op, true
}
