package ui

// The entry page (SPEC §5 "Entry detail"; prototype renderDetail).

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/saniyaa61/relic/core"
)

// heroStyle is the poster hero's look. The owner chose the prototype's (a
// blurred copy of the poster behind it and a fade into the page) on
// 2026-10-06, an agreed exception to "no gradients"; the flat band stays
// for snapshots comparing the two.
type heroStyle int

const (
	heroFlat heroStyle = iota
	heroPrototype
)

var detailHero = heroPrototype

type entryDetail struct {
	id       string
	list     widget.List
	back     widget.Clickable
	fav      widget.Clickable
	edit     widget.Clickable
	topFive  widget.Clickable
	logBtn   widget.Clickable
	del      widget.Clickable
	delArmed time.Time // "Tap again to confirm deletion" until 3s after
	rankAt   time.Time // when to offer the Top 5 after favouriting
	items    map[string]*journeyButtons
}

type journeyButtons struct{ edit, del widget.Clickable }

func newEntryDetail(id string) *entryDetail { return &entryDetail{id: id} }

func (p *entryDetail) Layout(gtx layout.Context, a *App) layout.Dimensions {
	e := a.Lib.Entry(p.id)
	if e == nil { // deleted
		a.Pop()
		return layout.Dimensions{}
	}
	catName := a.Lib.CategoryName(e.CategoryID)
	p.handle(gtx, a, e)
	if !p.rankAt.IsZero() {
		// The prototype offers the Top 5 a moment after the heart turns.
		if a.Now().Before(p.rankAt) {
			gtx.Execute(op.InvalidateCmd{At: p.rankAt})
		} else {
			p.rankAt = time.Time{}
			openRankPrompt(a, e.ID)
		}
	}

	size := gtx.Constraints.Max
	heroH := gtx.Dp(280)
	// The body scrolls under a fixed hero, as in the prototype.
	st := op.Offset(image.Pt(0, heroH)).Push(gtx.Ops)
	bgtx := gtx
	bgtx.Constraints = layout.Exact(image.Pt(size.X, size.Y-heroH))
	p.list.Axis = layout.Vertical
	rows := p.body(a, e, catName)
	p.list.List.Layout(bgtx, len(rows), func(gtx layout.Context, i int) layout.Dimensions { return rows[i](gtx) })
	st.Pop()
	p.hero(gtx, a, e, image.Pt(size.X, heroH))
	return layout.Dimensions{Size: size}
}

func (p *entryDetail) handle(gtx layout.Context, a *App, e *core.Entry) {
	if p.back.Clicked(gtx) {
		a.Back()
	}
	if p.edit.Clicked(gtx) {
		a.Push(editEntryPage(a, e))
	}
	if p.fav.Clicked(gtx) {
		if toggleFavourite(a, e) {
			p.rankAt = a.Now().Add(380 * time.Millisecond)
		} else {
			p.rankAt = time.Time{}
		}
	}
	if p.topFive.Clicked(gtx) {
		if _, ok := a.Lib.TopFiveRank(e.ID); ok {
			removeTopFive(a, e.ID)
		} else {
			addTopFive(a, e.ID)
		}
	}
	if p.del.Clicked(gtx) {
		if a.Now().Sub(p.delArmed) > 3*time.Second {
			p.delArmed = a.Now()
		} else {
			p.deleteEntry(a, e)
			return
		}
	}
	if p.logBtn.Clicked(gtx) {
		openLogSheet(a, e.ID, e.Status == core.Finished, "")
	}
	for id, b := range p.items {
		if b.edit.Clicked(gtx) {
			openLogSheet(a, e.ID, isRewatch(e, id), id)
		}
		if b.del.Clicked(gtx) {
			confirmDeleteJourneyItem(a, e.ID, id)
		}
	}
}

func (p *entryDetail) deleteEntry(a *App, e *core.Entry) {
	if a.Lib.DeleteEntry(e.ID) != nil {
		return
	}
	ok := a.Update(func(w storeWriter) error {
		if err := w.DeleteEntry(e.ID); err != nil {
			return err
		}
		return w.SaveFavourites(a.Lib.Favourites)
	})
	if ok {
		a.removeUnusedPosters()
		a.Pop()
		a.Toast("Entry removed from your library")
	}
}

// hero is the 280dp poster area with the back, favourite and Edit buttons.
func (p *entryDetail) hero(gtx layout.Context, a *App, e *core.Entry, size image.Point) {
	th := a.Theme
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.FillShape(gtx.Ops, th.Tag, clip.Rect{Max: size}.Op())
	if img, ok := a.posterImage(e.Poster); ok {
		if detailHero == heroPrototype {
			blurredBackdrop(gtx, a, e.Poster, size, th.Tag)
		}
		// The fade sits under the poster (the poster has z-index 1 in the
		// prototype).
		p.fade(gtx, th, size)
		// The poster, full height, at most 65% wide, contained.
		b := img.Bounds()
		h := size.Y
		w := b.Dx() * h / max(b.Dy(), 1)
		if maxW := size.X * 65 / 100; w > maxW {
			w, h = maxW, b.Dy()*maxW/max(b.Dx(), 1)
		}
		ps := image.Pt(w, h)
		st := op.Offset(image.Pt((size.X-w)/2, (size.Y-h)/2)).Push(gtx.Ops)
		boxShadow(gtx, ps, gtx.Dp(4), 8, 32, 0.45, th.Tag)
		a.drawPosterContain(gtx, e.Poster, ps, gtx.Dp(4))
		st.Pop()
	} else {
		p.fade(gtx, th, size)
		ic := heroIcons[e.Type]
		if ic == nil {
			ic = rowIcons[core.Film]
		}
		px := gtx.Dp(44)
		st := op.Offset(image.Pt((size.X-px)/2, (size.Y-px)/2)).Push(gtx.Ops)
		ic.Layout(gtx, 44, 1, withAlpha(th.Muted, 0.4))
		st.Pop()
	}
	// The buttons: 14dp in from the top corners, translucent dark discs.
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	dark := shade(0.28, th.Tag)
	disc := func(gtx layout.Context, c *widget.Clickable, bg color.NRGBA, w layout.Widget) layout.Dimensions {
		return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			sz := gtx.Dp(32)
			paint.FillShape(gtx.Ops, bg, clip.Ellipse{Max: image.Pt(sz, sz)}.Op(gtx.Ops))
			gtx.Constraints = layout.Exact(image.Pt(sz, sz))
			return layout.Center.Layout(gtx, w)
		})
	}
	bgtx := gtx
	bgtx.Constraints = layout.Constraints{Max: size}
	layout.Inset{Top: 14, Left: 14, Right: 14}.Layout(bgtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = image.Pt(gtx.Constraints.Max.X, 0)
		return layout.Flex{Spacing: layout.SpaceBetween, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return disc(gtx, &p.back, dark, func(gtx layout.Context) layout.Dimensions { return IconBack.Layout(gtx, 15, 2, white) })
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						bg := dark
						if e.Favorite {
							bg = color.NRGBA{R: 220, G: 80, B: 80, A: 140} // rgba(220,80,80,.55)
						}
						return disc(gtx, &p.fav, bg, func(gtx layout.Context) layout.Dimensions {
							if e.Favorite {
								IconFavorites.Fill(gtx, 15, white)
							}
							return IconFavorites.Layout(gtx, 15, 2, white)
						})
					}),
					layout.Rigid(layout.Spacer{Width: 7}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return p.edit.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							rec := op.Record(gtx.Ops)
							d := layout.Inset{Top: 6, Bottom: 6, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions { return IconEdit.Layout(gtx, 13, 2, white) }),
									layout.Rigid(layout.Spacer{Width: 5}.Layout),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return Text{Font: font.Font{Typeface: Sans}, Size: 11, Tracking: 0.03, Color: white}.Layout(gtx, th, "Edit")
									}))
							})
							call := rec.Stop()
							rrect(gtx, d.Size, d.Size.Y/2, dark)
							call.Add(gtx.Ops)
							return d
						})
					}))
			}))
	})
}

// fade is the prototype hero's .det-grad: 130px fading from the page
// colour up into the hero. Only in the prototype hero style.
func (p *entryDetail) fade(gtx layout.Context, th *Theme, size image.Point) {
	if detailHero != heroPrototype {
		return
	}
	gh := gtx.Dp(130)
	transparent := th.Bg
	transparent.A = 0
	paint.LinearGradientOp{Stop1: f32.Pt(0, float32(size.Y)), Color1: th.Bg,
		Stop2: f32.Pt(0, float32(size.Y-gh)), Color2: transparent}.Add(gtx.Ops)
	defer clip.Rect(image.Rect(0, size.Y-gh, size.X, size.Y)).Push(gtx.Ops).Pop()
	paint.PaintOp{}.Add(gtx.Ops)
}

// body is the scrolling part under the hero (.det-body).
func (p *entryDetail) body(a *App, e *core.Entry, catName string) []layout.Widget {
	th := a.Theme
	pad := func(w layout.Widget) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: gutter, Right: gutter}.Layout(gtx, w)
		}
	}
	// The rule (hr.det-div) has 14px above and below. The browser merges
	// its top margin with the bottom margin of what comes before (11px
	// under the meta line, 4px under the progress line), so the space
	// above is the larger of the two, not their sum.
	lead := 11
	divider := func() layout.Widget {
		top := 14 - lead
		lead = 0
		return pad(func(gtx layout.Context) layout.Dimensions {
			y := gtx.Dp(unitDp(float32(top)))
			fillRect(gtx, image.Rect(0, y, gtx.Constraints.Max.X, y+gtx.Dp(1)), th.Border)
			return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, y+gtx.Dp(15))}
		})
	}
	label := func(s string) layout.Widget {
		return pad(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.08, Upper: true, Color: th.Muted}.Layout(gtx, th, s)
			})
		})
	}
	rows := []layout.Widget{
		layout.Spacer{Height: 12}.Layout,
		pad(func(gtx layout.Context) layout.Dimensions { return p.badges(gtx, a, e) }),
		pad(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: 7}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Paragraph{Font: font.Font{Typeface: Display, Weight: font.SemiBold}, Size: 24, LineHeight: 1.2, Color: th.Text}.Layout(gtx, th, e.Title)
			})
		}),
		pad(func(gtx layout.Context) layout.Dimensions { return p.metas(gtx, a, e) }),
	}
	if done, total, ok := e.Progress(); ok && total > 0 {
		lead = 4
		rows = append(rows, pad(func(gtx layout.Context) layout.Dimensions {
			return p.progress(gtx, th, e, done, total)
		}))
	}
	if q, ok := e.YourWords(); ok {
		rows = append(rows, divider(), label("Your words"), pad(func(gtx layout.Context) layout.Dimensions {
			return yourWords(gtx, th, q.Label(a.Loc), q.Text)
		}))
	}
	if len(e.Tags) > 0 {
		rows = append(rows, divider(), label("How it felt"), pad(func(gtx layout.Context) layout.Dimensions {
			chips := make([]layout.Widget, len(e.Tags))
			for i, t := range e.Tags {
				chips[i] = func(gtx layout.Context) layout.Dimensions {
					return card(gtx, th.Tag, th.Accent, 20, layout.Inset{Top: 4, Bottom: 4, Left: 11, Right: 11}, 0, func(gtx layout.Context) layout.Dimensions {
						return Text{Font: font.Font{Typeface: Sans}, Size: 12, Color: th.Accent}.Layout(gtx, th, t)
					})
				}
			}
			return flow(gtx, gtx.Dp(5), chips)
		}))
	}
	if e.Fields.Cast != "" {
		rows = append(rows, divider(), label("Cast"), pad(func(gtx layout.Context) layout.Dimensions {
			return Paragraph{Font: font.Font{Typeface: Sans}, Size: 13, Color: th.Muted}.Layout(gtx, th, e.Fields.Cast)
		}))
	}
	journey := e.Journey(catName)
	title := "Add to your journey"
	if len(e.Sessions) > 0 || len(e.Rewatches) > 0 {
		title = "Your journey"
	}
	rows = append(rows, divider(), label(title), pad(func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return p.logButton(gtx, th, e)
		})
	}))
	if len(journey) == 0 {
		rows = append(rows, pad(func(gtx layout.Context) layout.Dimensions {
			// 10px above, merged with the button's 8px below it.
			return layout.Inset{Top: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 13, LineHeight: 1.6, Color: th.Muted}.Layout(gtx, th,
					"Your sessions and rewatches will appear here as a timeline.")
			})
		}))
	} else {
		if p.items == nil {
			p.items = map[string]*journeyButtons{}
		}
		rows = append(rows, layout.Spacer{Height: 4}.Layout)
		for i, it := range journey {
			b := p.items[it.ID]
			if b == nil {
				b = &journeyButtons{}
				p.items[it.ID] = b
			}
			last := i == len(journey)-1
			rows = append(rows, pad(func(gtx layout.Context) layout.Dimensions {
				return journeyItem(gtx, a, e, it, b, last)
			}))
		}
	}
	rows = append(rows, divider(), pad(func(gtx layout.Context) layout.Dimensions {
		armed := a.Now().Sub(p.delArmed) <= 3*time.Second
		if armed {
			gtx.Execute(op.InvalidateCmd{At: p.delArmed.Add(3 * time.Second)})
		}
		return deleteButton(gtx, th, &p.del, armed)
	}), layout.Spacer{Height: 40}.Layout)
	return rows
}

// badges is the type badge and, for favourites, the Top 5 pill.
func (p *entryDetail) badges(gtx layout.Context, a *App, e *core.Entry) layout.Dimensions {
	th := a.Theme
	children := []layout.Widget{func(gtx layout.Context) layout.Dimensions {
		// card's 1dp border is part of the prototype's 3px 9px padding.
		return card(gtx, th.Tag, th.Tag, 20, layout.Inset{Top: 2, Bottom: 2, Left: 8, Right: 8}, 0, func(gtx layout.Context) layout.Dimensions {
			return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.06, Upper: true, Color: th.Muted}.Layout(gtx, th, typeBadge(e.Type))
		})
	}}
	if e.Favorite {
		children = append(children, func(gtx layout.Context) layout.Dimensions {
			return p.topFive.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				// The same size as the type badge (SPEC §5).
				pad := layout.Inset{Top: 2, Bottom: 2, Left: 8, Right: 8}
				if rank, ok := a.Lib.TopFiveRank(e.ID); ok {
					return card(gtx, th.Accent, th.Accent, 20, pad, 0, func(gtx layout.Context) layout.Dimensions {
						return Text{Font: font.Font{Typeface: Display, Weight: font.SemiBold}, Size: 10, Color: th.BtnText}.Layout(gtx, th, fmt.Sprintf("#%d All-Time ✕", rank))
					})
				}
				bg, fg := th.Bg, th.Accent
				if p.topFive.Pressed() {
					bg, fg = th.Accent, th.BtnText
				}
				return card(gtx, bg, th.Accent, 20, pad, 0, func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Sans}, Size: 10, Color: fg}.Layout(gtx, th, "+ All-Time Top 5")
				})
			})
		})
	}
	return alignedRow(gtx, gtx.Dp(7), children)
}

// typeBadge is the prototype's typeLabel.
func typeBadge(t core.EntryType) string {
	switch t {
	case core.Film:
		return "Movie"
	case core.Series:
		return "Drama · Series"
	case core.Book:
		return "Book"
	}
	return "Entry"
}

// metas is the line under the title: stars, key fields, total time.
func (p *entryDetail) metas(gtx layout.Context, a *App, e *core.Entry) layout.Dimensions {
	th := a.Theme
	children := []layout.Widget{func(gtx layout.Context) layout.Dimensions { return stars(gtx, th, e.Rating, 11) }}
	chip := func(ic *Icon, s string) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ic.Layout(gtx, 12, 1.8, th.Muted) }),
				layout.Rigid(layout.Spacer{Width: 3}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Sans}, Size: 12, Color: th.Muted}.Layout(gtx, th, s)
				}))
		}
	}
	for _, row := range rowsFor(e.Type) {
		for _, f := range row {
			if detailSkips[f.id] || (e.Type.TracksProgress() && (f.id == "totalEpisodes" || f.id == "totalPages")) {
				continue
			}
			v := ""
			if f.kind == dateFieldKind {
				if d := fieldDate(e.Fields, f.id); !d.IsZero() {
					v = d.String()
				}
			} else {
				v = fieldText(e, f.id)
			}
			if v == "" {
				continue
			}
			ic := metaIcons[f.id]
			if ic == nil {
				ic = IconDot
			}
			children = append(children, chip(ic, v))
		}
	}
	if mins := a.timeIndex().Total(e.ID); mins > 0 {
		children = append(children, chip(IconClock, core.FormatDuration(mins)))
	}
	return layout.Inset{Bottom: 11}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return alignedRow(gtx, gtx.Dp(9), children)
	})
}

// detailSkips are fields the meta line leaves out (prototype's skip set).
var detailSkips = map[string]bool{"startEpisodes": true, "startPages": true, "cast": true, "url": true,
	"publisher": true, "genre": true, "duration": true, "episodeDuration": true, "durationText": true}

// progress is the bar and "Episode 4 of 16 · Netflix" line.
func (p *entryDetail) progress(gtx layout.Context, th *Theme, e *core.Entry, done, total int) layout.Dimensions {
	label := fmt.Sprintf("Page %d of %d", done, total)
	if e.Type.Progress() == core.EpisodeProgress {
		label = fmt.Sprintf("Episode %d of %d", done, total)
		if e.Fields.Platform != "" {
			label += " · " + e.Fields.Platform
		}
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			w, h := gtx.Constraints.Max.X, gtx.Dp(4)
			rrect(gtx, image.Pt(w, h), h/2, th.Border)
			if pct := e.ProgressPercent(); pct > 0 {
				rrect(gtx, image.Pt(w*pct/100, h), h/2, th.Accent2)
			}
			return layout.Dimensions{Size: image.Pt(w, h)}
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 5, Bottom: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Sans}, Size: 12, Color: th.Muted}.Layout(gtx, th, label)
			})
		}))
}

// yourWords is the quote with its date label and accent rule (.yw-item).
func yourWords(gtx layout.Context, th *Theme, label, quote string) layout.Dimensions {
	rec := op.Record(gtx.Ops)
	d := layout.Inset{Left: 15}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: 3}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.06, Upper: true, Color: th.Muted}.Layout(gtx, th, label)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 14, LineHeight: 1.78, Color: th.Text}.Layout(gtx, th, "“"+quote+"”")
			}))
	})
	call := rec.Stop()
	fillRect(gtx, image.Rect(0, 0, gtx.Dp(2), d.Size.Y), th.Accent2)
	call.Add(gtx.Ops)
	return d
}

// logButton is "Log a new session" (ongoing) or "Log a rewatch" and its
// kin (finished), in the .log-session-btn style.
func (p *entryDetail) logButton(gtx layout.Context, th *Theme, e *core.Entry) layout.Dimensions {
	ic, label := IconAlert, "Log a new session"
	if e.Status == core.Finished {
		ic = IconRewatch
		switch e.Type {
		case core.Book:
			label = "Log a reread"
		case core.Podcast:
			label = "Log another listen"
		case core.Music:
			label = "Log a relisten"
		default:
			label = "Log a rewatch"
		}
	}
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return p.logBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		bg, border := th.Tag, th.Border
		if p.logBtn.Pressed() {
			bg, border = th.Card, th.Accent
		}
		return card(gtx, bg, border, 11, layout.Inset{Top: 10, Bottom: 10, Left: 14, Right: 14}, 0, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ic.Layout(gtx, 16, 1.8, th.Accent2) }),
				layout.Rigid(layout.Spacer{Width: 8}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Display, Style: font.Italic}, Size: 13, Color: th.Accent}.Layout(gtx, th, label)
				}))
		})
	})
}

// journeyItem is one timeline row: dot and line, then the card with date,
// pill, edit / delete, label, stars and note.
func journeyItem(gtx layout.Context, a *App, e *core.Entry, it core.JourneyItem, b *journeyButtons, last bool) layout.Dimensions {
	th := a.Theme
	rec := op.Record(gtx.Ops)
	d := layout.Inset{Left: 15 + 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Bottom: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			border := th.Timeline
			if it.Rewatch {
				border = th.Border
			}
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return card(gtx, th.Timeline, border, 12, layout.Inset{Top: 11, Bottom: 11, Left: 13, Right: 13}, 0, func(gtx layout.Context) layout.Dimensions {
				var rows []layout.FlexChild
				rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Bottom: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, fillWidth(func(gtx layout.Context) layout.Dimensions {
								return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.06, Upper: true, Color: th.Muted}.Layout(gtx, th, core.FormatDate(it.At, a.Loc))
							})),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if it.Pill == "" {
									return layout.Dimensions{}
								}
								return layout.Inset{Right: 5}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return card(gtx, th.Tag, th.Tag, 20, layout.Inset{Top: 2, Bottom: 2, Left: 7, Right: 7}, 0, func(gtx layout.Context) layout.Dimensions {
										return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.06, Upper: true, Color: th.Accent}.Layout(gtx, th, it.Pill)
									})
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return smallAction(gtx, th, &b.edit, IconPen, false) }),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if !it.CanDelete {
									return layout.Dimensions{}
								}
								return layout.Inset{Left: 5}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return smallAction(gtx, th, &b.del, IconTrash, true)
								})
							}))
					})
				}))
				if it.Label != "" {
					rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return Text{Font: font.Font{Typeface: Sans, Weight: font.Medium}, Size: 12, Color: th.Accent}.Layout(gtx, th, it.Label)
						})
					}))
				}
				if it.Rewatch && it.Rating > 0 {
					rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return stars(gtx, th, it.Rating, 11) })
					}))
				}
				if it.Note != "" {
					rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 13, LineHeight: 1.65, Color: th.Text}.Layout(gtx, th, it.Note)
					}))
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
			})
		})
	})
	call := rec.Stop()
	// The line runs from under this dot to the next one.
	dot := gtx.Dp(15)
	if !last {
		fillRect(gtx, image.Rect(gtx.Dp(7), gtx.Dp(18), gtx.Dp(8), d.Size.Y+gtx.Dp(3)), th.Border)
	}
	dc := th.Accent2
	if it.Rewatch {
		dc = th.Accent
	}
	st := op.Offset(image.Pt(0, gtx.Dp(3))).Push(gtx.Ops)
	paint.FillShape(gtx.Ops, th.Bg, clip.Ellipse{Max: image.Pt(dot, dot)}.Op(gtx.Ops))
	b2 := gtx.Dp(2)
	paint.FillShape(gtx.Ops, dc, clip.Ellipse{Min: image.Pt(b2, b2), Max: image.Pt(dot-b2, dot-b2)}.Op(gtx.Ops))
	st.Pop()
	call.Add(gtx.Ops)
	return d
}

// smallAction is a 22dp timeline edit / delete button (.tl-action-btn).
func smallAction(gtx layout.Context, th *Theme, c *widget.Clickable, ic *Icon, danger bool) layout.Dimensions {
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		sz := gtx.Dp(22)
		col := th.Muted
		if c.Pressed() {
			rrect(gtx, image.Pt(sz, sz), gtx.Dp(7), th.Card)
			col = th.Accent
			if danger {
				col = dangerRed
			}
		}
		off := (sz - gtx.Dp(13)) / 2
		st := op.Offset(image.Pt(off, off)).Push(gtx.Ops)
		ic.Layout(gtx, 13, 2, col)
		st.Pop()
		return layout.Dimensions{Size: image.Pt(sz, sz)}
	})
}

// deleteButton is "Delete this entry", which asks for a second tap
// ("Tap again to confirm deletion") within 3 seconds, as the prototype.
func deleteButton(gtx layout.Context, th *Theme, c *widget.Clickable, armed bool) layout.Dimensions {
	label, fg, border := "Delete this entry", th.Muted, th.Border
	if armed || c.Pressed() {
		fg, border = dangerRed, dangerRed
	}
	if armed {
		label = "Tap again to confirm deletion"
	}
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return card(gtx, th.Bg, border, 10, layout.UniformInset(9), 0, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Sans}, Size: 12, Color: fg}.Layout(gtx, th, label)
			})
		})
	})
}

// alignedRow lays children in a wrapping row, vertically centred per line.
func alignedRow(gtx layout.Context, gap int, children []layout.Widget) layout.Dimensions {
	return flowCentered(gtx, gap, children)
}

func isRewatch(e *core.Entry, id string) bool {
	for _, r := range e.Rewatches {
		if r.ID == id {
			return true
		}
	}
	return false
}
