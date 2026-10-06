package core

import (
	"math"
	"testing"
	"time"
)

func total(e *Entry) float64 {
	var t float64
	for _, v := range e.TimeEvents() {
		t += v.Minutes
	}
	return t
}

func TestTimeEvents(t *testing.T) {
	yes, no := true, false
	day := Date{2026, 7, 1}
	tests := []struct {
		name  string
		typ   EntryType
		in    EntryInput
		build func(e *Entry)
		want  float64
	}{
		{"film counts its duration once", Film, EntryInput{Fields: Fields{Duration: 120}}, nil, 120},
		{"film rewatch adds duration", Film, EntryInput{Fields: Fields{Duration: 120}}, func(e *Entry) {
			e.LogRewatch(RewatchInput{Date: day}, time.UTC)
		}, 240},
		{"short", Short, EntryInput{Fields: Fields{Duration: 12}}, nil, 12},
		{"series sessions × episode length", Series, EntryInput{StartEpisodes: 3, Fields: Fields{EpisodeDuration: 60, TotalEpisodes: 16}},
			func(e *Entry) { e.LogSession(SessionInput{Episodes: ip(2)}, t0) }, 300},
		// Full rewatch doubles series time; partial adds nothing.
		{"full rewatch doubles series time", Series, EntryInput{Status: Finished, StartEpisodes: 10, Fields: Fields{EpisodeDuration: 45, TotalEpisodes: 10}},
			func(e *Entry) { e.LogRewatch(RewatchInput{Date: day, Full: &yes}, time.UTC) }, 900},
		{"partial rewatch counts episodes rewatched", Series, EntryInput{Status: Finished, StartEpisodes: 10, Fields: Fields{EpisodeDuration: 45, TotalEpisodes: 10}},
			func(e *Entry) { e.LogRewatch(RewatchInput{Date: day, Full: &no, Episodes: 4}, time.UTC) }, 450 + 180},
		{"partial rewatch with no episodes adds nothing", Series, EntryInput{Status: Finished, StartEpisodes: 10, Fields: Fields{EpisodeDuration: 45, TotalEpisodes: 10}},
			func(e *Entry) { e.LogRewatch(RewatchInput{Date: day, Full: &no}, time.UTC) }, 450},
		{"full rewatch ignores episodes field", Podcast, EntryInput{Status: Finished, StartEpisodes: 2, Fields: Fields{EpisodeDuration: 30, TotalEpisodes: 2}},
			func(e *Entry) { e.LogRewatch(RewatchInput{Date: day, Episodes: 9}, time.UTC) }, 120},
		{"podcast", Podcast, EntryInput{StartEpisodes: 2, Fields: Fields{EpisodeDuration: 30}}, nil, 60},
		// Book with pages but no minutes → estimated time; logged minutes override.
		{"book pages estimate 1.5 min/page", Book, EntryInput{StartPages: 0},
			func(e *Entry) { e.LogSession(SessionInput{FromPage: ip(0), ToPage: ip(40)}, t0) }, 60},
		{"book logged minutes override", Book, EntryInput{},
			func(e *Entry) { e.LogSession(SessionInput{FromPage: ip(0), ToPage: ip(40), Minutes: ip(35)}, t0) }, 35},
		{"book logged 0 minutes counts 0", Book, EntryInput{},
			func(e *Entry) { e.LogSession(SessionInput{FromPage: ip(0), ToPage: ip(40), Minutes: ip(0)}, t0) }, 0},
		{"book initial pages count", Book, EntryInput{StartPages: 80}, nil, 120},
		{"book backwards pages count nothing", Book, EntryInput{},
			func(e *Entry) { e.LogSession(SessionInput{FromPage: ip(50), ToPage: ip(10)}, t0) }, 0},
		{"other counts session minutes only", Other, EntryInput{Fields: Fields{DurationText: "2 hours"}},
			func(e *Entry) { e.LogSession(SessionInput{Minutes: ip(25)}, t0) }, 25},
		{"music counts its length", Music, EntryInput{Fields: Fields{Tracks: 11, Duration: 44}}, nil, 44},
		{"music without length estimates 3.5 min/track", Music, EntryInput{Fields: Fields{Tracks: 11}}, nil, 38.5},
		{"music relisten adds length again", Music, EntryInput{Fields: Fields{Duration: 44}},
			func(e *Entry) { e.LogRewatch(RewatchInput{Date: day}, time.UTC) }, 88},
		{"music with nothing counts nothing", Music, EntryInput{}, nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.in.Title = "x"
			if tt.in.Status == "" {
				tt.in.Status = Ongoing
			}
			e, err := NewEntry(&Category{ID: "c", Type: tt.typ}, tt.in, t0)
			if err != nil {
				t.Fatal(err)
			}
			if tt.build != nil {
				tt.build(e)
			}
			if got := total(e); math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("total = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTotalsAgree(t *testing.T) {
	// Category totals = sum of folder totals = all-time total.
	l := newLib(t)
	films := mustCat(t, l, "Movies", Film)
	books := mustCat(t, l, "Books", Book)
	music := mustCat(t, l, "Listening", Music)
	l.AddFolder(films.ID, "Bollywood")
	l.AddFolder(films.ID, "Classics")
	mustEntry(t, l, films, EntryInput{Title: "a", Folder: "Bollywood", Fields: Fields{Duration: 150}}, t0)
	mustEntry(t, l, films, EntryInput{Title: "b", Folder: "Classics", Fields: Fields{Duration: 120}}, t0)
	mustEntry(t, l, films, EntryInput{Title: "c", Fields: Fields{Duration: 90}}, t0)
	stale := mustEntry(t, l, films, EntryInput{Title: "d", Folder: "Gone", Fields: Fields{Duration: 10}}, t0)
	mustEntry(t, l, books, EntryInput{Title: "e", StartPages: 100}, t0)
	mustEntry(t, l, music, EntryInput{Title: "f", Fields: Fields{Duration: 40}}, t0)
	mustEntry(t, l, music, EntryInput{Title: "g"}, t0) // no time at all
	_ = stale

	x := NewTimeIndex(l.Entries)
	all := x.Sum(l.Entries)
	if all != 150+120+90+10+150+40 {
		t.Fatalf("all-time = %v", all)
	}
	cats := l.ConsumedByCategory(x)
	if len(cats) != 3 {
		t.Fatalf("got %d categories with time, want 3", len(cats))
	}
	var catSum float64
	for _, c := range cats {
		var fSum float64
		for _, f := range c.Folders {
			fSum += f.Minutes
		}
		if fSum != c.Minutes {
			t.Errorf("%s: folders sum %v ≠ category %v", c.Category.Name, fSum, c.Minutes)
		}
		catSum += c.Minutes
	}
	if catSum != all {
		t.Errorf("categories sum %v ≠ all-time %v", catSum, all)
	}
	if cats[0].Category.Name != "Movies" || cats[0].Folders[0].Name != "Bollywood" || cats[0].Folders[1].Name != "Classics" {
		t.Errorf("order: %+v", cats[0])
	}
	if f := cats[0].Folders[2]; f.Name != "" || f.Minutes != 100 {
		t.Errorf("Uncategorised = %+v, want 90 + stale 10", f)
	}
}

// A folder named in a different case counts under the category's own
// spelling; the "See all" page lists every entry of the group.
func TestConsumedFolder(t *testing.T) {
	l := newLib(t)
	films := mustCat(t, l, "Movies", Film)
	l.AddFolder(films.ID, "Classics")
	a := mustEntry(t, l, films, EntryInput{Title: "a", Folder: "Classics", Fields: Fields{Duration: 100}}, t0)
	b := mustEntry(t, l, films, EntryInput{Title: "b", Folder: "classics", Fields: Fields{Duration: 20}}, t0)
	c := mustEntry(t, l, films, EntryInput{Title: "c", Folder: "Classics"}, t0) // no time
	d := mustEntry(t, l, films, EntryInput{Title: "d", Folder: "Gone", Fields: Fields{Duration: 5}}, t0)

	cats := l.ConsumedByCategory(NewTimeIndex(l.Entries))
	if f := cats[0].Folders; len(f) != 2 || f[0].Name != "Classics" || f[0].Minutes != 120 {
		t.Errorf("folders = %+v", f)
	}
	tests := []struct {
		folder string
		want   []*Entry
	}{
		{"Classics", []*Entry{c, b, a}}, // library order: newest first
		{"", []*Entry{d}},
	}
	for _, tt := range tests {
		got, ok := l.ConsumedFolder(films.ID, tt.folder)
		if !ok || titles(got) != titles(tt.want) {
			t.Errorf("%q: %s, want %s", tt.folder, titles(got), titles(tt.want))
		}
	}
	if _, ok := l.ConsumedFolder("nope", ""); ok {
		t.Error("a missing category should report !ok")
	}
}

func TestShareBar(t *testing.T) {
	mk := func(name string, m float64) CategoryTime {
		return CategoryTime{Category: &Category{Name: name}, Minutes: m}
	}
	got := ShareBar([]CategoryTime{mk("A", 50), mk("B", 30), mk("C", 10), mk("D", 6), mk("E", 4)})
	if len(got) != 4 || got[3].Name != "Other" || !got[3].Other || got[3].Minutes != 10 {
		t.Errorf("ShareBar = %+v", got)
	}
	if got := ShareBar([]CategoryTime{mk("A", 1)}); len(got) != 1 {
		t.Errorf("one category: %+v", got)
	}
}

func TestTimeIndexRanges(t *testing.T) {
	l := newLib(t)
	films := mustCat(t, l, "Movies", Film)
	// Logged 23:30 local on 31 Jan in New York = 04:30 UTC on 1 Feb.
	mustEntry(t, l, films, EntryInput{Title: "late", Fields: Fields{Duration: 100}}, at(nyc, 2026, 1, 31, 23, 30))
	mustEntry(t, l, films, EntryInput{Title: "march", Fields: Fields{Duration: 60}}, at(nyc, 2026, 3, 10, 12, 0))
	x := NewTimeIndex(l.Entries)

	m := x.Monthly(l.Entries, 2026, nyc)
	if m[0] != 100 || m[1] != 0 || m[2] != 60 {
		t.Errorf("monthly = %v, want late log in January", m)
	}
	jan := MonthOfDigest(Month{2026, 1}, nyc)
	if got := x.SumInRange(l.Entries, jan.Start, jan.End); got != 100 {
		t.Errorf("January = %v", got)
	}
	now := at(nyc, 2026, 3, 12, 0, 0)
	w := x.Weekly(l.Entries, now, 8)
	if len(w) != 8 || w[7] != 60 || w[1] != 0 {
		t.Errorf("weekly = %v", w)
	}
	if PeakIndex(m[:]) != 0 || PeakIndex([]float64{0, 0}) != -1 {
		t.Error("PeakIndex")
	}
}
