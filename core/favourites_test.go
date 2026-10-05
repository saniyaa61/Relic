package core

import (
	"errors"
	"testing"
	"time"
)

func titles(es []*Entry) string {
	s := ""
	for _, e := range es {
		s += e.Title
	}
	return s
}

func TestTopFive(t *testing.T) {
	l := newLib(t)
	c := mustCat(t, l, "Movies", Film)
	var ids []string
	for _, name := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		e := mustEntry(t, l, c, EntryInput{Title: name}, t0)
		ids = append(ids, e.ID)
	}
	plain := mustEntry(t, l, c, EntryInput{Title: "p"}, t0)
	if err := l.AddTopFive(plain.ID); !errors.Is(err, ErrNotFavorite) {
		t.Errorf("non-favourite: %v", err)
	}
	for _, id := range ids {
		l.ToggleFavorite(id)
	}
	for _, id := range ids[:5] {
		if err := l.AddTopFive(id); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.AddTopFive(ids[5]); !errors.Is(err, ErrTopFiveFull) || err.Error() != "Your Top 5 is full — remove one to add this." {
		t.Errorf("6th: %v", err)
	}
	if got := titles(l.TopFive(c.ID)); got != "abcde" {
		t.Errorf("top five = %s", got)
	}
	// Top 5 entries aren't repeated below.
	if got := titles(l.FavouritesIn(c.ID)); got != "fg" {
		t.Errorf("favourites = %s", got)
	}
	tests := []struct {
		id   string
		dir  int
		want string
	}{
		{ids[0], -1, "abcde"}, // can't move past the start
		{ids[0], 1, "bacde"},
		{ids[4], 1, "bacde"}, // can't move past the end
		{ids[4], -1, "baced"},
	}
	for _, tt := range tests {
		l.MoveTopFive(tt.id, tt.dir)
		if got := titles(l.TopFive(c.ID)); got != tt.want {
			t.Errorf("after move: %s, want %s", got, tt.want)
		}
	}
	if r, ok := l.TopFiveRank(ids[1]); !ok || r != 1 {
		t.Errorf("rank of b = %d %v", r, ok)
	}
	// Removing drops it to the bottom of the favourites list.
	l.RemoveTopFive(ids[1])
	if got := titles(l.FavouritesIn(c.ID)); got != "fgb" {
		t.Errorf("after remove: %s", got)
	}
	// Unfavouriting takes it out of the Top 5 too.
	l.ToggleFavorite(ids[0])
	if got := titles(l.TopFive(c.ID)); got != "ced" {
		t.Errorf("after unfavourite: %s", got)
	}
	if _, ok := l.TopFiveRank(ids[0]); ok {
		t.Error("unfavourited entry still ranked")
	}
	if cats := l.FavouriteCategories(); len(cats) != 1 || cats[0] != c {
		t.Errorf("favourite categories = %v", cats)
	}
	l.DeleteEntry(ids[2])
	if got := titles(l.TopFive(c.ID)); got != "ed" {
		t.Errorf("after delete: %s", got)
	}
}

func TestFavouritesWithNilMaps(t *testing.T) {
	// A freshly loaded library may have no favourite maps yet.
	l := &Library{}
	c, _ := l.AddCategory("Movies", Film, t0)
	e, _ := l.AddEntry(c.ID, EntryInput{Title: "a", Status: Finished}, t0)
	if err := l.DeleteEntry(e.ID); err != nil {
		t.Fatal(err)
	}
	e, _ = l.AddEntry(c.ID, EntryInput{Title: "b", Status: Finished}, t0)
	if fav, _ := l.ToggleFavorite(e.ID); !fav {
		t.Error("should be favourite")
	}
}

func TestOneYearAgo(t *testing.T) {
	now := at(ist, 2026, 10, 5, 9, 0)
	tests := []struct {
		name    string
		created time.Time
		status  Status
		want    bool
	}{
		{"exactly a year", at(ist, 2025, 10, 5, 20, 0), Finished, true},
		{"three days before", at(ist, 2025, 10, 2, 0, 5), Finished, true},
		{"three days after", at(ist, 2025, 10, 8, 23, 55), Finished, true},
		{"four days off", at(ist, 2025, 10, 9, 0, 5), Finished, false},
		{"ongoing doesn't count", at(ist, 2025, 10, 5, 12, 0), Ongoing, false},
		{"two years ago", at(ist, 2024, 10, 5, 12, 0), Finished, false},
	}
	for _, tt := range tests {
		l := newLib(t)
		c := mustCat(t, l, "Movies", Film)
		mustEntry(t, l, c, EntryInput{Title: "x", Status: tt.status}, tt.created)
		if got := l.OneYearAgo(now, ist) != nil; got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
	// First match, newest first.
	l := newLib(t)
	c := mustCat(t, l, "Movies", Film)
	mustEntry(t, l, c, EntryInput{Title: "older", Status: Finished}, at(ist, 2025, 10, 3, 12, 0))
	newer := mustEntry(t, l, c, EntryInput{Title: "newer", Status: Finished}, at(ist, 2025, 10, 6, 12, 0))
	if got := l.OneYearAgo(now, ist); got != newer {
		t.Errorf("got %v", got)
	}
	// 29 Feb next year looks back to around 1 March.
	leap := at(ist, 2028, 2, 29, 9, 0)
	l2 := newLib(t)
	c2 := mustCat(t, l2, "Movies", Film)
	mustEntry(t, l2, c2, EntryInput{Title: "x", Status: Finished}, at(ist, 2027, 3, 3, 9, 0))
	if l2.OneYearAgo(leap, ist) == nil {
		t.Error("leap day lookup")
	}
}
