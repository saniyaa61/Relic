package ui

import (
	"io/fs"
	"math"
	"path"
	"testing"

	xfont "golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/saniyaa61/relic/assets"

	"gioui.org/font"
)

// Every font weight and style the prototype uses must be bundled.
func TestBundledFonts(t *testing.T) {
	faces, err := LoadFonts()
	if err != nil {
		t.Fatal(err)
	}
	have := map[font.Font]bool{}
	for _, f := range faces {
		have[f.Font] = true
	}
	want := []font.Font{
		{Typeface: Display, Weight: font.Normal},
		{Typeface: Display, Weight: font.Medium},
		{Typeface: Display, Weight: font.SemiBold},
		{Typeface: Display, Weight: font.Normal, Style: font.Italic},
		{Typeface: Display, Weight: font.Medium, Style: font.Italic},
		{Typeface: Serif, Weight: font.Normal},
		{Typeface: Serif, Weight: font.Medium},
		{Typeface: Serif, Weight: font.Normal, Style: font.Italic},
		{Typeface: Sans, Weight: font.Light},
		{Typeface: Sans, Weight: font.Normal},
		{Typeface: Sans, Weight: font.Medium},
		{Typeface: Symbols, Weight: font.Normal},
		{Typeface: Symbols, Weight: font.Normal, Style: font.Italic}, // same file, for italic text
	}
	for _, f := range want {
		if !have[f] {
			t.Errorf("missing font %s weight %v style %v", f.Typeface, f.Weight, f.Style)
		}
	}
	if len(faces) != len(want) {
		t.Errorf("bundled %d faces, want %d", len(faces), len(want))
	}
}

// normalLineHeight must match each family's vertical metrics, and the
// symbols font must never be taller than the text fonts it fills in for.
func TestFontLineMetrics(t *testing.T) {
	files, err := fs.Glob(assets.Fonts, "fonts/*.ttf")
	if err != nil {
		t.Fatal(err)
	}
	minAsc, minDesc := fixed.Int26_6(math.MaxInt32), fixed.Int26_6(math.MaxInt32)
	var symAsc, symDesc fixed.Int26_6
	for _, name := range files {
		data, _ := assets.Fonts.ReadFile(name)
		f, err := sfnt.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		m, err := f.Metrics(nil, fixed.I(1000), xfont.HintingNone)
		if err != nil {
			t.Fatal(err)
		}
		ff, _ := fontFromFilename(path.Base(name))
		if ff.Typeface == Symbols {
			symAsc, symDesc = m.Ascent, m.Descent
			continue
		}
		minAsc, minDesc = min(minAsc, m.Ascent), min(minDesc, m.Descent)
		got := normalLineHeight[ff.Typeface]
		want := float64(m.Height) / 64 / 1000
		if math.Abs(float64(got)-want) > 0.001 {
			t.Errorf("%s: normal line height %.3f, font says %.3f", name, got, want)
		}
	}
	if symAsc > minAsc || symDesc > minDesc {
		t.Errorf("symbols font ascent/descent %v/%v exceed the text fonts' %v/%v", symAsc, symDesc, minAsc, minDesc)
	}
}
