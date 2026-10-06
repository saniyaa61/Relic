package ui

import (
	"image"
	"image/color"
	"math"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
)

// dashedRRect strokes a rounded rectangle with a dashed line, like CSS
// "border: 1.5px dashed". Gio has no dashes, so we walk the outline and
// stroke each dash; Chrome's dashes at this width are 3px with 2px gaps.
func dashedRRect(gtx layout.Context, sz image.Point, radius int, width, dash, gap unit.Dp, c color.NRGBA) {
	w := gtx.Metric.PxPerDp * float32(width)
	h := w / 2
	r := float32(clampRadius(radius, sz))
	x0, y0, x1, y1 := h, h, float32(sz.X)-h, float32(sz.Y)-h
	r = max(r-h, 0)
	// The outline as a polyline: straight edges and quarter arcs.
	var pts []f32.Point
	arc := func(cx, cy, from float32) {
		const steps = 8
		for i := 0; i <= steps; i++ {
			a := float64(from) + float64(i)/steps*math.Pi/2
			pts = append(pts, f32.Pt(cx+r*float32(math.Cos(a)), cy+r*float32(math.Sin(a))))
		}
	}
	arc(x1-r, y0+r, -math.Pi/2)
	arc(x1-r, y1-r, 0)
	arc(x0+r, y1-r, math.Pi/2)
	arc(x0+r, y0+r, math.Pi)
	pts = append(pts, pts[0])

	dashPx, gapPx := float32(gtx.Metric.PxPerDp)*float32(dash), float32(gtx.Metric.PxPerDp)*float32(gap)
	var p clip.Path
	p.Begin(gtx.Ops)
	on, left := true, dashPx // drawing a dash?, length left in this dash or gap
	p.MoveTo(pts[0])
	for i := 1; i < len(pts); i++ {
		a, b := pts[i-1], pts[i]
		seg := b.Sub(a)
		l := float32(math.Hypot(float64(seg.X), float64(seg.Y)))
		pos := float32(0)
		for l-pos > left {
			pos += left
			pt := a.Add(seg.Mul(pos / l))
			if on {
				p.LineTo(pt)
			} else {
				p.MoveTo(pt)
			}
			on = !on
			if on {
				left = dashPx
			} else {
				left = gapPx
			}
		}
		left -= l - pos
		if on {
			p.LineTo(b)
		} else {
			p.MoveTo(b)
		}
	}
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: w}.Op())
}

// addTile is the prototype's dashed .add-tile ("+ New category"). The
// small form (.add-tile.small, "New folder" in the folder grid) stacks the
// plus above the words and fills its grid cell.
func addTile(gtx layout.Context, th *Theme, c *widget.Clickable, label string, small bool) layout.Dimensions {
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		border, bg := th.Border, color.NRGBA{}
		if c.Pressed() {
			border, bg = th.Accent, th.Tag
		}
		size := 14
		pad := layout.UniformInset(15)
		if small {
			size = 13
			pad = layout.Inset{Top: 16, Bottom: 16, Left: 13, Right: 13}
		}
		rec := op.Record(gtx.Ops)
		pgtx := gtx
		pgtx.Constraints.Min.Y = 0 // the tile centres its content in the cell instead
		d := pad.Layout(pgtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			txt := func(gtx layout.Context) layout.Dimensions {
				return Paragraph{Font: font.Font{Typeface: Display, Style: font.Italic}, Size: unit.Sp(size), LineHeight: 1.3, Alignment: text.Middle, Color: th.Accent}.Layout(gtx, th, label)
			}
			plus := func(gtx layout.Context) layout.Dimensions { return IconPlus.Layout(gtx, 15, 2, th.Accent) }
			if small {
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle, Spacing: layout.SpaceSides}.Layout(gtx,
					layout.Rigid(plus), layout.Rigid(layout.Spacer{Height: 7}.Layout), layout.Rigid(txt))
			}
			return layout.Flex{Alignment: layout.Middle, Spacing: layout.SpaceSides}.Layout(gtx,
				layout.Rigid(plus), layout.Rigid(layout.Spacer{Width: 8}.Layout), layout.Rigid(txt))
		})
		call := rec.Stop()
		sz := image.Pt(gtx.Constraints.Max.X, max(d.Size.Y, gtx.Constraints.Min.Y))
		rad := gtx.Dp(16)
		if bg.A > 0 {
			rrect(gtx, sz, rad, bg)
		}
		dashedRRect(gtx, sz, rad, 1.5, 3, 2, border)
		off := (sz.Y - d.Size.Y) / 2
		st := op.Offset(image.Pt(0, off)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		st.Pop()
		return layout.Dimensions{Size: sz}
	})
}

// cardDrop plays the prototype's cardDrop animation on the index-th card
// of a grid that appeared at start: each card falls 18px into place while
// growing from 97% and fading in, 0.5s, 55ms after the previous one
// (capped at 440ms) (SPEC §6).
func cardDrop(gtx layout.Context, now, start time.Time, index int, w layout.Widget) layout.Dimensions {
	delay := time.Duration(min(index*55, 440)) * time.Millisecond
	t := progress(gtx, now, start.Add(delay), 500*time.Millisecond)
	if t >= 1 {
		return w(gtx)
	}
	e := cardDropEase(t)
	opacity := min(t/0.55, 1)
	rec := op.Record(gtx.Ops)
	d := w(gtx)
	call := rec.Stop()
	scale := lerpf(0.97, 1, e)
	dy := float32(gtx.Dp(18)) * (e - 1)
	c := f32.Pt(float32(d.Size.X)/2, float32(d.Size.Y)/2)
	tr := f32.AffineId().Scale(c, f32.Pt(scale, scale)).Offset(f32.Pt(0, dy))
	defer paint.PushOpacity(gtx.Ops, opacity).Pop()
	defer op.Affine(tr).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	return d
}

var cardDropEase = cubicBezier(0.22, 1, 0.36, 1)

// cardMenu is the ☰ button in a card's top-right corner and the small
// pop-up menu it opens (prototype .card-menu-btn and .menu-pop.card).
type cardMenu struct {
	btn      widget.Clickable
	items    []menuItem
	open     bool
	openedAt time.Time
	scrim    int // tag for taps outside the menu
	box      int // tag that keeps taps on the menu from reaching the scrim
}

type menuItem struct {
	Icon   *Icon
	Label  string
	Danger bool
	click  widget.Clickable
}

// Layout draws the ☰ button at the top-right of a card of width cardW,
// and the menu when open. It returns the index of a chosen item, or -1.
// The menu is drawn after everything else (op.Defer), so it sits over the
// neighbouring cards, as in the prototype.
func (m *cardMenu) Layout(gtx layout.Context, a *App, cardW int) int {
	th := a.Theme
	chosen := -1
	if m.btn.Clicked(gtx) {
		m.open = !m.open
		m.openedAt = a.Now()
	}
	for i := range m.items {
		if m.items[i].click.Clicked(gtx) {
			m.open = false
			chosen = i
		}
	}
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &m.scrim, Kinds: pointer.Press})
		if !ok {
			break
		}
		if e, ok := ev.(pointer.Event); ok && e.Kind == pointer.Press {
			m.open = false
		}
	}
	for {
		if _, ok := gtx.Event(pointer.Filter{Target: &m.box, Kinds: pointer.Press | pointer.Release}); !ok {
			break
		}
	}

	// The button: 25×25, 8px from the top right, a 14px muted ☰.
	sz := gtx.Dp(25)
	st := op.Offset(image.Pt(cardW-gtx.Dp(8)-sz, gtx.Dp(8))).Push(gtx.Ops)
	m.btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		c := th.Muted
		if m.btn.Pressed() {
			rrect(gtx, image.Pt(sz, sz), gtx.Dp(8), th.Tag)
			c = th.Accent
		}
		off := (sz - gtx.Dp(14)) / 2
		defer op.Offset(image.Pt(off, off)).Push(gtx.Ops).Pop()
		IconMenu.Layout(gtx, 14, 1.9, c)
		return layout.Dimensions{Size: image.Pt(sz, sz)}
	})
	st.Pop()

	if !m.open {
		return chosen
	}
	rec := op.Record(gtx.Ops)
	// Taps anywhere else close the menu.
	big := clip.Rect{Min: image.Pt(-1<<20, -1<<20), Max: image.Pt(1<<20, 1<<20)}.Push(gtx.Ops)
	event.Op(gtx.Ops, &m.scrim)
	big.Pop()
	// The menu: 36px down, 6px in from the right; at least 158px wide.
	mgtx := gtx
	mgtx.Constraints = layout.Constraints{Min: image.Pt(gtx.Dp(158), 0), Max: image.Pt(gtx.Dp(158), gtx.Dp(400))}
	mrec := op.Record(gtx.Ops)
	md := card(mgtx, th.Surface, th.Border, 13, layout.UniformInset(5), 158, func(gtx layout.Context) layout.Dimensions {
		children := make([]layout.FlexChild, len(m.items))
		for i := range m.items {
			children[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return m.items[i].layout(gtx, th)
			})
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
	menu := mrec.Stop()
	t := easeCSS(progress(gtx, a.Now(), m.openedAt, 160*time.Millisecond))
	pos := image.Pt(cardW-gtx.Dp(6)-md.Size.X, gtx.Dp(36))
	// Grows from its top-right corner: 6px higher at 97% size.
	scale := lerpf(0.97, 1, t)
	origin := f32.Pt(float32(pos.X+md.Size.X), float32(pos.Y))
	tr := f32.AffineId().Offset(f32.Pt(float32(pos.X), float32(pos.Y)-float32(gtx.Dp(6))*(1-t))).Scale(origin, f32.Pt(scale, scale))
	o := paint.PushOpacity(gtx.Ops, t)
	at := op.Affine(tr).Push(gtx.Ops)
	boxShadow(gtx, md.Size, gtx.Dp(13), 12, 30, 0.2, th.Bg)
	area := clip.Rect{Max: md.Size}.Push(gtx.Ops)
	event.Op(gtx.Ops, &m.box)
	area.Pop()
	menu.Add(gtx.Ops)
	at.Pop()
	o.Pop()
	op.Defer(gtx.Ops, rec.Stop())
	return chosen
}

func (it *menuItem) layout(gtx layout.Context, th *Theme) layout.Dimensions {
	fg, ic := th.Text, th.Muted
	if it.Danger {
		fg, ic = dangerRed, dangerRed
	}
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return it.click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		rec := op.Record(gtx.Ops)
		d := layout.Inset{Top: 9, Bottom: 9, Left: 11, Right: 11}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return it.Icon.Layout(gtx, 14, 1.8, ic) }),
				layout.Rigid(layout.Spacer{Width: 10}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Sans}, Size: 13, Color: fg}.Layout(gtx, th, it.Label)
				}),
			)
		})
		call := rec.Stop()
		if it.click.Pressed() {
			rrect(gtx, d.Size, gtx.Dp(9), th.Tag)
		}
		call.Add(gtx.Ops)
		return d
	})
}

// renameDeleteItems are the ☰ menu's two choices on category and folder
// boxes.
func renameDeleteItems() []menuItem {
	return []menuItem{{Icon: IconPen, Label: "Rename"}, {Icon: IconTrash, Label: "Delete", Danger: true}}
}
