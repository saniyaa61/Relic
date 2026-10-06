package store

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/saniyaa61/relic/core"
)

func openTemp(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "relic.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// sample builds a library touching every stored field.
func sample(t *testing.T) *core.Library {
	t.Helper()
	loc := time.FixedZone("X", 3*3600)
	now := time.Date(2026, 9, 1, 10, 0, 0, 123456789, time.UTC)
	l := &core.Library{Profile: core.Profile{Name: "Haya", FirstUsedAt: now.Add(-time.Hour), Theme: "custom",
		Mode: "dark", CustomBase: "#bb356f", CustomAccent: "#6f75be"}}
	dramas, _ := l.AddCategory("Dramas", core.Series, now)
	books, _ := l.AddCategory("Books", core.Book, now)
	l.AddCategory("Empty", core.Music, now)
	l.AddFolder(dramas.ID, "K-dramas")

	d, err := l.AddEntry(dramas.ID, core.EntryInput{Title: "Signal", Folder: "K-dramas", Status: core.Ongoing,
		Rating: 4.5, Tags: []string{"Haunting", "Slow-burn"}, Review: "It stays.", StartEpisodes: 3, Poster: "p1.jpg",
		Fields: core.Fields{TotalEpisodes: 16, EpisodeDuration: 70, Platform: "Netflix", Cast: "Kim Hye-soo"}}, now)
	if err != nil {
		t.Fatal(err)
	}
	d.LogSession(core.SessionInput{Episodes: ptr(0), Note: "Nothing yet."}, now.Add(time.Hour))
	d.SetStatus(core.Finished, now.Add(2*time.Hour))
	no := false
	d.LogRewatch(core.RewatchInput{Date: core.Date{Year: 2026, Month: 9, Day: 5}, Rating: 5, Full: &no, Episodes: 4}, loc)

	b, _ := l.AddEntry(books.ID, core.EntryInput{Title: "Stoner", Status: core.Ongoing, StartPages: 0,
		Fields: core.Fields{Author: "John Williams", TotalPages: 288, StartDate: core.Date{Year: 2026, Month: 8, Day: 30}}}, now.Add(time.Minute))
	b.LogSession(core.SessionInput{FromPage: ptr(0), ToPage: ptr(40), Minutes: ptr(0)}, now.Add(3*time.Hour))
	b.LogSession(core.SessionInput{FromPage: ptr(40), ToPage: ptr(80)}, now.Add(4*time.Hour))
	b.LogRewatch(core.RewatchInput{Date: core.Date{Year: 2026, Month: 9, Day: 6},
		StartDate: core.Date{Year: 2026, Month: 9, Day: 1}, EndDate: core.Date{Year: 2026, Month: 9, Day: 6}}, loc)

	l.ToggleFavorite(d.ID)
	l.ToggleFavorite(b.ID)
	l.AddTopFive(d.ID)
	return l
}

func ptr(n int) *int { return &n }

// normalize makes nil and empty collections compare equal.
func normalize(l *core.Library) {
	for _, c := range l.Categories {
		if c.Folders == nil {
			c.Folders = []string{}
		}
	}
	for _, e := range l.Entries {
		if e.Tags == nil {
			e.Tags = []string{}
		}
	}
	for _, m := range []*map[string][]string{&l.Favourites.TopFive, &l.Favourites.Order} {
		if *m == nil {
			*m = map[string][]string{}
		}
		for k, v := range *m {
			if len(v) == 0 {
				delete(*m, k)
			}
		}
	}
}

func TestRoundTrip(t *testing.T) {
	db := openTemp(t)
	want := sample(t)
	if err := db.ReplaceAll(want); err != nil {
		t.Fatal(err)
	}
	got, err := db.Load()
	if err != nil {
		t.Fatal(err)
	}
	normalize(want)
	normalize(got)
	if !reflect.DeepEqual(got.Profile, want.Profile) {
		t.Errorf("profile:\n got %+v\nwant %+v", got.Profile, want.Profile)
	}
	if !reflect.DeepEqual(got.Categories, want.Categories) {
		t.Errorf("categories differ")
	}
	if len(got.Entries) != len(want.Entries) {
		t.Fatalf("%d entries, want %d", len(got.Entries), len(want.Entries))
	}
	for i := range want.Entries {
		if !reflect.DeepEqual(got.Entries[i], want.Entries[i]) {
			t.Errorf("entry %d:\n got %+v\nwant %+v", i, *got.Entries[i], *want.Entries[i])
		}
	}
	if !reflect.DeepEqual(got.Favourites, want.Favourites) {
		t.Errorf("favourites:\n got %+v\nwant %+v", got.Favourites, want.Favourites)
	}
	// Zero minutes and blank minutes stay different.
	s := got.Entries[0].Sessions
	if s[1].Minutes == nil || *s[1].Minutes != 0 || s[2].Minutes != nil {
		t.Errorf("minutes: %v, %v", s[1].Minutes, s[2].Minutes)
	}
}

func TestUpdateAndCascade(t *testing.T) {
	db := openTemp(t)
	lib := sample(t)
	if err := db.ReplaceAll(lib); err != nil {
		t.Fatal(err)
	}
	signal := lib.Entries[1]
	// Edit an entry and delete a session: the session must not come back.
	signal.Title = "Signal (2016)"
	if err := signal.DeleteSession(signal.Sessions[1].ID); err != nil {
		t.Fatal(err)
	}
	if err := db.Update(func(w Writer) error { return w.SaveEntry(signal) }); err != nil {
		t.Fatal(err)
	}
	got, _ := db.Load()
	if e := got.Entry(signal.ID); e.Title != "Signal (2016)" || len(e.Sessions) != 1 {
		t.Errorf("after update: %q, %d sessions", e.Title, len(e.Sessions))
	}
	if len(got.Favourites.TopFive[signal.CategoryID]) != 1 {
		t.Error("saving an entry must not drop its Top 5 place")
	}

	// A failed Update writes nothing.
	boom := db.Update(func(w Writer) error {
		w.DeleteEntry(signal.ID)
		return errBoom
	})
	if boom != errBoom {
		t.Fatalf("err = %v", boom)
	}
	if got, _ := db.Load(); got.Entry(signal.ID) == nil {
		t.Error("rolled-back delete took effect")
	}

	// Dropping a category deletes its entries, sessions and favourites.
	dramas := lib.Categories[0]
	if err := lib.DeleteCategory(dramas.ID); err != nil {
		t.Fatal(err)
	}
	err := db.Update(func(w Writer) error {
		if err := w.SaveCategories(lib.Categories); err != nil {
			return err
		}
		return w.SaveFavourites(lib.Favourites)
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ = db.Load()
	if len(got.Categories) != 2 || len(got.Entries) != 1 || got.Entry(signal.ID) != nil {
		t.Errorf("after category delete: %d categories, %d entries", len(got.Categories), len(got.Entries))
	}
	var n int
	db.sql.QueryRow(`SELECT count(*) FROM sessions WHERE entry_id = ?`, signal.ID).Scan(&n)
	if n != 0 {
		t.Errorf("%d orphan sessions", n)
	}
}

type boomErr struct{}

func (boomErr) Error() string { return "boom" }

var errBoom = boomErr{}

func TestLoadEmpty(t *testing.T) {
	got, err := openTemp(t).Load()
	if err != nil || len(got.Entries) != 0 || got.Profile.Name != "" {
		t.Errorf("empty load: %+v, %v", got, err)
	}
}

func TestEntriesNewestFirst(t *testing.T) {
	db := openTemp(t)
	l := &core.Library{}
	c, _ := l.AddCategory("Films", core.Film, time.Now())
	base := time.Date(2026, 1, 1, 0, 0, 5, 0, time.UTC)
	// 00:00:05 and 00:00:05.5 sort wrongly as variable-width RFC 3339 text.
	l.AddEntry(c.ID, core.EntryInput{Title: "a", Status: core.Ongoing}, base)
	l.AddEntry(c.ID, core.EntryInput{Title: "b", Status: core.Ongoing}, base.Add(500*time.Millisecond))
	db.ReplaceAll(l)
	got, _ := db.Load()
	if got.Entries[0].Title != "b" {
		t.Errorf("first = %q, want newest", got.Entries[0].Title)
	}
}
