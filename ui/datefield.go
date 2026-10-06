package ui

import (
	"fmt"
	"image"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"

	"github.com/saniyaa61/relic/core"
)

// dateField is a form date (the prototype's <input type="date">). The
// closed field shows the date as the browser did ("16/06/2026", or a
// "dd/mm/yyyy" placeholder); tapping it opens a month calendar in a
// dialog, our stand-in for the phone's own date picker.
type dateField struct {
	value core.Date
	click widget.Clickable
}

func (f *dateField) Layout(gtx layout.Context, a *App, title string) layout.Dimensions {
	th := a.Theme
	if f.click.Clicked(gtx) {
		openCalendar(a, title, f.value, func(d core.Date) { f.value = d })
	}
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return f.click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		st := formSelect
		return card(gtx, th.Bg, th.Border, unitDp(st.Radius), st.Pad, 0, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			text, c := "dd/mm/yyyy", th.Muted
			if !f.value.IsZero() {
				text, c = fmt.Sprintf("%02d/%02d/%04d", f.value.Day, int(f.value.Month), f.value.Year), th.Text
			}
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, fillWidth(func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Sans}, Size: 13, MaxLines: 1, Color: c}.Layout(gtx, th, text)
				})),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return IconCalendar.Layout(gtx, 14, 1.6, th.Muted) }),
			)
		})
	})
}

// IconCalendar marks date fields.
var IconCalendar = IconSVG(`<rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/>`)

// IconChevronLeft and IconChevronRight are the prototype's digest arrows.
var (
	IconChevronLeft  = IconSVG(`<polyline points="15 18 9 12 15 6"/>`)
	IconChevronRight = IconSVG(`<polyline points="9 18 15 12 9 6"/>`)
)

// calendar is the month grid inside the date dialog.
type calendar struct {
	month      core.Month
	picked     core.Date
	today      core.Date
	prev, next widget.Clickable
	days       [42]widget.Clickable
}

// openCalendar shows the date dialog; Done keeps the picked day, Clear
// empties the field.
func openCalendar(a *App, title string, value core.Date, set func(core.Date)) {
	today := core.DateOf(a.Now(), a.Loc)
	c := &calendar{picked: value, today: today}
	start := value
	if start.IsZero() {
		start = today
	}
	c.month = core.Month{Year: start.Year, Month: start.Month}
	d := &Dialog{Title: title, Cancel: "Clear", Confirm: "Done",
		OnConfirm: func(a *App, _ string) { set(c.picked) },
		OnCancel:  func(a *App) { set(core.Date{}) },
		Body:      c.Layout,
	}
	a.ShowDialog(d)
}

func (c *calendar) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	if c.prev.Clicked(gtx) {
		c.month = c.month.Add(-1)
	}
	if c.next.Clicked(gtx) {
		c.month = c.month.Add(1)
	}
	first := time.Date(c.month.Year, c.month.Month, 1, 0, 0, 0, 0, time.UTC)
	lead := (int(first.Weekday()) + 6) % 7 // weeks start on Monday
	days := first.AddDate(0, 1, -1).Day()
	for i := range c.days {
		if n := i - lead + 1; n >= 1 && n <= days && c.days[i].Clicked(gtx) {
			c.picked = core.Date{Year: c.month.Year, Month: c.month.Month, Day: n}
		}
	}
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	w := gtx.Constraints.Max.X
	cell := w / 7
	arrow := func(gtx layout.Context, b *widget.Clickable, ic *Icon) layout.Dimensions {
		return b.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			// The digest's round arrow buttons (.dnav-btn).
			sz := gtx.Dp(30)
			border := th.Border
			if b.Pressed() {
				border = th.Accent2
			}
			gtx.Constraints = layout.Exact(image.Pt(sz, sz))
			return card(gtx, th.Card, border, 15, layout.Inset{}, 0, func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return ic.Layout(gtx, 14, 2, th.Muted) })
			})
		})
	}
	rows := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return arrow(gtx, &c.prev, IconChevronLeft) }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Display, Weight: font.SemiBold}, Size: 14, Color: th.Text}.Layout(gtx, th, c.month.String())
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return arrow(gtx, &c.next, IconChevronRight) }),
			)
		}),
		layout.Rigid(layout.Spacer{Height: 10}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			h := 0
			for i, l := range []string{"M", "T", "W", "T", "F", "S", "S"} {
				rec := op.Record(gtx.Ops)
				d := Text{Font: font.Font{Typeface: Sans}, Size: 10, Color: th.Muted}.Layout(gtx, th, l)
				call := rec.Stop()
				st := op.Offset(image.Pt(i*cell+(cell-d.Size.X)/2, 0)).Push(gtx.Ops)
				call.Add(gtx.Ops)
				st.Pop()
				h = max(h, d.Size.Y)
			}
			return layout.Dimensions{Size: image.Pt(w, h+gtx.Dp(4))}
		}),
	}
	weeks := (lead + days + 6) / 7
	for wk := range weeks {
		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			sz := min(cell, gtx.Dp(36))
			for dow := range 7 {
				i := wk*7 + dow
				n := i - lead + 1
				if n < 1 || n > days {
					continue
				}
				day := core.Date{Year: c.month.Year, Month: c.month.Month, Day: n}
				st := op.Offset(image.Pt(dow*cell+(cell-sz)/2, 0)).Push(gtx.Ops)
				dgtx := gtx
				dgtx.Constraints = layout.Exact(image.Pt(sz, sz))
				c.days[i].Layout(dgtx, func(gtx layout.Context) layout.Dimensions {
					fg := th.Text
					switch {
					case day == c.picked:
						rrect(gtx, image.Pt(sz, sz), sz/2, th.Accent)
						fg = th.BtnText
					case c.days[i].Pressed():
						rrect(gtx, image.Pt(sz, sz), sz/2, th.Tag)
					case day == c.today:
						card(gtx, withAlpha(th.Bg, 0), th.Accent2, unitDp(float32(sz)/2/gtx.Metric.PxPerDp), layout.Inset{}, 0, func(gtx layout.Context) layout.Dimensions {
							return layout.Dimensions{Size: image.Pt(sz-2*gtx.Dp(1), sz-2*gtx.Dp(1))}
						})
					}
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return Text{Font: font.Font{Typeface: Sans}, Size: 13, Color: fg}.Layout(gtx, th, fmt.Sprint(n))
					})
				})
				st.Pop()
			}
			return layout.Dimensions{Size: image.Pt(w, sz)}
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
}
