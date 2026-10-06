package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

// PosterCard is the "Still with you" card on Home: poster area, title,
// folder name and progress. Prototype: .oc, .oc-ph, .oc-body, .prog-ring.
// The spike has no poster images yet, so it draws the no-poster state.
type PosterCard struct {
	Title, Section string
	// Progress is the text under the folder ("p.142/320"); empty hides it.
	Progress string
	// Percent drives the progress ring; 0 hides it, as in the prototype.
	Percent int
}

func (p PosterCard) Layout(gtx layout.Context, th *Theme) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Dp(130)
	gtx.Constraints.Max.X = gtx.Constraints.Min.X
	return card(gtx, th.Card, th.Border, 13, layout.Inset{}, 130, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return p.layoutPoster(gtx, th)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 7, Bottom: 9, Left: 9, Right: 9}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return p.layoutBody(gtx, th)
				})
			}),
		)
	})
}

func (p PosterCard) layoutPoster(gtx layout.Context, th *Theme) layout.Dimensions {
	size := image.Pt(gtx.Constraints.Max.X, gtx.Dp(175))
	fillRect(gtx, image.Rectangle{Max: size}, th.Tag)

	icon := gtx.Dp(26)
	st := op.Offset(image.Pt((size.X-icon)/2, (size.Y-icon)/2)).Push(gtx.Ops)
	IconBook.Layout(gtx, 26, 1.3, withAlpha(th.Muted, 0.7))
	st.Pop()

	if p.Percent > 0 {
		ring := gtx.Dp(ringSize)
		st := op.Offset(image.Pt(size.X-gtx.Dp(6)-ring, size.Y-gtx.Dp(6)-ring)).Push(gtx.Ops)
		progressRing(gtx, th, p.Percent)
		st.Pop()
	}
	return layout.Dimensions{Size: size}
}

func (p PosterCard) layoutBody(gtx layout.Context, th *Theme) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	children := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 11, MaxLines: 1, Color: th.Text}.Layout(gtx, th, p.Title)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.05, Upper: true, Color: th.Muted}.Layout(gtx, th, p.Section)
		}),
	}
	if p.Progress != "" {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Sans}, Size: 10, Color: th.Muted}.Layout(gtx, th, p.Progress)
			})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// The prototype's .prog-ring asks for 28px, but the general `.oc-ph svg`
// rule wins and draws it at 26px and 70% opacity. We match what's on screen,
// except the label: that same rule also outlines the text in the muted
// colour, which makes it muddy, so ours stays crisp white.
const ringSize unit.Dp = 26

// progressRing draws the white ring with the percentage in the middle.
func progressRing(gtx layout.Context, th *Theme, pct int) layout.Dimensions {
	defer paint.PushOpacity(gtx.Ops, 0.7).Pop()
	size := gtx.Dp(ringSize)
	px := float32(size) / 28 // the ring is designed in a 28-unit box
	centre := f32.Pt(14*px, 14*px)
	white := color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	strokeArc(gtx, centre, 11*px, 3*px, 0, 2*math.Pi, withAlpha(white, 0.25))
	strokeArc(gtx, centre, 11*px, 3*px, 0, 2*math.Pi*float32(min(pct, 100))/100, withAlpha(white, 0.85))

	// Centre the label in the ring.
	rec := op.Record(gtx.Ops)
	lgtx := gtx
	lgtx.Constraints = layout.Constraints{Max: image.Pt(size, size)}
	d := Text{Font: font.Font{Typeface: Sans, Weight: font.Medium}, Size: unit.Sp(7 * float32(ringSize) / 28), Color: white}.Layout(lgtx, th, fmt.Sprintf("%d%%", pct))
	call := rec.Stop()
	st := op.Offset(image.Pt((size-d.Size.X)/2, (size-d.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	st.Pop()
	return layout.Dimensions{Size: image.Pt(size, size)}
}
