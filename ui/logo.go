package ui

import (
	"bytes"
	"image"
	"sync"

	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"golang.org/x/image/draw"
	"golang.org/x/image/webp"

	"github.com/saniyaa61/relic/assets"
)

var (
	logoOnce sync.Once
	logoSrc  image.Image
	// logoOps caches the logo resized to each pixel height drawn. Gio's
	// GPU scaling has no mipmaps, so shrinking the 341px original on the
	// fly would look grainy; we resize once on the CPU instead.
	logoMu  sync.Mutex
	logoOps = map[int]paint.ImageOp{}
)

// logoImage returns the bundled logo at h pixels tall.
func logoImage(h int) (paint.ImageOp, bool) {
	logoOnce.Do(func() {
		img, err := webp.Decode(bytes.NewReader(assets.Logo))
		if err == nil {
			logoSrc = img
		}
	})
	if logoSrc == nil || h <= 0 {
		return paint.ImageOp{}, false
	}
	logoMu.Lock()
	defer logoMu.Unlock()
	if op, ok := logoOps[h]; ok {
		return op, true
	}
	b := logoSrc.Bounds()
	w := max(b.Dx()*h/b.Dy(), 1)
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), logoSrc, b, draw.Src, nil)
	op := paint.NewImageOp(dst)
	logoOps[h] = op
	return op, true
}

// Logo draws the Relic mark at the given height.
func Logo(gtx layout.Context, height unit.Dp) layout.Dimensions {
	img, ok := logoImage(gtx.Dp(height))
	if !ok {
		return layout.Dimensions{}
	}
	img.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: img.Size()}
}
