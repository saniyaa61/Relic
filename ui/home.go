package ui

// Home (SPEC §5; prototype renderHome) and its three "see all" pages:
// Still with you, Recently finished, All entries.

import (
	"image"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"

	"github.com/saniyaa61/relic/core"
)

// streakStyle is how the streak pill shows its 🔥 while the owner decides
// (Gio can't draw colour emoji from a font).
type streakStyle int

const (
	streakEmoji streakStyle = iota // the colour 🔥 as a bundled image
	streakIcon                     // a flame line icon in the accent colour
	streakPlain                    // words only
)

var homeStreak = streakEmoji

// IconFlame is a flame line icon (Lucide "flame", ISC licence).
var IconFlame = IconSVG(`<path d="M8.5 14.5A2.5 2.5 0 0011 12c0-1.38-.5-2-1-3-1.072-2.143-.224-4.054 2-6 .5 2.5 2 4.9 4 6.5 2 1.6 3 3.5 3 5.5a7 7 0 11-14 0c0-1.153.433-2.294 1-3a2.5 2.5 0 002.5 2.5z"/>`)

type homePage struct {
	settings IconButton
	list     widget.List
	chips    [4]widget.Clickable
	chipRow  widget.List
	add      widget.Clickable
	setup    widget.Clickable
	memory   widget.Clickable
	seeAll   widget.Clickable
	allLink  widget.Clickable
	ongoing  widget.List
	finished widget.List
	oCards   map[string]*ongoingCard
	fCards   map[string]*finishedCard
}

func (h *homePage) oCard(id string) *ongoingCard {
	if h.oCards == nil {
		h.oCards = map[string]*ongoingCard{}
	}
	if h.oCards[id] == nil {
		h.oCards[id] = &ongoingCard{}
	}
	return h.oCards[id]
}

func (h *homePage) fCard(id string) *finishedCard {
	if h.fCards == nil {
		h.fCards = map[string]*finishedCard{}
	}
	if h.fCards[id] == nil {
		h.fCards[id] = &finishedCard{}
	}
	return h.fCards[id]
}

func (h *homePage) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	lib := a.Lib
	if h.settings.Click.Clicked(gtx) {
		a.Push(&themePreview{})
	}
	if h.add.Clicked(gtx) {
		a.Go(TabNew)
	}
	if h.setup.Clicked(gtx) {
		a.Go(TabLibrary)
		newCategoryDialog(a)
	}
	if h.chips[0].Clicked(gtx) {
		a.Push(newPosterPage(pageAllEntries))
	}
	if h.chips[1].Clicked(gtx) {
		a.Push(&comingSoon{eyebrow: "Home", title: "Consumed", step: 7})
	}
	if h.chips[2].Clicked(gtx) || h.allLink.Clicked(gtx) {
		a.Push(newPosterPage(pageRecentlyFinished))
	}
	if h.chips[3].Clicked(gtx) || h.seeAll.Clicked(gtx) {
		a.Push(newPosterPage(pageStillWithYou))
	}
	header := []layout.Widget{func(gtx layout.Context) layout.Dimensions {
		return logoBar(gtx, th, func(gtx layout.Context) layout.Dimensions { return h.settings.Layout(gtx, th, IconSettings) })
	}}

	if len(lib.Categories) == 0 && len(lib.Entries) == 0 {
		return scrollPage(gtx, &h.list, header, []layout.Widget{
			func(gtx layout.Context) layout.Dimensions {
				return h.greeting(gtx, a, "Your library is waiting. Every great collection starts with one entry.", false, 18, 12)
			},
			func(gtx layout.Context) layout.Dimensions { return addButton(gtx, th, &h.add, "Add your first entry") },
			func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return wideButton(gtx, th, &h.setup, IconLibrary, "Set up your library categories")
				})
			},
		})
	}

	ongoing := lib.StillWithYou()
	finished := lib.RecentlyFinished()
	rows := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions {
			return h.greeting(gtx, a, plural(len(lib.Entries), "story", "stories")+" in your archive. Here's where you left off.", true, 18, 4)
		},
		func(gtx layout.Context) layout.Dimensions { return h.ribbon(gtx, a, len(ongoing), len(finished)) },
		func(gtx layout.Context) layout.Dimensions { return addButton(gtx, th, &h.add, "Log something new") },
	}
	if e := lib.OneYearAgo(a.Now(), a.Loc); e != nil {
		if h.memory.Clicked(gtx) {
			a.Push(newEntryDetail(e.ID))
		}
		rows = append(rows, func(gtx layout.Context) layout.Dimensions { return h.memoryCard(gtx, a, e) })
	}
	if len(ongoing) > 0 {
		rows = append(rows,
			func(gtx layout.Context) layout.Dimensions {
				return sectionHead(gtx, th, "Still with you", "See all", &h.seeAll)
			},
			func(gtx layout.Context) layout.Dimensions {
				tall := hasProgressLine(ongoing)
				return cardRow(gtx, &h.ongoing, 10, len(ongoing), func(gtx layout.Context, i int) layout.Dimensions {
					c := h.oCard(ongoing[i].ID)
					c.tall = tall
					return c.Layout(gtx, a, ongoing[i])
				}, 4)
			},
			layout.Spacer{Height: 18}.Layout,
		)
	}
	if len(finished) > 0 {
		recent := finished[:min(6, len(finished))]
		rows = append(rows,
			func(gtx layout.Context) layout.Dimensions {
				return sectionHead(gtx, th, "Recently finished", "All entries", &h.allLink)
			},
			func(gtx layout.Context) layout.Dimensions {
				return cardRow(gtx, &h.finished, 10, len(recent), func(gtx layout.Context, i int) layout.Dimensions {
					return h.fCard(recent[i].ID).Layout(gtx, a, recent[i], false)
				}, 20)
			},
		)
	}
	rows = append(rows, layout.Spacer{Height: 4}.Layout)
	return scrollPage(gtx, &h.list, header, rows)
}

// greeting is the date, "Good morning, Name" and the line under it, with
// the streak pill when there is a streak.
func (h *homePage) greeting(gtx layout.Context, a *App, sub string, streak bool, top, bottom float32) layout.Dimensions {
	th := a.Theme
	now := a.Now()
	return layout.Inset{Top: unitDp(top - 4), Bottom: unitDp(bottom), Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: 3}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Sans}, Size: 11, Tracking: 0.1, Upper: true, Color: th.Muted}.Layout(gtx, th, core.FormatToday(now, a.Loc))
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				greet := core.Greeting(now, a.Loc)
				name := strings.TrimSpace(a.Lib.Profile.Name)
				return richLine(gtx, th, 26, 1.2, []textRun{
					{greet, font.Font{Typeface: Display, Weight: font.Medium}, th.Text},
					{func() string {
						if name == "" {
							return ""
						}
						return ", "
					}(), font.Font{Typeface: Display, Weight: font.Medium}, th.Text},
					{name, font.Font{Typeface: Display, Weight: font.Medium, Style: font.Italic}, th.Accent},
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 5}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 13, LineHeight: 1.5, Color: th.Muted}.Layout(gtx, th, sub)
				})
			}),
		}
		if streak {
			if n, _ := core.Streak(a.Lib.Entries, now, a.Loc); n > 0 {
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return streakPill(gtx, th, n) })
				}))
			}
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

// streakPill is "🔥 4 days in a row" (.streak-pill).
func streakPill(gtx layout.Context, th *Theme, n int) layout.Dimensions {
	gtx.Constraints.Min = image.Point{} // inline: hugs its words
	return card(gtx, th.Tag, th.Border, 20, layout.Inset{Top: 5, Bottom: 5, Left: 12, Right: 12}, 0, func(gtx layout.Context) layout.Dimensions {
		var children []layout.FlexChild
		switch homeStreak {
		case streakEmoji:
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions { return Emoji(gtx, "1f525", 16) }),
				layout.Rigid(layout.Spacer{Width: 5}.Layout))
		case streakIcon:
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions { return IconFlame.Layout(gtx, 14, 1.8, th.Accent) }),
				layout.Rigid(layout.Spacer{Width: 5}.Layout))
		}
		days := "days in a row"
		if n == 1 {
			days = "day in a row"
		}
		children = append(children,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Display, Weight: font.SemiBold}, Size: 13, Color: th.Accent}.Layout(gtx, th, itoa(n))
			}),
			layout.Rigid(layout.Spacer{Width: 5}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Sans}, Size: 12, Color: th.Accent}.Layout(gtx, th, days)
			}))
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx, children...)
	})
}

// ribbon is the row of four stat chips, each opening its page.
func (h *homePage) ribbon(gtx layout.Context, a *App, ongoing, finished int) layout.Dimensions {
	th := a.Theme
	total := a.timeIndex().Sum(a.Lib.Entries)
	chips := []StatChip{
		{Value: itoa(len(a.Lib.Entries)), Label: "Total", Sub: "in archive"},
		{Value: core.FormatDurationOrZero(total), Label: "Consumed", Sub: "all time"},
		{Value: itoa(finished), Label: "Finished", Sub: "titles"},
		{Value: itoa(ongoing), Label: "Ongoing", Sub: "in progress"},
	}
	return layout.Inset{Top: 8, Bottom: 20}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return cardRow(gtx, &h.chipRow, 9, len(chips), func(gtx layout.Context, i int) layout.Dimensions {
			return pressable(gtx, &h.chips[i], func(gtx layout.Context) layout.Dimensions { return chips[i].Layout(gtx, th) })
		}, 0)
	})
}

// memoryCard is "One year ago today" (SPEC §4.15).
func (h *homePage) memoryCard(gtx layout.Context, a *App, e *core.Entry) layout.Dimensions {
	th := a.Theme
	return layout.Inset{Top: 4, Bottom: 20, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return pressable(gtx, &h.memory, func(gtx layout.Context) layout.Dimensions {
			return card(gtx, th.Card, th.Border, 14, layout.Inset{Top: 14, Bottom: 14, Left: 16, Right: 16}, 0, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				children := []layout.FlexChild{
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions { return IconClock.Layout(gtx, 13, 2, th.Accent) }),
								layout.Rigid(layout.Spacer{Width: 6}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return Text{Font: font.Font{Typeface: Serif, Weight: font.Medium}, Size: 10, Tracking: 0.08, Upper: true, Color: th.Accent}.Layout(gtx, th, "One year ago today")
								}))
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return Paragraph{Font: font.Font{Typeface: Display, Weight: font.SemiBold}, Size: 17, LineHeight: 1.3, Color: th.Text}.Layout(gtx, th, e.Title)
					}),
				}
				if r := e.Review; r != "" {
					if len([]rune(r)) > 160 {
						r = core.Excerpt(r, 157)
					}
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: 8, Bottom: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 13, LineHeight: 1.6, Color: th.Muted}.Layout(gtx, th, "\""+r+"\"")
						})
					}))
				}
				meta := typeShort(e.Type) + " · Finished " + core.FormatDate(e.CreatedAt, a.Loc)
				if e.Rating > 0 {
					meta += "  " + starString(e.Rating)
				}
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Serif}, Size: 11, Color: th.Muted}.Layout(gtx, th, meta)
				}))
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
			})
		})
	})
}

// sectionHead is a .sec-hd: Playfair heading and a small link on the right.
func sectionHead(gtx layout.Context, th *Theme, title, link string, c *widget.Clickable) layout.Dimensions {
	return layout.Inset{Bottom: 10, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Alignment: layout.Baseline, Spacing: layout.SpaceBetween}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 16, Color: th.Text}.Layout(gtx, th, title)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if link == "" {
					return layout.Dimensions{}
				}
				return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Sans}, Size: 12, Color: th.Accent2}.Layout(gtx, th, link)
				})
			}))
	})
}

// cardRow is a horizontally scrolling row of n cards with gap dp between,
// inside the page gutters, with bottom dp below (.ongoing-row, .ribbon).
func cardRow(gtx layout.Context, list *widget.List, gap float32, n int, card func(gtx layout.Context, i int) layout.Dimensions, bottom float32) layout.Dimensions {
	list.Axis = layout.Horizontal
	return layout.Inset{Bottom: unitDp(bottom)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = image.Point{}
		return list.List.Layout(gtx, n, func(gtx layout.Context, i int) layout.Dimensions {
			left, right := unitDp(gap/2), unitDp(gap/2)
			if i == 0 {
				left = gutter
			}
			if i == n-1 {
				right = gutter
			}
			return layout.Inset{Left: left, Right: right}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return card(gtx, i)
			})
		})
	})
}

// addButton is the accent "+ Log something new" button (.add-btn).
func addButton(gtx layout.Context, th *Theme, c *widget.Clickable, label string) layout.Dimensions {
	return layout.Inset{Bottom: 20, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return pressable(gtx, c, func(gtx layout.Context) layout.Dimensions {
			return card(gtx, th.Accent, th.Accent, 13, layout.Inset{Top: 12, Bottom: 12, Left: 17, Right: 17}, 0, func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions { return IconPlus.Layout(gtx, 16, 2, th.BtnText) }),
						layout.Rigid(layout.Spacer{Width: 9}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return Text{Font: font.Font{Typeface: Display, Style: font.Italic}, Size: 15, Color: th.BtnText}.Layout(gtx, th, label)
						}))
				})
			})
		})
	})
}

// ---------- The "see all" pages ----------

type posterPageKind int

const (
	pageStillWithYou posterPageKind = iota
	pageRecentlyFinished
	pageAllEntries
)

// posterPage is Still with you (3 cards a row) or Recently finished / All
// entries (4 a row): rows of poster cards that scroll sideways, no time
// shown, and search scoped to the page (SPEC §5).
type posterPage struct {
	kind   posterPageKind
	back   IconButton
	search Search
	list   widget.List
	rows   []widget.List
	oCards map[string]*ongoingCard
	fCards map[string]*finishedCard
}

func newPosterPage(k posterPageKind) *posterPage { return &posterPage{kind: k} }

func (p *posterPage) WantsBack() bool { return p.search.WantsBack() }
func (p *posterPage) Back(a *App)     { p.search.Back(a) }

func (p *posterPage) entries(a *App) []*core.Entry {
	switch p.kind {
	case pageStillWithYou:
		return a.Lib.StillWithYou()
	case pageRecentlyFinished:
		return a.Lib.RecentlyFinished()
	}
	return a.Lib.Entries
}

func (p *posterPage) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	es := p.entries(a)
	title, sub, placeholder, empty, emptyIcon := "All entries", plural(len(es), "title", "titles")+" in your archive", "Search all entries…", "Nothing logged yet.", IconBook
	switch p.kind {
	case pageStillWithYou:
		title, sub, placeholder, empty, emptyIcon = "Still with you", plural(len(es), "title", "titles")+" in progress", "Search still with you…", "Nothing in progress right now.", IconClock
	case pageRecentlyFinished:
		title, sub, placeholder, empty = "Recently finished", plural(len(es), "title", "titles")+" finished", "Search recently finished…", "Nothing finished yet."
	}
	var search *Search
	if len(es) > 0 {
		search = &p.search
	}
	header := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions { return subBar(gtx, a, &p.back, search) },
		func(gtx layout.Context) layout.Dimensions { return p.search.Bar(gtx, a, placeholder) },
	}
	if q := p.search.Query(); strings.TrimSpace(q) != "" {
		return scrollPage(gtx, &p.list, header, searchResultsWith(a, es, q, title, IconSearch, 14, func(m []*core.Entry) []layout.Widget {
			return p.cardRows(a, m)
		}))
	}
	rows := []layout.Widget{func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: -8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return pageHead(gtx, th, "Home", title, sub)
		})
	}}
	if len(es) == 0 {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions { return emptyState(gtx, th, emptyIcon, empty) })
	} else {
		rows = append(rows, p.cardRows(a, es)...)
	}
	rows = append(rows, layout.Spacer{Height: 14}.Layout)
	return scrollPage(gtx, &p.list, header, rows)
}

// cardRows chunks entries into rows (.fav-rows: 14dp apart).
func (p *posterPage) cardRows(a *App, es []*core.Entry) []layout.Widget {
	per := 4
	if p.kind == pageStillWithYou {
		per = 3
	}
	n := (len(es) + per - 1) / per
	for len(p.rows) < n {
		p.rows = append(p.rows, widget.List{})
	}
	if p.oCards == nil {
		p.oCards, p.fCards = map[string]*ongoingCard{}, map[string]*finishedCard{}
	}
	out := []layout.Widget{layout.Spacer{Height: 2}.Layout}
	for r := 0; r < n; r++ {
		row := es[r*per : min((r+1)*per, len(es))]
		tall := hasProgressLine(row)
		out = append(out, func(gtx layout.Context) layout.Dimensions {
			bottom := float32(14)
			if r == n-1 {
				bottom = 10
			}
			return cardRow(gtx, &p.rows[r], 10, len(row), func(gtx layout.Context, i int) layout.Dimensions {
				e := row[i]
				if p.kind == pageStillWithYou {
					if p.oCards[e.ID] == nil {
						p.oCards[e.ID] = &ongoingCard{}
					}
					p.oCards[e.ID].tall = tall
					return p.oCards[e.ID].Layout(gtx, a, e)
				}
				if p.fCards[e.ID] == nil {
					p.fCards[e.ID] = &finishedCard{}
				}
				return p.fCards[e.ID].Layout(gtx, a, e, false)
			}, bottom)
		})
	}
	return out
}

// comingSoon stands in for a page built in a later step.
type comingSoon struct {
	eyebrow, title string
	step           int
	back           IconButton
	list           widget.List
}

func (p *comingSoon) Layout(gtx layout.Context, a *App) layout.Dimensions {
	return scrollPage(gtx, &p.list, []layout.Widget{
		func(gtx layout.Context) layout.Dimensions { return subBar(gtx, a, &p.back, nil) },
	}, []layout.Widget{func(gtx layout.Context) layout.Dimensions {
		return pageHead(gtx, a.Theme, p.eyebrow, p.title, "This page arrives in step "+itoa(p.step)+" of Phase 3.")
	}})
}

// textRun is one styled piece of a richLine.
type textRun struct {
	s     string
	font  font.Font
	color colorNRGBA
}

// richLine lays out runs one after another on a baseline, wrapping whole
// runs to a new line when they don't fit ("Good morning, Saniya" with the
// name in italic accent).
func richLine(gtx layout.Context, th *Theme, size, lineHeight float32, runs []textRun) layout.Dimensions {
	maxW := gtx.Constraints.Max.X
	x, y, lineH := 0, 0, gtx.Sp(unitSp(size*lineHeight))
	w := 0
	cgtx := gtx
	cgtx.Constraints.Min = image.Point{}
	for _, r := range runs {
		if r.s == "" {
			continue
		}
		rec := op.Record(gtx.Ops)
		d := Text{Font: r.font, Size: unitSp(size), LineHeight: lineHeight, Color: r.color}.Layout(cgtx, th, r.s)
		call := rec.Stop()
		if x > 0 && x+d.Size.X > maxW {
			x, y = 0, y+lineH
		}
		st := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		st.Pop()
		x += d.Size.X
		w = max(w, x)
	}
	return layout.Dimensions{Size: image.Pt(w, y+lineH)}
}
