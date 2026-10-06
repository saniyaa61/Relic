package ui

import (
	"image"
	"time"

	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
)

// Sheet is the prototype's bottom sheet (.log-sheet): it slides up over
// a dimmed page with a spring (0.35s), has a small handle, a Playfair
// title and a scrolling body, and is at most 85% of the screen tall.
type Sheet struct {
	Title string
	Body  func(gtx layout.Context, a *App) layout.Dimensions

	openedAt, closedAt time.Time
	scrim              widget.Clickable
	list               widget.List
	box                int
}

// ShowSheet slides s up over the current page.
func (a *App) ShowSheet(s *Sheet) {
	s.openedAt = a.Now()
	a.sheet = s
}

// CloseSheet slides the open sheet away.
func (a *App) CloseSheet() {
	if a.sheet != nil && a.sheet.closedAt.IsZero() {
		a.sheet.closedAt = a.Now()
	}
}

func (s *Sheet) closing() bool { return !s.closedAt.IsZero() }

// Layout draws the sheet over the whole window, above the system bar
// inset bottom (px), and reports whether it has finished closing.
func (s *Sheet) Layout(gtx layout.Context, a *App, bottom int) (done bool) {
	th := a.Theme
	now := a.Now()
	var tScrim, tSlide float32
	if s.closing() {
		if now.Sub(s.closedAt) >= 350*time.Millisecond {
			return true
		}
		tScrim = 1 - easeCSS(progress(gtx, now, s.closedAt, 300*time.Millisecond))
		tSlide = 1 - easeCSS(progress(gtx, now, s.closedAt, 350*time.Millisecond))
	} else {
		tScrim = easeCSS(progress(gtx, now, s.openedAt, 300*time.Millisecond))
		tSlide = easeSpring(progress(gtx, now, s.openedAt, 350*time.Millisecond))
		if s.scrim.Clicked(gtx) {
			a.CloseSheet()
		}
	}
	for {
		if _, ok := gtx.Event(pointer.Filter{Target: &s.box, Kinds: pointer.Press | pointer.Release | pointer.Scroll | pointer.Drag}); !ok {
			break
		}
	}
	size := gtx.Constraints.Max
	s.scrim.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, shade(0.4*tScrim, th.Bg), clip.Rect{Max: size}.Op())
		return layout.Dimensions{Size: size}
	})

	maxH := size.Y * 85 / 100
	cgtx := gtx
	cgtx.Constraints = layout.Constraints{Min: image.Pt(size.X, 0), Max: image.Pt(size.X, maxH)}
	rec := op.Record(gtx.Ops)
	d := s.layoutBox(cgtx, a, bottom)
	call := rec.Stop()
	y := size.Y - d.Size.Y + int(float32(d.Size.Y)*(1-tSlide))
	defer op.Offset(image.Pt(0, y)).Push(gtx.Ops).Pop()
	area := clip.Rect{Max: d.Size}.Push(gtx.Ops)
	event.Op(gtx.Ops, &s.box)
	area.Pop()
	call.Add(gtx.Ops)
	return false
}

func (s *Sheet) layoutBox(gtx layout.Context, a *App, bottom int) layout.Dimensions {
	th := a.Theme
	r := gtx.Dp(20)
	// Measure the content to size the sheet, up to the maximum.
	pad := layout.Inset{Top: 18, Left: gutter, Right: gutter, Bottom: 32}
	s.list.Axis = layout.Vertical
	content := func(gtx layout.Context) layout.Dimensions {
		return pad.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					sz := image.Pt(gtx.Dp(34), gtx.Dp(3))
					rrect(gtx, sz, sz.Y, th.Border)
					return layout.Dimensions{Size: image.Pt(sz.X, sz.Y+gtx.Dp(16))}
				}),
				layout.Rigid(fillWidth(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Bottom: 16}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return Paragraph{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 17, Color: th.Text}.Layout(gtx, th, s.Title)
					})
				})),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return s.Body(gtx, a)
				}),
			)
		})
	}
	var scratch op.Ops
	mgtx := gtx
	mgtx.Ops = &scratch
	mgtx.Constraints.Max.Y = 1 << 20
	natural := content(mgtx).Size.Y + bottom
	h := min(natural, gtx.Constraints.Max.Y)
	size := image.Pt(gtx.Constraints.Max.X, h)

	// Surface with rounded top corners and a hairline on top.
	rr := clip.RRect{Rect: image.Rectangle{Max: image.Pt(size.X, size.Y+r)}, NE: r, NW: r}
	paint.FillShape(gtx.Ops, th.Border, rr.Op(gtx.Ops))
	inner := clip.RRect{Rect: image.Rect(0, gtx.Dp(1), size.X, size.Y+r), NE: r, NW: r}
	paint.FillShape(gtx.Ops, th.Surface, inner.Op(gtx.Ops))
	lgtx := gtx
	lgtx.Constraints = layout.Exact(image.Pt(size.X, h-bottom))
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	s.list.List.Layout(lgtx, 1, func(gtx layout.Context, _ int) layout.Dimensions { return content(gtx) })
	return layout.Dimensions{Size: size}
}
