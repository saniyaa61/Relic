// Package yearcard draws the Year in Review image (SPEC §7): a 1080 × 1350
// PNG in the theme's colours, laid out as the prototype's
// renderYearCardCanvas. It draws in plain Go, so it works the same on
// every platform and in tests.
package yearcard

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"strconv"
	"strings"

	"github.com/saniyaa61/relic/core"
)

// Size of the card in pixels.
const (
	Width  = 1080
	Height = 1350
)

// Colors are the theme colours the card uses.
type Colors struct {
	Bg, Text, Muted, Accent, Accent2, Border, Tag color.NRGBA
}

// Render draws the card. posters[i] is the poster of yc.TopRated[i], or
// nil to draw a tile with the title's first letter.
func Render(yc core.YearCard, c Colors, posters []image.Image) *image.RGBA {
	const (
		W, H = float64(Width), float64(Height)
		M    = 84.0
		CW   = W - 2*M
	)
	img := image.NewRGBA(image.Rect(0, 0, Width, Height))
	fillRect(img, 0, 0, W, H, c.Bg)
	strokeRRect(img, 36, 36, W-72, H-72, 40, 3, c.Border)

	sans := func(size float64, col color.Color) textStyle { return textStyle{sansMedium, size, col, 0} }
	spaced := func(size float64) textStyle { return textStyle{sansMedium, size, c.Muted, 3} }
	serif := func(size float64, col color.Color) textStyle { return textStyle{serifSemiBold, size, col, 0} }
	lora := func(size float64) textStyle { return textStyle{loraItalic, size, c.Muted, 0} }

	// Header.
	spaced(24).draw(img, "RELIC  ·  YEAR IN REVIEW", M, 120)
	serif(150, c.Text).draw(img, strconv.Itoa(yc.Year), M-4, 270)

	// Total time.
	spaced(22).draw(img, "TIME SPENT IN OTHER WORLDS", M, 326)
	num, unit := core.SplitDuration(core.FormatDurationOrZero(yc.Minutes))
	big := serif(120, c.Accent)
	big.draw(img, num, M, 442)
	lora(44).draw(img, unit, M+big.width(num)+16, 442)
	if p := core.Perspective(yc.Minutes); p != "" {
		lora(28).fit(p, CW, 18).draw(img, p, M, 488)
	}

	// Started · Finished · Top feeling.
	colW := CW / 3
	feeling := yc.TopTag
	if feeling == "" {
		feeling = "—"
	}
	stats := [3][2]string{{strconv.Itoa(len(yc.New)), "STARTED"}, {strconv.Itoa(len(yc.Finished)), "FINISHED"}, {feeling, "TOP FEELING"}}
	for i, st := range stats {
		x := M + float64(i)*colW
		if i > 0 {
			fillRect(img, x-1, 538, 2, 90, c.Border)
			x += 22
		}
		serif(64, c.Text).fit(st[0], colW-50, 28).draw(img, st[0], x, 592)
		spaced(19).draw(img, st[1], x, 626)
	}

	// Where it went: top 3 categories and Other, with a legend.
	spaced(20).draw(img, "WHERE IT WENT", M, 692)
	share := core.ShareBar(yc.Categories)
	cols := []color.NRGBA{c.Accent, c.Accent2, c.Muted, withAlpha(c.Muted, 0.4)}
	var total float64
	for _, s := range share {
		total += s.Minutes
	}
	fillRRect(img, M, 712, CW, 18, 9, c.Tag)
	if total > 0 {
		bar := image.NewRGBA(img.Bounds())
		sx := M
		for i, s := range share {
			w := CW * s.Minutes / total
			gap := 0.0
			if i < len(share)-1 {
				gap = 4
			}
			fillRect(bar, sx, 712, max(w-gap, 2), 18, cols[i])
			sx += w
		}
		// Clip the segments to the rounded bar.
		fillShape(img, bar, image.Point{}, func(z *rasterizer) { rrectPath(z, M, 712, CW, 18, 9) })
		lx := M
		legend := textStyle{sansMedium, 23, c.Text, 0}
		for i, s := range share {
			label := s.Name + "  " + core.FormatDuration(s.Minutes)
			lw := legend.width(label) + 34
			if lx+lw > M+CW {
				break
			}
			fillCircle(img, lx+8, 768, 8, cols[i])
			legend.draw(img, label, lx+24, 776)
			lx += lw + 14
		}
	}

	// Month by month.
	spaced(20).draw(img, "MONTH BY MONTH", M, 836)
	maxM, peak := 1.0, 0.0
	for _, m := range yc.Monthly {
		maxM, peak = max(maxM, m), max(peak, m)
	}
	slot := CW / 12
	const baseY, maxH = 968.0, 100.0
	for i, m := range yc.Monthly {
		h, col := 6.0, c.Tag
		if m > 0 {
			h, col = max(m/maxM*maxH, 8), c.Accent2
			if m == peak {
				col = c.Accent
			}
		}
		fillRRect(img, M+float64(i)*slot+(slot-44)/2, baseY-h, 44, h, min(8, h/2), col)
		sans(19, c.Muted).drawCentered(img, string("JFMAMJJASOND"[i]), M+float64(i)*slot+slot/2, baseY+32)
	}

	// Top rated.
	spaced(20).draw(img, "TOP RATED", M, 1050)
	if len(yc.TopRated) == 0 {
		lora(26).draw(img, "Rate what you finish and your favourites will live here.", M, 1114)
	}
	for i, e := range yc.TopRated {
		x, py, pw, ph := M+float64(i)*colW, 1074.0, 116.0, 164.0
		var poster image.Image
		if i < len(posters) {
			poster = posters[i]
		}
		if poster != nil {
			drawCover(img, poster, x, py, pw, ph, 14)
		} else {
			fillRRect(img, x, py, pw, ph, 14, c.Tag)
			initial := "?"
			if t := strings.TrimSpace(e.Title); t != "" {
				initial = strings.ToUpper(string([]rune(t)[:1]))
			}
			serif(48, c.Accent).drawCentered(img, initial, x+pw/2, py+ph/2+16)
		}
		title := e.Title
		if title == "" {
			title = "Untitled"
		}
		ts := serif(23, c.Text)
		lines := ts.wrap(title, colW-pw-36, 3)
		for k, ln := range lines {
			ts.draw(img, ln, x+pw+16, py+30+float64(k)*29)
		}
		textStyle{sansMedium, 22, c.Accent, 0}.draw(img, starString(e.Rating), x+pw+16, py+30+float64(len(lines))*29+10)
	}

	// Footer.
	lora(26).drawCentered(img, "your stories, preserved", W/2, 1294)
	return img
}

// starString is "★★★★½".
func starString(r float64) string {
	s := strings.Repeat("★", int(r))
	if r != float64(int(r)) {
		s += "½"
	}
	return s
}

// EncodePNG writes the card as a PNG.
func EncodePNG(w io.Writer, img image.Image) error {
	return (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(w, img)
}

// FileName is the prototype's download name, "relic-2026-in-review.png".
func FileName(year int) string { return "relic-" + strconv.Itoa(year) + "-in-review.png" }
