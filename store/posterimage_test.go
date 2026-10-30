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

// withOrientation inserts an EXIF block with the given orientation after
// a JPEG's start marker, in big- or little-endian TIFF order.
func withOrientation(jpg []byte, o uint16, little bool) []byte {
	var tiff []byte
	put16 := func(v uint16) {
		if little {
			tiff = append(tiff, byte(v), byte(v>>8))
		} else {
			tiff = append(tiff, byte(v>>8), byte(v))
		}
	}
	put32 := func(v uint32) { put16(uint16(v >> 16)); put16(uint16(v)) }
	if little {
		tiff = append(tiff, 'I', 'I')
		put16(42)
		tiff = append(tiff, 8, 0, 0, 0)
	} else {
		tiff = append(tiff, 'M', 'M')
		put16(42)
		put32(8)
	}
	put16(1)      // one entry
	put16(0x0112) // orientation
	put16(3)      // SHORT
	if little {
		tiff = append(tiff, 1, 0, 0, 0)
	} else {
		put32(1)
	}
	put16(o)
	put16(0)
	tiff = append(tiff, 0, 0, 0, 0) // no next IFD
	app1 := append([]byte("Exif\x00\x00"), tiff...)
	n := len(app1) + 2
	seg := append([]byte{0xFF, 0xE1, byte(n >> 8), byte(n)}, app1...)
	out := append([]byte{}, jpg[:2]...)
	out = append(out, seg...)
	return append(out, jpg[2:]...)
}

// A camera photo comes out the way it was meant to be seen.
func TestResizePosterOrientation(t *testing.T) {
	// 80 × 40, quadrants: red, green / blue, white.
	red, green := color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255}
	blue, white := color.RGBA{0, 0, 255, 255}, color.RGBA{255, 255, 255, 255}
	src := image.NewRGBA(image.Rect(0, 0, 80, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 80; x++ {
			c := map[[2]bool]color.RGBA{{false, false}: red, {true, false}: green, {false, true}: blue, {true, true}: white}[[2]bool{x >= 40, y >= 20}]
			src.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, src, &jpeg.Options{Quality: 100})
	near := func(a color.Color, b color.RGBA) bool {
		r, g, bl, _ := a.RGBA()
		d := func(x uint32, y uint8) bool { return int(x>>8)-int(y) < 60 && int(y)-int(x>>8) < 60 }
		return d(r, b.R) && d(g, b.G) && d(bl, b.B)
	}
	for o, want := range map[uint16]color.RGBA{1: red, 2: green, 3: white, 4: blue, 5: red, 6: blue, 7: white, 8: green} {
		for _, little := range []bool{false, true} {
			raw := withOrientation(buf.Bytes(), o, little)
			if got := jpegOrientation(raw); got != int(o) {
				t.Errorf("orientation %d (little %v): read %d", o, little, got)
			}
			out, err := ResizePoster(raw)
			if err != nil {
				t.Fatal(err)
			}
			img, _ := jpeg.Decode(bytes.NewReader(out))
			b := img.Bounds()
			wantW, wantH := 80, 40
			if o >= 5 {
				wantW, wantH = 40, 80
			}
			if b.Dx() != wantW || b.Dy() != wantH {
				t.Errorf("orientation %d: %dx%d, want %dx%d", o, b.Dx(), b.Dy(), wantW, wantH)
			}
			if c := img.At(b.Dx()/4, b.Dy()/4); !near(c, want) {
				t.Errorf("orientation %d (little %v): top-left %v, want %v", o, little, c, want)
			}
		}
	}
	if jpegOrientation(buf.Bytes()) != 1 || jpegOrientation([]byte("junk")) != 1 {
		t.Error("no EXIF should read as upright")
	}
}
