package ui

import (
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"
)

// StatChip is a Home ribbon chip: a big number, an uppercase label and an
// italic sub-line ("128 / TOTAL / in archive"). Prototype: .chip, .chip-v,
// .chip-l and the inline Lora sub-line.
type StatChip struct {
	Value, Label, Sub string
	// PadX is the side padding; 0 means the prototype's 14dp.
	PadX unit.Dp
	// Scale shrinks the text to fit a narrow row; 0 means 1.
	Scale float32
}

func (c StatChip) Layout(gtx layout.Context, th *Theme) layout.Dimensions {
	px := c.PadX
	if px == 0 {
		px = 14
	}
	k := c.Scale
	if k == 0 {
		k = 1
	}
	sz := func(v float32) unit.Sp { return unit.Sp(v * k) }
	pad := layout.Inset{Top: 10, Bottom: 10, Left: px, Right: px}
	return card(gtx, th.Card, th.Border, 13, pad, unit.Dp(80*k), func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Display, Weight: font.SemiBold}, Size: sz(21), LineHeight: 1, Color: th.Accent}.Layout(gtx, th, c.Value)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Sans}, Size: sz(10), Tracking: 0.06, Upper: true, Color: th.Muted}.Layout(gtx, th, c.Label)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 1}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: sz(9), Color: th.Muted}.Layout(gtx, th, c.Sub)
				})
			}),
		)
	})
}
