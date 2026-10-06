package importer

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"net/url"
	"strings"

	_ "image/gif"
	_ "image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// PosterLongEdge is the largest width or height a stored poster keeps.
const PosterLongEdge = 600

// resizePoster decodes a data: URL image, shrinks it so its long edge is at
// most PosterLongEdge, and re-encodes it as JPEG. Transparent areas become white.
func resizePoster(dataURL string) ([]byte, error) {
	raw, err := decodeDataURL(dataURL)
	if err != nil {
		return nil, err
	}
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

func decodeDataURL(s string) ([]byte, error) {
	if !strings.HasPrefix(s, "data:") {
		return nil, errors.New("not an embedded image")
	}
	meta, data, ok := strings.Cut(s[len("data:"):], ",")
	if !ok {
		return nil, errors.New("malformed data URL")
	}
	if strings.HasSuffix(meta, ";base64") {
		return base64.StdEncoding.DecodeString(strings.TrimSpace(data))
	}
	d, err := url.PathUnescape(data)
	return []byte(d), err
}
