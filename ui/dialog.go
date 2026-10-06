package ui

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"strings"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/saniyaa61/relic/core"
)

// Dialog is the prototype's in-page modal (.rmodal): a centred card over a
// dimmed page, with a title, an optional italic sub-line, an optional text
// field, and Cancel / Confirm buttons. Build one with PromptDialog or
// ConfirmDialog and open it with App.ShowDialog.
type Dialog struct {
	Title, Sub string
	// Label is the field's label; a dialog without one has no field.
	Label, Placeholder string
	Cancel, Confirm    string
	Danger             bool
	// Type, if set, adds the "What lives here" picker (category dialogs);
	// OnConfirm reads the choice with Dialog.ChosenType.
	Type *core.EntryType
	// OnConfirm gets the trimmed field text. A prompt with an empty field
	// doesn't confirm; it puts the cursor back in the field instead.
	OnConfirm func(a *App, value string)
	// OnCancel, if set, runs when the Cancel button (not the scrim) is
	// tapped; the calendar uses it for "Clear".
	OnCancel func(a *App)
	// Body, if set, is drawn under the title and sub-line (the calendar).
	Body func(gtx layout.Context, a *App) layout.Dimensions
	// KeepOpen leaves the dialog up after Confirm (the Year card's "Save
	// image"); Cancel and the scrim still close it.
	KeepOpen bool
	// Wide is the Year card preview's box: min(380px, 100% − 28px) with
	// 16px padding and the buttons 14px below, instead of the usual
	// min(342px, 100% − 40px), 21px and 19px.
	Wide bool

	input      Input
	cancel, ok Button
	scrim      widget.Clickable
	openedAt   time.Time
	closedAt   time.Time
	focusField bool
	cardTag    int
	picker     typePicker
}

// ChosenType is the type picked in a dialog with a Type picker.
func (d *Dialog) ChosenType() core.EntryType { return d.picker.value }

// PromptDialog asks for a name (prototype openPromptModal). label defaults
// to "Name" and confirm to "Save".
func PromptDialog(title, sub, label, placeholder, value, confirm string, onConfirm func(a *App, value string)) *Dialog {
	if label == "" {
		label = "Name"
	}
	if confirm == "" {
		confirm = "Save"
	}
	d := &Dialog{Title: title, Sub: sub, Label: label, Placeholder: placeholder,
		Cancel: "Cancel", Confirm: confirm, OnConfirm: onConfirm, focusField: true}
	d.input.Editor.SetText(value)
	d.input.Editor.Submit = true
	return d
}

// ConfirmDialog asks before something destructive (prototype
// openConfirmModal): "Keep it" and a red "Delete" unless given other words.
func ConfirmDialog(title, sub, cancel, confirm string, onConfirm func(a *App)) *Dialog {
	if cancel == "" {
		cancel = "Keep it"
	}
	if confirm == "" {
		confirm = "Delete"
	}
	return &Dialog{Title: title, Sub: sub, Cancel: cancel, Confirm: confirm, Danger: true,
		OnConfirm: func(a *App, _ string) { onConfirm(a) }}
}

func (d *Dialog) closing() bool { return !d.closedAt.IsZero() }

func (d *Dialog) close(a *App) {
	if !d.closing() {
		d.closedAt = a.Now()
	}
}

func (d *Dialog) submit(gtx layout.Context, a *App) {
	v := strings.TrimSpace(d.input.Editor.Text())
	if d.Label != "" && v == "" {
		d.input.Focus(gtx)
		return
	}
	if !d.KeepOpen {
		d.close(a)
	}
	if d.OnConfirm != nil {
		d.OnConfirm(a, v)
	}
}

// Layout draws the dialog over the whole window and reports whether it
// has finished closing.
func (d *Dialog) Layout(gtx layout.Context, a *App) (done bool) {
	th := a.Theme
	// Opening: scrim fades in over 0.2s; the card fades in and rises from
	// 4% of its height lower over 0.22s with a slight overshoot. Closing
	// plays the same in reverse, and the dialog is gone after 0.24s.
	now := a.Now()
	var tScrim, tCard, tRise float32
	if d.closing() {
		if now.Sub(d.closedAt) >= 240*time.Millisecond {
			return true
		}
		progress(gtx, now, d.closedAt, 240*time.Millisecond)
		tScrim = 1 - easeCSS(progress(gtx, now, d.closedAt, 200*time.Millisecond))
		tCard = 1 - easeCSS(progress(gtx, now, d.closedAt, 220*time.Millisecond))
		tRise = 1 - easeSpring(progress(gtx, now, d.closedAt, 220*time.Millisecond))
	} else {
		tScrim = easeCSS(progress(gtx, now, d.openedAt, 200*time.Millisecond))
		tCard = easeCSS(progress(gtx, now, d.openedAt, 220*time.Millisecond))
		tRise = easeSpring(progress(gtx, now, d.openedAt, 220*time.Millisecond))
	}

	if !d.closing() {
		if d.cancel.Click.Clicked(gtx) {
			d.close(a)
			if d.OnCancel != nil {
				d.OnCancel(a)
			}
		}
		if d.scrim.Clicked(gtx) {
			d.close(a)
		}
		if d.ok.Click.Clicked(gtx) {
			d.submit(gtx, a)
		}
		for {
			ev, ok := d.input.Editor.Update(gtx)
			if !ok {
				break
			}
			if _, ok := ev.(widget.SubmitEvent); ok {
				d.submit(gtx, a)
			}
		}
		if d.focusField {
			// Like the prototype: cursor in the field, any text selected
			// (so a rename can be typed straight over).
			d.input.Focus(gtx)
			d.input.Editor.SetCaret(d.input.Editor.Len(), 0)
			d.focusField = false
		}
	}

	size := gtx.Constraints.Max
	// The scrim: rgba(0,0,0,.45), and taps on it close the dialog.
	d.scrim.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, shade(0.45*tScrim, th.Bg), clip.Rect{Max: size}.Op())
		return layout.Dimensions{Size: size}
	})

	// The card: min(342px, 100% - 40px) wide, centred.
	w := min(gtx.Dp(342), size.X-gtx.Dp(40))
	if d.Wide {
		w = min(gtx.Dp(380), size.X-gtx.Dp(28))
	}
	cgtx := gtx
	cgtx.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, size.Y)}
	rec := op.Record(gtx.Ops)
	cd := d.layoutCard(cgtx, a)
	call := rec.Stop()
	rise := roundi(float32(cd.Size.Y) * 0.04 * (1 - tRise))
	pos := image.Pt((size.X-cd.Size.X)/2, (size.Y-cd.Size.Y)/2+rise)

	defer paint.PushOpacity(gtx.Ops, tCard).Pop()
	defer op.Offset(pos).Push(gtx.Ops).Pop()
	boxShadow(gtx, cd.Size, gtx.Dp(19), 18, 50, 0.25, th.Bg)
	// The card takes taps that miss its buttons, so they don't fall
	// through to the scrim and close the dialog.
	area := clip.Rect{Max: cd.Size}.Push(gtx.Ops)
	event.Op(gtx.Ops, &d.cardTag)
	for {
		if _, ok := gtx.Event(pointer.Filter{Target: &d.cardTag, Kinds: pointer.Press | pointer.Release}); !ok {
			break
		}
	}
	area.Pop()
	call.Add(gtx.Ops)
	return false
}

func (d *Dialog) layoutCard(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	pad, actsTop := unitDp(21), unitDp(19)
	if d.Wide {
		pad, actsTop = 16, 14
	}
	return card(gtx, th.Surface, th.Border, 19, layout.UniformInset(pad), 0, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		var rows []layout.FlexChild
		if d.Title != "" {
			rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 19, Color: th.Text}.Layout(gtx, th, d.Title)
			}))
		}
		if d.Sub != "" {
			rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 5}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 12.5, LineHeight: 1.6, Color: th.Muted}.Layout(gtx, th, d.Sub)
				})
			}))
		}
		if d.Label != "" {
			rows = append(rows,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: 14, Bottom: 5}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.07, Upper: true, Color: th.Muted}.Layout(gtx, th, d.Label)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return d.input.Layout(gtx, th, InputStyle{
						Bg: th.Bg, Radius: 11, Size: 14,
						Pad:              layout.Inset{Top: 11, Bottom: 11, Left: 13, Right: 13},
						Placeholder:      d.Placeholder,
						PlaceholderAlpha: 1,
					})
				}),
			)
		}
		if d.Body != nil {
			top := unitDp(14)
			if len(rows) == 0 {
				top = 0
			}
			rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: top}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return d.Body(gtx, a)
				})
			}))
		}
		if d.Type != nil {
			rows = append(rows,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: 14, Bottom: 5}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.07, Upper: true, Color: th.Muted}.Layout(gtx, th, "What lives here")
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return d.picker.Layout(gtx, a) }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 11.5, LineHeight: 1.6, Color: th.Muted}.Layout(gtx, th, "This only decides which fields appear when you log an entry here.")
					})
				}),
			)
		}
		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: actsTop}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				style := ButtonAccent
				if d.Danger {
					style = ButtonDanger
				}
				return layout.Flex{}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return d.cancel.Layout(gtx, th, ButtonQuiet, d.Cancel)
					}),
					layout.Rigid(layout.Spacer{Width: 8}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return d.ok.Layout(gtx, th, style, d.Confirm)
					}),
				)
			})
		}))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
	})
}
