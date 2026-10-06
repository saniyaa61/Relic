package ui

// The Favorites tab and the All-Time Top 5 (SPEC §4.14, §5 "Favorites";
// prototype renderFavorites).

import (
	"errors"
	"image"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/saniyaa61/relic/core"
)

// IconTrophy is the prototype's Top 5 cup.
var IconTrophy = IconSVG(`<path d="M8 21h8M12 17v4M7 4h10l-1 8a4 4 0 01-8 0L7 4z"/><path d="M17 5h2a2 2 0 010 4h-1.3M7 5H5a2 2 0 000 4h1.3"/>`)

// glideEase is the prototype's Top 5 reorder curve.
var glideEase = cubicBezier(0.22, 0.8, 0.22, 1)

type favoritesPage struct {
	list   widget.List
	tabs   tabStrip
	catID  string
	top    widget.List
	rows   []widget.List
	tCards map[string]*topCard
	fCards map[string]*finishedCard

	// After a Top 5 change the cards glide from their old places.
	glideAt   time.Time
	glideFrom map[string]int
}

type topCard struct {
	poster, title       widget.Clickable
	left, right, remove widget.Clickable
}

func (p *favoritesPage) topCard(id string) *topCard {
	if p.tCards == nil {
		p.tCards = map[string]*topCard{}
	}
	if p.tCards[id] == nil {
		p.tCards[id] = &topCard{}
	}
	return p.tCards[id]
}

func (p *favoritesPage) favCard(id string) *finishedCard {
	if p.fCards == nil {
		p.fCards = map[string]*finishedCard{}
	}
	if p.fCards[id] == nil {
		p.fCards[id] = &finishedCard{withPlus: true}
	}
	return p.fCards[id]
}

// category returns the tab being shown: the last one picked, or the first.
func (p *favoritesPage) category(lib *core.Library) *core.Category {
	for _, c := range lib.Categories {
		if c.ID == p.catID {
			return c
		}
	}
	if len(lib.Categories) == 0 {
		return nil
	}
	p.catID = lib.Categories[0].ID
	return lib.Categories[0]
}

func (p *favoritesPage) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	lib := a.Lib
	header := []layout.Widget{func(gtx layout.Context) layout.Dimensions { return logoBar(gtx, th, nil) }}
	cat := p.category(lib)
	if cat == nil {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(header[0]),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return noCategories(gtx, th) }))
	}
	p.handle(gtx, a, cat)
	cat = p.category(lib) // a tab may have been picked

	names := make([]string, len(lib.Categories))
	sel := 0
	for i, c := range lib.Categories {
		names[i] = c.Name
		if c.ID == cat.ID {
			sel = i
		}
	}
	if i := p.tabs.Clicked(gtx, a); i >= 0 && i < len(lib.Categories) && i != sel {
		p.catID, sel, cat = lib.Categories[i].ID, i, lib.Categories[i]
		p.glideFrom = nil
		p.top.Position = layout.Position{}
		for i := range p.rows {
			p.rows[i].Position = layout.Position{}
		}
		p.tabs.scrollTo(a, i)
	}

	top := lib.TopFive(cat.ID)
	favs := lib.FavouritesIn(cat.ID)
	rows := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions { return p.tabs.Layout(gtx, a, names, sel) },
		func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 14}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return sectionHead(gtx, th, "All-Time Top 5 · "+cat.Name, "", nil)
			})
		},
	}
	if len(top) == 0 {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return emptyStyled(gtx, th, layout.Inset{Top: 22, Bottom: 8, Left: gutter, Right: gutter}, func(gtx layout.Context) layout.Dimensions {
				defer paint.PushOpacity(gtx.Ops, 0.8).Pop()
				return IconTrophy.Layout(gtx, 38, 1.2, th.Accent2)
			}, "Crown your all-time "+cat.Name+" favourites. Pick up to 5 from the list below — order is up to you.")
		})
	} else {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions { return p.topRow(gtx, a, top) })
	}
	rows = append(rows, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 22}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return sectionHead(gtx, th, "Favourites · "+cat.Name, "", nil)
		})
	})
	if len(favs) == 0 {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return emptyStyled(gtx, th, layout.Inset{Top: 20, Bottom: 20, Left: gutter, Right: gutter}, func(gtx layout.Context) layout.Dimensions {
				return IconFavorites.Layout(gtx, 40, 1, withAlpha(th.Border, 0.7))
			}, "No favourites in "+cat.Name+" yet. Tap the heart on an entry to save it here.")
		})
	} else {
		rows = append(rows, p.favRows(a, favs)...)
	}
	return scrollPage(gtx, &p.list, header, rows)
}

// handle acts on the card buttons before anything is drawn.
func (p *favoritesPage) handle(gtx layout.Context, a *App, cat *core.Category) {
	for id, c := range p.tCards {
		if c.poster.Clicked(gtx) || c.title.Clicked(gtx) {
			a.Push(newEntryDetail(id))
		}
		if c.left.Clicked(gtx) {
			p.changeTop(a, cat.ID, func() bool { a.Lib.MoveTopFive(id, -1); return a.save(false, nil, true) })
		}
		if c.right.Clicked(gtx) {
			p.changeTop(a, cat.ID, func() bool { a.Lib.MoveTopFive(id, 1); return a.save(false, nil, true) })
		}
		if c.remove.Clicked(gtx) {
			p.changeTop(a, cat.ID, func() bool { return removeTopFive(a, id) })
		}
	}
	for id, c := range p.fCards {
		if c.plus.Clicked(gtx) {
			p.changeTop(a, cat.ID, func() bool { return addTopFive(a, id) })
		}
	}
}

// changeTop runs a Top 5 change and, if it took, glides the cards from
// where they were (the prototype's flipRenderFavorites).
func (p *favoritesPage) changeTop(a *App, catID string, change func() bool) {
	before := map[string]int{}
	for i, e := range a.Lib.TopFive(catID) {
		before[e.ID] = i
	}
	if change() {
		p.glideFrom, p.glideAt = before, a.Now()
	}
}

// topRow is the sideways row of Top 5 cards (.top-grid).
func (p *favoritesPage) topRow(gtx layout.Context, a *App, top []*core.Entry) layout.Dimensions {
	step := gtx.Dp(130 + 10)
	t := float32(1)
	if p.glideFrom != nil {
		t = glideEase(progress(gtx, a.Now(), p.glideAt, 320*time.Millisecond))
		if t >= 1 {
			p.glideFrom = nil
		}
	}
	return layout.Inset{Top: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return cardRow(gtx, &p.top, 10, len(top), func(gtx layout.Context, i int) layout.Dimensions {
			e := top[i]
			if from, ok := p.glideFrom[e.ID]; ok && from != i {
				dx := float32((from - i) * step)
				defer op.Offset(image.Pt(roundi(dx*(1-t)), 0)).Push(gtx.Ops).Pop()
			}
			return p.topCard(e.ID).Layout(gtx, a, e, i, len(top))
		}, 8)
	})
}

// favRows chunks the favourites into rows of four (.fav-rows).
func (p *favoritesPage) favRows(a *App, es []*core.Entry) []layout.Widget {
	const per = 4
	n := (len(es) + per - 1) / per
	for len(p.rows) < n {
		p.rows = append(p.rows, widget.List{})
	}
	out := []layout.Widget{layout.Spacer{Height: 2}.Layout}
	for r := 0; r < n; r++ {
		row := es[r*per : min((r+1)*per, len(es))]
		bottom := float32(14)
		if r == n-1 {
			bottom = 10
		}
		out = append(out, func(gtx layout.Context) layout.Dimensions {
			return cardRow(gtx, &p.rows[r], 10, len(row), func(gtx layout.Context, i int) layout.Dimensions {
				return p.favCard(row[i].ID).Layout(gtx, a, row[i], false)
			}, bottom)
		})
	}
	return out
}

// Layout draws a Top 5 card (.top-card): 130dp wide, the 175dp poster,
// a two-line title, the type, and ‹ › ✕ to move it or take it out.
func (c *topCard) Layout(gtx layout.Context, a *App, e *core.Entry, i, n int) layout.Dimensions {
	th := a.Theme
	w := gtx.Dp(130)
	gtx.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, gtx.Constraints.Max.Y)}
	rec := op.Record(gtx.Ops)
	d := card(gtx, th.Card, th.Border, 13, layout.Inset{}, 130, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return c.poster.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					size := image.Pt(gtx.Constraints.Max.X, gtx.Dp(175))
					fillRect(gtx, image.Rectangle{Max: size}, th.Tag)
					if !a.drawPoster(gtx, e.Poster, size, 0) {
						icon := gtx.Dp(26)
						st := op.Offset(image.Pt((size.X-icon)/2, (size.Y-icon)/2)).Push(gtx.Ops)
						cardPosterIcon(e.Type).Layout(gtx, 26, 1.3, withAlpha(th.Muted, 0.7))
						st.Pop()
					}
					return layout.Dimensions{Size: size}
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 7, Bottom: 9, Left: 9, Right: 9}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							// Two lines, always 27.5px tall, so the buttons line up.
							return c.title.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								h := gtx.Dp(27.5)
								gtx.Constraints.Max.Y = h
								Paragraph{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 11, LineHeight: 1.25, MaxLines: 2, Color: th.Text}.Layout(gtx, th, e.Title)
								return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Top: 2, Bottom: 7}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return Text{Font: font.Font{Typeface: Sans}, Size: 10, LineHeight: 1.2, Tracking: 0.05, Upper: true, Color: th.Muted}.Layout(gtx, th, typeShort(e.Type))
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Spacing: layout.SpaceBetween}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return rankButton(gtx, th, &c.left, "‹", i == 0, false)
										}),
										layout.Rigid(layout.Spacer{Width: 2}.Layout),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return rankButton(gtx, th, &c.right, "›", i == n-1, false)
										}))
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return rankButton(gtx, th, &c.remove, "✕", false, true)
								}))
						}))
				})
			}))
	})
	call := rec.Stop()
	boxShadow(gtx, d.Size, gtx.Dp(13), 3, 10, 0.08, th.Bg)
	call.Add(gtx.Ops)
	return d
}

// rankButton is the small square ‹ › ✕ (.rank-btn); pressed it turns
// accent, or red for ✕.
func rankButton(gtx layout.Context, th *Theme, c *widget.Clickable, label string, disabled, danger bool) layout.Dimensions {
	size := gtx.Dp(20)
	draw := func(gtx layout.Context) layout.Dimensions {
		border, fg := th.Border, th.Muted
		if c.Pressed() && !disabled {
			border, fg = th.Accent, th.Accent
			if danger {
				border, fg = dangerRed, dangerRed
			}
		}
		rrect(gtx, image.Pt(size, size), gtx.Dp(6), border)
		b := gtx.Dp(1)
		st := op.Offset(image.Pt(b, b)).Push(gtx.Ops)
		rrect(gtx, image.Pt(size-2*b, size-2*b), gtx.Dp(5), th.Surface)
		st.Pop()
		rec := op.Record(gtx.Ops)
		lgtx := gtx
		lgtx.Constraints = layout.Constraints{Max: image.Pt(size, size)}
		d := Text{Font: font.Font{Typeface: Sans}, Size: 10, Color: fg}.Layout(lgtx, th, label)
		call := rec.Stop()
		st = op.Offset(image.Pt((size-d.Size.X)/2, (size-d.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		st.Pop()
		return layout.Dimensions{Size: image.Pt(size, size)}
	}
	gtx.Constraints = layout.Exact(image.Pt(size, size))
	if disabled {
		defer paint.PushOpacity(gtx.Ops, 0.3).Pop()
		return draw(gtx)
	}
	return c.Layout(gtx, draw)
}

// noCategories is the Favorites page before any category exists.
func noCategories(gtx layout.Context, th *Theme) layout.Dimensions {
	// The prototype centres this over the whole screen below the top bar,
	// behind its fixed bottom bar; ours ends at the bar, so add the bar's
	// height (66dp) on top to land in the same place.
	return layout.Inset{Top: 40 + 66, Bottom: 40, Left: 24, Right: 24}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle, Spacing: layout.SpaceSides}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: 16}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return IconFavorites.Layout(gtx, 48, 1.2, th.Border)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Display}, Size: 18, Color: th.Text}.Layout(gtx, th, "No categories yet")
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = 0
				gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(240))
				return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 13, LineHeight: 1.7, Alignment: textMiddle, Color: th.Muted}.Layout(gtx, th,
					"Set up your library categories, then favourite and rank the titles you love.")
			}))
	})
}

// ---------- Category tabs ----------

// tabStrip is the row of category pills (.stabs) that scrolls sideways;
// a picked tab glides into the middle (scrollIntoView, inline: center).
type tabStrip struct {
	list   widget.List
	clicks []widget.Clickable
	widths []int // each tab's slot, gutters included, from the last frame
	view   int

	animAt           time.Time
	animFrom, animTo int
}

// Clicked returns the tab tapped since the last frame, or -1.
func (s *tabStrip) Clicked(gtx layout.Context, a *App) int {
	for i := range s.clicks {
		if s.clicks[i].Clicked(gtx) {
			return i
		}
	}
	return -1
}

// scrollTo starts gliding tab i to the middle of the strip.
func (s *tabStrip) scrollTo(a *App, i int) {
	if i >= len(s.widths) {
		return
	}
	start, total := 0, 0
	for k, w := range s.widths {
		if k < i {
			start += w
		}
		total += w
	}
	to := start + s.widths[i]/2 - s.view/2
	to = max(0, min(to, total-s.view))
	from := 0
	for k := 0; k < s.list.Position.First && k < len(s.widths); k++ {
		from += s.widths[k]
	}
	from += s.list.Position.Offset
	s.animFrom, s.animTo, s.animAt = from, to, a.Now()
}

func (s *tabStrip) Layout(gtx layout.Context, a *App, names []string, sel int) layout.Dimensions {
	th := a.Theme
	for len(s.clicks) < len(names) {
		s.clicks = append(s.clicks, widget.Clickable{})
	}
	if len(s.widths) != len(names) {
		s.widths = make([]int, len(names))
	}
	s.view = gtx.Constraints.Max.X
	if !s.animAt.IsZero() {
		t := easeCSS(progress(gtx, a.Now(), s.animAt, 300*time.Millisecond))
		s.list.Position = layout.Position{Offset: roundi(lerpf(float32(s.animFrom), float32(s.animTo), t))}
		if t >= 1 {
			s.animAt = time.Time{}
		}
	}
	return layout.Inset{Top: 12, Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return cardRow(gtx, &s.list, 6, len(names), func(gtx layout.Context, i int) layout.Dimensions {
			d := tabPill(gtx, th, &s.clicks[i], names[i], i == sel)
			left, right := gtx.Dp(3), gtx.Dp(3)
			if i == 0 {
				left = gtx.Dp(gutter)
			}
			if i == len(names)-1 {
				right = gtx.Dp(gutter)
			}
			s.widths[i] = d.Size.X + left + right
			return d
		}, 0)
	})
}

// tabPill is one .stab: outlined, accent-filled when picked.
func tabPill(gtx layout.Context, th *Theme, c *widget.Clickable, label string, on bool) layout.Dimensions {
	bg, fg, border := th.Bg, th.Muted, th.Border
	if on {
		bg, fg, border = th.Accent, th.BtnText, th.Accent
	}
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return card(gtx, bg, border, 20, layout.Inset{Top: 4, Bottom: 4, Left: 12, Right: 12}, 0, func(gtx layout.Context) layout.Dimensions {
			return Text{Font: font.Font{Typeface: Sans}, Size: 12, Color: fg}.Layout(gtx, th, label)
		})
	})
}

// ---------- Favourite and Top 5 actions (shared with the entry page) ----------

// toggleFavourite flips the heart, saves and says so; it reports whether
// the entry is now a favourite.
func toggleFavourite(a *App, e *core.Entry) bool {
	fav, err := a.Lib.ToggleFavorite(e.ID)
	if err != nil || !a.save(false, []*core.Entry{e}, true) {
		return false
	}
	if fav {
		a.Toast("Added to favourites ♥")
	} else {
		a.Toast("Removed from favourites")
	}
	return fav
}

// addTopFive ranks a favourite at the end of its Top 5 and says so.
func addTopFive(a *App, id string) bool {
	if err := a.Lib.AddTopFive(id); err != nil {
		if errors.Is(err, core.ErrTopFiveFull) {
			a.ToastError(err.Error())
		}
		return false
	}
	if !a.save(false, nil, true) {
		return false
	}
	a.Toast("Added to your All-Time Top 5 ✦")
	return true
}

// removeTopFive takes an entry out of its Top 5 and says so.
func removeTopFive(a *App, id string) bool {
	if a.Lib.RemoveTopFive(id) != nil || !a.save(false, nil, true) {
		return false
	}
	a.Toast("Removed from All-Time Top 5")
	return true
}

// rankPrompt is the sheet's two buttons.
type rankPrompt struct{ rank, keep widget.Clickable }

// openRankPrompt offers a just-favourited entry a place in its Top 5,
// unless it already has one or the Top 5 is full (prototype openRankPrompt).
func openRankPrompt(a *App, id string) *rankPrompt {
	e := a.Lib.Entry(id)
	if e == nil || !e.Favorite || a.Lib.CategoryName(e.CategoryID) == "" {
		return nil
	}
	if _, ranked := a.Lib.TopFiveRank(id); ranked || len(a.Lib.TopFive(e.CategoryID)) >= core.TopFiveMax {
		return nil
	}
	rp := &rankPrompt{}
	rank, keep := &rp.rank, &rp.keep
	cat := a.Lib.CategoryName(e.CategoryID)
	a.ShowSheet(&Sheet{Body: func(gtx layout.Context, a *App) layout.Dimensions {
		th := a.Theme
		if rank.Clicked(gtx) {
			addTopFive(a, id)
			a.CloseSheet()
		}
		if keep.Clicked(gtx) {
			a.CloseSheet()
		}
		centred := func(w layout.Widget) layout.FlexChild {
			return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return layout.Center.Layout(gtx, w)
			})
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			centred(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 2, Bottom: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return IconTrophy.Layout(gtx, 32, 1.3, th.Accent2)
				})
			}),
			centred(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: 6, Left: 6, Right: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Paragraph{Font: font.Font{Typeface: Display}, Size: 16, LineHeight: 1.3, Alignment: textMiddle, Color: th.Text}.Layout(gtx, th, "Rank it among your all-time favourites?")
				})
			}),
			centred(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: 18, Left: 6, Right: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 13, LineHeight: 1.65, Alignment: textMiddle, Color: th.Muted}.Layout(gtx, th,
						"“"+e.Title+"” is now a favourite. Give it a place in your "+cat+" Top 5?")
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return bigButton(gtx, th, rank, "Rank it ✦")
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return pressable(gtx, keep, func(gtx layout.Context) layout.Dimensions {
					return card(gtx, th.Tag, th.Border, 10, layout.Inset{Top: 10, Bottom: 10, Left: 15, Right: 15}, 0, func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return Text{Font: font.Font{Typeface: Sans}, Size: 13, Color: th.Muted}.Layout(gtx, th, "Just keep as favourite")
						})
					})
				})
			}))
	}})
	return rp
}
