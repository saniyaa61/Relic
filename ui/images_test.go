package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// On phones a poster is decoded and resized off the UI goroutine: the
// first lookup says it's loading (and its box shows a plain tile), the
// worker asks for a redraw, and the next lookup has it. A missing poster
// is reported at once.
func TestPosterLoadsInBackground(t *testing.T) {
	a, _ := testApp(t, nil)
	a.posters.sync = false
	a.PosterDir = t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 300, 450))
	for i := range img.Pix {
		img.Pix[i] = 200
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, nil)
	os.WriteFile(filepath.Join(a.PosterDir, "p.jpg"), buf.Bytes(), 0o600)
	redrawn := make(chan bool, 4)
	a.Invalidate = func() { redrawn <- true }

	if _, st := a.poster("gone.jpg", image.Pt(60, 90)); st != posterNone {
		t.Errorf("a missing poster: state %v", st)
	}
	if d, ok := a.posterSize("p.jpg"); !ok || d != image.Pt(300, 450) {
		t.Errorf("size from the header: %v %v", d, ok)
	}
	if _, st := a.poster("p.jpg", image.Pt(60, 90)); st != posterLoading {
		t.Fatalf("first lookup: state %v, want loading", st)
	}
	select {
	case <-redrawn:
	case <-time.After(5 * time.Second):
		t.Fatal("the worker never asked for a redraw")
	}
	op, st := a.poster("p.jpg", image.Pt(60, 90))
	if st != posterReady || op.Size() != image.Pt(60, 90) {
		t.Fatalf("after the worker: state %v, size %v", st, op.Size())
	}
	if c := color.RGBAModel.Convert(coverScale(img, image.Pt(10, 10)).At(5, 5)).(color.RGBA); c.R < 190 {
		t.Errorf("resized colour %v", c)
	}
	// Posters changing on disk drop everything, including work in flight.
	a.forgetPosters()
	if _, st := a.poster("p.jpg", image.Pt(60, 90)); st != posterLoading {
		t.Errorf("after forgetPosters: state %v, want loading again", st)
	}
}
