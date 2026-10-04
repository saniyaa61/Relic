package ui

import (
	"gioui.org/font"
	"gioui.org/layout"
)

// StatChip is a Home ribbon chip: a big number, an uppercase label and an
// italic sub-line ("128 / TOTAL / in archive"). Prototype: .chip, .chip-v,
// .chip-l and the inline Lora sub-line.
type StatChip struct {
	Value, Label, Sub string
}

func (c StatChip) Layout(gtx layout.Context, th *Theme) layout.Dimensions {
	pad := layout.Inset{Top: 10, Bottom: 10, Left: 14, Right: 14}
	return card(gtx, th.Card, th.Border, 13, pad, 80, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Display, Weight: font.SemiBold}, Size: 21, LineHeight: 1, Color: th.Accent}.Layout(gtx, th, c.Value)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.06, Upper: true, Color: th.Muted}.Layout(gtx, th, c.Label)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 1}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 9, Color: th.Muted}.Layout(gtx, th, c.Sub)
				})
			}),
		)
	})
}
