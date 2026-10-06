package ui

// Mood Trends (SPEC §4.13, §5 "Mood Trends"; prototype renderMoods).

import (
	"fmt"
	"image"
	"strconv"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"

	"github.com/saniyaa61/relic/core"
)

// IconSmile is the empty state's face.
var IconSmile = IconSVG(`<circle cx="12" cy="12" r="10"/><path d="M8 14s1.5 2 4 2 4-2 4-2"/><line x1="9" y1="9" x2="9.01" y2="9"/><line x1="15" y1="9" x2="15.01" y2="9"/>`)

type moodsPage struct {
	back IconButton
	list widget.List
}

func (p *moodsPage) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	d := a.Lib.MoodTrends(a.Now(), a.Loc)
	sub := "Based on the feelings you tagged when you logged each entry"
	switch n := len(d.Months); {
	case n == 12:
		sub += ", over the last 12 months"
	case n > 0:
		sub += ", since " + d.Months[0].String()
	}
	sub += "."
	header := []layout.Widget{func(gtx layout.Context) layout.Dimensions { return subBar(gtx, a, &p.back, nil) }}
	rows := []layout.Widget{func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: -8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return pageHead(gtx, th, "Mood trends", "How you've felt", sub)
		})
	}}
	if len(d.Tags) == 0 {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 30}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return emptyState(gtx, th, IconSmile, "Tag how an entry felt when you log it, and your mood trends will appear here.")
			})
		})
		return scrollPage(gtx, &p.list, header, append(rows, layout.Spacer{Height: 14}.Layout))
	}
	if ins := d.Insights(); len(ins) > 0 {
		lines := ""
		for i, in := range ins {
			if i > 0 {
				lines += "\n"
			}
			lines += in.Before + in.Tag + in.After
		}
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return dcard(gtx, th, nil, "What stands out", "", lines)
		})
	}
	rows = append(rows, func(gtx layout.Context) layout.Dimensions { return feelingsCard(gtx, th, d) })

	var months []layout.Widget
	for i := len(d.Months) - 1; i >= 0; i-- {
		top := d.MonthTop(i, 3)
		if len(top) == 0 {
			continue
		}
		m := d.Months[i]
		months = append(months, func(gtx layout.Context) layout.Dimensions { return moodMonth(gtx, th, m, top) })
	}
	if len(months) > 0 {
		rows = append(rows,
			func(gtx layout.Context) layout.Dimensions {
				// Its 6px top margin merges with the card's 10px above.
				return sectionHead(gtx, th, "Month by month", "", nil)
			},
			func(gtx layout.Context) layout.Dimensions {
				return ruledCard(gtx, th, layout.UniformInset(15), months)
			})
	}
	rows = append(rows, layout.Spacer{Height: 14}.Layout)
	return scrollPage(gtx, &p.list, header, rows)
}

// ruledCard is a dcard whose rows are split by hairlines (.mood-row,
// .mood-month: 11px above and below each, no line after the last).
func ruledCard(gtx layout.Context, th *Theme, pad layout.Inset, rows []layout.Widget) layout.Dimensions {
	return layout.Inset{Bottom: 10, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return card(gtx, th.Card, th.Border, 14, pad, 0, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			children := make([]layout.FlexChild, 0, len(rows))
			for i, r := range rows {
				last := i == len(rows)-1
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					d := layout.Inset{Top: 11, Bottom: 11}.Layout(gtx, r)
					d.Size.X = gtx.Constraints.Max.X
					if !last {
						d.Size.Y += gtx.Dp(1)
						fillRect(gtx, image.Rect(0, d.Size.Y-gtx.Dp(1), d.Size.X, d.Size.Y), th.Border)
					}
					return d
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}

// feelingsCard is "Feelings over time": a row of month bars per feeling,
// its peak month in accent, and the month initials underneath.
func feelingsCard(gtx layout.Context, th *Theme, d core.MoodData) layout.Dimensions {
	shown := d.Shown()
	maxCount := max(1, d.MaxCount())
	n := len(d.Months)
	// bars lays out n equal columns 3dp apart across the width.
	bars := func(gtx layout.Context, h int, draw func(i int, r image.Rectangle)) {
		gap := gtx.Dp(3)
		w := gtx.Constraints.Max.X
		for i := range n {
			x0 := (w + gap) * i / n
			x1 := (w+gap)*(i+1)/n - gap
			draw(i, image.Rect(x0, 0, x1, h))
		}
	}
	var rows []layout.Widget
	for _, t := range shown {
		peak := t.Peak()
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					w := gtx.Dp(92)
					gtx.Constraints = layout.Constraints{Max: image.Pt(w, gtx.Constraints.Max.Y)}
					d := Text{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 13, MaxLines: 1, Color: th.Text}.Layout(gtx, th, t.Tag)
					d.Size.X = w
					return d
				}),
				layout.Rigid(layout.Spacer{Width: 10}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					h := gtx.Dp(28)
					bars(gtx, h, func(i int, r image.Rectangle) {
						c := t.Counts[i]
						frac := 0.08
						col := th.Tag
						if c > 0 {
							frac = max(float64(c)/float64(maxCount), 0.16)
							col = th.Accent2
							if c == peak {
								col = th.Accent
							}
						}
						bh := int(float64(h) * frac)
						r.Min.Y = h - bh
						st := op.Offset(r.Min).Push(gtx.Ops)
						rrect(gtx, r.Size(), min(gtx.Dp(2), r.Dx()/2), col)
						st.Pop()
					})
					return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
				}),
				layout.Rigid(layout.Spacer{Width: 10}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints = layout.Constraints{Min: image.Pt(gtx.Dp(24), 0), Max: image.Pt(gtx.Dp(24), gtx.Constraints.Max.Y)}
					return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return Text{Font: font.Font{Typeface: Sans, Weight: font.Medium}, Size: 12, Color: th.Accent}.Layout(gtx, th, strconv.Itoa(t.Total))
					})
				}))
		})
	}
	axis := func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 2, Left: 102, Right: 34}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			hMax := 0
			bars(gtx, 0, func(i int, r image.Rectangle) {
				rec := op.Record(gtx.Ops)
				lgtx := gtx
				lgtx.Constraints = layout.Constraints{Max: image.Pt(max(r.Dx(), gtx.Dp(3)), gtx.Constraints.Max.Y)}
				ld := Text{Font: font.Font{Typeface: Sans}, Size: 8.5, Color: th.Muted}.Layout(lgtx, th, d.Months[i].Initial())
				call := rec.Stop()
				st := op.Offset(image.Pt(r.Min.X+(r.Dx()-ld.Size.X)/2, 0)).Push(gtx.Ops)
				call.Add(gtx.Ops)
				st.Pop()
				hMax = max(hMax, ld.Size.Y)
			})
			return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, hMax)}
		})
	}
	return layout.Inset{Bottom: 10, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return card(gtx, th.Card, th.Border, 14, layout.UniformInset(15), 0, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			children := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					// The rows' 6px top margin merges with the eyebrow's 4px.
					return layout.Inset{Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.07, Upper: true, Color: th.Muted}.Layout(gtx, th, "Feelings over time")
					})
				}),
			}
			for i, r := range rows {
				last := i == len(rows)-1
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					d := layout.Inset{Top: 11, Bottom: 11}.Layout(gtx, r)
					d.Size.X = gtx.Constraints.Max.X
					if !last {
						d.Size.Y += gtx.Dp(1)
						fillRect(gtx, image.Rect(0, d.Size.Y-gtx.Dp(1), d.Size.X, d.Size.Y), th.Border)
					}
					return d
				}))
			}
			children = append(children, layout.Rigid(axis))
			if !d.EnoughMonths() {
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 13, LineHeight: 1.65, Color: th.Muted}.Layout(gtx, th,
							"Trends take shape after a couple of months of logging.")
					})
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}

// moodMonth is one "Month by month" row: the month and its top feelings.
func moodMonth(gtx layout.Context, th *Theme, m core.Month, top []core.TagCount) layout.Dimensions {
	chips := make([]layout.Widget, len(top))
	for i, t := range top {
		label := t.Tag
		if t.Count > 1 {
			label += fmt.Sprintf(" · %d", t.Count)
		}
		chips[i] = func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return card(gtx, th.Card, th.Border, 20, layout.Inset{Top: 3, Bottom: 3, Left: 11, Right: 11}, 0, func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Sans}, Size: 12, Color: th.Accent}.Layout(gtx, th, label)
				})
			})
		}
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.08, Upper: true, Color: th.Muted}.Layout(gtx, th, m.String())
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return flow(gtx, gtx.Dp(6), chips) }))
}
