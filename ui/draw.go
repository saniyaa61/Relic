package ui

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

// card draws the shared Relic surface: a rounded rectangle with a 1dp
// border, like CSS `background:var(--card);border:1px solid var(--border)`.
// content is laid out inside the border, clipped to the inner corners
// (CSS overflow:hidden), and padded by pad. The card is at least minW wide
// and at least as big as the constraints' minimum.
func card(gtx layout.Context, bg, border color.NRGBA, radius unit.Dp, pad layout.Inset, minW unit.Dp, content layout.Widget) layout.Dimensions {
	b := gtx.Dp(1)
	r := gtx.Dp(radius)

	// Measure the content first so we know the card's size.
	// The content gets the card's minimum size less the border, so a card
	// stretched by its parent can centre or fill its content.
	cgtx := gtx
	cgtx.Constraints.Max = cgtx.Constraints.Max.Sub(image.Pt(2*b, 2*b))
	// layout.Inset only shrinks the maximum, so take the padding off the
	// minimum here, or a card asked to be 110dp tall would grow by its
	// padding.
	padX := gtx.Dp(pad.Left) + gtx.Dp(pad.Right)
	padY := gtx.Dp(pad.Top) + gtx.Dp(pad.Bottom)
	cgtx.Constraints.Min = image.Pt(max(cgtx.Constraints.Min.X-2*b-padX, 0), max(cgtx.Constraints.Min.Y-2*b-padY, 0))
	rec := op.Record(gtx.Ops)
	cd := pad.Layout(cgtx, content)
	call := rec.Stop()

	size := image.Pt(max(cd.Size.X+2*b, gtx.Dp(minW), gtx.Constraints.Min.X), max(cd.Size.Y+2*b, gtx.Constraints.Min.Y))
	outer := image.Rectangle{Max: size}
	inner := outer.Inset(b)
	// Like CSS, a radius can't exceed half the box (pills).
	r = clampRadius(r, size)

	paint.FillShape(gtx.Ops, border, clip.UniformRRect(outer, r).Op(gtx.Ops))
	cl := clip.UniformRRect(inner, max(r-b, 0)).Push(gtx.Ops)
	paint.Fill(gtx.Ops, bg)
	st := op.Offset(inner.Min).Push(gtx.Ops)
	call.Add(gtx.Ops)
	st.Pop()
	cl.Pop()
	return layout.Dimensions{Size: size}
}

// fillRect paints r in c.
func fillRect(gtx layout.Context, r image.Rectangle, c color.NRGBA) {
	paint.FillShape(gtx.Ops, c, clip.Rect(r).Op())
}

// strokeArc strokes a circular arc of radius rad (px) around centre,
// starting at angle `from` and sweeping `sweep` radians clockwise, where 0
// points up (12 o'clock). A full turn draws a circle.
func strokeArc(gtx layout.Context, centre f32.Point, rad, width, from, sweep float32, c color.NRGBA) {
	const segPerTurn = 96
	n := max(int(math.Ceil(float64(segPerTurn*sweep/(2*math.Pi)))), 1)
	pt := func(a float32) f32.Point {
		s, co := math.Sincos(float64(a))
		return f32.Pt(centre.X+rad*float32(s), centre.Y-rad*float32(co))
	}
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(pt(from))
	for i := 1; i <= n; i++ {
		p.LineTo(pt(from + sweep*float32(i)/float32(n)))
	}
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: width}.Op())
}

// clampRadius limits a corner radius to half of sz's shorter side, as CSS
// does; a larger radius makes Gio draw spiky corners.
func clampRadius(r int, sz image.Point) int {
	return max(min(r, sz.X/2, sz.Y/2), 0)
}
