package ui

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// Five quick taps in the top-left corner turn the timing display on, and
// taps still reach the page underneath (the Home gear opens Settings).
func TestPerfMeterToggleAndPassThrough(t *testing.T) {
	a, _ := libApp(t)
	var r input.Router
	var ops op.Ops
	draw := func() {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Source: r.Source(), Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2},
			Constraints: layout.Exact(image.Pt(800, 1600))}
		a.Layout(gtx, layout.Inset{})
		r.Frame(gtx.Ops)
	}
	tap := func(x, y float32) {
		draw()
		r.Queue(
			pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Position: f32.Pt(x, y)},
			pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: f32.Pt(x, y)},
		)
		draw()
	}
	for i := 0; i < 5; i++ {
		tap(40, 60)
	}
	if !a.perf.on {
		t.Fatal("five taps in the corner should show the timing display")
	}
	gear := a.guideRects.gear.Min.Add(a.guideRects.gear.Size().Div(2))
	tap(float32(gear.X), float32(gear.Y))
	draw()
	if _, ok := a.top().(*settingsPage); !ok {
		t.Errorf("a tap on the gear should still open Settings, got %T", a.top())
	}
	if a.perf.waiting {
		t.Error("the page appeared: the tap's time should be measured")
	}
}
