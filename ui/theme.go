// Package ui holds Relic's screens and components, drawn with Gio.
package ui

import (
	"image/color"

	"gioui.org/text"
)

// Palette is one theme variant. Fields match the keys in
// docs/design-tokens.json (btn-text → BtnText).
type Palette struct {
	Bg, Surface, Card, Text, Muted, Accent, Accent2, Border, Tag, BtnText, Timeline color.NRGBA
}

// LinenLight and LinenDark are the default theme, copied from
// docs/design-tokens.json. theme_test.go keeps them in step with that file.
var (
	LinenLight = Palette{
		Bg: rgb(0xF7F3EC), Surface: rgb(0xFDFAF5), Card: rgb(0xFFFFFF),
		Text: rgb(0x2C2016), Muted: rgb(0x8B7355), Accent: rgb(0x7A5C3A),
		Accent2: rgb(0xC4956A), Border: rgb(0xE2D9C8), Tag: rgb(0xEDE4D3),
		BtnText: rgb(0xFFF8F0), Timeline: rgb(0xF0EAE0),
	}
	LinenDark = Palette{
		Bg: rgb(0x18140D), Surface: rgb(0x221C12), Card: rgb(0x2C2418),
		Text: rgb(0xF0EAE0), Muted: rgb(0x9A8468), Accent: rgb(0xD4A870),
		Accent2: rgb(0xE8C090), Border: rgb(0x3A3020), Tag: rgb(0x2A2215),
		BtnText: rgb(0x1A1008), Timeline: rgb(0x1E1A10),
	}
)

// Theme is what every component draws with: the active palette and a text
// shaper loaded with the bundled fonts only.
type Theme struct {
	Palette
	Shaper *text.Shaper
}

// NewTheme loads the bundled fonts and returns a theme using p.
func NewTheme(p Palette) (*Theme, error) {
	faces, err := LoadFonts()
	if err != nil {
		return nil, err
	}
	return &Theme{
		Palette: p,
		Shaper:  text.NewShaper(text.NoSystemFonts(), text.WithCollection(faces)),
	}, nil
}

func rgb(v uint32) color.NRGBA {
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xFF}
}

// withAlpha returns c at the given opacity (0–1), like CSS opacity on a colour.
func withAlpha(c color.NRGBA, a float32) color.NRGBA {
	c.A = uint8(float32(c.A)*a + 0.5)
	return c
}
