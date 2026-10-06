package core

import (
	"fmt"
	"strings"
	"time"
)

// Store UTC, bucket locally (SPEC §4.11): every "which day / month / year"
// question takes the user's *time.Location and converts the stored instant
// into it before asking.

// Date is a calendar day with no time or zone. The zero Date means "not set".
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// DateOf is the local calendar day of t in loc.
func DateOf(t time.Time, loc *time.Location) Date {
	y, m, d := t.In(loc).Date()
	return Date{y, m, d}
}

// ParseDate reads "YYYY-MM-DD". An empty string gives the zero Date.
func ParseDate(s string) (Date, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Date{}, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return Date{}, err
	}
	return DateOf(t, time.UTC), nil
}

// IsZero reports whether d is unset.
func (d Date) IsZero() bool { return d == Date{} }

// String formats d as "YYYY-MM-DD".
func (d Date) String() string { return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day) }

// MarshalText writes "YYYY-MM-DD", or "" for the zero Date.
func (d Date) MarshalText() ([]byte, error) {
	if d.IsZero() {
		return []byte{}, nil
	}
	return []byte(d.String()), nil
}

// UnmarshalText reads what MarshalText writes.
func (d *Date) UnmarshalText(b []byte) error {
	p, err := ParseDate(string(b))
	if err != nil {
		return err
	}
	*d = p
	return nil
}

func (d Date) utc() time.Time { return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC) }

// AddDays returns d moved by n days (n may be negative).
func (d Date) AddDays(n int) Date { return DateOf(d.utc().AddDate(0, 0, n), time.UTC) }

// DaysSince returns how many days d is after e (negative if before).
func (d Date) DaysSince(e Date) int { return int(d.utc().Sub(e.utc()).Hours() / 24) }

// Before reports whether d is earlier than e.
func (d Date) Before(e Date) bool { return d.utc().Before(e.utc()) }

// Start is the instant local midnight begins d in loc.
func (d Date) Start(loc *time.Location) time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, loc)
}

// Noon is local midday on d in loc: the instant used for logs given only a
// date (rewatches), so they land on that day in any nearby zone.
func (d Date) Noon(loc *time.Location) time.Time {
	return time.Date(d.Year, d.Month, d.Day, 12, 0, 0, 0, loc)
}

// Month is a calendar month.
type Month struct {
	Year  int
	Month time.Month
}

// MonthOf is the local month of t in loc.
func MonthOf(t time.Time, loc *time.Location) Month {
	y, m, _ := t.In(loc).Date()
	return Month{y, m}
}

// Add returns m moved by n months.
func (m Month) Add(n int) Month {
	t := time.Date(m.Year, m.Month+time.Month(n), 1, 0, 0, 0, 0, time.UTC)
	return Month{t.Year(), t.Month()}
}

// Before reports whether m is earlier than o.
func (m Month) Before(o Month) bool {
	return m.Year < o.Year || (m.Year == o.Year && m.Month < o.Month)
}

// Start is the instant m begins in loc; End is the instant the next month begins.
func (m Month) Start(loc *time.Location) time.Time {
	return time.Date(m.Year, m.Month, 1, 0, 0, 0, 0, loc)
}

func (m Month) End(loc *time.Location) time.Time { return m.Add(1).Start(loc) }

// String is "October 2026".
func (m Month) String() string { return fmt.Sprintf("%s %d", m.Month, m.Year) }

// Initial is the month's first letter, for chart axes.
func (m Month) Initial() string { return m.Month.String()[:1] }

// inRange reports whether start <= t < end.
func inRange(t, start, end time.Time) bool { return !t.Before(start) && t.Before(end) }
