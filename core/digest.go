package core

import (
	"fmt"
	"sort"
	"time"
)

// PeriodKind is the Digest toggle: Week, Month or Year.
type PeriodKind int

const (
	WeekPeriod PeriodKind = iota
	MonthPeriod
	YearPeriod
)

// Period is a span of time the Digest reports on: Start <= t < End.
type Period struct {
	Kind  PeriodKind
	Start time.Time
	End   time.Time
	Month Month // MonthPeriod only
	Year  int   // YearPeriod only
}

// LastWeek is the rolling last 7 days ending now (no navigation).
func LastWeek(now time.Time) Period {
	return Period{Kind: WeekPeriod, Start: now.Add(-7 * 24 * time.Hour), End: now}
}

// MonthOfDigest is the calendar month m in the user's zone.
func MonthOfDigest(m Month, loc *time.Location) Period {
	return Period{Kind: MonthPeriod, Start: m.Start(loc), End: m.End(loc), Month: m}
}

// YearOfDigest is the calendar year y in the user's zone.
func YearOfDigest(y int, loc *time.Location) Period {
	return Period{Kind: YearPeriod, Start: time.Date(y, 1, 1, 0, 0, 0, 0, loc),
		End: time.Date(y+1, 1, 1, 0, 0, 0, 0, loc), Year: y}
}

// Contains reports whether t falls in the period.
func (p Period) Contains(t time.Time) bool { return inRange(t, p.Start, p.End) }

// DigestBounds are how far the month/year arrows may go (SPEC §4.12): from
// the month of first activity to the current month.
type DigestBounds struct {
	Floor   Month
	Ceiling Month
}

// DigestBounds works out the navigation floor and ceiling.
func (l *Library) DigestBounds(now time.Time, loc *time.Location) DigestBounds {
	ceil := MonthOf(now, loc)
	first := l.FirstActivity()
	if first.IsZero() {
		return DigestBounds{ceil, ceil}
	}
	floor := MonthOf(first, loc)
	if ceil.Before(floor) {
		floor = ceil
	}
	return DigestBounds{floor, ceil}
}

// CanStepMonth reports whether m moved by delta stays within bounds; the
// arrow is disabled when it doesn't.
func (b DigestBounds) CanStepMonth(m Month, delta int) bool {
	c := m.Add(delta)
	return !c.Before(b.Floor) && !b.Ceiling.Before(c)
}

// StepMonth moves m by delta, or leaves it unchanged if that would leave bounds.
func (b DigestBounds) StepMonth(m Month, delta int) Month {
	if b.CanStepMonth(m, delta) {
		return m.Add(delta)
	}
	return m
}

// CanStepYear and StepYear are the same for years.
func (b DigestBounds) CanStepYear(y, delta int) bool {
	c := y + delta
	return c >= b.Floor.Year && c <= b.Ceiling.Year
}

func (b DigestBounds) StepYear(y, delta int) int {
	if b.CanStepYear(y, delta) {
		return y + delta
	}
	return y
}

// DigestStats is what happened in a period (SPEC §4.12).
type DigestStats struct {
	New         []*Entry // createdAt in period
	Finished    []*Entry // finishedAt in period
	Minutes     float64  // time events in period
	Pages       int      // sum of toPage − fromPage over book sessions in period
	Sessions    int      // sessions + rewatches in period
	TopCategory string   // category id with the most new entries + sessions; "" if none
	TopTag      string   // most frequent tag among new entries; "" if none
	TopRated    []*Entry // rated entries active in period, best first (at most 3)
}

// Digest gathers the stats for one period.
func (l *Library) Digest(x *TimeIndex, p Period) DigestStats {
	var st DigestStats
	catCount := map[string]int{}
	tagCount := map[string]int{}
	var tagOrder []string
	var active []*Entry
	for _, e := range l.Entries {
		isActive := false
		if p.Contains(e.CreatedAt) {
			st.New = append(st.New, e)
			catCount[e.CategoryID]++
			isActive = true
			for _, t := range e.Tags {
				if tagCount[t] == 0 {
					tagOrder = append(tagOrder, t)
				}
				tagCount[t]++
			}
		}
		if !e.FinishedAt.IsZero() && p.Contains(e.FinishedAt) {
			st.Finished = append(st.Finished, e)
		}
		st.Minutes += x.InRange(e.ID, p.Start, p.End)
		for _, s := range e.Sessions {
			if p.Contains(s.At) {
				st.Sessions++
				catCount[e.CategoryID]++
				isActive = true
				if e.Type == Book && s.ToPage > s.FromPage {
					st.Pages += s.ToPage - s.FromPage
				}
			}
		}
		for _, r := range e.Rewatches {
			if p.Contains(r.At) {
				st.Sessions++
				catCount[e.CategoryID]++
				isActive = true
			}
		}
		if isActive && e.Rating > 0 {
			active = append(active, e)
		}
	}
	best := 0
	for _, c := range l.Categories {
		if n := catCount[c.ID]; n > best {
			best, st.TopCategory = n, c.ID
		}
	}
	st.TopTag = topCount(tagOrder, tagCount)
	sort.SliceStable(active, func(i, j int) bool { return active[i].Rating > active[j].Rating })
	if len(active) > 3 {
		active = active[:3]
	}
	st.TopRated = active
	return st
}

// topCount returns the key with the highest count; ties go to the first seen.
func topCount(order []string, count map[string]int) string {
	top, n := "", 0
	for _, k := range order {
		if count[k] > n {
			top, n = k, count[k]
		}
	}
	return top
}

// StillGoing is the ongoing progress entry furthest along (Digest "Still
// going" card), or nil.
func (l *Library) StillGoing() *Entry {
	var best *Entry
	var bestRatio float64 = -1
	for _, e := range l.Entries {
		done, total, ok := e.Progress()
		if !ok || e.Status == Finished {
			continue
		}
		if total <= 0 {
			total = 1
		}
		if r := float64(done) / float64(total); r > bestRatio {
			best, bestRatio = e, r
		}
	}
	return best
}

// PeakIndex returns the index of the largest value, or -1 if all are zero.
func PeakIndex(v []float64) int {
	idx, max := -1, 0.0
	for i, x := range v {
		if x > max {
			idx, max = i, x
		}
	}
	return idx
}

// YearHasData reports whether a year has anything for the Year in Review card.
func (st DigestStats) YearHasData() bool {
	return st.Minutes > 0 || len(st.New) > 0 || len(st.Finished) > 0
}

// DigestNarrative is the sentence in the Digest hero (SPEC §5 "Digest";
// the prototype's wording). streak is the current day streak.
func (l *Library) DigestNarrative(st DigestStats, p Period, now time.Time, streak int) string {
	if p.Start.After(now) {
		return "This chapter hasn't been written yet."
	}
	spent := func(fallback string) string {
		if s := FormatDuration(st.Minutes); s != "" {
			return s
		}
		return fallback
	}
	cat := l.CategoryName(st.TopCategory)
	switch p.Kind {
	case WeekPeriod:
		if len(st.New) == 0 && st.Sessions == 0 {
			return "A quiet week. Sometimes stillness is its own kind of nourishment."
		}
		line := "You spent " + spent("a little time") + " in other worlds this week."
		if cat != "" {
			line += " Mostly in " + cat + "."
		}
		if streak > 1 {
			line += fmt.Sprintf(" %d days in a row now.", streak)
		}
		return line
	case MonthPeriod:
		if len(st.New) == 0 && st.Minutes == 0 {
			return "A quiet month in the library. Your next story is out there, waiting."
		}
		noun := "entries"
		if len(st.New) == 1 {
			noun = "entry"
		}
		line := fmt.Sprintf("%d new %s, %s spent living inside them.", len(st.New), noun, spent("a little time"))
		if cat != "" {
			line += " Most of it in " + cat + "."
		}
		if st.TopTag != "" {
			line += " The feeling that kept returning: " + st.TopTag + "."
		}
		return line
	}
	if len(st.New) == 0 && st.Minutes == 0 {
		return fmt.Sprintf("%d hasn't started yet, or nothing was logged. Either way, the page is blank and waiting.", p.Year)
	}
	line := fmt.Sprintf("You gave %s to other worlds in %d.", spent("some time"), p.Year)
	if n := len(st.Finished); n > 0 {
		noun := "stories"
		if n == 1 {
			noun = "story"
		}
		line += fmt.Sprintf(" You finished %d %s.", n, noun)
	}
	if cat != "" {
		line += " More than anywhere else, you lived in " + cat + "."
	}
	if st.TopTag != "" {
		line += " And the feeling you returned to most: " + st.TopTag + "."
	}
	return line
}

// ProgressSummary is the Digest's "Still going" line: "10 episodes in ·
// 83% through", or just "120 pages in" without a total.
func (e *Entry) ProgressSummary() string {
	done, total, ok := e.Progress()
	if !ok {
		return ""
	}
	unit := "episodes"
	if e.Type.Progress() == PageProgress {
		unit = "pages"
	}
	s := fmt.Sprintf("%d %s in", done, unit)
	if total > 0 {
		s += fmt.Sprintf(" · %d%% through", e.ProgressPercent())
	}
	return s
}
