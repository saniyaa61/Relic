package core

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// FormatDuration writes minutes the way the app shows time (SPEC §4.6):
// "45 min", "5.4 hrs", "1 hr", "3.1 days". Zero or less gives "".
func FormatDuration(mins float64) string {
	if mins <= 0 {
		return ""
	}
	if mins < 60 {
		return fmt.Sprintf("%d min", int(math.Round(mins)))
	}
	if hrs := mins / 60; hrs <= 72 {
		return oneDecimal(hrs, "hr", "hrs")
	}
	return oneDecimal(mins/60/24, "day", "days")
}

// FormatDurationOrZero is FormatDuration, but "0 min" where a value is required.
func FormatDurationOrZero(mins float64) string {
	if s := FormatDuration(mins); s != "" {
		return s
	}
	return "0 min"
}

func oneDecimal(v float64, one, many string) string {
	// Halves round up, as the prototype's toFixed does (6.25 → "6.3"),
	// not to even.
	s := strconv.FormatFloat(math.Floor(v*10+0.5)/10, 'f', 1, 64)
	s = strings.TrimSuffix(s, ".0")
	if s == "1" {
		return s + " " + one
	}
	return s + " " + many
}

// SplitDuration splits "5.4 hrs" into "5.4" and "hrs", for the large number
// with a smaller italic unit.
func SplitDuration(s string) (num, unit string) {
	num, unit, _ = strings.Cut(s, " ")
	return num, unit
}

// Perspective is the "In perspective" line (SPEC §4.6), e.g. "about 2.8 days
// nonstop, like 33 films back to back". "" when nothing applies.
func Perspective(mins float64) string {
	var parts []string
	if mins >= 24*60 && mins <= 72*60 {
		parts = append(parts, fmt.Sprintf("about %.1f days nonstop", mins/(24*60)))
	}
	if films := int(math.Round(mins / 120)); films >= 1 {
		w := "films"
		if films == 1 {
			w = "film"
		}
		parts = append(parts, fmt.Sprintf("like %d %s back to back", films, w))
	}
	return strings.Join(parts, ", ")
}

// FormatDate is "16 June 2026", the local day of t.
func FormatDate(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("2 January 2006")
}

// FormatToday is the Home date line, "Monday, 5 October".
func FormatToday(now time.Time, loc *time.Location) string {
	return now.In(loc).Format("Monday 2 January") // as the prototype (en-GB): "Tuesday 6 October"
}

// Greeting is "Good morning" before noon, "Good afternoon" before 5pm,
// otherwise "Good evening", by local time.
func Greeting(now time.Time, loc *time.Location) string {
	switch h := now.In(loc).Hour(); {
	case h < 12:
		return "Good morning"
	case h < 17:
		return "Good afternoon"
	}
	return "Good evening"
}

// Excerpt shortens s to at most n characters, adding "…" when cut.
func Excerpt(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "…"
}
