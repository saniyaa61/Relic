package core

import (
	"testing"
	"time"
)

func TestDigestBounds(t *testing.T) {
	now := at(ist, 2026, 10, 5, 12, 0)
	l := newLib(t)
	l.Profile.FirstUsedAt = at(ist, 2026, 6, 10, 0, 0)
	movies := mustCat(t, l, "Movies", Film)
	// An imported entry older than first use moves the floor back.
	mustEntry(t, l, movies, EntryInput{Title: "old"}, at(ist, 2026, 3, 1, 0, 30))

	b := l.DigestBounds(now, ist)
	if b.Floor != (Month{2026, 3}) || b.Ceiling != (Month{2026, 10}) {
		t.Fatalf("bounds = %+v", b)
	}
	tests := []struct {
		name  string
		from  Month
		delta int
		want  Month
		can   bool
	}{
		{"back from now", Month{2026, 10}, -1, Month{2026, 9}, true},
		{"can't pass today", Month{2026, 10}, 1, Month{2026, 10}, false},
		{"can't go before first use", Month{2026, 3}, -1, Month{2026, 3}, false},
		{"forward from floor", Month{2026, 3}, 1, Month{2026, 4}, true},
		{"big jump refused", Month{2026, 5}, -12, Month{2026, 5}, false},
	}
	for _, tt := range tests {
		if got := b.StepMonth(tt.from, tt.delta); got != tt.want || b.CanStepMonth(tt.from, tt.delta) != tt.can {
			t.Errorf("%s: StepMonth = %v (can %v), want %v (can %v)", tt.name, got, b.CanStepMonth(tt.from, tt.delta), tt.want, tt.can)
		}
	}
	if b.StepYear(2026, 1) != 2026 || b.StepYear(2026, -1) != 2026 || b.CanStepYear(2026, -1) {
		t.Error("year should be pinned to 2026")
	}
	// Floor uses local time: 00:30 IST on 1 March is still February in UTC.
	if b2 := l.DigestBounds(now, time.UTC); b2.Floor != (Month{2026, 2}) {
		t.Errorf("UTC floor = %v", b2.Floor)
	}
	empty := &Library{}
	if b := empty.DigestBounds(now, ist); b.Floor != b.Ceiling {
		t.Errorf("empty library bounds = %+v", b)
	}
}

func TestDigestStats(t *testing.T) {
	l := newLib(t)
	movies := mustCat(t, l, "Movies", Film)
	books := mustCat(t, l, "Books", Book)
	dramas := mustCat(t, l, "Dramas", Series)

	sep := at(ist, 2026, 9, 20, 12, 0)
	oct := at(ist, 2026, 10, 2, 12, 0)

	mustEntry(t, l, movies, EntryInput{Title: "Sept film", Status: Finished, Rating: 5, Tags: []string{"Quiet"},
		Fields: Fields{Duration: 100}}, sep)
	f := mustEntry(t, l, movies, EntryInput{Title: "Oct film", Status: Finished, Rating: 4, Tags: []string{"Haunting", "Quiet"},
		Fields: Fields{Duration: 120}}, oct)
	b := mustEntry(t, l, books, EntryInput{Title: "Book", StartPages: 20, Tags: []string{"Haunting"}}, sep)
	b.LogSession(SessionInput{FromPage: ip(20), ToPage: ip(70), Minutes: ip(40)}, oct)
	b.Rating = 4.5
	d := mustEntry(t, l, dramas, EntryInput{Title: "Drama", StartEpisodes: 1, Fields: Fields{TotalEpisodes: 2, EpisodeDuration: 60}}, sep)
	d.LogSession(SessionInput{}, oct.Add(time.Hour))
	d.SetStatus(Finished, oct.Add(2*time.Hour))
	f.LogRewatch(RewatchInput{Date: Date{2026, 10, 3}}, ist)

	x := NewTimeIndex(l.Entries)
	st := l.Digest(x, MonthOfDigest(Month{2026, 10}, ist))

	if len(st.New) != 1 || st.New[0] != f {
		t.Errorf("new = %v", st.New)
	}
	if len(st.Finished) != 2 {
		t.Errorf("finished = %d, want Oct film + Drama", len(st.Finished))
	}
	if st.Minutes != 120+40+60+120 {
		t.Errorf("minutes = %v", st.Minutes)
	}
	if st.Pages != 50 {
		t.Errorf("pages = %d", st.Pages)
	}
	if st.Sessions != 3 { // book session, drama session, rewatch
		t.Errorf("sessions = %d", st.Sessions)
	}
	if st.TopCategory != movies.ID { // new film + rewatch = 2
		t.Errorf("top category = %q", l.CategoryName(st.TopCategory))
	}
	if st.TopTag != "Haunting" {
		t.Errorf("top tag = %q", st.TopTag)
	}
	if len(st.TopRated) != 2 || st.TopRated[0] != b || st.TopRated[1] != f {
		t.Errorf("top rated = %v", st.TopRated)
	}
	if !st.YearHasData() {
		t.Error("YearHasData")
	}

	sepStats := l.Digest(x, MonthOfDigest(Month{2026, 9}, ist))
	if len(sepStats.New) != 3 || sepStats.Pages != 20 {
		t.Errorf("Sept: new %d pages %d", len(sepStats.New), sepStats.Pages)
	}
	week := l.Digest(x, LastWeek(at(ist, 2026, 10, 5, 12, 0)))
	if len(week.New) != 1 || week.Sessions != 3 {
		t.Errorf("week: new %d sessions %d", len(week.New), week.Sessions)
	}
	year := l.Digest(x, YearOfDigest(2026, ist))
	if len(year.New) != 4 || len(year.TopRated) != 3 || year.TopRated[0].Title != "Sept film" {
		t.Errorf("year: new %d top %v", len(year.New), year.TopRated)
	}
	if (DigestStats{}).YearHasData() {
		t.Error("empty YearHasData")
	}
}

func TestStillGoing(t *testing.T) {
	l := newLib(t)
	books := mustCat(t, l, "Books", Book)
	dramas := mustCat(t, l, "Dramas", Series)
	mustEntry(t, l, books, EntryInput{Title: "a", StartPages: 50, Fields: Fields{TotalPages: 100}}, t0)
	want := mustEntry(t, l, dramas, EntryInput{Title: "b", StartEpisodes: 12, Fields: Fields{TotalEpisodes: 16}}, t0)
	mustEntry(t, l, dramas, EntryInput{Title: "c", Status: Finished, StartEpisodes: 16, Fields: Fields{TotalEpisodes: 16}}, t0)
	if got := l.StillGoing(); got != want {
		t.Errorf("StillGoing = %v", got)
	}
	if (&Library{}).StillGoing() != nil {
		t.Error("empty")
	}
}
