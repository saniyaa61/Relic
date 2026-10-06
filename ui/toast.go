package ui

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/op/paint"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
)

// toast is the prototype's #relic-toast: a pill at the top centre in the
// text colour with the page colour for its words, a tick (or a cross for
// errors), shown for 2.2s. It fades in and settles from 8px higher at 97%
// size over 0.25s, and fades back out the same way.
type toast struct {
	msg     string
	isErr   bool
	shownAt time.Time
}

const (
	toastFade = 250 * time.Millisecond
	toastShow = 2200 * time.Millisecond
)

func (t *toast) show(now time.Time, msg string, isErr bool) {
	// A toast that is already showing swaps its words without animating
	// in again, and its timer restarts (as the prototype's does).
	if t.visible(now) {
		now = now.Add(-toastFade)
	}
	t.msg, t.isErr, t.shownAt = msg, isErr, now
}

func (t *toast) visible(now time.Time) bool {
	return !t.shownAt.IsZero() && now.Sub(t.shownAt) < toastShow+toastFade
}

// Layout draws the toast below the status bar (topInset px).
func (t *toast) Layout(gtx layout.Context, a *App, topInset int) {
	now := a.Now()
	if !t.visible(now) {
		return
	}
	var k float32
	if hide := t.shownAt.Add(toastShow); now.Before(hide) {
		k = easeCSS(progress(gtx, now, t.shownAt, toastFade))
		// Wake up again when it's time to fade out.
		if k == 1 {
			gtx.Execute(op.InvalidateCmd{At: hide})
		}
	} else {
		k = 1 - easeCSS(progress(gtx, now, hide, toastFade))
	}
	th := a.Theme
	// The pill hugs its words, up to 280px wide; longer messages wrap.
	cgtx := gtx
	cgtx.Constraints = layout.Constraints{Max: image.Pt(min(gtx.Dp(280), gtx.Constraints.Max.X-2*gtx.Dp(gutter)), gtx.Constraints.Max.Y)}
	rec := op.Record(gtx.Ops)
	// card adds a 1dp border (here the same colour as the fill), so the
	// padding is 1dp less than the prototype's 9px 18px.
	d := card(cgtx, th.Text, th.Text, 30, layout.Inset{Top: 8, Bottom: 8, Left: 17, Right: 17}, 0, func(gtx layout.Context) layout.Dimensions {
		ic := IconCheck
		if t.isErr {
			ic = IconXCircle
		}
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ic.Layout(gtx, 14, 2, th.Bg)
			}),
			layout.Rigid(layout.Spacer{Width: 6}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return Paragraph{Font: font.Font{Typeface: Sans}, Size: 13, Color: th.Bg}.Layout(gtx, th, t.msg)
			}),
		)
	})
	call := rec.Stop()

	// Scale about the pill's centre, as CSS transforms do.
	scale := lerpf(0.97, 1, k)
	x := float32(gtx.Constraints.Max.X-d.Size.X) / 2
	y := float32(topInset+gtx.Dp(12)) - float32(gtx.Dp(8))*(1-k)
	c := f32.Pt(x+float32(d.Size.X)/2, y+float32(d.Size.Y)/2)
	tr := f32.AffineId().Offset(f32.Pt(x, y)).Scale(c, f32.Pt(scale, scale))
	defer paint.PushOpacity(gtx.Ops, k).Pop()
	defer op.Affine(tr).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
}
