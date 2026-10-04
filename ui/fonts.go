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
)

// Bundled font files are named Family-WeightStyle.ttf. We set the family,
// weight and style from the filename rather than trusting each file's
// internal names, which differ between static font builds.
var fontFamilies = map[string]font.Typeface{
	"PlayfairDisplay": Display,
	"Lora":            Serif,
	"DMSans":          Sans,
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
