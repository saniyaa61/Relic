package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/saniyaa61/relic/core"
)

// cardPosterIcon is the glyph a card shows without a poster (the
// prototype's film / series / book icons, a circle for the rest).
func cardPosterIcon(t core.EntryType) *Icon {
	if ic := rowIcons[t]; ic != nil {
		return ic
	}
	return IconCircle
}

// IconCircle is the plain fallback poster glyph.
var IconCircle = IconSVG(`<circle cx="12" cy="12" r="9"/>`)

// ongoingCard is the "Still with you" card (.oc): 130dp wide, a 175dp
// poster with the progress ring and, once the end is reached, the
// "✓ Finished?" badge; then title, category and "Ep 4/16" / "p.80/321".
// It lifts 3dp with a soft shadow while pressed (the prototype's hover).
type ongoingCard struct {
	click  widget.Clickable
	finish widget.Clickable
	// tall keeps room for the progress line even without one, so every
	// card in a row is as tall as the tallest (CSS flex stretch).
	tall bool
}

func (c *ongoingCard) Layout(gtx layout.Context, a *App, e *core.Entry) layout.Dimensions {
	th := a.Theme
	if c.click.Clicked(gtx) {
		a.Push(newEntryDetail(e.ID))
	}
	if c.finish.Clicked(gtx) {
		confirmMarkFinished(a, e.ID)
	}
	w := gtx.Dp(130)
	gtx.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, gtx.Constraints.Max.Y)}
	return liftOnPress(gtx, th, &c.click, 13, func(gtx layout.Context) layout.Dimensions {
		return card(gtx, th.Card, th.Border, 13, layout.Inset{}, 130, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return c.poster(gtx, a, e) }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: 7, Bottom: 9, Left: 9, Right: 9}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return c.body(gtx, a, e)
					})
				}),
			)
		})
	})
}

func (c *ongoingCard) poster(gtx layout.Context, a *App, e *core.Entry) layout.Dimensions {
	th := a.Theme
	size := image.Pt(gtx.Constraints.Max.X, gtx.Dp(175))
	fillRect(gtx, image.Rectangle{Max: size}, th.Tag)
	if !a.drawPoster(gtx, e.Poster, size, 0) {
		icon := gtx.Dp(26)
		st := op.Offset(image.Pt((size.X-icon)/2, (size.Y-icon)/2)).Push(gtx.Ops)
		cardPosterIcon(e.Type).Layout(gtx, 26, 1.3, withAlpha(th.Muted, 0.7))
		st.Pop()
	}
	if pct := e.ProgressPercent(); pct > 0 {
		ring := gtx.Dp(ringSize)
		st := op.Offset(image.Pt(size.X-gtx.Dp(6)-ring, size.Y-gtx.Dp(6)-ring)).Push(gtx.Ops)
		progressRing(gtx, th, pct)
		st.Pop()
	}
	if e.ShouldPromptFinished() {
		st := op.Offset(image.Pt(gtx.Dp(6), gtx.Dp(6))).Push(gtx.Ops)
		bgtx := gtx
		bgtx.Constraints.Min = image.Point{}
		c.finish.Layout(bgtx, func(gtx layout.Context) layout.Dimensions {
			rec := op.Record(gtx.Ops)
			d := card(gtx, th.Accent, th.Accent, 20, layout.Inset{Top: 2, Bottom: 2, Left: 7, Right: 7}, 0, func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Sans}, Size: 9, Color: th.BtnText}.Layout(gtx, th, "✓ Finished?")
			})
			call := rec.Stop()
			boxShadow(gtx, d.Size, d.Size.Y/2, 2, 6, 0.25, th.Tag)
			if c.finish.Pressed() {
				defer paint.PushOpacity(gtx.Ops, 0.85).Pop()
			}
			call.Add(gtx.Ops)
			return d
		})
		st.Pop()
	}
	return layout.Dimensions{Size: size}
}

func (c *ongoingCard) body(gtx layout.Context, a *App, e *core.Entry) layout.Dimensions {
	th := a.Theme
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	children := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 11, MaxLines: 1, Color: th.Text}.Layout(gtx, th, e.Title)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.05, Upper: true, Color: th.Muted}.Layout(gtx, th, a.Lib.CategoryName(e.CategoryID))
		}),
	}
	if done, total, ok := e.Progress(); (ok && total > 0) || c.tall {
		label := fmt.Sprintf("p.%d/%d", done, total)
		if e.Type.Progress() == core.EpisodeProgress {
			label = fmt.Sprintf("Ep %d/%d", done, total)
		}
		if !ok || total == 0 {
			label = " "
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Sans}, Size: 10, Color: th.Muted}.Layout(gtx, th, label)
			})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// confirmMarkFinished is the "✓ Finished?" badge's question.
func confirmMarkFinished(a *App, id string) {
	e := a.Lib.Entry(id)
	if e == nil {
		return
	}
	d := ConfirmDialog("Mark as finished?", "You've reached the end of "+e.Title+". Mark it as finished?",
		"Not yet", "Mark as finished", func(a *App) {
			e := a.Lib.Entry(id)
			if e == nil {
				return
			}
			e.SetStatus(core.Finished, a.Now())
			if a.save(false, []*core.Entry{e}, false) {
				a.Toast("Marked as finished ✦")
			}
		})
	d.Danger = markFinishedRed
	a.ShowDialog(d)
}

// finishedCard is the small poster card in "Recently finished" and the
// "see all" pages (.fin-card): a 90×126 poster with a soft shadow, the
// type, title, stars and, on Consumed, the entry's time.
type finishedCard struct {
	click widget.Clickable
	// withPlus adds the Favourites "+" (add to Top 5) at the poster's
	// top right; the caller handles plus.Clicked.
	withPlus bool
	plus     widget.Clickable
}

func (c *finishedCard) Layout(gtx layout.Context, a *App, e *core.Entry, showTime bool) layout.Dimensions {
	th := a.Theme
	if c.click.Clicked(gtx) {
		a.Push(newEntryDetail(e.ID))
	}
	w := gtx.Dp(90)
	gtx.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, gtx.Constraints.Max.Y)}
	return c.click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// Pressed: shrink to 96% (the prototype's mousedown).
		rec := op.Record(gtx.Ops)
		d := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				size := image.Pt(w, gtx.Dp(126))
				rad := gtx.Dp(10)
				boxShadow(gtx, size, rad, 3, 10, 0.12, th.Bg)
				if !a.drawPoster(gtx, e.Poster, size, rad) {
					rrect(gtx, size, rad, th.Tag)
					rec := op.Record(gtx.Ops)
					d := layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return finIcon(e.Type).Layout(gtx, 22, 1.3, withAlpha(th.Muted, 0.7))
						}),
						layout.Rigid(layout.Spacer{Height: 6}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return Text{Font: font.Font{Typeface: Serif}, Size: 9, Tracking: 0.07, Upper: true, Color: th.Muted}.Layout(gtx, th, typeShort(e.Type))
						}))
					call := rec.Stop()
					st := op.Offset(image.Pt((size.X-d.Size.X)/2, (size.Y-d.Size.Y)/2)).Push(gtx.Ops)
					call.Add(gtx.Ops)
					st.Pop()
				}
				if c.withPlus {
					b := gtx.Dp(22)
					st := op.Offset(image.Pt(size.X-gtx.Dp(6)-b, gtx.Dp(6))).Push(gtx.Ops)
					plusBadge(gtx, th, &c.plus, b)
					st.Pop()
				}
				return layout.Dimensions{Size: size}
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 5}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Serif}, Size: 9, Tracking: 0.07, Upper: true, Color: th.Muted}.Layout(gtx, th, typeShort(e.Type))
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 11, MaxLines: 1, Color: th.Text}.Layout(gtx, th, e.Title)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if e.Rating <= 0 {
					return layout.Dimensions{}
				}
				return layout.Inset{Top: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Sans}, Size: 10, Color: th.Accent}.Layout(gtx, th, starString(e.Rating))
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if !showTime {
					return layout.Dimensions{}
				}
				return layout.Inset{Top: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Sans, Weight: font.Medium}, Size: 9.5, FakeBold: true, Color: th.Accent}.Layout(gtx, th,
						core.FormatDurationOrZero(a.timeIndex().Total(e.ID)))
				})
			}),
		)
		call := rec.Stop()
		if c.click.Pressed() {
			s := float32(0.96)
			ctr := f32.Pt(float32(d.Size.X)/2, float32(d.Size.Y)/2)
			defer op.Affine(f32.AffineId().Scale(ctr, f32.Pt(s, s))).Push(gtx.Ops).Pop()
		}
		call.Add(gtx.Ops)
		return d
	})
}

// plusBadge is the Favourites card's round "+" (.fav-card-plus): dark
// glass over the poster, accent while pressed.
func plusBadge(gtx layout.Context, th *Theme, c *widget.Clickable, size int) layout.Dimensions {
	gtx.Constraints = layout.Exact(image.Pt(size, size))
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		bg := shade(0.45, th.Tag)
		if c.Pressed() {
			bg = th.Accent
		}
		rrect(gtx, image.Pt(size, size), size/2, bg)
		ic := gtx.Dp(11)
		st := op.Offset(image.Pt((size-ic)/2, (size-ic)/2)).Push(gtx.Ops)
		IconPlus.Layout(gtx, 11, 1.8, rgb(0xFFFFFF))
		st.Pop()
		return layout.Dimensions{Size: image.Pt(size, size)}
	})
}

// starString is the prototype's compact rating: one ★ per whole star and
// "½" for a half ("★★★½").
func starString(r float64) string {
	s := ""
	for i := 0; i < int(r); i++ {
		s += "★"
	}
	if r != math.Trunc(r) {
		s += "½"
	}
	return s
}

// typeShort is the prototype's short type word for cards and the memory
// card ("Movie", "Drama", "Album").
func typeShort(t core.EntryType) string {
	switch t {
	case core.Film:
		return "Movie"
	case core.Series:
		return "Drama"
	case core.Book:
		return "Book"
	case core.Podcast:
		return "Podcast"
	case core.Short:
		return "Short"
	case core.Music:
		return "Album"
	}
	return "Entry"
}

// finIcon is the no-poster glyph on a finished card (the prototype has
// a few more here than elsewhere: podcast and music too).
func finIcon(t core.EntryType) *Icon {
	switch t {
	case core.Podcast:
		return iconPodcastSmall
	case core.Music:
		return typeIcons[core.Music]
	}
	return cardPosterIcon(t)
}

var iconPodcastSmall = IconSVG(`<circle cx="12" cy="11" r="4"/><path d="M4 11a8 8 0 0016 0"/>`)

// liftOnPress draws w raised 3dp with a soft shadow while c is pressed
// (the prototype's .oc:hover).
func liftOnPress(gtx layout.Context, th *Theme, c *widget.Clickable, radius unit.Dp, w layout.Widget) layout.Dimensions {
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if !c.Pressed() {
			return w(gtx)
		}
		rec := op.Record(gtx.Ops)
		d := w(gtx)
		call := rec.Stop()
		defer op.Offset(image.Pt(0, -gtx.Dp(3))).Push(gtx.Ops).Pop()
		boxShadow(gtx, d.Size, gtx.Dp(radius), 8, 20, 0.12, th.Bg)
		call.Add(gtx.Ops)
		return d
	})
}

// The prototype's .prog-ring asks for 28px, but the general `.oc-ph svg`
// rule wins and draws it at 26px and 70% opacity. That rule also outlines
// the white label in the muted colour, thick enough to cover the white,
// so it reads as bold brown. The owner likes that, so we draw it solid
// muted with the same 1.3-unit outline.
const ringSize unit.Dp = 26

// progressRing draws the white ring with the percentage in the middle.
func progressRing(gtx layout.Context, th *Theme, pct int) layout.Dimensions {
	defer paint.PushOpacity(gtx.Ops, 0.7).Pop()
	size := gtx.Dp(ringSize)
	px := float32(size) / 28 // the ring is designed in a 28-unit box
	centre := f32.Pt(14*px, 14*px)
	white := color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	strokeArc(gtx, centre, 11*px, 3*px, 0, 2*math.Pi, withAlpha(white, 0.25))
	strokeArc(gtx, centre, 11*px, 3*px, 0, 2*math.Pi*float32(min(pct, 100))/100, withAlpha(white, 0.85))

	// Centre the label in the ring.
	rec := op.Record(gtx.Ops)
	lgtx := gtx
	lgtx.Constraints = layout.Constraints{Max: image.Pt(size, size)}
	d := outlinedText(lgtx, th, font.Font{Typeface: Sans, Weight: font.Medium}, unit.Sp(7*float32(ringSize)/28), fmt.Sprintf("%d%%", pct),
		th.Muted, th.Muted, 1.3*px)
	call := rec.Stop()
	st := op.Offset(image.Pt((size-d.Size.X)/2, (size-d.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	st.Pop()
	return layout.Dimensions{Size: image.Pt(size, size)}
}

// hasProgressLine reports whether any of es shows "Ep 4/16" / "p.80/321".
func hasProgressLine(es []*core.Entry) bool {
	for _, e := range es {
		if _, total, ok := e.Progress(); ok && total > 0 {
			return true
		}
	}
	return false
}
