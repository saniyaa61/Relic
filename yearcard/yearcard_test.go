package yearcard

import (
	"bytes"
	"image"
	"image/color"
	"os"
	"testing"
	"time"

	"github.com/saniyaa61/relic/core"
	"github.com/saniyaa61/relic/importer"
)

var linen = Colors{
	Bg: rgb(0xF7F3EC), Text: rgb(0x2C2016), Muted: rgb(0x8B7355), Accent: rgb(0x7A5C3A),
	Accent2: rgb(0xC4956A), Border: rgb(0xE2D9C8), Tag: rgb(0xEDE4D3),
}

func rgb(v uint32) color.NRGBA {
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xFF}
}

func fixture(t *testing.T) core.YearCard {
	t.Helper()
	f, err := os.Open("../importer/testdata/scrubbed-archive.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	res, err := importer.Read(f, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return res.Library.YearCard(core.NewTimeIndex(res.Library.Entries), 2026, time.Local)
}

// The card renders at 1080 × 1350 in the theme's colours. Set
// YEARCARD_OUT=path.png to look at it.
func TestRender(t *testing.T) {
	yc := fixture(t)
	posters := make([]image.Image, len(yc.TopRated))
	if len(posters) > 1 {
		p := image.NewRGBA(image.Rect(0, 0, 60, 90))
		for i := range p.Pix {
			p.Pix[i] = 0x80
		}
		posters[1] = p
	}
	img := Render(yc, linen, posters)
	if b := img.Bounds(); b.Dx() != Width || b.Dy() != Height {
		t.Fatalf("size %v", b)
	}
	if got := img.RGBAAt(5, 5); got != (color.RGBA{0xF7, 0xF3, 0xEC, 0xFF}) {
		t.Errorf("background %v", got)
	}
	var buf bytes.Buffer
	if err := EncodePNG(&buf, img); err != nil {
		t.Fatal(err)
	}
	if out := os.Getenv("YEARCARD_OUT"); out != "" {
		os.WriteFile(out, buf.Bytes(), 0o644)
	}
}

// An empty year and long titles draw without trouble.
func TestRenderEdgeCases(t *testing.T) {
	Render(core.YearCard{Year: 2025}, linen, nil)
	yc := fixture(t)
	if len(yc.TopRated) > 0 {
		yc.TopRated[0].Title = "A very long title that will certainly need more than three lines to fit in its narrow column"
	}
	Render(yc, linen, nil)
}

func TestWrapAndFit(t *testing.T) {
	s := textStyle{serifSemiBold, 23, linen.Text, 0}
	lines := s.wrap("one two three four five six seven eight nine ten eleven twelve", 150, 3)
	if len(lines) != 3 || lines[2][len(lines[2])-len("…"):] != "…" {
		t.Errorf("wrap: %q", lines)
	}
	for _, l := range lines {
		if s.width(l) > 150 {
			t.Errorf("%q is %.0f wide", l, s.width(l))
		}
	}
	// Like the prototype's ycFit: 2px steps while above the minimum, so it
	// can end just under it (23 → 17 for a minimum of 18).
	if f := s.fit("Comforting and bittersweet", 100, 18); f.size != 17 {
		t.Errorf("fit stopped at %v", f.size)
	}
	if FileName(2026) != "relic-2026-in-review.png" {
		t.Error(FileName(2026))
	}
}
