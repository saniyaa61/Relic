package ui

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/paint"
)

// Spike is the Phase 0 test screen: one Home stat chip and one poster card
// on the theme background, with the prototype's 18dp page gutter.
// safe is the system-bar inset (status bar, navigation bar).
func Spike(gtx layout.Context, th *Theme, safe layout.Inset) layout.Dimensions {
	paint.Fill(gtx.Ops, th.Bg)
	return safe.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 18, Left: 18, Right: 18}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min = image.Point{} // let cards hug their content
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return StatChip{Value: "128", Label: "Total", Sub: "in archive"}.Layout(gtx, th)
				}),
				layout.Rigid(layout.Spacer{Height: 20}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return PosterCard{
						Title:    "The Remains of the Day",
						Section:  "Books",
						Progress: "p.142/320",
						Percent:  44,
					}.Layout(gtx, th)
				}),
			)
		})
	})
}
