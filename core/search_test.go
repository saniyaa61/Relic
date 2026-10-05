package core

import "testing"

func TestMatches(t *testing.T) {
	e := &Entry{Type: Film, Title: "Dune: Part Two", Review: "Vast and quiet.", Folder: "Sci-fi",
		Tags: []string{"Haunting"}, Fields: Fields{Director: "Denis Villeneuve", Duration: 166,
			WatchedDate: Date{2024, 3, 1}, Author: "not a film field"}}
	tests := []struct {
		q    string
		want bool
	}{
		{"", true},
		{"dune", true},
		{"  DUNE ", true},
		{"quiet", true},       // review
		{"haunt", true},       // tag
		{"movies", true},      // category name
		{"sci-fi", true},      // folder
		{"villeneuve", true},  // type field
		{"166", true},         // number field
		{"2024-03", true},     // date field
		{"not a film", false}, // field of another type
		{"arrival", false},
	}
	for _, tt := range tests {
		if got := Matches(e, "Movies", tt.q); got != tt.want {
			t.Errorf("Matches(%q) = %v, want %v", tt.q, got, tt.want)
		}
	}
}

func TestSearchScopes(t *testing.T) {
	l := newLib(t)
	movies := mustCat(t, l, "Movies", Film)
	books := mustCat(t, l, "Books", Book)
	l.AddFolder(movies.ID, "Bollywood")
	a := mustEntry(t, l, movies, EntryInput{Title: "Dune", Folder: "Bollywood"}, t0)
	b := mustEntry(t, l, movies, EntryInput{Title: "Dune 2", Status: Finished}, t0)
	c := mustEntry(t, l, books, EntryInput{Title: "Dune (novel)"}, t0)
	tests := []struct {
		name  string
		scope []*Entry
		want  []*Entry
	}{
		{"library root", l.Entries, []*Entry{c, b, a}},
		{"category", l.InCategory(movies.ID), []*Entry{b, a}},
		{"folder", l.InFolder(movies.ID, "Bollywood"), []*Entry{a}},
		{"uncategorised", l.InFolder(movies.ID, ""), []*Entry{b}},
		{"still with you", l.StillWithYou(), []*Entry{c, a}},
		{"recently finished", l.RecentlyFinished(), []*Entry{b}},
	}
	for _, tt := range tests {
		got := l.Search(tt.scope, "dune")
		if len(got) != len(tt.want) {
			t.Errorf("%s: %d results, want %d", tt.name, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("%s: result %d = %q, want %q", tt.name, i, got[i].Title, tt.want[i].Title)
			}
		}
	}
	if got := l.Search(l.Entries, "books"); len(got) != 1 || got[0] != c {
		t.Errorf("category-name search: %v", got)
	}
}

func TestResultsHeader(t *testing.T) {
	tests := []struct {
		n     int
		q     string
		scope string
		want  string
	}{
		{3, "dune", "Movies", "3 results for “dune” in Movies"},
		{1, "dune", "", "1 result for “dune” across your library"},
		{0, " x ", "Bollywood", "0 results for “x” in Bollywood"},
	}
	for _, tt := range tests {
		if got := ResultsHeader(tt.n, tt.q, tt.scope); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
	if got := NoResults("dune"); got != "Nothing matches “dune”." {
		t.Errorf("NoResults = %q", got)
	}
}
