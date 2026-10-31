package main

import (
	"image"
	"math"
	"testing"

	"gioui.org/unit"
)

func TestFitWidth(t *testing.T) {
	for _, tc := range []struct {
		goos      string
		pxPerDp   float32
		widthPx   int
		wantScale float32
	}{
		{"android", 2.8125, 1080, 384 / 390.0}, // the owner's Android: 384dp, zooms out a little
		{"ios", 3, 1290, 430 / 390.0},          // a 430pt iPhone zooms in
		{"ios", 3, 1170, 1},                    // a 390pt iPhone is the design width
		{"android", 1, 1800, 1.2},              // tablets stop at 1.2
		{"android", 3, 900, 0.85},              // tiny screens stop at 0.85
		{"windows", 1, 400, 1},                 // desktops keep their size
	} {
		m, k := fitWidth(tc.goos, unit.Metric{PxPerDp: tc.pxPerDp, PxPerSp: tc.pxPerDp}, tc.widthPx)
		if math.Abs(float64(k-tc.wantScale)) > 1e-4 || math.Abs(float64(m.PxPerDp-tc.pxPerDp*k)) > 1e-4 || m.PxPerSp != m.PxPerDp {
			t.Errorf("%s %v×%d: scale %v, metric %+v; want scale %v", tc.goos, tc.pxPerDp, tc.widthPx, k, m, tc.wantScale)
		}
	}
}

func TestKeyboardWatch(t *testing.T) {
	var k keyboardWatch
	const dp = 3
	steps := []struct {
		w, h, inset int
		want        bool
	}{
		{1170, 2532, 34 * dp, false},  // iPhone, home indicator
		{1170, 2532 - 336*dp, 0, true}, // iOS shrinks the window for the keyboard
		{1170, 2532, 34 * dp, false},   // keyboard gone
		{1170, 2532, 300 * dp, true},   // Android-style: keyboard as bottom inset
		{2532, 1170, 21 * dp, false},   // rotated: starts again
		{2532, 1170 - 40*dp, 0, false}, // a small change isn't the keyboard
	}
	for i, s := range steps {
		if got := k.open(image.Pt(s.w, s.h), s.inset, dp); got != s.want {
			t.Errorf("step %d: open = %v, want %v", i, got, s.want)
		}
	}
}
