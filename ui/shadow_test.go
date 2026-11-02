package ui

import (
	"image"
	"testing"
)

// The entry page's poster shadow (blur 32dp, ~96px on a phone) used to be
// worked out pixel by pixel at full size as the page opened, ~1M pixels.
// It's now worked out small and stretched by the GPU.
func TestWideShadowsAreWorkedOutSmall(t *testing.T) {
	img, pad, full := shadowImage(shadowKey{image.Pt(560, 840), 11, 96, 0.45, rgb(0xEEE8DD)})
	if want := image.Pt(560+2*pad, 840+2*pad); full != want {
		t.Errorf("covers %v, want %v", full, want)
	}
	if sz := img.Size(); sz.X*sz.Y > 20000 {
		t.Errorf("shadow image %v: %d pixels, want a small one", sz, sz.X*sz.Y)
	}
	// A narrow blur stays full size, where its edge needs the detail.
	if img, _, full := shadowImage(shadowKey{image.Pt(90, 126), 10, 6, 0.12, rgb(0xEEE8DD)}); img.Size() != full {
		t.Errorf("narrow shadow %v, want full size %v", img.Size(), full)
	}
}
