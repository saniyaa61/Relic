package ui

import (
	"fmt"
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// Real taps on a phone: the finger lands, wobbles a pixel or two and
// lifts. Every one must open its page on the next frame, tap after tap,
// for buttons and cards inside scrolling pages as well as the bottom bar
// (owner, 2026-11-01: taps not responsive).
func TestRealTapsOpenPagesEveryTime(t *testing.T) {
	a, _ := libApp(t)
	var r input.Router
	var ops op.Ops
	clock := time.Duration(0)
	draw := func() {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Source: r.Source(), Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2},
			Constraints: layout.Exact(image.Pt(800, 1600))}
		a.Layout(gtx, layout.Inset{})
		r.Frame(gtx.Ops)
	}
	tap := func(p f32.Point) {
		ev := func(k pointer.Kind, d f32.Point) pointer.Event {
			clock += 16 * time.Millisecond
			return pointer.Event{Kind: k, Source: pointer.Touch, Position: p.Add(d), Time: clock}
		}
		r.Queue(ev(pointer.Press, f32.Pt(0, 0)))
		draw()
		r.Queue(ev(pointer.Move, f32.Pt(1, 2)), ev(pointer.Move, f32.Pt(2, 3)))
		draw()
		r.Queue(ev(pointer.Release, f32.Pt(2, 3)))
		draw()
	}
	targets := []struct {
		name string
		at   f32.Point
		want string
	}{
		{"Log something new", f32.Pt(400, 526), "*ui.entryForm"},
		{"a Still with you card", f32.Pt(160, 860), "*ui.entryDetail"},
		{"See all", f32.Pt(728, 638), "*ui.posterPage"},
		{"the Total chip", f32.Pt(114, 368), "*ui.posterPage"},
		{"the gear", f32.Pt(730, 64), "*ui.settingsPage"},
		{"the Library tab", f32.Pt(240, 1530), "*ui.libraryRoot"},
	}
	for round := 0; round < 3; round++ {
		for _, tg := range targets {
			a.Go(TabHome)
			a.stack = nil
			draw()
			tap(tg.at)
			if got := fmt.Sprintf("%T", a.top()); got != tg.want {
				t.Errorf("round %d: tapping %s showed %s, want %s", round, tg.name, got, tg.want)
			}
		}
	}
}
