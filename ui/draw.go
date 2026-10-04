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
// (CSS overflow:hidden), and padded by pad. The card is at least minW wide.
func card(gtx layout.Context, bg, border color.NRGBA, radius unit.Dp, pad layout.Inset, minW unit.Dp, content layout.Widget) layout.Dimensions {
	b := gtx.Dp(1)
	r := gtx.Dp(radius)

	// Measure the content first so we know the card's size.
	cgtx := gtx
	cgtx.Constraints.Min = image.Point{}
	cgtx.Constraints.Max.X -= 2 * b
	cgtx.Constraints.Max.Y -= 2 * b
	rec := op.Record(gtx.Ops)
	cd := pad.Layout(cgtx, content)
	call := rec.Stop()

	size := image.Pt(max(cd.Size.X+2*b, gtx.Dp(minW), gtx.Constraints.Min.X), max(cd.Size.Y+2*b, gtx.Constraints.Min.Y))
	outer := image.Rectangle{Max: size}
	inner := outer.Inset(b)

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

// svgPath is a tiny subset of SVG path drawing in a 24×24 icon box, enough
// for the prototype's line icons. Coordinates are in icon units.
type svgPath struct {
	p     clip.Path
	scale float32
}

func (s *svgPath) pt(x, y float32) f32.Point { return f32.Pt(x*s.scale, y*s.scale) }
func (s *svgPath) M(x, y float32)            { s.p.MoveTo(s.pt(x, y)) }
func (s *svgPath) L(x, y float32)            { s.p.LineTo(s.pt(x, y)) }
func (s *svgPath) Z()                        { s.p.Close() }

// Q90 draws a quarter-circle from the pen to (x, y) bending around (cx, cy),
// the corner the arc rounds off (where the two straight edges would meet).
func (s *svgPath) Q90(cx, cy, x, y float32) {
	const k = 0.5523 // cubic Bézier constant for a quarter circle
	from := s.p.Pos()
	c := s.pt(cx, cy)
	to := s.pt(x, y)
	c1 := from.Add(c.Sub(from).Mul(k))
	c2 := to.Add(c.Sub(to).Mul(k))
	s.p.CubeTo(c1, c2, to)
}

// strokeIcon draws icon (defined in a 24×24 box) at size, centred in the
// current constraints' minimum, with the given stroke width in icon units.
func strokeIcon(gtx layout.Context, size unit.Dp, strokeW float32, c color.NRGBA, icon func(*svgPath)) layout.Dimensions {
	px := gtx.Dp(size)
	s := &svgPath{scale: float32(px) / 24}
	s.p.Begin(gtx.Ops)
	icon(s)
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: s.p.End(), Width: strokeW * s.scale}.Op())
	return layout.Dimensions{Size: image.Pt(px, px)}
}

// bookIcon is the prototype's book glyph (Feather "book").
func bookIcon(s *svgPath) {
	// M4 19.5 A2.5 2.5 0 0 1 6.5 17 H20
	s.M(4, 19.5)
	s.Q90(4, 17, 6.5, 17)
	s.L(20, 17)
	// M6.5 2 H20 V22 H6.5 A2.5 2.5 0 0 1 4 19.5 V4.5 A2.5 2.5 0 0 1 6.5 2 Z
	s.M(6.5, 2)
	s.L(20, 2)
	s.L(20, 22)
	s.L(6.5, 22)
	s.Q90(4, 22, 4, 19.5)
	s.L(4, 4.5)
	s.Q90(4, 2, 6.5, 2)
	s.Z()
}
