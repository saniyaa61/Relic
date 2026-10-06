package core

import (
	"reflect"
	"testing"
)

// moodLib builds entries tagged in the given months (offset from Oct 2026).
func moodLib(t *testing.T, firstUse Month, tagged map[int][]string) *Library {
	l := newLib(t)
	l.Profile.FirstUsedAt = firstUse.Start(ist)
	c := mustCat(t, l, "All", Film)
	for off, tags := range tagged {
		m := Month{2026, 10}.Add(off)
		mustEntry(t, l, c, EntryInput{Title: "x", Tags: tags}, m.Start(ist).Add(36*3600e9))
	}
	return l
}

func TestMoodWindow(t *testing.T) {
	now := at(ist, 2026, 10, 5, 12, 0)
	tests := []struct {
		name     string
		firstUse Month
		months   int
		enough   bool
	}{
		{"long-time user gets 12", Month{2024, 1}, 12, true},
		{"floor limits window", Month{2026, 7}, 4, true},
		{"one month isn't enough", Month{2026, 10}, 1, false},
		{"two months is", Month{2026, 9}, 2, true},
	}
	for _, tt := range tests {
		l := moodLib(t, tt.firstUse, nil)
		d := l.MoodTrends(now, ist)
		if len(d.Months) != tt.months || d.EnoughMonths() != tt.enough {
			t.Errorf("%s: %d months (enough %v)", tt.name, len(d.Months), d.EnoughMonths())
		}
		if d.Months[len(d.Months)-1] != (Month{2026, 10}) {
			t.Errorf("%s: window should end this month", tt.name)
		}
	}
}

func TestMoodTrends(t *testing.T) {
	now := at(ist, 2026, 10, 5, 12, 0)
	l := moodLib(t, Month{2026, 5}, map[int][]string{
		-5: {"Quiet", "Gothic"}, // May
		-4: {"Quiet"},
		-3: {"Quiet"},
		-2: {"Haunting"},
		-1: {"Haunting", "Quiet"},
		0:  {"Haunting"},
	})
	d := l.MoodTrends(now, ist)
	if len(d.Months) != 6 {
		t.Fatalf("months = %d", len(d.Months))
	}
	if d.Tags[0].Tag != "Quiet" || d.Tags[0].Total != 4 {
		t.Errorf("top = %+v", d.Tags[0])
	}
	if !reflect.DeepEqual(d.Tags[1].Counts, []int{0, 0, 0, 1, 1, 1}) {
		t.Errorf("Haunting counts = %v", d.Tags[1].Counts)
	}
	if d.MaxCount() != 1 {
		t.Errorf("MaxCount = %d", d.MaxCount())
	}
	if got := d.MonthTop(0, 3); !reflect.DeepEqual(got, []TagCount{{"Quiet", 1}, {"Gothic", 1}}) {
		t.Errorf("May top = %v", got)
	}
	// 6 months: recent = last 3, earlier = first 3.
	// Haunting: 3/3 − 0/3 = +1. Quiet: 1/3 − 3/3 = −0.67. Gothic: 0 − 1/3 = −0.33.
	want := []Insight{
		{"Your most frequent feeling: ", "Quiet", " (4 entries)."},
		{"Lately you're leaning toward ", "Haunting", "."},
		{"", "Quiet", " has faded from your recent entries."},
	}
	if got := d.Insights(); !reflect.DeepEqual(got, want) {
		t.Errorf("insights = %q", got)
	}
}

func TestMoodWindowDropsOlderThanTwelveMonths(t *testing.T) {
	now := at(ist, 2026, 10, 5, 12, 0)
	l := moodLib(t, Month{2024, 1}, map[int][]string{-12: {"Too old"}, -11: {"Kept"}})
	d := l.MoodTrends(now, ist)
	if len(d.Tags) != 1 || d.Tags[0].Tag != "Kept" || d.Months[0] != (Month{2025, 11}) {
		t.Errorf("tags = %+v, first month %v", d.Tags, d.Months[0])
	}
}

func TestMoodInsightsThresholds(t *testing.T) {
	now := at(ist, 2026, 10, 5, 12, 0)
	// Under 4 months: only the most-frequent line.
	l := moodLib(t, Month{2026, 8}, map[int][]string{-2: {"Quiet"}, 0: {"Haunting"}})
	if got := l.MoodTrends(now, ist).Insights(); len(got) != 1 || got[0].After != " (1 entry)." {
		t.Errorf("short window = %q", got)
	}
	// Small changes (< 0.3/month) say nothing.
	l = moodLib(t, Month{2025, 11}, map[int][]string{-11: {"Quiet"}, -1: {"Quiet"}})
	if got := l.MoodTrends(now, ist).Insights(); len(got) != 1 {
		t.Errorf("flat = %q", got)
	}
	if got := (MoodData{}).Insights(); got != nil {
		t.Errorf("empty = %q", got)
	}
}

func TestMoodShownCapsAtEight(t *testing.T) {
	now := at(ist, 2026, 10, 5, 12, 0)
	l := moodLib(t, Month{2026, 1}, map[int][]string{0: {"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}})
	if got := len(l.MoodTrends(now, ist).Shown()); got != MoodTrendTags {
		t.Errorf("shown = %d", got)
	}
}
