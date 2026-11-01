package ui

// The launch splash (the prototype's #splash): the framed rose and RELIC on
// black, pulsing gently, for 1.6s, then fading out over 0.8s.

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"sync"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/saniyaa61/relic/assets"
)

const (
	splashHold  = 1600 * time.Millisecond
	splashFade  = 800 * time.Millisecond
	splashPulse = 2200 * time.Millisecond
)

var (
	splashOnce sync.Once
	splashImg  paint.ImageOp
	splashOK   bool
	easeInOut  = cubicBezier(0.42, 0, 0.58, 1) // CSS "ease-in-out"
)

// ShowSplash shows the launch splash from now (cmd/relic calls it at start).
func (a *App) ShowSplash() { a.splashAt = a.Now() }

// splashing reports whether the splash is still on screen.
func (a *App) splashing() bool {
	return !a.splashAt.IsZero() && a.Now().Sub(a.splashAt) < splashHold+splashFade
}

// layoutSplash draws the splash over everything and swallows touches
// while it's up.
func (a *App) layoutSplash(gtx layout.Context) {
	if !a.splashing() {
		a.splashAt = time.Time{}
		return
	}
	splashOnce.Do(func() {
		img, err := png.Decode(bytes.NewReader(assets.Splash))
		if err == nil {
			splashImg, splashOK = paint.NewImageOp(img), true
		}
	})
	now := a.Now()
	size := gtx.Constraints.Max
	// The whole splash fades after the hold (#splash.fade, opacity .8s ease).
	alpha := float32(1)
	if since := now.Sub(a.splashAt); since > splashHold {
		alpha = 1 - easeCSS(progress(gtx, now, a.splashAt.Add(splashHold), splashFade))
	} else {
		gtx.Execute(op.InvalidateCmd{})
	}
	defer paint.PushOpacity(gtx.Ops, alpha).Pop()

	area := clip.Rect{Max: size}.Push(gtx.Ops)
	event.Op(gtx.Ops, &a.splashAt)
	for {
		if _, ok := gtx.Event(pointer.Filter{Target: &a.splashAt, Kinds: pointer.Press | pointer.Release | pointer.Cancel | pointer.Drag}); !ok {
			break
		}
	}
	paint.ColorOp{Color: color.NRGBA{A: 0xFF}}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	area.Pop()

	if !splashOK {
		return
	}
	// width: min(340px, 80vw), height auto; relicPulse: scale 1 → 1.07 and
	// opacity 1 → 0.85 at the half-way point, ease-in-out, every 2.2s.
	src := splashImg.Size()
	w := min(float32(gtx.Dp(340)), 0.8*float32(size.X))
	phase := float32(now.Sub(a.splashAt)%splashPulse) / float32(splashPulse)
	e := easeInOut(1 - abs32(2*phase-1)) // 0 → 1 → 0 over the cycle
	scale := 1 + 0.07*e
	centre := f32.Pt(float32(size.X)/2, float32(size.Y)/2)
	k := scale * w / float32(src.X)
	tr := f32.AffineId().
		Offset(f32.Pt(-float32(src.X)/2, -float32(src.Y)/2)).
		Scale(f32.Point{}, f32.Pt(k, k)).
		Offset(centre)
	st := op.Affine(tr).Push(gtx.Ops)
	pulse := paint.PushOpacity(gtx.Ops, 1-0.15*e)
	cl := clip.Rect{Max: image.Pt(src.X, src.Y)}.Push(gtx.Ops)
	splashImg.Filter = paint.FilterLinear
	splashImg.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	cl.Pop()
	pulse.Pop()
	st.Pop()
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
