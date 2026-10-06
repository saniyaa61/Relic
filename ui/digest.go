package ui

// The Digest tab (SPEC §4.12, §5 "Digest"; prototype renderDigest).

import (
	"fmt"
	"image"
	"strconv"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/saniyaa61/relic/core"
)

// yearBarStyle is how the Year digest's month bars are coloured. The
// owner chose the prototype's gradient (A) on 2026-10-06, an agreed
// exception to "no gradients"; the other two stay for snapshots.
type yearBarStyle int

const (
	barsGradient yearBarStyle = iota // A: the prototype's accent2→accent gradient
	barsPeak                         // B: flat accent2, the peak month in accent (like Mood Trends)
	barsFlat                         // C: flat accent, no highlight
)

var yearBars = barsGradient

var (
	IconStar  = IconSVG(`<polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>`)
	IconShare = IconSVG(`<path d="M4 12v7a2 2 0 002 2h12a2 2 0 002-2v-7"/><polyline points="16 6 12 2 8 6"/><line x1="12" y1="2" x2="12" y2="15"/>`)
	// iconDigestEmpty is the digest icon without its lines (the empty state).
	iconDigestEmpty = IconSVG(`<path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/>`)
)

type digestPage struct {
	list     widget.List
	kind     core.PeriodKind
	month    core.Month // zero: this month
	year     int        // zero: this year
	toggle   [3]widget.Clickable
	prev     widget.Clickable
	next     widget.Clickable
	moods    widget.Clickable
	share    widget.Clickable
	topRated [3]widget.Clickable
}

func newDigestPage() *digestPage { return &digestPage{kind: core.MonthPeriod} }

// period is the span on show, kept within the navigation bounds.
func (p *digestPage) period(a *App, b core.DigestBounds) core.Period {
	switch p.kind {
	case core.WeekPeriod:
		return core.LastWeek(a.Now())
	case core.MonthPeriod:
		if p.month == (core.Month{}) || b.Ceiling.Before(p.month) || p.month.Before(b.Floor) {
			p.month = b.Ceiling
		}
		return core.MonthOfDigest(p.month, a.Loc)
	}
	if p.year == 0 || p.year > b.Ceiling.Year || p.year < b.Floor.Year {
		p.year = b.Ceiling.Year
	}
	return core.YearOfDigest(p.year, a.Loc)
}

func (p *digestPage) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	lib := a.Lib
	header := []layout.Widget{func(gtx layout.Context) layout.Dimensions { return logoBar(gtx, th, nil) }}
	if len(lib.Entries) == 0 {
		return scrollPage(gtx, &p.list, header, []layout.Widget{func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 60}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return emptyState(gtx, th, iconDigestEmpty, "Your digest will come alive once you start logging entries. Come back here to see your story.")
			})
		}})
	}

	b := lib.DigestBounds(a.Now(), a.Loc)
	for i := range p.toggle {
		if p.toggle[i].Clicked(gtx) && core.PeriodKind(i) != p.kind {
			// Switching resets to the current month / year (prototype).
			p.kind, p.month, p.year = core.PeriodKind(i), core.Month{}, 0
			p.list.Position = layout.Position{}
		}
	}
	per := p.period(a, b)
	canPrev, canNext := false, false
	switch p.kind {
	case core.MonthPeriod:
		canPrev, canNext = b.CanStepMonth(p.month, -1), b.CanStepMonth(p.month, 1)
		if p.prev.Clicked(gtx) {
			p.month = b.StepMonth(p.month, -1)
		}
		if p.next.Clicked(gtx) {
			p.month = b.StepMonth(p.month, 1)
		}
	case core.YearPeriod:
		canPrev, canNext = b.CanStepYear(p.year, -1), b.CanStepYear(p.year, 1)
		if p.prev.Clicked(gtx) {
			p.year = b.StepYear(p.year, -1)
		}
		if p.next.Clicked(gtx) {
			p.year = b.StepYear(p.year, 1)
		}
	}
	per = p.period(a, b)
	if p.moods.Clicked(gtx) {
		a.Push(&moodsPage{})
	}
	if p.share.Clicked(gtx) {
		openYearCard(a, per.Year)
	}

	x := a.timeIndex()
	st := lib.Digest(x, per)
	streak, longest := core.Streak(lib.Entries, a.Now(), a.Loc)
	word := map[core.PeriodKind]string{core.WeekPeriod: "week", core.MonthPeriod: "month", core.YearPeriod: "year"}[p.kind]

	label := "This Week"
	switch p.kind {
	case core.MonthPeriod:
		label = per.Month.String()
	case core.YearPeriod:
		label = strconv.Itoa(per.Year)
	}
	eyebrow := map[core.PeriodKind]string{core.WeekPeriod: "Weekly digest", core.MonthPeriod: "Monthly digest", core.YearPeriod: "Year in review"}[p.kind]

	rows := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions { return p.toggleBar(gtx, th) },
	}
	if p.kind != core.WeekPeriod {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 14, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return digestNav(gtx, th, &p.prev, &p.next, label, canPrev, canNext)
			})
		})
	}
	rows = append(rows,
		func(gtx layout.Context) layout.Dimensions {
			return digestHero(gtx, th, eyebrow, label, lib.DigestNarrative(st, per, a.Now(), streak))
		},
		func(gtx layout.Context) layout.Dimensions { return p.stats(gtx, a, st, word, streak, longest) },
	)
	if p.kind == core.YearPeriod {
		monthly := x.Monthly(lib.Entries, per.Year, a.Loc)
		rows = append(rows, func(gtx layout.Context) layout.Dimensions { return yearChart(gtx, th, monthly) })
	}
	if len(st.TopRated) > 0 {
		e := st.TopRated[0]
		rows = append(rows, func(gtx layout.Context) layout.Dimensions { return bestCard(gtx, a, e, word) })
	}
	if p.kind == core.YearPeriod && len(st.TopRated) > 0 {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			// Its 4px top margin merges with the card's 10px above.
			return sectionHead(gtx, th, "Top rated this year", "", nil)
		})
		for i, e := range st.TopRated {
			rows = append(rows, func(gtx layout.Context) layout.Dimensions { return p.rankRow(gtx, a, i, e) })
		}
		rows = append(rows, layout.Spacer{Height: 2}.Layout)
	}
	if p.kind != core.YearPeriod {
		if e := lib.StillGoing(); e != nil {
			rows = append(rows, func(gtx layout.Context) layout.Dimensions {
				return dcard(gtx, th, nil, "Still going", e.Title, e.ProgressSummary())
			})
		}
		if n := st.Sessions; n > 0 {
			rows = append(rows, func(gtx layout.Context) layout.Dimensions {
				return dcard(gtx, th, nil, "Sessions this "+word, fmt.Sprintf("%d %s logged", n, pluralWord(n, "session", "sessions")),
					fmt.Sprintf("You sat down and lived inside something %d %s this %s. That counts for something.", n, pluralWord(n, "time", "times"), word))
			})
		}
	}
	if p.kind == core.YearPeriod && !per.Start.After(a.Now()) && (st.Minutes > 0 || len(st.New) > 0) {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return linkCard(gtx, th, &p.share, "Year in review card", "Share your "+strconv.Itoa(per.Year),
				"Turn this year into an image you can save or send.", IconShare, 18, 1.8, th.Accent)
		})
	}
	if p.kind != core.WeekPeriod {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return linkCard(gtx, th, &p.moods, "Mood trends", "How your feelings have shifted",
				"See which feelings keep returning, and which are fading.", IconChevronRight, 16, 2, th.Muted)
		})
	}
	rows = append(rows,
		func(gtx layout.Context) layout.Dimensions {
			return dcard(gtx, th, nil, "A note from Relic", "You kept something that mattered.",
				"Every entry here is a little act of remembrance. You're building a library that is entirely yours — one feeling at a time.")
		},
		layout.Spacer{Height: 4}.Layout,
	)
	return scrollPage(gtx, &p.list, header, rows)
}

func pluralWord(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// toggleBar is the Week / Month / Year switch (.digest-toggle).
func (p *digestPage) toggleBar(gtx layout.Context, th *Theme) layout.Dimensions {
	return layout.Inset{Top: 12, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return card(gtx, th.Tag, th.Tag, 11, layout.UniformInset(2), 0, func(gtx layout.Context) layout.Dimensions {
			gap := gtx.Dp(3)
			w := (gtx.Constraints.Max.X - 2*gap) / 3
			h := 0
			for i, name := range []string{"Week", "Month", "Year"} {
				on := core.PeriodKind(i) == p.kind
				st := op.Offset(image.Pt(i*(w+gap), 0)).Push(gtx.Ops)
				cgtx := gtx
				cgtx.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, gtx.Constraints.Max.Y)}
				d := p.toggle[i].Layout(cgtx, func(gtx layout.Context) layout.Dimensions {
					bg, fg := th.Tag, th.Muted
					if on {
						bg, fg = th.Card, th.Accent
					}
					return card(gtx, bg, bg, 8, layout.Inset{Top: 6, Bottom: 6}, 0, func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return Text{Font: font.Font{Typeface: Sans}, Size: 12, Color: fg}.Layout(gtx, th, name)
						})
					})
				})
				st.Pop()
				h = max(h, d.Size.Y)
			}
			return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
		})
	})
}

// roundArrow is the digest's round ‹ › button (.dnav-btn); disabled ones
// fade to 35% and ignore taps.
func roundArrow(gtx layout.Context, th *Theme, b *widget.Clickable, ic *Icon, disabled bool) layout.Dimensions {
	draw := func(gtx layout.Context) layout.Dimensions {
		sz := gtx.Dp(30)
		border, fg := th.Border, th.Muted
		if b.Pressed() && !disabled {
			border, fg = th.Accent2, th.Accent
		}
		gtx.Constraints = layout.Exact(image.Pt(sz, sz))
		return card(gtx, th.Card, border, 15, layout.Inset{}, 0, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return ic.Layout(gtx, 14, 2, fg) })
		})
	}
	if disabled {
		defer paint.PushOpacity(gtx.Ops, 0.35).Pop()
		return draw(gtx)
	}
	return b.Layout(gtx, draw)
}

// digestNav is ‹ September 2026 › (.dnav).
func digestNav(gtx layout.Context, th *Theme, prev, next *widget.Clickable, label string, canPrev, canNext bool) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return layout.Flex{Alignment: layout.Middle, Spacing: layout.SpaceSides}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return roundArrow(gtx, th, prev, IconChevronLeft, !canPrev)
		}),
		layout.Rigid(layout.Spacer{Width: 16}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Dp(130)
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Display, Weight: font.SemiBold}, Size: 14, Color: th.Text}.Layout(gtx, th, label)
			})
		}),
		layout.Rigid(layout.Spacer{Width: 16}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return roundArrow(gtx, th, next, IconChevronRight, !canNext)
		}),
	)
}

// digestHero is the period's card (.dhero): eyebrow, label, narrative.
func digestHero(gtx layout.Context, th *Theme, eyebrow, title, body string) layout.Dimensions {
	return layout.Inset{Top: 12, Bottom: 12, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return card(gtx, th.Card, th.Border, 16, layout.Inset{Top: 18, Bottom: 18, Left: 16, Right: 16}, 0, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Bottom: 3}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.08, Upper: true, Color: th.Muted}.Layout(gtx, th, eyebrow)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Bottom: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return Paragraph{Font: font.Font{Typeface: Display, Weight: font.SemiBold}, Size: 20, LineHeight: 1.2, Color: th.Text}.Layout(gtx, th, title)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 13, LineHeight: 1.75, Color: th.Muted}.Layout(gtx, th, body)
				}))
		})
	})
}

// stats is the 2×2 grid (.dgrid): new, time, pages, and the streak (or
// sessions) — finished on the Year page.
func (p *digestPage) stats(gtx layout.Context, a *App, st core.DigestStats, word string, streak, longest int) layout.Dimensions {
	th := a.Theme
	dash := func(n int) string {
		if n == 0 {
			return "—"
		}
		return strconv.Itoa(n)
	}
	type stat struct {
		value, label, sub string
		flame             bool
	}
	var cells [4]stat
	if p.kind == core.YearPeriod {
		cells = [4]stat{
			{dash(len(st.New)), "New this year", "entries started", false},
			{core.FormatDurationOrZero(st.Minutes), "Time this year", "in other worlds", false},
			{dash(st.Pages), "Pages this year", "across all books", false},
			{dash(len(st.Finished)), "Finished this year", "stories completed", false},
		}
	} else {
		sub := "nothing yet"
		switch n := len(st.New); {
		case n == 1:
			sub = "1 story added"
		case n > 1:
			sub = "entries logged"
		}
		last := stat{dash(st.Sessions), "Sessions", "this " + word, false}
		if streak > 0 {
			last = stat{strconv.Itoa(streak), "Day streak", fmt.Sprintf("best: %d days", longest), true}
		}
		cells = [4]stat{
			{dash(len(st.New)), "New this " + word, sub, false},
			{core.FormatDurationOrZero(st.Minutes), "Time this " + word, "in other worlds", false},
			{dash(st.Pages), "Pages this " + word, "across all books", false},
			last,
		}
	}
	cell := func(s stat) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			return card(gtx, th.Card, th.Border, 13, layout.UniformInset(13), 0, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						value := func(gtx layout.Context) layout.Dimensions {
							return Text{Font: font.Font{Typeface: Display, Weight: font.SemiBold}, Size: 22, LineHeight: 1, Color: th.Accent}.Layout(gtx, th, s.value)
						}
						if !s.flame {
							return value(gtx)
						}
						// The 🔥 may be taller than the 22px line; like the
						// browser's, it overflows instead of growing the card.
						rec := op.Record(gtx.Ops)
						fd := Emoji(gtx, "1f525", 22)
						flame := rec.Stop()
						st := op.Offset(image.Pt(fd.Size.X, 0)).Push(gtx.Ops)
						vd := value(gtx)
						st.Pop()
						st = op.Offset(image.Pt(0, (vd.Size.Y-fd.Size.Y)/2)).Push(gtx.Ops)
						flame.Add(gtx.Ops)
						st.Pop()
						return layout.Dimensions{Size: image.Pt(fd.Size.X+vd.Size.X, vd.Size.Y), Baseline: vd.Baseline}
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: 3}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return Text{Font: font.Font{Typeface: Sans}, Size: 10, LineHeight: normalLineHeight[Sans], Tracking: 0.05, Upper: true, Color: th.Muted}.Layout(gtx, th, s.label)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: 1}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return Text{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 11, LineHeight: normalLineHeight[Serif], Color: th.Muted}.Layout(gtx, th, s.sub)
						})
					}))
			})
		}
	}
	return layout.Inset{Bottom: 10, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gap := gtx.Dp(9)
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return gridRow(gtx, gap, 0, []layout.Widget{cell(cells[0]), cell(cells[1])})
			}),
			layout.Rigid(layout.Spacer{Height: 9}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return gridRow(gtx, gap, 0, []layout.Widget{cell(cells[2]), cell(cells[3])})
			}))
	})
}

// dcard is a digest card (.dcard): eyebrow, Playfair title, italic body.
// lead, if given, sits to the left (the best entry's poster).
func dcard(gtx layout.Context, th *Theme, lead layout.Widget, eyebrow, title, body string, extra ...layout.Widget) layout.Dimensions {
	return layout.Inset{Bottom: 10, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return card(gtx, th.Card, th.Border, 14, layout.UniformInset(15), 0, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			text := func(gtx layout.Context) layout.Dimensions {
				children := []layout.FlexChild{
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Bottom: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.07, Upper: true, Color: th.Muted}.Layout(gtx, th, eyebrow)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						// Without a title the body's 4px top margin merges with
						// the eyebrow's ("What stands out").
						if title == "" {
							return layout.Dimensions{}
						}
						return layout.Inset{Bottom: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return Paragraph{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 14, Color: th.Text}.Layout(gtx, th, title)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 13, LineHeight: 1.65, Color: th.Muted}.Layout(gtx, th, body)
					}),
				}
				for _, w := range extra {
					children = append(children, layout.Rigid(w))
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
			}
			if lead == nil {
				return text(gtx)
			}
			return layout.Flex{}.Layout(gtx,
				layout.Rigid(lead),
				layout.Rigid(layout.Spacer{Width: 12}.Layout),
				layout.Flexed(1, text))
		})
	})
}

// linkCard is a dcard that opens something, with an icon on the right.
func linkCard(gtx layout.Context, th *Theme, c *widget.Clickable, eyebrow, title, body string, ic *Icon, size, stroke float32, col colorNRGBA) layout.Dimensions {
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Bottom: 10, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return card(gtx, th.Card, th.Border, 14, layout.UniformInset(15), 0, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Bottom: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.07, Upper: true, Color: th.Muted}.Layout(gtx, th, eyebrow)
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Bottom: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return Paragraph{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 14, Color: th.Text}.Layout(gtx, th, title)
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 13, LineHeight: 1.65, Color: th.Muted}.Layout(gtx, th, body)
							}))
					}),
					layout.Rigid(layout.Spacer{Width: 12}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ic.Layout(gtx, unitDp(size), stroke, col) }),
				)
			})
		})
	})
}

// bestCard is "Best this month": poster, title, a review excerpt and the
// rating badge.
func bestCard(gtx layout.Context, a *App, e *core.Entry, word string) layout.Dimensions {
	th := a.Theme
	var lead layout.Widget
	if e.Poster != "" {
		lead = func(gtx layout.Context) layout.Dimensions {
			size := image.Pt(gtx.Dp(48), gtx.Dp(68))
			if !a.drawPoster(gtx, e.Poster, size, gtx.Dp(8)) {
				rrect(gtx, size, gtx.Dp(8), th.Tag)
			}
			return layout.Dimensions{Size: size}
		}
	}
	body := "No review written yet."
	if e.Review != "" {
		body = "\"" + core.Excerpt(e.Review, 120) + "\""
	}
	badge := func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = 0
		return layout.Inset{Top: 7}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return card(gtx, th.Tag, th.Tag, 20, layout.Inset{Top: 2, Bottom: 2, Left: 9, Right: 9}, 0, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return IconStar.Layout(gtx, 11, 2, th.Accent) }),
					layout.Rigid(layout.Spacer{Width: 4}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return Text{Font: font.Font{Typeface: Sans}, Size: 11, Color: th.Accent}.Layout(gtx, th, strconv.FormatFloat(e.Rating, 'f', -1, 64)+" stars")
					}))
			})
		})
	}
	return dcard(gtx, th, lead, "Best this "+word, e.Title, body, badge)
}

// rankRow is one "Top rated this year" row (.drank-row).
func (p *digestPage) rankRow(gtx layout.Context, a *App, i int, e *core.Entry) layout.Dimensions {
	th := a.Theme
	c := &p.topRated[i]
	if c.Clicked(gtx) {
		a.Push(newEntryDetail(e.ID))
	}
	return layout.Inset{Bottom: 8, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return pressable(gtx, c, func(gtx layout.Context) layout.Dimensions {
			return card(gtx, th.Card, th.Border, 12, layout.Inset{Top: 10, Bottom: 10, Left: 12, Right: 12}, 0, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(16)
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return Text{Font: font.Font{Typeface: Display, Weight: font.SemiBold}, Size: 13, Color: th.Accent}.Layout(gtx, th, strconv.Itoa(i+1))
						})
					}),
					layout.Rigid(layout.Spacer{Width: 10}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						size := image.Pt(gtx.Dp(30), gtx.Dp(42))
						if !a.drawPoster(gtx, e.Poster, size, gtx.Dp(6)) {
							rrect(gtx, size, gtx.Dp(6), th.Tag)
						}
						return layout.Dimensions{Size: size}
					}),
					layout.Rigid(layout.Spacer{Width: 10}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return Text{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 12.5, MaxLines: 1, Color: th.Text}.Layout(gtx, th, e.Title)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: 1}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return Text{Font: font.Font{Typeface: Sans}, Size: 10, Color: th.Accent}.Layout(gtx, th, starString(e.Rating))
								})
							}))
					}))
			})
		})
	})
}

// yearChart is the Year's "Month by month" bars (.dmonth-chart).
func yearChart(gtx layout.Context, th *Theme, monthly [12]float64) layout.Dimensions {
	peak := core.PeakIndex(monthly[:])
	maxM := 1.0
	for _, m := range monthly {
		maxM = max(maxM, m)
	}
	chart := func(gtx layout.Context) layout.Dimensions {
		// 74px tall, 18px in from the card's padding, 4px between columns;
		// each column is a bar (at most 16px wide) over its initial.
		// Its 4px top margin merges with the eyebrow's 4px.
		return layout.Inset{Bottom: 2, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			w := gtx.Constraints.Max.X
			h := gtx.Dp(74)
			gap := gtx.Dp(4)
			col := (w - 11*gap) / 12
			for i, m := range monthly {
				x := i * (col + gap)
				rec := op.Record(gtx.Ops)
				lgtx := gtx
				lgtx.Constraints = layout.Constraints{Max: image.Pt(col, h)}
				ld := Text{Font: font.Font{Typeface: Sans}, Size: 8.5, Color: th.Muted}.Layout(lgtx, th, []string{"J", "F", "M", "A", "M", "J", "J", "A", "S", "O", "N", "D"}[i])
				call := rec.Stop()
				st := op.Offset(image.Pt(x+(col-ld.Size.X)/2, h-ld.Size.Y)).Push(gtx.Ops)
				call.Add(gtx.Ops)
				st.Pop()

				// The bar's height is a share of the 74px column, but the
				// column also holds the label, so a tall bar shrinks to fit
				// (CSS flex-shrink); 3px at least.
				bw := min(col, gtx.Dp(16))
				bottom := h - ld.Size.Y - gtx.Dp(5)
				bh := min(int(float64(h)*m/maxM), bottom)
				bh = max(bh, gtx.Dp(3))
				r := image.Rect(x+(col-bw)/2, bottom-bh, x+(col-bw)/2+bw, bottom)
				monthBar(gtx, th, r, m > 0, i == peak)
			}
			return layout.Dimensions{Size: image.Pt(w, h)}
		})
	}
	return layout.Inset{Bottom: 10, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return card(gtx, th.Card, th.Border, 14, layout.Inset{Top: 15, Bottom: 10, Left: 15, Right: 15}, 0, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Bottom: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.07, Upper: true, Color: th.Muted}.Layout(gtx, th, "Month by month")
					})
				}),
				layout.Rigid(chart))
		})
	})
}

// monthBar paints one bar in the chosen yearBars style, rounded 4px on
// top and 2px at the foot.
func monthBar(gtx layout.Context, th *Theme, r image.Rectangle, hasData, peak bool) {
	rr := clip.RRect{Rect: r, NW: gtx.Dp(4), NE: gtx.Dp(4), SW: gtx.Dp(2), SE: gtx.Dp(2)}
	if r.Dy() < gtx.Dp(6) {
		rr.NW, rr.NE, rr.SW, rr.SE = r.Dy()/2, r.Dy()/2, r.Dy()/2, r.Dy()/2
	}
	defer rr.Push(gtx.Ops).Pop()
	if !hasData {
		paint.ColorOp{Color: th.Tag}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		return
	}
	switch yearBars {
	case barsGradient:
		paint.LinearGradientOp{Stop1: f32.Pt(0, float32(r.Min.Y)), Color1: th.Accent2,
			Stop2: f32.Pt(0, float32(r.Max.Y)), Color2: th.Accent}.Add(gtx.Ops)
	case barsPeak:
		c := th.Accent2
		if peak {
			c = th.Accent
		}
		paint.ColorOp{Color: c}.Add(gtx.Ops)
	default:
		paint.ColorOp{Color: th.Accent}.Add(gtx.Ops)
	}
	paint.PaintOp{}.Add(gtx.Ops)
}
