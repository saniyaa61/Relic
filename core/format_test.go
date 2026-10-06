package core

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		mins float64
		want string
	}{
		{0, ""},
		{-5, ""},
		{1, "1 min"},
		{45, "45 min"},
		{44.6, "45 min"},
		{60, "1 hr"},
		{90, "1.5 hrs"},
		{324, "5.4 hrs"},
		{375, "6.3 hrs"},   // exactly 6.25: rounds up, like the prototype
		{3699, "61.6 hrs"}, // 61.65 is really 61.6499…, so it rounds down (prototype too)
		{3 * 60, "3 hrs"},
		{120, "2 hrs"},
		{72 * 60, "72 hrs"},
		{72*60 + 1, "3 days"},
		{4464, "3.1 days"},
		{24 * 60 * 10, "10 days"},
	}
	for _, tt := range tests {
		if got := FormatDuration(tt.mins); got != tt.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", tt.mins, got, tt.want)
		}
	}
	if got := FormatDurationOrZero(0); got != "0 min" {
		t.Errorf("FormatDurationOrZero(0) = %q", got)
	}
}

func TestSplitDuration(t *testing.T) {
	if n, u := SplitDuration("5.4 hrs"); n != "5.4" || u != "hrs" {
		t.Errorf("got %q %q", n, u)
	}
}

func TestPerspective(t *testing.T) {
	tests := []struct {
		mins float64
		want string
	}{
		{0, ""},
		{59, ""},
		{60, "like 1 film back to back"}, // round(0.5) = 1
		{240, "like 2 films back to back"},
		{4032, "about 2.8 days nonstop, like 34 films back to back"},
		{24 * 60, "about 1.0 days nonstop, like 12 films back to back"},
		{80 * 60, "like 40 films back to back"},
	}
	for _, tt := range tests {
		if got := Perspective(tt.mins); got != tt.want {
			t.Errorf("Perspective(%v) = %q, want %q", tt.mins, got, tt.want)
		}
	}
}

func TestDatesAndGreeting(t *testing.T) {
	when := time.Date(2026, 10, 4, 20, 0, 0, 0, time.UTC) // 01:30 on 5 Oct in IST
	if got := FormatDate(when, ist); got != "5 October 2026" {
		t.Errorf("FormatDate = %q", got)
	}
	if got := FormatToday(when, ist); got != "Monday 5 October" {
		t.Errorf("FormatToday = %q", got)
	}
	tests := []struct {
		hour int
		want string
	}{{0, "Good morning"}, {11, "Good morning"}, {12, "Good afternoon"}, {16, "Good afternoon"}, {17, "Good evening"}, {23, "Good evening"}}
	for _, tt := range tests {
		if got := Greeting(at(ist, 2026, 10, 5, tt.hour, 0), ist); got != tt.want {
			t.Errorf("Greeting(%d:00) = %q, want %q", tt.hour, got, tt.want)
		}
	}
}

func TestExcerpt(t *testing.T) {
	if got := Excerpt("short", 160); got != "short" {
		t.Errorf("got %q", got)
	}
	if got := Excerpt("héllo world", 5); got != "héllo…" {
		t.Errorf("got %q", got)
	}
}
