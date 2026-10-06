package ui

import (
	"bytes"
	"image"
	"image/jpeg"
	"log"
	"os"
	"path/filepath"
	"sync"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"golang.org/x/image/draw"
)

// posterCache holds decoded posters, and copies resized to each box they
// are drawn in. Gio's GPU scaling has no mipmaps, so shrinking a 600px
// poster to a 76px thumbnail on the fly would look grainy; resizing once
// on the CPU keeps them crisp.
type posterCache struct {
	mu      sync.Mutex
	src     map[string]image.Image // nil value: failed to load
	resized map[posterKey]paint.ImageOp
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
	a.posters.src, a.posters.resized = nil, nil
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
