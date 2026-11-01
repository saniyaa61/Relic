package ui

// A hidden timing display, for finding out on a real phone why something
// feels slow (owner, 2026-11-01: taps "still not responsive"). Five quick
// taps in the top-left corner (over the Relic logo) turn it on or off. It
// shows how long the last tap took to bring up its page, and how long
// frames take to lay out and to arrive.

import (
	"fmt"
	"image"
	"time"

	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

type perfMeter struct {
	on        bool
	taps      []time.Time // recent presses in the corner
	lastPress time.Time   // the last press anywhere
	waiting   bool        // a press, and its page hasn't shown yet
	shownTop  Screen      // the page on top when the press came
	tapToPage time.Duration
	lastFrame time.Time
	gaps      [60]time.Duration // time between frames, while busy
	layouts   [60]time.Duration // time to lay a frame out
	n         int
	area      clip.Stack
	frameTop  Screen // the page on top when this frame started
}

// perfFrameStart notes a frame starting and returns the time. It also
// opens a window-sized area the whole frame is drawn inside, so the meter
// hears every press as a parent area does (without passing events
// through, which upset the clicks underneath); perfFrameEnd closes it.
func (a *App) perfFrameStart(gtx layout.Context) time.Time {
	a.perf.area = clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	event.Op(gtx.Ops, &a.perf)
	now := time.Now()
	p := &a.perf
	if !p.lastFrame.IsZero() {
		if gap := now.Sub(p.lastFrame); gap < time.Second {
			p.gaps[p.n%len(p.gaps)] = gap
		}
	}
	p.lastFrame = now
	if p.waiting && a.top() != p.shownTop {
		p.tapToPage = now.Sub(p.lastPress)
		p.waiting = false
	}
	p.frameTop = a.top()
	return now
}

// perfFrameEnd notes how long the frame took to lay out, listens for taps
// (passing them on to the page underneath) and draws the display if on.
func (a *App) perfFrameEnd(gtx layout.Context, start time.Time) {
	p := &a.perf
	p.layouts[p.n%len(p.layouts)] = time.Since(start)
	p.n++
	size := gtx.Constraints.Max
	p.area.Pop()
	corner := image.Rect(0, 0, gtx.Dp(140), gtx.Dp(110))
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: p, Kinds: pointer.Press | pointer.Release | pointer.Cancel})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok || e.Kind != pointer.Press {
			continue
		}
		now := time.Now()
		p.lastPress, p.waiting, p.shownTop = now, true, p.frameTop
		if e.Position.Round().In(corner) {
			p.taps = append(p.taps, now)
			if len(p.taps) > 5 {
				p.taps = p.taps[len(p.taps)-5:]
			}
			if len(p.taps) == 5 && now.Sub(p.taps[0]) < 3*time.Second {
				p.on, p.taps = !p.on, nil
			}
		}
	}
	if !p.on {
		return
	}
	var gapSum, gapMax, laySum, layMax time.Duration
	count := min(p.n, len(p.gaps))
	for i := 0; i < count; i++ {
		gapSum += p.gaps[i]
		gapMax = max(gapMax, p.gaps[i])
		laySum += p.layouts[i]
		layMax = max(layMax, p.layouts[i])
	}
	ms := func(d time.Duration) string { return fmt.Sprintf("%.0f", float64(d.Microseconds())/1000) }
	lines := []string{
		"tap → page " + ms(p.tapToPage) + " ms",
		"frame gap " + ms(gapSum/time.Duration(max(count, 1))) + " ms (max " + ms(gapMax) + ")",
		"layout " + ms(laySum/time.Duration(max(count, 1))) + " ms (max " + ms(layMax) + ")",
	}
	th := a.Theme
	defer op.Offset(image.Pt(gtx.Dp(8), size.Y/2)).Push(gtx.Ops).Pop()
	rec := op.Record(gtx.Ops)
	d := layout.UniformInset(8).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		var children []layout.FlexChild
		for _, l := range lines {
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Sans}, Size: 11, Color: th.BtnText}.Layout(gtx, th, l)
			}))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
	call := rec.Stop()
	paint.FillShape(gtx.Ops, withAlpha(th.Text, 0.85), clip.UniformRRect(image.Rectangle{Max: d.Size}, gtx.Dp(8)).Op(gtx.Ops))
	call.Add(gtx.Ops)
}
