package ui

import (
	"bytes"
	"image"
	"image/png"
	"sync"

	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"golang.org/x/image/draw"

	"github.com/saniyaa61/relic/assets"
)

var (
	emojiMu  sync.Mutex
	emojiSrc = map[string]image.Image{}
	emojiOps = map[string]map[int]paint.ImageOp{}
)

// Emoji draws a bundled colour emoji ("1f525" for 🔥) at size dp tall,
// resized once per size like the logo.
func Emoji(gtx layout.Context, code string, size unit.Dp) layout.Dimensions {
	h := gtx.Dp(size)
	emojiMu.Lock()
	defer emojiMu.Unlock()
	src, ok := emojiSrc[code]
	if !ok {
		data, err := assets.Emoji.ReadFile("emoji/" + code + ".png")
		if err == nil {
			src, err = png.Decode(bytes.NewReader(data))
		}
		if err != nil {
			src = nil
		}
		emojiSrc[code] = src
	}
	if src == nil || h <= 0 {
		return layout.Dimensions{}
	}
	if emojiOps[code] == nil {
		emojiOps[code] = map[int]paint.ImageOp{}
	}
	op, ok := emojiOps[code][h]
	if !ok {
		b := src.Bounds()
		w := max(b.Dx()*h/b.Dy(), 1)
		dst := image.NewNRGBA(image.Rect(0, 0, w, h))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
		op = paint.NewImageOp(dst)
		emojiOps[code][h] = op
	}
	op.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: op.Size()}
}
