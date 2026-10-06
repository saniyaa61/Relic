package ui

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/saniyaa61/relic/core"
)

type tokenFile struct {
	Themes        map[string]map[string]map[string]string `json:"themes"`
	CustomPalette struct {
		Defaults struct{ Base, Accent string } `json:"defaults"`
	} `json:"custom_palette"`
}

func readTokens(t *testing.T) tokenFile {
	t.Helper()
	data, err := os.ReadFile("../docs/design-tokens.json")
	if err != nil {
		t.Fatal(err)
	}
	var tokens tokenFile
	if err := json.Unmarshal(data, &tokens); err != nil {
		t.Fatal(err)
	}
	return tokens
}

func hex(c color.NRGBA) string { return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B) }

func paletteMap(p Palette) map[string]color.NRGBA {
	return map[string]color.NRGBA{
		"bg": p.Bg, "surface": p.Surface, "card": p.Card,
		"text": p.Text, "muted": p.Muted, "accent": p.Accent,
		"accent2": p.Accent2, "border": p.Border, "tag": p.Tag,
		"btn-text": p.BtnText, "timeline": p.Timeline,
	}
}

// Every built-in theme × mode must match docs/design-tokens.json exactly,
// and ThemeNames must list exactly the themes in that file.
func TestThemesMatchDesignTokens(t *testing.T) {
	tokens := readTokens(t)
	if len(tokens.Themes) != len(ThemeNames) {
		t.Errorf("tokens have %d themes, ThemeNames has %d", len(tokens.Themes), len(ThemeNames))
	}
	for _, name := range ThemeNames {
		for _, mode := range []string{"light", "dark"} {
			want := tokens.Themes[name][mode]
			if want == nil {
				t.Errorf("%s %s: not in design tokens", name, mode)
				continue
			}
			got := paletteMap(PaletteFor(core.Profile{Theme: name, Mode: mode}))
			if len(want) != len(got) {
				t.Errorf("%s %s: tokens have %d colours, Palette has %d", name, mode, len(want), len(got))
			}
			for key, h := range want {
				if g := hex(got[key]); g != strings.ToUpper(h) {
					t.Errorf("%s %s %s = %s, want %s", name, mode, key, g, h)
				}
			}
		}
	}
}

func TestPaletteForFallbacks(t *testing.T) {
	for _, tc := range []struct {
		name string
		p    core.Profile
		want Palette
	}{
		{"fresh install", core.Profile{}, LinenLight},
		{"unknown theme", core.Profile{Theme: "sepia", Mode: "dark"}, LinenDark},
		{"unknown mode is light", core.Profile{Theme: "linen", Mode: "dusk"}, LinenLight},
		{"custom without colours", core.Profile{Theme: "custom"}, CustomPalette(DefaultCustomBase, DefaultCustomAccent, false)},
	} {
		if got := PaletteFor(tc.p); got != tc.want {
			t.Errorf("%s: got %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

// Expected values were computed with the prototype's own applyCustomPalette
// (colorLerp + isColorDark), so these pin the formulas to what the
// prototype shows.
func TestCustomPalette(t *testing.T) {
	tokens := readTokens(t)
	base, _ := ParseHex(tokens.CustomPalette.Defaults.Base)
	accent, _ := ParseHex(tokens.CustomPalette.Defaults.Accent)
	for _, tc := range []struct {
		name         string
		base, accent string
		dark         bool
		want         map[string]string
	}{
		{"defaults light", "#7A5C3A", "#C4956A", false, map[string]string{
			"bg": "#F5F2F0", "surface": "#FAF8F7", "card": "#FFFFFF", "text": "#251C11",
			"muted": "#98734D", "accent": "#C4956A", "accent2": "#7A5C3A", "border": "#DFD8CF",
			"tag": "#EAE5E0", "btn-text": "#1A1A1A", "timeline": "#EFECE8"}},
		{"defaults dark", "#7A5C3A", "#C4956A", true, map[string]string{
			"bg": "#0D0A06", "surface": "#17110B", "card": "#211910", "text": "#EFEBE7",
			"muted": "#AE845C", "accent": "#C4956A", "accent2": "#7A5C3A", "border": "#342719",
			"tag": "#15100A", "btn-text": "#0D0A06", "timeline": "#18120B"}},
		{"dark accent gets white button text", "#BB356F", "#2E3A8C", false, map[string]string{
			"bg": "#FAEFF4", "text": "#381021", "muted": "#83377B", "border": "#EFCEDC",
			"tag": "#F4DFE8", "btn-text": "#FFFFFF", "timeline": "#F7E7EE"}},
	} {
		b, _ := ParseHex(tc.base)
		a, _ := ParseHex(tc.accent)
		got := paletteMap(CustomPalette(b, a, tc.dark))
		for key, want := range tc.want {
			if g := hex(got[key]); g != want {
				t.Errorf("%s %s = %s, want %s", tc.name, key, g, want)
			}
		}
	}
	if base != DefaultCustomBase || accent != DefaultCustomAccent {
		t.Errorf("default custom colours %s/%s, tokens say %s/%s", hex(DefaultCustomBase), hex(DefaultCustomAccent),
			tokens.CustomPalette.Defaults.Base, tokens.CustomPalette.Defaults.Accent)
	}
}

func TestParseHex(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
		ok   bool
	}{
		{"#7a5c3a", "#7A5C3A", true},
		{"C4956A", "#C4956A", true},
		{"#abc", "#AABBCC", true},
		{"", "", false},
		{"#12345", "", false},
		{"#zzzzzz", "", false},
	} {
		c, ok := ParseHex(tc.in)
		if ok != tc.ok || (ok && hex(c) != tc.want) {
			t.Errorf("ParseHex(%q) = %s, %v; want %s, %v", tc.in, hex(c), ok, tc.want, tc.ok)
		}
	}
}

// A shade over a colour must darken it as a browser would: by (1-a) in
// gamma-encoded sRGB, when Gio blends in linear light.
func TestShadeMatchesCSS(t *testing.T) {
	toSRGB := func(l float64) float64 {
		if l <= 0.0031308 {
			return 255 * 12.92 * l
		}
		return 255 * (1.055*math.Pow(l, 1/2.4) - 0.055)
	}
	for _, name := range ThemeNames {
		for _, mode := range []string{"light", "dark"} {
			bg := PaletteFor(core.Profile{Theme: name, Mode: mode}).Bg
			for _, a := range []float32{0.25, 0.45} {
				s := shade(a, bg)
				k := 1 - float64(s.A)/255
				for _, c := range []uint8{bg.R, bg.G, bg.B} {
					gio := toSRGB(srgbToLinear(float64(c)/255) * k)
					css := float64(c) * (1 - float64(a))
					if math.Abs(gio-css) > 2.5 {
						t.Errorf("%s %s alpha %v: channel %d becomes %.1f, CSS %.1f", name, mode, a, c, gio, css)
					}
				}
			}
		}
	}
}
