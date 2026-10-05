package core

import (
	"testing"
	"time"
)

// Test zones: fixed offsets so tests don't depend on the OS time zone database.
var (
	ist = time.FixedZone("IST", 5*3600+1800) // UTC+5:30
	nyc = time.FixedZone("EST", -5*3600)     // UTC−5
)

// at builds an instant from a local wall-clock time in loc.
func at(loc *time.Location, y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, loc)
}

func ip(n int) *int { return &n }

func newLib(t *testing.T) *Library {
	t.Helper()
	return &Library{}
}

func mustCat(t *testing.T, l *Library, name string, typ EntryType) *Category {
	t.Helper()
	c, err := l.AddCategory(name, typ, at(time.UTC, 2026, 1, 1, 0, 0))
	if err != nil {
		t.Fatalf("AddCategory(%q): %v", name, err)
	}
	return c
}

func mustEntry(t *testing.T, l *Library, c *Category, in EntryInput, now time.Time) *Entry {
	t.Helper()
	if in.Status == "" {
		in.Status = Ongoing
	}
	e, err := l.AddEntry(c.ID, in, now)
	if err != nil {
		t.Fatalf("AddEntry(%q): %v", in.Title, err)
	}
	return e
}
