package ui

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"testing"

	"gioui.org/font"
)

// The Go palettes must match docs/design-tokens.json exactly.
func TestLinenMatchesDesignTokens(t *testing.T) {
	data, err := os.ReadFile("../docs/design-tokens.json")
	if err != nil {
		t.Fatal(err)
	}
	var tokens struct {
		Themes map[string]map[string]map[string]string `json:"themes"`
	}
	if err := json.Unmarshal(data, &tokens); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		variant string
		p       Palette
	}{
		{"light", LinenLight},
		{"dark", LinenDark},
	} {
		want := tokens.Themes["linen"][tc.variant]
		got := map[string]color.NRGBA{
			"bg": tc.p.Bg, "surface": tc.p.Surface, "card": tc.p.Card,
			"text": tc.p.Text, "muted": tc.p.Muted, "accent": tc.p.Accent,
			"accent2": tc.p.Accent2, "border": tc.p.Border, "tag": tc.p.Tag,
			"btn-text": tc.p.BtnText, "timeline": tc.p.Timeline,
		}
		if len(want) != len(got) {
			t.Errorf("linen %s: tokens have %d colours, Palette has %d", tc.variant, len(want), len(got))
		}
		for key, hex := range want {
			c := got[key]
			if g := fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B); g != hex {
				t.Errorf("linen %s %s = %s, want %s", tc.variant, key, g, hex)
			}
		}
	}
}

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
