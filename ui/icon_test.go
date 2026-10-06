package ui

import (
	"math"
	"testing"

	"gioui.org/f32"
)

// Icons must stay inside their 24×24 box (with a little room for curves).
func TestIconsInBox(t *testing.T) {
	icons := map[string]*Icon{
		"home": IconHome, "library": IconLibrary, "add": IconAdd, "favorites": IconFavorites,
		"digest": IconDigest, "back": IconBack, "search": IconSearch, "settings": IconSettings,
		"check": IconCheck, "x-circle": IconXCircle, "plus": IconPlus, "clock": IconClock, "book": IconBook,
	}
	for name, ic := range icons {
		if len(ic.segs) == 0 {
			t.Errorf("%s: no segments", name)
		}
		for _, s := range ic.segs {
			for _, p := range s.pts {
				if p.X < -0.5 || p.X > 24.5 || p.Y < -0.5 || p.Y > 24.5 {
					t.Errorf("%s: point %v outside the 24×24 box", name, p)
				}
			}
		}
	}
}

func near(a, b f32.Point) bool {
	return math.Abs(float64(a.X-b.X)) < 0.01 && math.Abs(float64(a.Y-b.Y)) < 0.01
}

func TestIconPathParsing(t *testing.T) {
	// Compact SVG forms: glued arc flags ("016.5"), implicit linetos,
	// relative commands and a "-" separating numbers.
	ic := IconSVG(`<path d="M4 19.5A2.5 2.5 0 016.5 17H20"/>`)
	last := ic.segs[len(ic.segs)-1].pts[0]
	if !near(last, f32.Pt(20, 17)) {
		t.Errorf("H after arc ended at %v, want (20,17)", last)
	}
	arcEnd := ic.segs[len(ic.segs)-2].pts[2]
	if !near(arcEnd, f32.Pt(6.5, 17)) {
		t.Errorf("arc ended at %v, want (6.5,17)", arcEnd)
	}
	// A quarter circle of radius 2.5 bulges toward (4,17)'s corner: its
	// midpoint sits about 0.73 in from the corner along the diagonal.
	c := ic.segs[1].pts
	mid := f32.Pt((4+3*c[0].X+3*c[1].X+c[2].X)/8, (19.5+3*c[0].Y+3*c[1].Y+c[2].Y)/8)
	if !near(mid, f32.Pt(6.5-2.5*float32(math.Sqrt2)/2, 19.5-2.5*float32(math.Sqrt2)/2)) {
		t.Errorf("arc midpoint %v off the circle", mid)
	}

	poly := IconSVG(`<polyline points="12 19 5 12 12 5"/>`)
	if len(poly.segs) != 3 || poly.segs[0].op != 'M' || poly.segs[2].op != 'L' || !near(poly.segs[2].pts[0], f32.Pt(12, 5)) {
		t.Errorf("polyline parsed as %+v", poly.segs)
	}

	rel := IconSVG(`<path d="M12 20h9l-2.83-2.83"/>`)
	if got := rel.segs[2].pts[0]; !near(got, f32.Pt(18.17, 17.17)) {
		t.Errorf("relative line ended at %v", got)
	}
}

func TestIconRejectsUnknownMarkup(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("want a panic for <ellipse>")
		}
	}()
	IconSVG(`<ellipse cx="1" cy="1" rx="1" ry="1"/>`)
}
