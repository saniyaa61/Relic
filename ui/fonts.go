package ui

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"gioui.org/font"
	"gioui.org/font/opentype"

	"github.com/saniyaa61/relic/assets"
)

// The three bundled typefaces, as the prototype uses them.
const (
	Display font.Typeface = "Playfair Display" // headings, numbers, titles
	Serif   font.Typeface = "Lora"             // italic sub-lines and prose
	Sans    font.Typeface = "DM Sans"          // UI labels
	// Symbols fills in ★ ☆ ♥ ✓ ✕ ✦ ✧, which the three fonts above lack.
	// Browsers borrow these from a system font; we bundle a small subset
	// of Noto Sans Symbols 2 instead, and the shaper falls back to it.
	Symbols font.Typeface = "Noto Sans Symbols 2"
)

// normalLineHeight is CSS "line-height: normal" for each typeface: the
// font's ascent + descent + line gap, as a multiple of the font size.
// fonts_test.go checks these against the font files.
var normalLineHeight = map[font.Typeface]float32{
	Display: 1.333,
	Serif:   1.280,
	Sans:    1.302,
}

// Bundled font files are named Family-WeightStyle.ttf. We set the family,
// weight and style from the filename rather than trusting each file's
// internal names, which differ between static font builds.
var fontFamilies = map[string]font.Typeface{
	"PlayfairDisplay":  Display,
	"Lora":             Serif,
	"DMSans":           Sans,
	"NotoSansSymbols2": Symbols,
}

var fontWeights = map[string]font.Weight{
	"Light":    font.Light,
	"Regular":  font.Normal,
	"":         font.Normal, // "Lora-Italic" is regular weight
	"Medium":   font.Medium,
	"SemiBold": font.SemiBold,
}

// LoadFonts parses every bundled font into a Gio font collection.
func LoadFonts() ([]font.FontFace, error) {
	files, err := fs.Glob(assets.Fonts, "fonts/*.ttf")
	if err != nil {
		return nil, err
	}
	var faces []font.FontFace
	for _, name := range files {
		f, err := fontFromFilename(path.Base(name))
		if err != nil {
			return nil, err
		}
		data, err := assets.Fonts.ReadFile(name)
		if err != nil {
			return nil, err
		}
		face, err := opentype.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		faces = append(faces, font.FontFace{Font: f, Face: face})
	}
	return faces, nil
}

func fontFromFilename(name string) (font.Font, error) {
	family, variant, ok := strings.Cut(strings.TrimSuffix(name, ".ttf"), "-")
	if !ok {
		return font.Font{}, fmt.Errorf("font %q: want Family-Variant.ttf", name)
	}
	typeface, ok := fontFamilies[family]
	if !ok {
		return font.Font{}, fmt.Errorf("font %q: unknown family %q", name, family)
	}
	f := font.Font{Typeface: typeface}
	if w, found := strings.CutSuffix(variant, "Italic"); found {
		f.Style = font.Italic
		variant = w
	}
	weight, ok := fontWeights[variant]
	if !ok {
		return font.Font{}, fmt.Errorf("font %q: unknown weight %q", name, variant)
	}
	f.Weight = weight
	return f, nil
}
