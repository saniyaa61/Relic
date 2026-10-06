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

func TestDigestNarrative(t *testing.T) {
	l := newLib(t)
	films := mustCat(t, l, "Films", Film)
	now := at(ist, 2026, 9, 29, 9, 0)
	e := &Entry{}
	week := LastWeek(now)
	month := MonthOfDigest(Month{2026, 9}, ist)
	year := YearOfDigest(2026, ist)
	tests := []struct {
		name   string
		st     DigestStats
		p      Period
		streak int
		want   string
	}{
		{"quiet week", DigestStats{}, week, 0, "A quiet week. Sometimes stillness is its own kind of nourishment."},
		{"week", DigestStats{Sessions: 2, Minutes: 702, TopCategory: films.ID}, week, 3,
			"You spent 11.7 hrs in other worlds this week. Mostly in Films. 3 days in a row now."},
		{"week, no time, streak of one", DigestStats{Sessions: 1}, week, 1, "You spent a little time in other worlds this week."},
		{"quiet month", DigestStats{}, month, 0, "A quiet month in the library. Your next story is out there, waiting."},
		{"month", DigestStats{New: []*Entry{e}, Minutes: 30, TopCategory: films.ID, TopTag: "Quiet"}, month, 0,
			"1 new entry, 30 min spent living inside them. Most of it in Films. The feeling that kept returning: Quiet."},
		{"month, time only", DigestStats{Minutes: 90}, month, 0, "0 new entries, 1.5 hrs spent living inside them."},
		{"empty year", DigestStats{}, year, 0, "2026 hasn't started yet, or nothing was logged. Either way, the page is blank and waiting."},
		{"year", DigestStats{New: []*Entry{e, e}, Finished: []*Entry{e}, Minutes: 4752, TopCategory: films.ID, TopTag: "Beautiful"}, year, 0,
			"You gave 3.3 days to other worlds in 2026. You finished 1 story. More than anywhere else, you lived in Films. And the feeling you returned to most: Beautiful."},
		{"future", DigestStats{}, MonthOfDigest(Month{2026, 10}, ist), 0, "This chapter hasn't been written yet."},
	}
	for _, tt := range tests {
		if got := l.DigestNarrative(tt.st, tt.p, now, tt.streak); got != tt.want {
			t.Errorf("%s:\n got %q\nwant %q", tt.name, got, tt.want)
		}
	}
}

func TestProgressSummary(t *testing.T) {
	tests := []struct {
		e    *Entry
		want string
	}{
		{&Entry{Type: Series, Fields: Fields{TotalEpisodes: 12}, Sessions: []Session{{Episodes: 10}}}, "10 episodes in · 83% through"},
		{&Entry{Type: Book, Sessions: []Session{{ToPage: 120}}}, "120 pages in"},
		{&Entry{Type: Book, Fields: Fields{TotalPages: 300}}, "0 pages in · 0% through"},
		{&Entry{Type: Film}, ""},
	}
	for _, tt := range tests {
		if got := tt.e.ProgressSummary(); got != tt.want {
			t.Errorf("%v: %q, want %q", tt.e.Type, got, tt.want)
		}
	}
}

func TestYearCard(t *testing.T) {
	l := newLib(t)
	films := mustCat(t, l, "Films", Film)
	books := mustCat(t, l, "Books", Book)
	mustEntry(t, l, films, EntryInput{Title: "old", Fields: Fields{Duration: 300}}, at(ist, 2025, 12, 31, 23, 0))
	mustEntry(t, l, films, EntryInput{Title: "a", Fields: Fields{Duration: 100}, Tags: []string{"Quiet"}, Rating: 4}, at(ist, 2026, 3, 1, 10, 0))
	mustEntry(t, l, books, EntryInput{Title: "b", StartPages: 200}, at(ist, 2026, 5, 1, 10, 0)) // 300 min
	x := NewTimeIndex(l.Entries)
	yc := l.YearCard(x, 2026, ist)
	if yc.Minutes != 400 || len(yc.New) != 2 || yc.TopTag != "Quiet" || len(yc.TopRated) != 1 {
		t.Fatalf("stats: %v min, %d new, tag %q, %d rated", yc.Minutes, len(yc.New), yc.TopTag, len(yc.TopRated))
	}
	if len(yc.Categories) != 2 || yc.Categories[0].Category != books || yc.Categories[1].Minutes != 100 {
		t.Errorf("categories: %+v", yc.Categories)
	}
	if yc.Monthly[2] != 100 || yc.Monthly[4] != 300 {
		t.Errorf("monthly: %v", yc.Monthly)
	}
	if !yc.YearHasData() || l.YearCard(x, 2024, ist).YearHasData() {
		t.Error("YearHasData")
	}
}
