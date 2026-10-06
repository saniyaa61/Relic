// Package ui holds Relic's screens and components, drawn with Gio.
package ui

import (
	"gioui.org/text"
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
