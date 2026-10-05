package core

import (
	"sort"
	"time"
)

// ActiveDays returns the local dates with any entry created, session or
// rewatch, newest first (SPEC §4.10).
func ActiveDays(entries []*Entry, loc *time.Location) []Date {
	set := map[Date]bool{}
	for _, e := range entries {
		set[DateOf(e.CreatedAt, loc)] = true
		for _, s := range e.Sessions {
			set[DateOf(s.At, loc)] = true
		}
		for _, r := range e.Rewatches {
			set[DateOf(r.At, loc)] = true
		}
	}
	days := make([]Date, 0, len(set))
	for d := range set {
		days = append(days, d)
	}
	sort.Slice(days, func(i, j int) bool { return days[j].Before(days[i]) })
	return days
}

// Streak returns the current and longest runs of consecutive active days.
// The current streak only counts if the latest active day is today or
// yesterday (local).
func Streak(entries []*Entry, now time.Time, loc *time.Location) (current, longest int) {
	days := ActiveDays(entries, loc)
	if len(days) == 0 {
		return 0, 0
	}
	run := 1
	longest = 1
	first := true
	for i := 1; i <= len(days); i++ {
		if i < len(days) && days[i-1].DaysSince(days[i]) == 1 {
			run++
		} else {
			if first {
				if today := DateOf(now, loc); days[0] == today || days[0] == today.AddDays(-1) {
					current = run
				}
				first = false
			}
			run = 1
		}
		if run > longest {
			longest = run
		}
	}
	return current, longest
}
