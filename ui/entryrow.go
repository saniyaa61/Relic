package ui

import (
	"image"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"

	"github.com/saniyaa61/relic/core"
)

// entryRow is the prototype's .erow: a list row with a small poster,
// title, "Category · Folder · status", stars and feeling tags.
type entryRow struct {
	click widget.Clickable
}

func (r *entryRow) Layout(gtx layout.Context, a *App, e *core.Entry) layout.Dimensions {
	th := a.Theme
	if r.click.Clicked(gtx) {
		a.Push(&entryPlaceholder{entry: e})
	}
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return r.click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// Hover nudged the row 2px right; on touch it's press feedback.
		if r.click.Pressed() {
			defer op.Offset(image.Pt(gtx.Dp(2), 0)).Push(gtx.Ops).Pop()
		}
		return card(gtx, th.Card, th.Border, 13, layout.Inset{Top: 11, Bottom: 11, Left: 13, Right: 13}, 0, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Start}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return rowPoster(gtx, a, e) }),
				layout.Rigid(layout.Spacer{Width: 11}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return r.body(gtx, a, e) }),
			)
		})
	})
}

// rowPoster is the 42×56 thumbnail, or the type icon on a tag tile.
func rowPoster(gtx layout.Context, a *App, e *core.Entry) layout.Dimensions {
	th := a.Theme
	sz := image.Pt(gtx.Dp(42), gtx.Dp(56))
	rad := gtx.Dp(7)
	if !a.drawPoster(gtx, e.Poster, sz, rad) {
		rrect(gtx, sz, rad, th.Tag)
		if ic := rowIcons[e.Type]; ic != nil {
			st := op.Offset(image.Pt((sz.X-gtx.Dp(18))/2, (sz.Y-gtx.Dp(18))/2)).Push(gtx.Ops)
			ic.Layout(gtx, 18, 1.3, th.Muted)
			st.Pop()
		}
	}
	return layout.Dimensions{Size: sz}
}

func (r *entryRow) body(gtx layout.Context, a *App, e *core.Entry) layout.Dimensions {
	th := a.Theme
	meta := a.Lib.CategoryName(e.CategoryID)
	if e.Folder != "" {
		meta += " · " + e.Folder
	}
	meta += " · " + string(e.Status)
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 13, MaxLines: 1, Color: th.Text}.Layout(gtx, th, e.Title)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: 3}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Sans}, Size: 11, MaxLines: 1, Color: th.Muted}.Layout(gtx, th, meta)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return stars(gtx, th, e.Rating, 11) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(e.Tags) == 0 {
				return layout.Dimensions{}
			}
			return layout.Inset{Top: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				chips := make([]layout.Widget, len(e.Tags))
				for i, t := range e.Tags {
					chips[i] = func(gtx layout.Context) layout.Dimensions { return tagChip(gtx, th, t) }
				}
				return flow(gtx, gtx.Dp(4), chips)
			})
		}),
	)
}

// stars is the prototype's starsHTML: five stars, filled in accent2, a
// "½" for a half star, the rest in the border colour.
func stars(gtx layout.Context, th *Theme, rating float64, size float32) layout.Dimensions {
	children := make([]layout.FlexChild, 0, 9)
	for i := 1; i <= 5; i++ {
		glyph, c := "★", th.Border
		switch {
		case rating >= float64(i):
			c = th.Accent2
		case rating >= float64(i)-0.5:
			glyph, c = "½", th.Accent2
		}
		if i > 1 {
			children = append(children, layout.Rigid(layout.Spacer{Width: 1}.Layout))
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return Text{Font: font.Font{Typeface: Sans}, Size: unitSp(size), Color: c}.Layout(gtx, th, glyph)
		}))
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx, children...)
}

// tagChip is a feeling tag on a row (.etag): tag fill, accent outline.
func tagChip(gtx layout.Context, th *Theme, s string) layout.Dimensions {
	return card(gtx, th.Tag, th.Accent, 20, layout.Inset{Top: 2, Bottom: 2, Left: 7, Right: 7}, 0, func(gtx layout.Context) layout.Dimensions {
		return Text{Font: font.Font{Typeface: Sans}, Size: 10, Color: th.Accent}.Layout(gtx, th, s)
	})
}

// emptyState is the prototype's .empty: a faint 40px line icon over an
// italic note, centred.
func emptyState(gtx layout.Context, th *Theme, ic *Icon, note string) layout.Dimensions {
	return layout.Inset{Top: 28, Bottom: 28, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if ic == nil {
					return layout.Dimensions{}
				}
				return layout.Inset{Bottom: 9}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return ic.Layout(gtx, 40, 1, withAlpha(th.Border, 0.7))
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(200))
				return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 13, LineHeight: 1.6, Alignment: textMiddle, Color: th.Muted}.Layout(gtx, th, note)
			}),
		)
	})
}

// entryList lays out rows for entries with 9dp between them (.elist).
type entryList struct {
	rows map[string]*entryRow
}

func (l *entryList) row(id string) *entryRow {
	if l.rows == nil {
		l.rows = map[string]*entryRow{}
	}
	r := l.rows[id]
	if r == nil {
		r = &entryRow{}
		l.rows[id] = r
	}
	return r
}

// widgets returns one widget per entry, for a page's scrolling list.
func (l *entryList) widgets(a *App, entries []*core.Entry) []layout.Widget {
	out := make([]layout.Widget, len(entries))
	for i, e := range entries {
		out[i] = func(gtx layout.Context) layout.Dimensions {
			bottom := unitDp(9)
			if i == len(entries)-1 {
				bottom = 0
			}
			return layout.Inset{Left: gutter, Right: gutter, Bottom: bottom}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return l.row(e.ID).Layout(gtx, a, e)
			})
		}
	}
	return out
}

// searchResults is the shared results layout (prototype
// searchResultsHTML): the "3 results for “x” in Movies" line, then the
// rows, or an empty state. Pages pass their own entry set and scope label;
// the matching itself is core.Library.Search (SPEC §4.9).
func searchResults(a *App, list *entryList, scopeEntries []*core.Entry, query, scope string, emptyIcon *Icon) []layout.Widget {
	th := a.Theme
	matched := a.Lib.Search(scopeEntries, query)
	header := func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 12, Bottom: 8, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 12, Color: th.Muted}.Layout(gtx, th, core.ResultsHeader(len(matched), query, scope))
		})
	}
	rows := []layout.Widget{header}
	if len(matched) == 0 {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return emptyState(gtx, th, emptyIcon, core.NoResults(query))
		})
	} else {
		rows = append(rows, list.widgets(a, matched)...)
	}
	return append(rows, layout.Spacer{Height: 10}.Layout)
}

// plural is "1 entry" / "3 entries" style counting.
func plural(n int, one, many string) string {
	if n == 1 {
		return itoa(n) + " " + one
	}
	return itoa(n) + " " + many
}
