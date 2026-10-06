package ui

import (
	"image"
	"image/color"
	"strconv"

	"github.com/saniyaa61/relic/store"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
)

// pageHead is the prototype's .lib-head: a small letter-spaced eyebrow, a
// serif title and an italic sub-line.
func pageHead(gtx layout.Context, th *Theme, eyebrow, title, sub string) layout.Dimensions {
	return layout.Inset{Top: 16, Bottom: 12, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: 5}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.12, Upper: true, Color: th.Muted}.Layout(gtx, th, eyebrow)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return Paragraph{Font: font.Font{Typeface: Display, Weight: font.SemiBold}, Size: 25, LineHeight: 1.18, Color: th.Text}.Layout(gtx, th, title)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if sub == "" {
					return layout.Dimensions{}
				}
				return layout.Inset{Top: 5}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 12.5, LineHeight: 1.55, Color: th.Muted}.Layout(gtx, th, sub)
				})
			}),
		)
	})
}

// scrollPage lays out a page's rows in a vertical scroller (the prototype's
// .scroll), with the top bar and search bar fixed above it.
func scrollPage(gtx layout.Context, list *widget.List, header []layout.Widget, rows []layout.Widget) layout.Dimensions {
	list.Axis = layout.Vertical
	children := make([]layout.FlexChild, 0, len(header)+1)
	for _, h := range header {
		children = append(children, layout.Rigid(h))
	}
	children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
		return list.List.Layout(gtx, len(rows), func(gtx layout.Context, i int) layout.Dimensions {
			return rows[i](gtx)
		})
	}))
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// wideButton is the prototype's quiet full-width card button ("Set up
// your library categories"): card fill, thin border, Playfair italic in
// accent2, with an optional leading icon.
func wideButton(gtx layout.Context, th *Theme, c *widget.Clickable, ic *Icon, label string) layout.Dimensions {
	return layout.Inset{Left: gutter, Right: gutter, Bottom: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return pressable(gtx, c, func(gtx layout.Context) layout.Dimensions {
			return card(gtx, th.Card, th.Border, 13, layout.UniformInset(13), 0, func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if ic == nil {
								return layout.Dimensions{}
							}
							return layout.Inset{Right: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return ic.Layout(gtx, 15, 2, th.Accent2)
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return Text{Font: font.Font{Typeface: Display, Style: font.Italic}, Size: 14, Color: th.Accent2}.Layout(gtx, th, label)
						}),
					)
				})
			})
		})
	})
}

// pill is the prototype's .stab: a rounded outline chip that fills with
// the accent colour when selected.
func pill(gtx layout.Context, th *Theme, c *widget.Clickable, label string, on bool) layout.Dimensions {
	bg, fg, border := th.Bg, th.Muted, th.Border
	if on {
		bg, fg, border = th.Accent, th.BtnText, th.Accent
	}
	return pressable(gtx, c, func(gtx layout.Context) layout.Dimensions {
		return card(gtx, bg, border, 20, layout.Inset{Top: 5, Bottom: 5, Left: 13, Right: 13}, 0, func(gtx layout.Context) layout.Dimensions {
			return Text{Font: font.Font{Typeface: Sans}, Size: 12, Color: fg}.Layout(gtx, th, label)
		})
	})
}

// flow lays children out left to right, wrapping onto new rows, with gap
// dp between them both ways (CSS flex-wrap with gap).
func flow(gtx layout.Context, gap int, children []layout.Widget) layout.Dimensions {
	maxW := gtx.Constraints.Max.X
	cgtx := gtx
	cgtx.Constraints.Min = image.Point{}
	x, y, rowH, w := 0, 0, 0, 0
	for _, ch := range children {
		rec := op.Record(gtx.Ops)
		d := ch(cgtx)
		call := rec.Stop()
		if x > 0 && x+d.Size.X > maxW {
			x, y = 0, y+rowH+gap
			rowH = 0
		}
		st := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		st.Pop()
		x += d.Size.X + gap
		rowH = max(rowH, d.Size.Y)
		w = max(w, x-gap)
	}
	return layout.Dimensions{Size: image.Pt(w, y+rowH)}
}

// eyebrow is a small letter-spaced label (the prototype's .edit-label).
func eyebrow(gtx layout.Context, th *Theme, s string) layout.Dimensions {
	return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.07, Upper: true, Color: th.Muted}.Layout(gtx, th, s)
}

func unitSp(v float32) unit.Sp { return unit.Sp(v) }
func unitDp(v float32) unit.Dp { return unit.Dp(v) }
func itoa(n int) string        { return strconv.Itoa(n) }

const textMiddle = text.Middle

// fillWidth lays out w and reports the full available width, so a
// Flexed child pushes the next child to the far edge even when w (text,
// say) is narrower.
func fillWidth(w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		d := w(gtx)
		d.Size.X = max(d.Size.X, gtx.Constraints.Max.X)
		return d
	}
}

type colorNRGBA = color.NRGBA

const keyHintNumeric = key.HintNumeric

// gtxSize is the full available width at height h.
func gtxSize(gtx layout.Context, h int) image.Point { return image.Pt(gtx.Constraints.Max.X, h) }

type storeWriter = store.Writer

// flowCentered is flow with each line's items centred vertically on it.
func flowCentered(gtx layout.Context, gap int, children []layout.Widget) layout.Dimensions {
	maxW := gtx.Constraints.Max.X
	cgtx := gtx
	cgtx.Constraints.Min = image.Point{}
	type item struct {
		call op.CallOp
		size image.Point
	}
	var lines [][]item
	var line []item
	x := 0
	for _, ch := range children {
		rec := op.Record(gtx.Ops)
		d := ch(cgtx)
		it := item{rec.Stop(), d.Size}
		if x > 0 && x+d.Size.X > maxW {
			lines, line, x = append(lines, line), nil, 0
		}
		line = append(line, it)
		x += d.Size.X + gap
	}
	if len(line) > 0 {
		lines = append(lines, line)
	}
	y, w := 0, 0
	for li, l := range lines {
		h := 0
		for _, it := range l {
			h = max(h, it.size.Y)
		}
		x := 0
		for _, it := range l {
			st := op.Offset(image.Pt(x, y+(h-it.size.Y)/2)).Push(gtx.Ops)
			it.call.Add(gtx.Ops)
			st.Pop()
			x += it.size.X + gap
		}
		w = max(w, x-gap)
		y += h
		if li < len(lines)-1 {
			y += gap
		}
	}
	return layout.Dimensions{Size: image.Pt(w, y)}
}
