package ui

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/saniyaa61/relic/core"
)

// typePicker is the "What lives here" field in the category dialogs. The
// prototype used the browser's own <select>; here the closed field looks
// like it (the dialog's input style with a chevron) and opens a list in
// the style of the ☰ card menu.
type typePicker struct {
	value    core.EntryType
	field    widget.Clickable
	items    [7]widget.Clickable
	open     bool
	openedAt time.Time
	scrim    int
	box      int
}

func (p *typePicker) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	if p.field.Clicked(gtx) {
		p.open = !p.open
		p.openedAt = a.Now()
	}
	for i := range p.items {
		if p.items[i].Clicked(gtx) {
			p.value = core.EntryTypes[i]
			p.open = false
		}
	}
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &p.scrim, Kinds: pointer.Press})
		if !ok {
			break
		}
		if e, ok := ev.(pointer.Event); ok && e.Kind == pointer.Press {
			p.open = false
		}
	}
	for {
		if _, ok := gtx.Event(pointer.Filter{Target: &p.box, Kinds: pointer.Press | pointer.Release}); !ok {
			break
		}
	}

	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	d := p.field.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		border := th.Border
		if p.open {
			border = th.Accent
		}
		return card(gtx, th.Bg, border, 11, layout.Inset{Top: 11, Bottom: 11, Left: 13, Right: 13}, 0, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, fillWidth(func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Sans}, Size: 14, Color: th.Text}.Layout(gtx, th, p.value.Label())
				})),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return IconChevronDown.Layout(gtx, 16, 2, th.Muted)
				}),
			)
		})
	})
	if p.open {
		p.layoutList(gtx, a, d.Size)
	}
	return d
}

// layoutList draws the open list just under the field, over everything.
func (p *typePicker) layoutList(gtx layout.Context, a *App, field image.Point) {
	th := a.Theme
	rec := op.Record(gtx.Ops)
	big := clip.Rect{Min: image.Pt(-1<<20, -1<<20), Max: image.Pt(1<<20, 1<<20)}.Push(gtx.Ops)
	event.Op(gtx.Ops, &p.scrim)
	big.Pop()

	lgtx := gtx
	lgtx.Constraints = layout.Exact(image.Pt(field.X, 0))
	lgtx.Constraints.Max.Y = 1 << 20
	lrec := op.Record(gtx.Ops)
	ld := card(lgtx, th.Surface, th.Border, 13, layout.UniformInset(5), 0, func(gtx layout.Context) layout.Dimensions {
		children := make([]layout.FlexChild, len(p.items))
		for i := range p.items {
			t := core.EntryTypes[i]
			children[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return p.items[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					on := t == p.value
					rec := op.Record(gtx.Ops)
					d := layout.Inset{Top: 9, Bottom: 9, Left: 11, Right: 11}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return typeIcon(t).Layout(gtx, 14, 1.8, th.Muted) }),
							layout.Rigid(layout.Spacer{Width: 10}.Layout),
							layout.Flexed(1, fillWidth(func(gtx layout.Context) layout.Dimensions {
								c := th.Text
								if on {
									c = th.Accent
								}
								return Text{Font: font.Font{Typeface: Sans}, Size: 13, Color: c}.Layout(gtx, th, t.Label())
							})),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if !on {
									return layout.Dimensions{}
								}
								return IconCheck.Layout(gtx, 14, 2, th.Accent)
							}),
						)
					})
					call := rec.Stop()
					if on || p.items[i].Pressed() {
						rrect(gtx, d.Size, gtx.Dp(9), th.Tag)
					}
					call.Add(gtx.Ops)
					return d
				})
			})
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
	list := lrec.Stop()

	t := easeCSS(progress(gtx, a.Now(), p.openedAt, 160*time.Millisecond))
	y := float32(field.Y+gtx.Dp(6)) - float32(gtx.Dp(6))*(1-t)
	o := paint.PushOpacity(gtx.Ops, t)
	at := op.Affine(f32.AffineId().Offset(f32.Pt(0, y))).Push(gtx.Ops)
	boxShadow(gtx, ld.Size, gtx.Dp(13), 12, 30, 0.2, th.Bg)
	area := clip.Rect{Max: ld.Size}.Push(gtx.Ops)
	event.Op(gtx.Ops, &p.box)
	area.Pop()
	list.Add(gtx.Ops)
	at.Pop()
	o.Pop()
	op.Defer(gtx.Ops, rec.Stop())
}
