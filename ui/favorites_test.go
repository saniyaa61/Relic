package ui

import (
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/saniyaa61/relic/core"
)

// favPage opens the Favorites tab on the category that has entry e.
func favPage(t *testing.T, a *App, catID string) *favoritesPage {
	t.Helper()
	a.Go(TabFavorites)
	p := a.roots[TabFavorites].(*favoritesPage)
	p.catID = catID
	frame(a)
	return p
}

// The "+" on a favourite ranks it; ‹ › reorder; ✕ sends it back to the
// favourites list. Each change is saved and announced.
func TestTopFiveOnFavoritesPage(t *testing.T) {
	a, reload := libApp(t)
	// The series category has one ranked entry and four other favourites.
	var cat string
	for c, ids := range a.Lib.Favourites.TopFive {
		if len(ids) > 0 {
			cat = c
		}
	}
	if cat == "" {
		t.Fatal("fixture: no Top 5")
	}
	p := favPage(t, a, cat)
	favs := a.Lib.FavouritesIn(cat)
	if len(favs) < 2 {
		t.Fatal("fixture: want two unranked favourites")
	}
	first := a.Lib.TopFive(cat)[0].ID
	add := favs[0].ID

	p.favCard(add).plus.Click()
	frame(a)
	if got := ids(reload().TopFive(cat)); len(got) != 2 || got[1] != add {
		t.Fatalf("after +: Top 5 %v", got)
	}
	if a.toast.msg != "Added to your All-Time Top 5 ✦" {
		t.Errorf("toast %q", a.toast.msg)
	}
	if p.glideFrom == nil {
		t.Error("cards should glide after a change")
	}
	for _, e := range a.Lib.FavouritesIn(cat) {
		if e.ID == add {
			t.Error("a ranked entry is not repeated in the favourites list")
		}
	}

	p.topCard(add).left.Click()
	frame(a)
	if got := ids(reload().TopFive(cat)); got[0] != add || got[1] != first {
		t.Errorf("after ‹: %v", got)
	}

	p.topCard(add).remove.Click()
	frame(a)
	lib := reload()
	if got := ids(lib.TopFive(cat)); len(got) != 1 || got[0] != first {
		t.Errorf("after ✕: %v", got)
	}
	if fs := lib.FavouritesIn(cat); fs[len(fs)-1].ID != add {
		t.Error("a removed entry drops to the bottom of the favourites")
	}
	if a.toast.msg != "Removed from All-Time Top 5" {
		t.Errorf("toast %q", a.toast.msg)
	}
}

func TestTopFiveFull(t *testing.T) {
	a, reload := libApp(t)
	cat := a.Lib.Categories[1].ID
	n := 0
	for _, e := range a.Lib.Entries {
		if e.CategoryID == cat && !e.Favorite {
			a.Lib.ToggleFavorite(e.ID)
		}
		if e.CategoryID == cat {
			n++
		}
	}
	if n <= core.TopFiveMax {
		t.Skip("fixture: category too small")
	}
	p := favPage(t, a, cat)
	for range 8 {
		favs := a.Lib.FavouritesIn(cat)
		if len(favs) == 0 {
			break
		}
		p.favCard(favs[0].ID).plus.Click()
		frame(a)
	}
	if got := len(reload().TopFive(cat)); got != core.TopFiveMax {
		t.Errorf("Top 5 holds %d", got)
	}
	if !a.toast.isErr || a.toast.msg != "Your Top 5 is full — remove one to add this." {
		t.Errorf("toast %q (error %v)", a.toast.msg, a.toast.isErr)
	}
}

// Tapping a tab shows that category.
func TestFavoritesTabs(t *testing.T) {
	a, _ := libApp(t)
	p := favPage(t, a, "")
	if p.catID != a.Lib.Categories[0].ID {
		t.Fatal("the first category is shown first")
	}
	p.tabs.clicks[2].Click()
	frame(a)
	if p.catID != a.Lib.Categories[2].ID {
		t.Error("tab 3 should show the third category")
	}
	if p.tabs.animAt.IsZero() {
		t.Error("the picked tab should glide into view")
	}
	// A tab off the right edge glides towards the middle.
	now := a.Now()
	a.Now = func() time.Time { return now }
	last := len(a.Lib.Categories) - 1
	p.tabs.clicks[last].Click()
	frame(a)
	later := now.Add(time.Second)
	a.Now = func() time.Time { return later }
	frame(a)
	if pos := p.tabs.list.Position; pos.First == 0 && pos.Offset == 0 {
		t.Error("the strip didn't scroll to the last tab")
	}

	// No categories: the "No categories yet" page, no panic.
	b, _ := testApp(t, nil)
	b.Go(TabFavorites)
	frame(b)
}

// Favouriting on the entry page says so, then offers the Top 5; "Rank it"
// ranks it.
func TestRankPrompt(t *testing.T) {
	a, reload := libApp(t)
	e := findEntry(a.Lib, func(e *core.Entry) bool { return !e.Favorite && len(a.Lib.TopFive(e.CategoryID)) < core.TopFiveMax })
	p := newEntryDetail(e.ID)
	a.Push(p)
	frame(a)
	now := a.Now()
	a.Now = func() time.Time { return now }
	p.fav.Click()
	frame(a)
	if a.toast.msg != "Added to favourites ♥" {
		t.Errorf("toast %q", a.toast.msg)
	}
	if a.sheet != nil {
		t.Fatal("the prompt waits a moment")
	}
	later := now.Add(400 * time.Millisecond)
	a.Now = func() time.Time { return later }
	frame(a)
	if a.sheet == nil {
		t.Fatal("no Top 5 prompt")
	}
	rp := openRankPrompt(a, e.ID) // the same sheet again, to reach its buttons
	rp.rank.Click()
	var ops op.Ops
	a.sheet.Body(layout.Context{Ops: &ops}, a)
	if a.toast.msg != "Added to your All-Time Top 5 ✦" || !a.sheet.closing() {
		t.Errorf("Rank it: toast %q, sheet closing %v", a.toast.msg, a.sheet.closing())
	}
	if r, ok := reload().TopFiveRank(e.ID); !ok || r != len(a.Lib.TopFive(e.CategoryID)) {
		t.Errorf("rank %d %v", r, ok)
	}

	// Unfavouriting says so and takes it out of the Top 5.
	p.fav.Click()
	frame(a)
	if a.toast.msg != "Removed from favourites" {
		t.Errorf("toast %q", a.toast.msg)
	}
	if _, ok := reload().TopFiveRank(e.ID); ok {
		t.Error("unfavourited entry still ranked")
	}
}

func ids(es []*core.Entry) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.ID
	}
	return out
}
