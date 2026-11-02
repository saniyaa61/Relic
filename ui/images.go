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
//
// Decoding and resizing take a phone tens of milliseconds a poster, so
// they run on a background worker: until a poster is ready its box shows a
// plain tile, and the frame (and the tap that opened the page) never
// waits. Snapshots and the screen tour set sync to draw them at once.
type posterCache struct {
	mu        sync.Mutex
	sync      bool
	gen       int                    // bumped when posters change on disk
	dims      map[string]image.Point // size from the file header; zero: none
	src       map[string]image.Image // decoded (as *image.RGBA); nil value: failed
	resized   map[posterKey]paint.ImageOp
	backdrops map[backdropKey]paint.ImageOp
	pending   map[any]bool
	jobs      chan func()
}

type posterKey struct {
	name string
	size image.Point
}

// posterState is what a poster lookup found.
type posterState int

const (
	posterNone    posterState = iota // no poster, or it can't be read
	posterLoading                    // being prepared in the background
	posterReady
)

// initLocked makes the maps; c.mu must be held.
func (c *posterCache) initLocked() {
	if c.dims == nil {
		c.dims = map[string]image.Point{}
		c.src = map[string]image.Image{}
		c.resized = map[posterKey]paint.ImageOp{}
		c.backdrops = map[backdropKey]paint.ImageOp{}
		c.pending = map[any]bool{}
	}
}

// posterSize is the poster's pixel size, read from its file header (fast,
// no decoding), or false if there's no readable poster.
func (a *App) posterSize(name string) (image.Point, bool) {
	if name == "" || a.PosterDir == "" {
		return image.Point{}, false
	}
	c := &a.posters
	c.mu.Lock()
	defer c.mu.Unlock()
	c.initLocked()
	if d, ok := c.dims[name]; ok {
		return d, d != (image.Point{})
	}
	var d image.Point
	if f, err := os.Open(filepath.Join(a.PosterDir, filepath.Base(name))); err == nil {
		if cfg, err := jpeg.DecodeConfig(f); err == nil && cfg.Width > 0 && cfg.Height > 0 {
			d = image.Pt(cfg.Width, cfg.Height)
		} else if err != nil {
			log.Printf("relic: poster %s: %v", name, err)
		}
		f.Close()
	}
	c.dims[name] = d
	return d, d != (image.Point{})
}

// decoded returns the decoded poster, decoding it if need be. c.mu must
// not be held: reading and decoding happen without it, so frames never
// wait on them (two callers may both decode once; the first kept wins).
func (a *App) decoded(name string) image.Image {
	c := &a.posters
	c.mu.Lock()
	c.initLocked()
	img, seen := c.src[name]
	gen := c.gen
	c.mu.Unlock()
	if seen {
		return img
	}
	var out image.Image
	data, err := os.ReadFile(filepath.Join(a.PosterDir, filepath.Base(name)))
	if err == nil {
		var img image.Image
		if img, err = jpeg.Decode(bytes.NewReader(data)); err == nil {
			// As RGBA, so resizing takes x/image's fast path.
			rgba := image.NewRGBA(img.Bounds())
			draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
			out = rgba
		}
	}
	if err != nil {
		log.Printf("relic: poster %s: %v", name, err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.initLocked()
	if prev, ok := c.src[name]; ok {
		return prev
	}
	if c.gen == gen {
		c.src[name] = out
	}
	return out
}

// prepare makes the value for key: build works it out (without c.mu, so
// frames don't wait on it) and save stores it (with c.mu held). It runs
// now when sync, else on the worker, and reports whether it's ready.
// c.mu must be held by the caller; with sync it is released while build
// runs.
func (a *App) prepare(key any, build func() any, save func(any)) posterState {
	c := &a.posters
	if c.sync {
		c.mu.Unlock()
		v := build()
		c.mu.Lock()
		c.initLocked()
		save(v)
		return posterReady
	}
	if c.pending[key] {
		return posterLoading
	}
	c.pending[key] = true
	if c.jobs == nil {
		c.jobs = make(chan func(), 256)
		go func() {
			for job := range c.jobs {
				job()
			}
		}()
	}
	gen := c.gen
	job := func() {
		c.mu.Lock()
		stale := c.gen != gen
		c.mu.Unlock()
		if stale {
			return
		}
		v := build()
		c.mu.Lock()
		if c.gen == gen {
			save(v)
			delete(c.pending, key)
		}
		c.mu.Unlock()
		if a.Invalidate != nil {
			a.Invalidate()
		}
	}
	select {
	case c.jobs <- job:
	default:
		go func() { c.jobs <- job }()
	}
	return posterLoading
}

// poster returns the named poster cropped and scaled to fill size px (CSS
// object-fit: cover), or whether it's still loading or missing.
func (a *App) poster(name string, size image.Point) (paint.ImageOp, posterState) {
	if size.X <= 0 || size.Y <= 0 {
		return paint.ImageOp{}, posterNone
	}
	if _, ok := a.posterSize(name); !ok {
		return paint.ImageOp{}, posterNone
	}
	c := &a.posters
	c.mu.Lock()
	defer c.mu.Unlock()
	c.initLocked()
	k := posterKey{name, size}
	if op, ok := c.resized[k]; ok {
		return op, posterReady
	}
	st := a.prepare(k, func() any {
		if src := a.decoded(name); src != nil {
			return paint.NewImageOp(coverScale(src, size))
		}
		return paint.ImageOp{}
	}, func(v any) { c.resized[k] = v.(paint.ImageOp) })
	if op, ok := c.resized[k]; ok && st == posterReady {
		if op == (paint.ImageOp{}) {
			return op, posterNone
		}
		return op, posterReady
	}
	return paint.ImageOp{}, posterLoading
}

// forgetPosters drops cached images, after posters change on disk.
func (a *App) forgetPosters() {
	c := &a.posters
	c.mu.Lock()
	c.gen++
	c.dims, c.src, c.resized, c.backdrops, c.pending = nil, nil, nil, nil, nil
	c.mu.Unlock()
}

// coverScale crops src to size's aspect ratio around its centre and
// scales it to size.
func coverScale(src image.Image, size image.Point) *image.RGBA {
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
	dst := image.NewRGBA(image.Rectangle{Max: size})
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, crop, draw.Src, nil)
	return dst
}

// drawPoster fills a rounded box of size px with the poster, if there is
// one, and reports whether it drew anything. A poster still being
// prepared shows as a plain tile (and counts as drawn, so callers don't
// flash their no-poster look).
func (a *App) drawPoster(gtx layout.Context, name string, size image.Point, radius int) bool {
	img, st := a.poster(name, size)
	switch st {
	case posterNone:
		return false
	case posterLoading:
		rrect(gtx, size, clampRadius(radius, size), a.Theme.Tag)
		return true
	}
	defer clip.UniformRRect(image.Rectangle{Max: size}, clampRadius(radius, size)).Push(gtx.Ops).Pop()
	img.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return true
}

// posterImage returns the decoded poster, decoding it now if need be (for
// the Year card, which is drawn off the UI goroutine).
func (a *App) posterImage(name string) (image.Image, bool) {
	if _, ok := a.posterSize(name); !ok {
		return nil, false
	}
	img := a.decoded(name)
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

// backdrop returns the tiny poster mixed 55% over bg, cached; false while
// it's being prepared or when there's no poster.
func (a *App) backdrop(name string, size image.Point, bg color.NRGBA) (paint.ImageOp, bool) {
	if _, ok := a.posterSize(name); !ok {
		return paint.ImageOp{}, false
	}
	c := &a.posters
	c.mu.Lock()
	defer c.mu.Unlock()
	c.initLocked()
	k := backdropKey{name, size, bg}
	if op, ok := c.backdrops[k]; ok {
		return op, op != (paint.ImageOp{})
	}
	a.prepare(k, func() any {
		src := a.decoded(name)
		if src == nil {
			return paint.ImageOp{}
		}
		img := coverScale(src, size)
		mix := func(p, q uint8) uint8 { return uint8(0.55*float32(p) + 0.45*float32(q) + 0.5) }
		for i := 0; i < len(img.Pix); i += 4 {
			img.Pix[i] = mix(img.Pix[i], bg.R)
			img.Pix[i+1] = mix(img.Pix[i+1], bg.G)
			img.Pix[i+2] = mix(img.Pix[i+2], bg.B)
			img.Pix[i+3] = 255
		}
		return paint.NewImageOp(img)
	}, func(v any) { c.backdrops[k] = v.(paint.ImageOp) })
	op, ok := c.backdrops[k]
	return op, ok && op != (paint.ImageOp{})
}
