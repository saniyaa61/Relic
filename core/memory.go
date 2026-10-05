package core

import "time"

// MemoryExcerptLen is how much of the review the memory card shows.
const MemoryExcerptLen = 160

// OneYearAgo returns the first finished entry (newest first) created within
// three days either side of this day last year, for Home's "One year ago
// today" card (SPEC §4.15). Days are local.
func (l *Library) OneYearAgo(now time.Time, loc *time.Location) *Entry {
	target := DateOf(now, loc)
	target.Year--
	// Normalise 29 February to 1 March in a non-leap year, as the calendar does.
	target = DateOf(time.Date(target.Year, target.Month, target.Day, 12, 0, 0, 0, time.UTC), time.UTC)
	for _, e := range l.Entries {
		if e.Status != Finished {
			continue
		}
		if d := DateOf(e.CreatedAt, loc).DaysSince(target); d >= -3 && d <= 3 {
			return e
		}
	}
	return nil
}
