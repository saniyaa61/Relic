package ui

import (
	"image"
	"testing"
	"time"

	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// Once its animations have finished, no screen keeps redrawing: the only
// wake-ups left are a focused field's caret blink (½s), a toast's timer
// and the minute tick. A screen that asked for the next frame straight
// away would keep the phone busy and make taps lag.
func TestIdleScreensDontRedraw(t *testing.T) {
	for _, screen := range SnapshotScreens {
		if screen == "splash" { // the splash pulses on purpose while it's up
			continue
		}
		a, _ := libApp(t)
		if err := a.ShowForSnapshot(screen); err != nil {
			continue
		}
		var r input.Router
		var ops op.Ops
		frameAt := a.Now()
		for i := 0; i < 3; i++ {
			ops.Reset()
			gtx := layout.Context{Ops: &ops, Now: frameAt, Source: r.Source(), Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2},
				Constraints: layout.Exact(image.Pt(800, 1600))}
			a.Layout(gtx, layout.Inset{})
			r.Frame(gtx.Ops)
		}
		if wt, ok := r.WakeupTime(); ok && wt.Sub(frameAt) < 300*time.Millisecond && wt.Sub(time.Now()) < 300*time.Millisecond {
			t.Errorf("%s: asks for another frame %v after the last", screen, wt.Sub(frameAt))
		}
	}
}
