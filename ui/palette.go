package ui

import (
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/saniyaa61/relic/core"
)

// Palette is one theme variant. Fields match the keys in
// docs/design-tokens.json (btn-text → BtnText).
type Palette struct {
	Bg, Surface, Card, Text, Muted, Accent, Accent2, Border, Tag, BtnText, Timeline color.NRGBA
}

// ThemeNames lists the built-in themes in the prototype's picker order.
// "custom" is the seventh choice, built from two colours by CustomPalette.
var ThemeNames = []string{"linen", "midnight", "blush", "forest", "rose", "slate"}

type themeKey struct{ name, mode string }

// themes is copied from docs/design-tokens.json; palette_test.go keeps the
// two in step.
var themes = map[themeKey]Palette{
	{"linen", "light"}:    {Bg: rgb(0xF7F3EC), Surface: rgb(0xFDFAF5), Card: rgb(0xFFFFFF), Text: rgb(0x2C2016), Muted: rgb(0x8B7355), Accent: rgb(0x7A5C3A), Accent2: rgb(0xC4956A), Border: rgb(0xE2D9C8), Tag: rgb(0xEDE4D3), BtnText: rgb(0xFFF8F0), Timeline: rgb(0xF0EAE0)},
	{"linen", "dark"}:     {Bg: rgb(0x18140D), Surface: rgb(0x221C12), Card: rgb(0x2C2418), Text: rgb(0xF0EAE0), Muted: rgb(0x9A8468), Accent: rgb(0xD4A870), Accent2: rgb(0xE8C090), Border: rgb(0x3A3020), Tag: rgb(0x2A2215), BtnText: rgb(0x1A1008), Timeline: rgb(0x1E1A10)},
	{"midnight", "light"}: {Bg: rgb(0xF2EEF8), Surface: rgb(0xF8F5FD), Card: rgb(0xFFFFFF), Text: rgb(0x1A1028), Muted: rgb(0x7060A0), Accent: rgb(0x5A3E9A), Accent2: rgb(0x9B74D4), Border: rgb(0xDDD5F0), Tag: rgb(0xEBE6F8), BtnText: rgb(0xFFFFFF), Timeline: rgb(0xEDE8F8)},
	{"midnight", "dark"}:  {Bg: rgb(0x07050F), Surface: rgb(0x0F0A1C), Card: rgb(0x160F2A), Text: rgb(0xEDE8F8), Muted: rgb(0x9078C8), Accent: rgb(0xB090E8), Accent2: rgb(0xCDB0F5), Border: rgb(0x2A1E48), Tag: rgb(0x130B22), BtnText: rgb(0x07050F), Timeline: rgb(0x110A20)},
	{"blush", "light"}:    {Bg: rgb(0xFDF0EF), Surface: rgb(0xFEF6F5), Card: rgb(0xFFFFFF), Text: rgb(0x2C1A18), Muted: rgb(0xA0706A), Accent: rgb(0xC4756A), Accent2: rgb(0xE8A89F), Border: rgb(0xF0D5D0), Tag: rgb(0xF5E2DF), BtnText: rgb(0xFFFFFF), Timeline: rgb(0xFAEAE8)},
	{"blush", "dark"}:     {Bg: rgb(0x160A09), Surface: rgb(0x201210), Card: rgb(0x2C1816), Text: rgb(0xF5EAE8), Muted: rgb(0xC09088), Accent: rgb(0xD49088), Accent2: rgb(0xEAB8B0), Border: rgb(0x3C2220), Tag: rgb(0x221210), BtnText: rgb(0x160A09), Timeline: rgb(0x1E1210)},
	{"forest", "light"}:   {Bg: rgb(0xEFF4EB), Surface: rgb(0xF6F9F3), Card: rgb(0xFFFFFF), Text: rgb(0x162116), Muted: rgb(0x5C7A5A), Accent: rgb(0x3D6B3A), Accent2: rgb(0x7AAD76), Border: rgb(0xD0E0CB), Tag: rgb(0xDCE9D8), BtnText: rgb(0xFFFFFF), Timeline: rgb(0xE8F0E4)},
	{"forest", "dark"}:    {Bg: rgb(0x060E06), Surface: rgb(0x0C160C), Card: rgb(0x121E12), Text: rgb(0xE8F0E5), Muted: rgb(0x6A9A66), Accent: rgb(0x7AAD76), Accent2: rgb(0xA0CC9C), Border: rgb(0x1E321E), Tag: rgb(0x0A140A), BtnText: rgb(0x060E06), Timeline: rgb(0x0E180E)},
	{"rose", "light"}:     {Bg: rgb(0xFEF0F5), Surface: rgb(0xFFF5F9), Card: rgb(0xFFFFFF), Text: rgb(0x2A1020), Muted: rgb(0xA06080), Accent: rgb(0xC05880), Accent2: rgb(0xE090B0), Border: rgb(0xF0D0E0), Tag: rgb(0xF8E0EC), BtnText: rgb(0xFFFFFF), Timeline: rgb(0xFAE8F2)},
	{"rose", "dark"}:      {Bg: rgb(0x110618), Surface: rgb(0x1C0C22), Card: rgb(0x28142E), Text: rgb(0xF5E8F0), Muted: rgb(0xC080A8), Accent: rgb(0xD890B8), Accent2: rgb(0xEAB8D4), Border: rgb(0x3C1840), Tag: rgb(0x1A0A20), BtnText: rgb(0x110618), Timeline: rgb(0x1A0C20)},
	{"slate", "light"}:    {Bg: rgb(0xEEF2F7), Surface: rgb(0xF5F8FC), Card: rgb(0xFFFFFF), Text: rgb(0x141C28), Muted: rgb(0x607090), Accent: rgb(0x3A5A8A), Accent2: rgb(0x7090C0), Border: rgb(0xCCD8E8), Tag: rgb(0xDDE5F2), BtnText: rgb(0xFFFFFF), Timeline: rgb(0xE5EAF5)},
	{"slate", "dark"}:     {Bg: rgb(0x060C14), Surface: rgb(0x0C1420), Card: rgb(0x121C2E), Text: rgb(0xE8EEF8), Muted: rgb(0x7090C0), Accent: rgb(0x90B0E0), Accent2: rgb(0xB0C8F0), Border: rgb(0x1E2E44), Tag: rgb(0x0A1018), BtnText: rgb(0x060C14), Timeline: rgb(0x0E1622)},
}

// The default theme, also used before the profile is loaded.
var (
	LinenLight = themes[themeKey{"linen", "light"}]
	LinenDark  = themes[themeKey{"linen", "dark"}]
)

// Default custom colours, offered when the user first picks Custom.
var (
	DefaultCustomBase   = rgb(0x7A5C3A)
	DefaultCustomAccent = rgb(0xC4956A)
)

// PaletteFor returns the palette a profile asks for. Unknown themes fall
// back to Linen, and any mode other than "dark" (including none, as after
// a fresh install) is light. A custom theme with a missing or unreadable
// colour uses the default custom colour in its place.
func PaletteFor(p core.Profile) Palette {
	dark := p.Mode == "dark"
	if p.Theme == "custom" {
		base, ok := ParseHex(p.CustomBase)
		if !ok {
			base = DefaultCustomBase
		}
		accent, ok := ParseHex(p.CustomAccent)
		if !ok {
			accent = DefaultCustomAccent
		}
		return CustomPalette(base, accent, dark)
	}
	mode := "light"
	if dark {
		mode = "dark"
	}
	if pal, ok := themes[themeKey{p.Theme, mode}]; ok {
		return pal
	}
	return themes[themeKey{"linen", mode}]
}

// CustomPalette derives a full palette from the user's two colours, with
// the formulas in docs/design-tokens.json ("custom_palette").
func CustomPalette(base, accent color.NRGBA, dark bool) Palette {
	white, black := rgb(0xFFFFFF), rgb(0x000000)
	if dark {
		bg := lerp(base, black, 0.894)
		return Palette{
			Bg:       bg,
			Surface:  lerp(base, black, 0.814),
			Card:     lerp(base, black, 0.731),
			Text:     lerp(base, white, 0.879),
			Muted:    lerp(base, accent, 0.70),
			Accent:   accent,
			Accent2:  base,
			Border:   lerp(base, black, 0.573),
			Tag:      lerp(base, black, 0.825),
			BtnText:  bg,
			Timeline: lerp(base, black, 0.802),
		}
	}
	btn := rgb(0x1A1A1A)
	if isDark(accent) {
		btn = white
	}
	return Palette{
		Bg:       lerp(base, white, 0.923),
		Surface:  lerp(base, white, 0.959),
		Card:     white,
		Text:     lerp(base, black, 0.700),
		Muted:    lerp(base, accent, 0.40),
		Accent:   accent,
		Accent2:  base,
		Border:   lerp(base, white, 0.758),
		Tag:      lerp(base, white, 0.841),
		BtnText:  btn,
		Timeline: lerp(base, white, 0.883),
	}
}

// lerp moves a toward b by t in plain RGB, rounding like the prototype's
// colorLerp.
func lerp(a, b color.NRGBA, t float64) color.NRGBA {
	ch := func(x, y uint8) uint8 {
		return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t))
	}
	return color.NRGBA{R: ch(a.R, b.R), G: ch(a.G, b.G), B: ch(a.B, b.B), A: 0xFF}
}

// isDark is the prototype's isColorDark: perceived brightness below half.
func isDark(c color.NRGBA) bool {
	return (int(c.R)*299+int(c.G)*587+int(c.B)*114)/1000 < 128
}

// ParseHex reads "#RRGGBB" or "#RGB" (the # is optional, case ignored).
func ParseHex(s string) (color.NRGBA, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return color.NRGBA{}, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return color.NRGBA{}, false
	}
	return rgb(uint32(v)), true
}

func rgb(v uint32) color.NRGBA {
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xFF}
}

// withAlpha returns c at the given opacity (0–1), like CSS opacity on a colour.
func withAlpha(c color.NRGBA, a float32) color.NRGBA {
	c.A = uint8(float32(c.A)*a + 0.5)
	return c
}

// shade is black at CSS opacity a (rgba(0,0,0,a)) laid over bg, for
// scrims and shadows. Gio blends colours in linear light, so the same
// rgba looks much paler than in a browser, which blends the gamma-encoded
// values. We work out how much black Gio needs to darken bg as far as
// the browser would, so the page behind a dialog matches the prototype.
func shade(a float32, bg color.NRGBA) color.NRGBA {
	var sum float64
	for _, c := range []uint8{bg.R, bg.G, bg.B} {
		v := float64(c) / 255
		if v < 0.01 {
			sum += float64(a) // black stays black either way
			continue
		}
		sum += 1 - srgbToLinear(v*(1-float64(a)))/srgbToLinear(v)
	}
	return color.NRGBA{A: uint8(math.Round(sum / 3 * 255))}
}

func srgbToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}
