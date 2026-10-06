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

// dropdown is a field that opens a list to choose from. The prototype used
// the browser's own <select>; here the closed field looks like it (an input
// box with a chevron) and opens a list in the style of the ☰ card menu
// (owner's choice, option A).
type dropdown struct {
	selected int
	field    widget.Clickable
	items    []widget.Clickable
	open     bool
	openedAt time.Time
	scrim    int
	box      int
}

// dropdownStyle is the closed field's look: the dialog's input (.rmodal-
// select) or the form's (.finput).
type dropdownStyle struct {
	Radius float32
	Pad    layout.Inset
	Size   float32
}

var (
	dialogSelect = dropdownStyle{Radius: 11, Pad: layout.Inset{Top: 11, Bottom: 11, Left: 13, Right: 13}, Size: 14}
	formSelect   = dropdownStyle{Radius: 9, Pad: layout.Inset{Top: 9, Bottom: 9, Left: 11, Right: 11}, Size: 13}
)

// Layout draws the field with labels[selected] and, when open, the list
// (icons may be nil). It also reports whether the choice changed.
func (p *dropdown) Layout(gtx layout.Context, a *App, labels []string, icons []*Icon, st dropdownStyle) (dims layout.Dimensions, changed bool) {
	th := a.Theme
	if len(p.items) != len(labels) {
		p.items = make([]widget.Clickable, len(labels))
	}
	p.selected = min(max(p.selected, 0), max(len(labels)-1, 0))
	if p.field.Clicked(gtx) && len(labels) > 0 {
		p.open = !p.open
		p.openedAt = a.Now()
	}
	for i := range p.items {
		if p.items[i].Clicked(gtx) {
			changed = changed || i != p.selected
			p.selected = i
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
		return card(gtx, th.Bg, border, unitDp(st.Radius), st.Pad, 0, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			label := ""
			if len(labels) > 0 {
				label = labels[p.selected]
			}
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, fillWidth(func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Sans}, Size: unitSp(st.Size), MaxLines: 1, Color: th.Text}.Layout(gtx, th, label)
				})),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return IconChevronDown.Layout(gtx, unitDp(st.Size+2), 2, th.Muted)
				}),
			)
		})
	})
	if p.open {
		p.layoutList(gtx, a, d.Size, labels, icons)
	}
	return d, changed
}

// layoutList draws the open list just under the field, over everything.
func (p *dropdown) layoutList(gtx layout.Context, a *App, field image.Point, labels []string, icons []*Icon) {
	th := a.Theme
	rec := op.Record(gtx.Ops)
	big := clip.Rect{Min: image.Pt(-1<<20, -1<<20), Max: image.Pt(1<<20, 1<<20)}.Push(gtx.Ops)
	event.Op(gtx.Ops, &p.scrim)
	big.Pop()

	lgtx := gtx
	lgtx.Constraints = layout.Constraints{Min: image.Pt(max(field.X, gtx.Dp(158)), 0), Max: image.Pt(max(field.X, gtx.Dp(158)), 1<<20)}
	lrec := op.Record(gtx.Ops)
	ld := card(lgtx, th.Surface, th.Border, 13, layout.UniformInset(5), 0, func(gtx layout.Context) layout.Dimensions {
		children := make([]layout.FlexChild, len(labels))
		for i := range labels {
			children[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return p.items[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					on := i == p.selected
					rec := op.Record(gtx.Ops)
					d := layout.Inset{Top: 9, Bottom: 9, Left: 11, Right: 11}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if i >= len(icons) || icons[i] == nil {
									return layout.Dimensions{}
								}
								return layout.Inset{Right: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return icons[i].Layout(gtx, 14, 1.8, th.Muted)
								})
							}),
							layout.Flexed(1, fillWidth(func(gtx layout.Context) layout.Dimensions {
								c := th.Text
								if on {
									c = th.Accent
								}
								return Text{Font: font.Font{Typeface: Sans}, Size: 13, MaxLines: 1, Color: c}.Layout(gtx, th, labels[i])
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

// typePicker is the "What lives here" field in the category dialogs.
type typePicker struct {
	dropdown
	value core.EntryType
}

func (p *typePicker) Layout(gtx layout.Context, a *App) layout.Dimensions {
	labels := make([]string, len(core.EntryTypes))
	icons := make([]*Icon, len(core.EntryTypes))
	for i, t := range core.EntryTypes {
		labels[i], icons[i] = t.Label(), typeIcon(t)
		if t == p.value {
			p.selected = i
		}
	}
	d, changed := p.dropdown.Layout(gtx, a, labels, icons, dialogSelect)
	if changed {
		p.value = core.EntryTypes[p.selected]
	}
	return d
}
