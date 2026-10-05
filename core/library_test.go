package core

import (
	"errors"
	"testing"
)

func TestCategories(t *testing.T) {
	l := newLib(t)
	movies := mustCat(t, l, "Movies", Film)
	tests := []struct {
		name string
		do   func() error
		want error
	}{
		{"duplicate ignores case", func() error { _, err := l.AddCategory(" movies ", Film, t0); return err }, ErrNameTaken},
		{"empty name", func() error { _, err := l.AddCategory(" ", Film, t0); return err }, ErrNameRequired},
		{"bad type", func() error { _, err := l.AddCategory("X", "game", t0); return err }, ErrBadType},
		{"rename to own name in new case", func() error { return l.RenameCategory(movies.ID, "MOVIES") }, nil},
		{"folder", func() error { return l.AddFolder(movies.ID, "Bollywood") }, nil},
		{"duplicate folder", func() error { return l.AddFolder(movies.ID, "bollywood") }, ErrNameTaken},
		{"rename missing folder", func() error { return l.RenameFolder(movies.ID, "Nope", "X") }, ErrNotFound},
	}
	for _, tt := range tests {
		if err := tt.do(); !errors.Is(err, tt.want) {
			t.Errorf("%s: err = %v, want %v", tt.name, err, tt.want)
		}
	}
	books := mustCat(t, l, "Books", Book)
	if err := l.RenameCategory(books.ID, "movies"); !errors.Is(err, ErrNameTaken) {
		t.Errorf("rename clash: %v", err)
	}
}

func TestFolderRenameAndDelete(t *testing.T) {
	l := newLib(t)
	movies := mustCat(t, l, "Movies", Film)
	l.AddFolder(movies.ID, "Bollywood")
	e := mustEntry(t, l, movies, EntryInput{Title: "a", Folder: "Bollywood"}, t0)
	if err := l.RenameFolder(movies.ID, "bollywood", "Hindi"); err != nil {
		t.Fatal(err)
	}
	if e.Folder != "Hindi" || movies.Folders[0] != "Hindi" {
		t.Errorf("rename: entry %q, folders %v", e.Folder, movies.Folders)
	}
	if err := l.DeleteFolder(movies.ID, "Hindi"); err != nil {
		t.Fatal(err)
	}
	if e.Folder != "" || len(movies.Folders) != 0 || l.Entry(e.ID) == nil {
		t.Errorf("delete folder should move entry to Uncategorised, got %q", e.Folder)
	}
}

func TestDeleteCategory(t *testing.T) {
	l := newLib(t)
	movies := mustCat(t, l, "Movies", Film)
	books := mustCat(t, l, "Books", Book)
	a := mustEntry(t, l, movies, EntryInput{Title: "a"}, t0)
	b := mustEntry(t, l, books, EntryInput{Title: "b"}, t0)
	l.ToggleFavorite(a.ID)
	l.AddTopFive(a.ID)
	if err := l.DeleteCategory(movies.ID); err != nil {
		t.Fatal(err)
	}
	if l.Entry(a.ID) != nil || l.Entry(b.ID) == nil || len(l.Categories) != 1 {
		t.Error("category delete should remove only its entries")
	}
	if _, ok := l.Favourites.TopFive[movies.ID]; ok {
		t.Error("Top 5 left behind")
	}
}

func TestNewEntriesGoToTop(t *testing.T) {
	l := newLib(t)
	c := mustCat(t, l, "Movies", Film)
	a := mustEntry(t, l, c, EntryInput{Title: "a"}, t0)
	b := mustEntry(t, l, c, EntryInput{Title: "b"}, t0)
	if l.Entries[0] != b || l.Entries[1] != a {
		t.Error("newest should be first")
	}
}

func TestMoveEntryChangesTypeAndFavourites(t *testing.T) {
	l := newLib(t)
	movies := mustCat(t, l, "Movies", Film)
	dramas := mustCat(t, l, "Dramas", Series)
	e := mustEntry(t, l, movies, EntryInput{Title: "a"}, t0)
	l.ToggleFavorite(e.ID)
	l.AddTopFive(e.ID)
	err := l.UpdateEntry(e.ID, dramas.ID, EntryInput{Title: "a", Status: Ongoing, StartEpisodes: 2}, t0)
	if err != nil {
		t.Fatal(err)
	}
	if e.Type != Series || e.WatchedEpisodes() != 2 {
		t.Errorf("type %s, eps %d", e.Type, e.WatchedEpisodes())
	}
	if len(l.TopFive(movies.ID)) != 0 || len(l.FavouritesIn(dramas.ID)) != 1 {
		t.Error("favourite should follow the entry to its new category")
	}
}

func TestRecentlyFinishedOrder(t *testing.T) {
	l := newLib(t)
	c := mustCat(t, l, "Movies", Film)
	a := mustEntry(t, l, c, EntryInput{Title: "a", Status: Finished}, t0)
	b := mustEntry(t, l, c, EntryInput{Title: "b"}, t0.Add(1))
	b.SetStatus(Finished, t0.Add(2))
	a.SetStatus(Ongoing, t0.Add(3))
	a.SetStatus(Finished, t0.Add(4))
	got := l.RecentlyFinished()
	if len(got) != 2 || got[0] != a || got[1] != b {
		t.Errorf("order = %q, %q", got[0].Title, got[1].Title)
	}
}

func TestFieldsFor(t *testing.T) {
	for _, typ := range EntryTypes {
		if len(FieldsFor(typ, Fields{})) == 0 {
			t.Errorf("%s has no fields", typ)
		}
		for _, f := range FieldsFor(typ, Fields{}) {
			if f.Value != "" {
				t.Errorf("%s.%s: empty fields should give \"\", got %q", typ, f.ID, f.Value)
			}
		}
		if typ.Label() == "" {
			t.Errorf("%s has no label", typ)
		}
	}
	if Other.Label() != "Other" || EntryType("x").Valid() {
		t.Error("labels/valid")
	}
}
