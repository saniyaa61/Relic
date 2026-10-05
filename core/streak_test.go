package core

import (
	"testing"
	"time"
)

func TestStreak(t *testing.T) {
	now := at(ist, 2026, 10, 5, 10, 0)
	day := func(d int) time.Time { return at(ist, 2026, 10, d, 9, 0) }
	tests := []struct {
		name          string
		times         []time.Time
		current, long int
	}{
		{"nothing", nil, 0, 0},
		{"today only", []time.Time{day(5)}, 1, 1},
		{"yesterday keeps it alive", []time.Time{day(4), day(3)}, 2, 2},
		{"two days ago breaks it", []time.Time{day(3), day(2)}, 0, 2},
		{"gap ends current run", []time.Time{day(5), day(4), day(2), day(1), at(ist, 2026, 9, 30, 9, 0)}, 2, 3},
		{"many logs one day", []time.Time{day(5), day(5).Add(time.Hour)}, 1, 1},
		{"longest earlier", []time.Time{day(5), at(ist, 2026, 9, 1, 9, 0), at(ist, 2026, 9, 2, 9, 0), at(ist, 2026, 9, 3, 9, 0)}, 1, 3},
		{"across month end", []time.Time{day(1), at(ist, 2026, 9, 30, 9, 0), day(5), day(4), day(3), day(2)}, 6, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var entries []*Entry
			for _, ti := range tt.times {
				entries = append(entries, &Entry{CreatedAt: ti})
			}
			c, l := Streak(entries, now, ist)
			if c != tt.current || l != tt.long {
				t.Errorf("Streak = %d, %d; want %d, %d", c, l, tt.current, tt.long)
			}
		})
	}
}

func TestStreakCountsSessionsAndRewatches(t *testing.T) {
	now := at(ist, 2026, 10, 5, 10, 0)
	e := &Entry{CreatedAt: at(ist, 2026, 10, 1, 9, 0),
		Sessions:  []Session{{At: at(ist, 2026, 10, 2, 9, 0)}, {At: at(ist, 2026, 10, 3, 9, 0)}},
		Rewatches: []Rewatch{{At: at(ist, 2026, 10, 4, 12, 0)}}}
	if c, l := Streak([]*Entry{e}, now, ist); c != 4 || l != 4 {
		t.Errorf("Streak = %d, %d; want 4, 4", c, l)
	}
}

func TestLateNightLogLandsOnLocalDate(t *testing.T) {
	// 23:50 on 4 Oct in New York is 04:50 on 5 Oct in UTC. The prototype
	// bucketed by UTC and put this on the 5th.
	late := at(nyc, 2026, 10, 4, 23, 50)
	if got := DateOf(late, nyc); got != (Date{2026, 10, 4}) {
		t.Errorf("DateOf = %v, want 2026-10-04", got)
	}
	if got := DateOf(late, time.UTC); got != (Date{2026, 10, 5}) {
		t.Errorf("UTC date = %v (sanity)", got)
	}
	// 00:20 on 5 Oct in IST is still 4 Oct in UTC.
	early := at(ist, 2026, 10, 5, 0, 20)
	if got := DateOf(early, ist); got != (Date{2026, 10, 5}) {
		t.Errorf("DateOf = %v, want 2026-10-05", got)
	}
	// Streak: logs late on the 3rd and 4th (local) are two consecutive days,
	// and "today" is the 5th, so the streak is alive.
	entries := []*Entry{{CreatedAt: at(nyc, 2026, 10, 3, 23, 50)}, {CreatedAt: late}}
	if c, _ := Streak(entries, at(nyc, 2026, 10, 5, 8, 0), nyc); c != 2 {
		t.Errorf("streak = %d, want 2", c)
	}
	// Monthly bucketing: 23:30 local on 31 Oct counts in October.
	if got := MonthOf(at(nyc, 2026, 10, 31, 23, 30), nyc); got != (Month{2026, 10}) {
		t.Errorf("MonthOf = %v", got)
	}
}

func TestDateHelpers(t *testing.T) {
	d := Date{2026, 3, 1}
	if got := d.AddDays(-1); got != (Date{2026, 2, 28}) {
		t.Errorf("AddDays = %v", got)
	}
	if got := (Date{2026, 1, 1}).DaysSince(Date{2025, 12, 25}); got != 7 {
		t.Errorf("DaysSince = %d", got)
	}
	p, err := ParseDate("2024-02-29")
	if err != nil || p != (Date{2024, 2, 29}) || p.String() != "2024-02-29" {
		t.Errorf("ParseDate = %v, %v", p, err)
	}
	if z, err := ParseDate(""); err != nil || !z.IsZero() {
		t.Errorf("empty ParseDate = %v, %v", z, err)
	}
	if _, err := ParseDate("bad"); err == nil {
		t.Error("expected error")
	}
	if got := (Month{2026, 12}).Add(1); got != (Month{2027, 1}) {
		t.Errorf("Month.Add = %v", got)
	}
	if got := (Month{2026, 1}).Add(-13); got != (Month{2024, 12}) {
		t.Errorf("Month.Add = %v", got)
	}
	if s := (Month{2026, 10}).String(); s != "October 2026" {
		t.Errorf("Month.String = %q", s)
	}
	if got := d.Noon(nyc).UTC(); got.Hour() != 17 {
		t.Errorf("Noon = %v", got)
	}
}
