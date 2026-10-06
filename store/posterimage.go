package store

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"

	_ "image/gif"
	_ "image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// PosterLongEdge is the largest width or height a stored poster keeps.
const PosterLongEdge = 600

// ResizePoster decodes an image (JPEG, PNG, GIF or WebP), shrinks it so its
// long edge is at most PosterLongEdge, and re-encodes it as JPEG.
// Transparent areas become white.
func ResizePoster(raw []byte) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, errors.New("empty image")
	}
	if long := max(w, h); long > PosterLongEdge {
		w, h = max(1, w*PosterLongEdge/long), max(1, h*PosterLongEdge/long)
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
