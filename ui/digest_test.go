package ui

import (
	"testing"

	"github.com/saniyaa61/relic/core"
)

// The toggle switches period and resets to the current month / year; the
// arrows stop at the floor and the ceiling.
func TestDigestPeriods(t *testing.T) {
	a, _ := libApp(t)
	a.Go(TabDigest)
	p := a.roots[TabDigest].(*digestPage)
	frame(a)
	b := a.Lib.DigestBounds(a.Now(), a.Loc)
	if p.kind != core.MonthPeriod || p.month != b.Ceiling {
		t.Fatalf("opens on %v %v, want this month", p.kind, p.month)
	}
	p.next.Click()
	frame(a)
	if p.month != b.Ceiling {
		t.Error("› past this month")
	}
	for range 40 {
		p.prev.Click()
		frame(a)
	}
	if p.month != b.Floor {
		t.Errorf("‹ stopped at %v, want the floor %v", p.month, b.Floor)
	}
	p.toggle[core.YearPeriod].Click()
	frame(a)
	if p.kind != core.YearPeriod || p.year != b.Ceiling.Year {
		t.Errorf("Year shows %v", p.year)
	}
	p.toggle[core.MonthPeriod].Click()
	frame(a)
	if p.month != b.Ceiling {
		t.Error("switching back should show this month again")
	}
	p.toggle[core.WeekPeriod].Click()
	frame(a)

	// Month and Year link to Mood Trends.
	p.toggle[core.MonthPeriod].Click()
	frame(a)
	p.moods.Click()
	frame(a)
	if _, ok := a.top().(*moodsPage); !ok {
		t.Fatalf("Mood trends opened %T", a.top())
	}
	frame(a)
}

// Every bar style and an empty library draw without trouble.
func TestDigestEmptyAndBars(t *testing.T) {
	a, _ := testApp(t, nil)
	a.Go(TabDigest)
	frame(a)
	a.Push(&moodsPage{})
	frame(a)

	b, _ := libApp(t)
	b.Go(TabDigest)
	b.roots[TabDigest].(*digestPage).kind = core.YearPeriod
	defer func(s yearBarStyle) { yearBars = s }(yearBars)
	for _, s := range []yearBarStyle{barsGradient, barsPeak, barsFlat} {
		yearBars = s
		frame(b)
	}
}
