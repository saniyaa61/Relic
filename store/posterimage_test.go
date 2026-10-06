package store

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestResizePoster(t *testing.T) {
	for _, tc := range []struct {
		w, h, wantW, wantH int
	}{
		{1200, 800, 600, 400}, // big: long edge to 600
		{300, 900, 200, 600},  // tall
		{200, 300, 200, 300},  // small: kept
	} {
		src := image.NewNRGBA(image.Rect(0, 0, tc.w, tc.h))
		src.Set(0, 0, color.NRGBA{A: 0}) // transparent corner becomes white
		var buf bytes.Buffer
		png.Encode(&buf, src)
		out, err := ResizePoster(buf.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		img, err := jpeg.Decode(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("not a JPEG: %v", err)
		}
		if b := img.Bounds(); b.Dx() != tc.wantW || b.Dy() != tc.wantH {
			t.Errorf("%dx%d → %dx%d, want %dx%d", tc.w, tc.h, b.Dx(), b.Dy(), tc.wantW, tc.wantH)
		}
	}
	if _, err := ResizePoster([]byte("not an image")); err == nil {
		t.Error("want an error for junk")
	}
}
