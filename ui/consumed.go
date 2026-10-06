package ui

// Consumed: where the time went (SPEC §5 "Consumed"; prototype
// renderConsumedRoot and renderConsumedFolder), opened from Home's
// Consumed chip.

import (
	"image"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/saniyaa61/relic/core"
)

// posterRows lays entries out four to a row (.fav-rows), each row
// scrolling sideways, with each card's time under it when showTime.
type posterRows struct {
	rows  []widget.List
	cards map[string]*finishedCard
}

func (p *posterRows) card(id string) *finishedCard {
	if p.cards == nil {
		p.cards = map[string]*finishedCard{}
	}
	if p.cards[id] == nil {
		p.cards[id] = &finishedCard{}
	}
	return p.cards[id]
}

func (p *posterRows) widgets(a *App, es []*core.Entry, showTime bool) []layout.Widget {
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
				return p.card(row[i].ID).Layout(gtx, a, row[i], showTime)
			}, bottom)
		})
	}
	return out
}

// folderKey names a Consumed folder group: category id and folder name
// ("" = Uncategorised).
type folderKey struct{ cat, folder string }

type consumedPage struct {
	back    IconButton
	search  Search
	list    widget.List
	results posterRows
	seeAll  map[folderKey]*widget.Clickable
	rows    map[folderKey]*widget.List
	cards   map[string]*finishedCard
}

func (p *consumedPage) WantsBack() bool { return p.search.WantsBack() }
func (p *consumedPage) Back(a *App)     { p.search.Back(a) }

func (p *consumedPage) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	lib := a.Lib
	x := a.timeIndex()
	cats := lib.ConsumedByCategory(x)
	for k, c := range p.seeAll {
		if c.Clicked(gtx) {
			a.Push(&consumedFolderPage{key: k})
		}
	}
	var search *Search
	if len(cats) > 0 {
		search = &p.search
	}
	header := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions { return subBar(gtx, a, &p.back, search) },
		func(gtx layout.Context) layout.Dimensions { return p.search.Bar(gtx, a, "Search your library…") },
	}
	if q := p.search.Query(); strings.TrimSpace(q) != "" {
		return scrollPage(gtx, &p.list, header, searchResultsWith(a, lib.Entries, q, "", IconSearch, 14, func(m []*core.Entry) []layout.Widget {
			return p.results.widgets(a, m, true)
		}))
	}

	var total float64
	for _, e := range lib.Entries {
		total += x.Total(e.ID)
	}
	m := core.MonthOf(a.Now(), a.Loc)
	month := x.SumInRange(lib.Entries, m.Start(a.Loc), m.End(a.Loc))
	weekly := x.Weekly(lib.Entries, a.Now(), 8)
	rows := []layout.Widget{func(gtx layout.Context) layout.Dimensions {
		return consumedHero(gtx, th, total, month, weekly, core.ShareBar(cats))
	}}
	if len(cats) == 0 {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return emptyState(gtx, th, IconClock, "Nothing logged yet.")
			})
		})
	}
	for _, ct := range cats {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			// 20px above (the hero's 4px below merges into it); its 10px
			// below merges with the folder heading's 12px.
			return layout.Inset{Top: 20}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return timeHead(gtx, th, ct.Category.Name, core.FormatDuration(ct.Minutes))
			})
		})
		for _, g := range ct.Folders {
			k := folderKey{ct.Category.ID, g.Name}
			see := p.click(k)
			preview := g.Entries[:min(4, len(g.Entries))]
			name := g.Name
			if name == "" {
				name = "Uncategorised"
			}
			rows = append(rows,
				func(gtx layout.Context) layout.Dimensions {
					return miniHead(gtx, th, name, core.FormatDuration(g.Minutes), see)
				},
				func(gtx layout.Context) layout.Dimensions {
					return cardRow(gtx, p.row(k), 10, len(preview), func(gtx layout.Context, i int) layout.Dimensions {
						return p.card(preview[i].ID).Layout(gtx, a, preview[i], true)
					}, 0)
				})
		}
	}
	rows = append(rows, layout.Spacer{Height: 14}.Layout)
	return scrollPage(gtx, &p.list, header, rows)
}

func (p *consumedPage) click(k folderKey) *widget.Clickable {
	if p.seeAll == nil {
		p.seeAll = map[folderKey]*widget.Clickable{}
	}
	if p.seeAll[k] == nil {
		p.seeAll[k] = &widget.Clickable{}
	}
	return p.seeAll[k]
}

func (p *consumedPage) row(k folderKey) *widget.List {
	if p.rows == nil {
		p.rows = map[folderKey]*widget.List{}
	}
	if p.rows[k] == nil {
		p.rows[k] = &widget.List{}
	}
	return p.rows[k]
}

func (p *consumedPage) card(id string) *finishedCard {
	if p.cards == nil {
		p.cards = map[string]*finishedCard{}
	}
	if p.cards[id] == nil {
		p.cards[id] = &finishedCard{}
	}
	return p.cards[id]
}

// timeHead is a category heading with its time on the right (.sec-hd).
func timeHead(gtx layout.Context, th *Theme, title, time string) layout.Dimensions {
	return layout.Inset{Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Alignment: layout.Baseline, Spacing: layout.SpaceBetween}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 16, Color: th.Text}.Layout(gtx, th, title)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Sans, Weight: font.Medium}, Size: 12, Color: th.Accent}.Layout(gtx, th, time)
			}))
	})
}

// miniHead is a folder heading: name, its time, and "See all" at the
// right (.mini-sec-hd).
func miniHead(gtx layout.Context, th *Theme, name, time string, see *widget.Clickable) layout.Dimensions {
	return layout.Inset{Top: 12, Bottom: 8, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Alignment: layout.Baseline}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 13.5, MaxLines: 1, Color: th.Text}.Layout(gtx, th, name)
			}),
			layout.Rigid(layout.Spacer{Width: 8}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Sans, Weight: font.Medium}, Size: 11, Color: th.Accent}.Layout(gtx, th, time)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return see.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return Text{Font: font.Font{Typeface: Sans}, Size: 11, Color: th.Accent2}.Layout(gtx, th, "See all")
					})
				})
			}))
	})
}

// consumedHero is "All time, all worlds" (.consumed-hero): the total, the
// month so far, the perspective line, the last 8 weeks and the share bar.
func consumedHero(gtx layout.Context, th *Theme, total, month float64, weekly []float64, share []core.ShareSlice) layout.Dimensions {
	// Its 4px bottom margin merges with what follows, which has more.
	return layout.Inset{Top: 14, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		// A 1.5px border, so drawn here rather than with card().
		rec := op.Record(gtx.Ops)
		bw := gtx.Dp(1.5)
		igtx := gtx
		igtx.Constraints.Max.X -= 2 * bw
		igtx.Constraints.Min.X = igtx.Constraints.Max.X
		st := op.Offset(image.Pt(bw, bw)).Push(gtx.Ops)
		d := layout.Inset{Top: 18, Bottom: 18, Left: 20, Right: 20}.Layout(igtx, func(gtx layout.Context) layout.Dimensions {
			return heroBody(gtx, th, total, month, weekly, share)
		})
		st.Pop()
		body := rec.Stop()
		size := image.Pt(gtx.Constraints.Max.X, d.Size.Y+2*bw)
		r := gtx.Dp(18)
		rrect(gtx, size, r, th.Border)
		st = op.Offset(image.Pt(bw, bw)).Push(gtx.Ops)
		rrect(gtx, size.Sub(image.Pt(2*bw, 2*bw)), r-bw, th.Card)
		st.Pop()
		body.Add(gtx.Ops)
		return layout.Dimensions{Size: size}
	})
}

func heroBody(gtx layout.Context, th *Theme, total, month float64, weekly []float64, share []core.ShareSlice) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	num, unit := core.SplitDuration(core.FormatDurationOrZero(total))
	children := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Spacing: layout.SpaceBetween}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.1, Upper: true, Color: th.Muted}.Layout(gtx, th, "All time, all worlds")
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Alignment: layout.Baseline}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return Text{Font: font.Font{Typeface: Display, Weight: font.SemiBold}, Size: 32, LineHeight: 1.05, Color: th.Accent}.Layout(gtx, th, num)
								}),
								layout.Rigid(layout.Spacer{Width: 8}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return Text{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 17, Color: th.Muted}.Layout(gtx, th, unit)
								}))
						}))
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if month <= 0 {
						return layout.Dimensions{}
					}
					return layout.Inset{Left: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return card(gtx, th.Tag, th.Tag, 20, layout.Inset{Top: 2, Bottom: 2, Left: 9, Right: 9}, 0, func(gtx layout.Context) layout.Dimensions {
							return Text{Font: font.Font{Typeface: Sans, Weight: font.Medium}, Size: 11.5, Color: th.Accent}.Layout(gtx, th, "+"+core.FormatDuration(month)+" this month")
						})
					})
				}))
		}),
	}
	if p := core.Perspective(total); p != "" {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 13, LineHeight: 1.5, Color: th.Muted}.Layout(gtx, th, p)
			})
		}))
	}
	maxW := 0.0
	for _, w := range weekly {
		maxW = max(maxW, w)
	}
	if maxW > 0 {
		children = append(children,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 14}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return sparkline(gtx, th, weekly, maxW)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Sans}, Size: 10.5, Color: th.Muted}.Layout(gtx, th, "Last 8 weeks")
				})
			}))
	}
	var shareTotal float64
	for _, s := range share {
		shareTotal += s.Minutes
	}
	if shareTotal > 0 {
		colors := []colorNRGBA{th.Accent, th.Accent2, th.Muted, th.Border}
		children = append(children,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 14}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return shareBar(gtx, share, shareTotal, colors)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				items := make([]layout.Widget, len(share))
				for i, s := range share {
					items[i] = func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								sz := gtx.Dp(8)
								rrect(gtx, image.Pt(sz, sz), sz/2, colors[i])
								return layout.Dimensions{Size: image.Pt(sz, sz)}
							}),
							layout.Rigid(layout.Spacer{Width: 6}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return Text{Font: font.Font{Typeface: Sans}, Size: 11.5, Color: th.Muted}.Layout(gtx, th, s.Name+" "+core.FormatDuration(s.Minutes))
							}))
					}
				}
				return layout.Inset{Top: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return flowGaps(gtx, gtx.Dp(14), gtx.Dp(4), items)
				})
			}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// sparkline is the last 8 weeks (.consumed-spark): faded accent2 bars,
// this week in accent.
func sparkline(gtx layout.Context, th *Theme, weekly []float64, maxW float64) layout.Dimensions {
	w, h := gtx.Constraints.Max.X, gtx.Dp(30)
	gap := gtx.Dp(4)
	n := len(weekly)
	faded := mixSRGB(th.Accent2, th.Card, 0.45)
	for i, m := range weekly {
		x0 := (w + gap) * i / n
		x1 := (w+gap)*(i+1)/n - gap
		bh := int(float64(h) * max(m/maxW, 0.06))
		c := faded
		if i == n-1 {
			c = th.Accent
		}
		st := op.Offset(image.Pt(x0, h-bh)).Push(gtx.Ops)
		rrect(gtx, image.Pt(x1-x0, bh), min(gtx.Dp(3), bh/2), c)
		st.Pop()
	}
	return layout.Dimensions{Size: image.Pt(w, h)}
}

// shareBar is the category split (.consumed-seg): an 8px rounded bar of
// segments 2px apart, each at least 3px.
func shareBar(gtx layout.Context, share []core.ShareSlice, total float64, colors []colorNRGBA) layout.Dimensions {
	w, h := gtx.Constraints.Max.X, gtx.Dp(8)
	gap := gtx.Dp(2)
	avail := w - gap*(len(share)-1)
	defer clip.UniformRRect(image.Rectangle{Max: image.Pt(w, h)}, h/2).Push(gtx.Ops).Pop()
	x := 0
	for i, s := range share {
		sw := max(int(float64(avail)*s.Minutes/total+0.5), gtx.Dp(3))
		if i == len(share)-1 {
			sw = w - x
		}
		r := clip.Rect(image.Rect(x, 0, x+sw, h)).Push(gtx.Ops)
		paint.ColorOp{Color: colors[i]}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		r.Pop()
		x += sw + gap
	}
	return layout.Dimensions{Size: image.Pt(w, h)}
}

// flowGaps is flow with different gaps across (colGap) and down (rowGap).
func flowGaps(gtx layout.Context, colGap, rowGap int, children []layout.Widget) layout.Dimensions {
	maxW := gtx.Constraints.Max.X
	cgtx := gtx
	cgtx.Constraints.Min = image.Point{}
	x, y, rowH, w := 0, 0, 0, 0
	for _, ch := range children {
		rec := op.Record(gtx.Ops)
		d := ch(cgtx)
		call := rec.Stop()
		if x > 0 && x+d.Size.X > maxW {
			x, y = 0, y+rowH+rowGap
			rowH = 0
		}
		st := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		st.Pop()
		x += d.Size.X + colGap
		rowH = max(rowH, d.Size.Y)
		w = max(w, x-colGap)
	}
	return layout.Dimensions{Size: image.Pt(w, y+rowH)}
}

// ---------- A Consumed folder ("See all") ----------

type consumedFolderPage struct {
	key    folderKey
	back   IconButton
	search Search
	list   widget.List
	grid   posterRows
}

func (p *consumedFolderPage) WantsBack() bool { return p.search.WantsBack() }
func (p *consumedFolderPage) Back(a *App)     { p.search.Back(a) }

func (p *consumedFolderPage) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	es, ok := a.Lib.ConsumedFolder(p.key.cat, p.key.folder)
	c := a.Lib.Category(p.key.cat)
	if !ok {
		a.Pop()
		return layout.Dimensions{}
	}
	label := p.key.folder
	if label == "" {
		label = "Uncategorised"
	}
	var search *Search
	if len(es) > 0 {
		search = &p.search
	}
	header := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions { return subBar(gtx, a, &p.back, search) },
		func(gtx layout.Context) layout.Dimensions { return p.search.Bar(gtx, a, "Search in "+label+"…") },
	}
	if q := p.search.Query(); strings.TrimSpace(q) != "" {
		return scrollPage(gtx, &p.list, header, searchResultsWith(a, es, q, label, IconSearch, 14, func(m []*core.Entry) []layout.Widget {
			return p.grid.widgets(a, m, true)
		}))
	}
	x := a.timeIndex()
	rows := []layout.Widget{func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: -8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return pageHead(gtx, th, c.Name+" · Time spent", label,
				core.FormatDurationOrZero(x.Sum(es))+" across "+plural(len(es), "title", "titles"))
		})
	}}
	if len(es) == 0 {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return emptyState(gtx, th, IconClock, "Nothing logged here yet.")
		})
	} else {
		rows = append(rows, p.grid.widgets(a, es, true)...)
	}
	rows = append(rows, layout.Spacer{Height: 14}.Layout)
	return scrollPage(gtx, &p.list, header, rows)
}
