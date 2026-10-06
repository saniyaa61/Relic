package ui

// The Year in Review image (SPEC §7): made from the Year digest's "Share
// your year", previewed in a dialog with Close and Save image.

import (
	"errors"
	"fmt"
	"image"
	"log"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	xdraw "golang.org/x/image/draw"

	"github.com/saniyaa61/relic/yearcard"
)

// openYearCard draws the year's card in the background, then previews it.
func openYearCard(a *App, year int) {
	yc := a.Lib.YearCard(a.timeIndex(), year, a.Loc)
	if !yc.YearHasData() {
		a.ToastError(fmt.Sprintf("Nothing logged in %d yet", year))
		return
	}
	a.Toast("Making your card…")
	posters := make([]image.Image, len(yc.TopRated))
	for i, e := range yc.TopRated {
		if e.Poster != "" {
			if img, ok := a.posterImage(e.Poster); ok {
				posters[i] = img
			}
		}
	}
	colors := cardColors(a.Theme)
	a.background(func() func(*App) {
		img := yearcard.Render(yc, colors, posters)
		return func(a *App) { showYearCard(a, year, img) }
	})
}

// yearPreview keeps the card scaled to the size it's shown at.
type yearPreview struct {
	img    *image.RGBA
	scaled paint.ImageOp
	size   image.Point
}

func (p *yearPreview) layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	// max-width 100%, max-height 62% of the window, centred, with a 1px
	// border and 12px corners.
	w := gtx.Constraints.Max.X
	h := w * yearcard.Height / yearcard.Width
	if maxH := gtx.Constraints.Max.Y * 62 / 100; h > maxH {
		h = maxH
		w = h * yearcard.Width / yearcard.Height
	}
	size := image.Pt(w, h)
	b := gtx.Dp(1)
	inner := size.Sub(image.Pt(2*b, 2*b))
	if inner != p.size {
		dst := image.NewRGBA(image.Rectangle{Max: inner})
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), p.img, p.img.Bounds(), xdraw.Src, nil)
		p.scaled, p.size = paint.NewImageOp(dst), inner
	}
	defer op.Offset(image.Pt((gtx.Constraints.Max.X-w)/2, 0)).Push(gtx.Ops).Pop()
	r := gtx.Dp(12)
	rrect(gtx, size, r, th.Border)
	st := op.Offset(image.Pt(b, b)).Push(gtx.Ops)
	cl := clip.UniformRRect(image.Rectangle{Max: inner}, r-b).Push(gtx.Ops)
	p.scaled.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	cl.Pop()
	st.Pop()
	return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
}

func showYearCard(a *App, year int, img *image.RGBA) {
	p := &yearPreview{img: img}
	d := &Dialog{Cancel: "Close", Confirm: "Save image", Wide: true, KeepOpen: true,
		Body:      p.layout,
		OnConfirm: func(a *App, _ string) { saveYearCard(a, year, img) }}
	a.ShowDialog(d)
}

// saveYearCard asks where to save the PNG and writes it.
func saveYearCard(a *App, year int, img *image.RGBA) {
	if a.CreateFile == nil {
		a.ToastError("Saving isn't available here yet.")
		return
	}
	a.background(func() func(*App) {
		wc, err := a.CreateFile(yearcard.FileName(year))
		if errors.Is(err, ErrNoFile) {
			return nil
		}
		if err == nil {
			err = yearcard.EncodePNG(wc, img)
			if cerr := wc.Close(); err == nil {
				err = cerr
			}
		}
		if err != nil {
			log.Printf("relic: year card: %v", err)
			return func(a *App) { a.ToastError("The image couldn't be saved.") }
		}
		return func(a *App) { a.Toast("Saved ✦") }
	})
}

// cardColors are the theme's colours for the card.
func cardColors(th *Theme) yearcard.Colors {
	return yearcard.Colors{Bg: th.Bg, Text: th.Text, Muted: th.Muted, Accent: th.Accent,
		Accent2: th.Accent2, Border: th.Border, Tag: th.Tag}
}
