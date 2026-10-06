package ui

import (
	"math"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
)

// progress returns how far an animation of length d that started at
// start has run at now, from 0 to 1. While it is still running it asks
// Gio for another frame.
func progress(gtx layout.Context, now, start time.Time, d time.Duration) float32 {
	t := float32(now.Sub(start)) / float32(d)
	if t >= 1 {
		return 1
	}
	gtx.Execute(op.InvalidateCmd{})
	return max(t, 0)
}

// cubicBezier is CSS cubic-bezier(x1,y1,x2,y2) as an easing function.
func cubicBezier(x1, y1, x2, y2 float64) func(float32) float32 {
	bez := func(a, b, t float64) float64 {
		return 3*a*t*(1-t)*(1-t) + 3*b*t*t*(1-t) + t*t*t
	}
	return func(x float32) float32 {
		if x <= 0 || x >= 1 {
			return x
		}
		// Find t for x by bisection; x(t) is monotonic for 0≤x1,x2≤1.
		lo, hi := 0.0, 1.0
		for range 30 {
			mid := (lo + hi) / 2
			if bez(x1, x2, mid) < float64(x) {
				lo = mid
			} else {
				hi = mid
			}
		}
		return float32(bez(y1, y2, (lo+hi)/2))
	}
}

// The prototype's CSS timing functions.
var (
	easeCSS    = cubicBezier(0.25, 0.1, 0.25, 1) // CSS "ease"
	easeSpring = cubicBezier(0.34, 1.1, 0.64, 1) // dialogs and sheets: a slight overshoot
)

func lerpf(a, b, t float32) float32 { return a + (b-a)*t }

func roundi(f float32) int { return int(math.Round(float64(f))) }
