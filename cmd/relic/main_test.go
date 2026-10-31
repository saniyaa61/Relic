package main

import (
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
