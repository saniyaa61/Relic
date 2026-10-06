package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/saniyaa61/relic/core"
	"github.com/saniyaa61/relic/importer"
	"github.com/saniyaa61/relic/store"
)

// libApp opens a temporary store holding the scrubbed archive and returns
// an app on it, plus a function that reopens the store as a fresh launch
// would, to check what was saved.
func libApp(t *testing.T) (*App, func() *core.Library) {
	t.Helper()
	f, err := os.Open("../importer/testdata/scrubbed-archive.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	res, err := importer.Read(f, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "relic.db")
	db, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.ReplaceAll(res.Library); err != nil {
		t.Fatal(err)
	}
	a, _ := testApp(t, db)
	reload := func() *core.Library {
		lib, err := db.Load()
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
	return a, reload
}

// confirm submits the open dialog with value, as tapping its button does.
func confirm(t *testing.T, a *App, value string) {
	t.Helper()
	d := a.dialog
	if d == nil {
		t.Fatal("no dialog open")
	}
	d.input.Editor.SetText(value)
	var ops op.Ops
	d.submit(layout.Context{Ops: &ops}, a)
	a.dialog = nil
}

func categoryNamed(lib *core.Library, name string) *core.Category {
	for _, c := range lib.Categories {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestLibraryCategories(t *testing.T) {
	a, reload := libApp(t)
	n := len(a.Lib.Categories)

	// New category, with its type from the picker.
	newCategoryDialog(a)
	a.dialog.picker.value = core.Podcast
	confirm(t, a, "  Late-night radio ")
	if c := categoryNamed(reload(), "Late-night radio"); c == nil || c.Type != core.Podcast {
		t.Fatalf("new category not saved as a podcast category: %+v", c)
	}
	if a.toast.msg != "Category created" || a.toast.isErr {
		t.Errorf("toast %q", a.toast.msg)
	}

	// A name already in use (any case) is refused with the prototype's words.
	newCategoryDialog(a)
	confirm(t, a, "late-night RADIO")
	if len(reload().Categories) != n+1 || a.toast.msg != "You already have that category" || !a.toast.isErr {
		t.Errorf("duplicate: %d categories, toast %q", len(reload().Categories), a.toast.msg)
	}

	// Rename and change type; entries keep their own type.
	series := a.Lib.Categories[1]
	entries := a.Lib.InCategory(series.ID)
	renameCategoryDialog(a, series.ID)
	if got := a.dialog.input.Editor.Text(); got != series.Name {
		t.Errorf("rename dialog starts with %q", got)
	}
	a.dialog.picker.value = core.Film
	confirm(t, a, "Dramas")
	lib := reload()
	c := lib.Category(series.ID)
	if c.Name != "Dramas" || c.Type != core.Film || a.toast.msg != "Category updated" {
		t.Errorf("after rename: %+v, toast %q", c, a.toast.msg)
	}
	if e := lib.Entry(entries[0].ID); e.Type != core.Series {
		t.Errorf("entry type changed to %s", e.Type)
	}

	// Rename to another category's name: refused, nothing changes.
	renameCategoryDialog(a, series.ID)
	a.dialog.picker.value = core.Book
	confirm(t, a, "films 1")
	if c := reload().Category(series.ID); c.Name != "Dramas" || c.Type != core.Film {
		t.Errorf("clashing rename changed the category: %+v", c)
	}

	// Delete removes the category and its entries.
	deleteCategoryDialog(a, series.ID)
	if a.dialog.Title != `Delete "Dramas"?` || a.dialog.Sub != "This category holds 6 entries. Deleting it removes them too — this can't be undone." {
		t.Errorf("delete dialog %q / %q", a.dialog.Title, a.dialog.Sub)
	}
	a.dialog.OnConfirm(a, "")
	lib = reload()
	if lib.Category(series.ID) != nil || len(lib.InCategory(series.ID)) != 0 || lib.Entry(entries[0].ID) != nil {
		t.Error("deleted category or its entries still stored")
	}
	if a.toast.msg != "Category deleted" {
		t.Errorf("toast %q", a.toast.msg)
	}
}

func TestLibraryFolders(t *testing.T) {
	a, reload := libApp(t)
	series := a.Lib.Categories[1]
	first := series.Folders[0]
	inFirst := len(a.Lib.InFolder(series.ID, first))
	loose := len(a.Lib.InFolder(series.ID, ""))

	newFolderDialog(a, series.ID)
	if a.dialog.Sub != "A shelf inside "+series.Name+" — by mood, year, whatever you like." {
		t.Errorf("sub %q", a.dialog.Sub)
	}
	confirm(t, a, "Rainy Sundays")
	if c := reload().Category(series.ID); c.Folders[len(c.Folders)-1] != "Rainy Sundays" || a.toast.msg != "Folder created" {
		t.Errorf("folders %v, toast %q", c.Folders, a.toast.msg)
	}
	newFolderDialog(a, series.ID)
	confirm(t, a, "rainy sundays")
	if a.toast.msg != "That folder already exists" || !a.toast.isErr {
		t.Errorf("duplicate folder toast %q", a.toast.msg)
	}

	// Renaming carries the entries along, and that is saved too.
	renameFolderDialog(a, series.ID, first)
	confirm(t, a, "Comfort")
	lib := reload()
	if got := len(lib.InFolder(series.ID, "Comfort")); got != inFirst || len(lib.InFolder(series.ID, first)) != 0 {
		t.Errorf("after rename %d entries in Comfort, want %d", got, inFirst)
	}

	// Deleting moves them to Uncategorised: nothing is lost.
	deleteFolderDialog(a, series.ID, "Comfort")
	if a.dialog.Sub != "5 entries will move back to Uncategorised — nothing is lost." {
		t.Errorf("sub %q", a.dialog.Sub)
	}
	a.dialog.OnConfirm(a, "")
	lib = reload()
	if got := len(lib.InFolder(series.ID, "")); got != loose+inFirst {
		t.Errorf("Uncategorised has %d, want %d", got, loose+inFirst)
	}
	if folderIndexOf(lib.Category(series.ID), "Comfort") >= 0 || a.toast.msg != "Folder deleted" {
		t.Errorf("folder still there or toast %q", a.toast.msg)
	}
}

func folderIndexOf(c *core.Category, name string) int {
	for i, f := range c.Folders {
		if f == name {
			return i
		}
	}
	return -1
}

// Back closes a card's menu first, then the search, then leaves the page;
// the category and folder pages keep the bottom bar, as in the prototype.
func TestLibraryNavigation(t *testing.T) {
	a, _ := libApp(t)
	a.Go(TabLibrary)
	root := a.roots[TabLibrary].(*libraryRoot)
	frame(a)
	m := &root.card(a.Lib.Categories[0].ID).menu
	m.open = true
	root.search.Toggle(a)
	if !a.canGoBack() {
		t.Fatal("back should close the menu")
	}
	a.Back()
	if m.open || !root.search.IsOpen() {
		t.Fatalf("first back: menu open %v, search open %v; want menu closed, search still open", m.open, root.search.IsOpen())
	}
	a.Back()
	if root.search.IsOpen() || a.Tab() != TabLibrary {
		t.Fatal("second back should close the search and stay in the Library")
	}

	series := a.Lib.Categories[1]
	a.Push(&categoryPage{catID: series.ID})
	a.Push(&folderPage{catID: series.ID, folder: series.Folders[0]})
	if !a.navShown() {
		t.Error("folder page should keep the bottom bar")
	}
	frame(a)
	a.Back()
	if _, ok := a.top().(*categoryPage); !ok {
		t.Fatal("back from a folder should return to its category")
	}
	a.Back()
	if a.top() != root {
		t.Fatal("back from a category should return to the categories")
	}

	// A category page whose category was deleted closes itself.
	a.Push(&categoryPage{catID: series.ID})
	a.Lib.DeleteCategory(series.ID)
	frame(a)
	if a.top() != root {
		t.Error("page for a deleted category should close")
	}
}

// Search on a category page only finds that category's entries.
func TestLibrarySearchScope(t *testing.T) {
	a, _ := libApp(t)
	series := a.Lib.Categories[1]
	var list entryList
	rows := searchResults(a, &list, a.Lib.InCategory(series.ID), "entry", series.Name, IconBook)
	// header + one row per match + closing spacer
	if want := len(a.Lib.InCategory(series.ID)) + 2; len(rows) != want {
		t.Errorf("%d rows, want %d", len(rows), want)
	}
	rows = searchResults(a, &list, a.Lib.Entries, "no such thing", "", IconBook)
	if len(rows) != 3 { // header, empty state, spacer
		t.Errorf("no-match search gave %d rows", len(rows))
	}
}
