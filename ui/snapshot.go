package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/saniyaa61/relic/core"
)

// SnapshotScreens lists the states cmd/snapshot can render.
var SnapshotScreens = []string{"home", "library", "new", "favorites", "digest", "subpage", "search", "prompt", "confirm", "toast", "toast-error",
	"lib-menu", "lib-new-category", "lib-picker", "lib-rename", "lib-delete", "lib-search",
	"lib-category", "lib-category-films", "lib-category-empty", "lib-new-folder", "lib-delete-folder",
	"lib-folder", "lib-folder-search", "lib-folder-empty",
	"form-new", "form-series", "form-podcast", "form-edit", "form-calendar",
	"home-empty", "home-memory", "home-streak-icon", "home-streak-plain", "swy", "rf", "ae", "swy-search", "home-finished-badge",
	"fav-series", "fav-top3", "fav-podcasts", "fav-rank", "fav-nocats",
	"detail", "detail-hero-prototype", "detail-noposter", "detail-book", "detail-film",
	"log-session", "log-book", "log-rewatch", "log-rewatch-partial", "log-edit", "reached-end", "reached-end-accent"}

// ShowForSnapshot puts the app into one of SnapshotScreens with every
// animation finished, for rendering to a PNG.
func (a *App) ShowForSnapshot(screen string) error {
	t0 := a.Now()
	a.Now = func() time.Time { return t0 }
	a.shownAt = t0
	settle := func() {
		// Jump the clock past every animation.
		later := t0.Add(time.Second)
		a.Now = func() time.Time { return later }
	}
	defer settle()
	switch screen {
	case "home":
	case "library":
		a.Go(TabLibrary)
	case "new":
		a.Go(TabNew)
	case "favorites", "fav-series", "fav-top3", "fav-podcasts", "fav-rank", "fav-nocats":
		return a.favoritesForSnapshot(screen)
	case "digest":
		a.Go(TabDigest)
	case "subpage", "search":
		p := &themePreview{}
		a.Push(p)
		if screen == "search" {
			p.search.Toggle(a)
			p.search.input.Editor.SetText("dune")
		}
	case "prompt":
		a.ShowDialog(PromptDialog("New category",
			"Call it whatever makes sense to you — Films, Comfort Rewatches, Books for the train.",
			"Category name", "e.g. Films", "", "Create", nil))
	case "confirm":
		a.ShowDialog(ConfirmDialog("Delete this entry?",
			"This piece of your journey will be removed. This can't be undone.", "", "Delete", func(*App) {}))
	case "toast":
		a.Toast("Session logged ✦")
	case "toast-error":
		a.ToastError("You already have that category")
	case "home-empty":
		a.Lib.Categories, a.Lib.Entries = nil, nil
	case "home-memory":
		// Pretend the newest finished entry was logged a year ago today.
		for _, e := range a.Lib.Entries {
			if e.Status == core.Finished {
				e.CreatedAt = t0.AddDate(-1, 0, 0)
				break
			}
		}
	case "home-streak-icon":
		homeStreak = streakIcon
	case "home-streak-plain":
		homeStreak = streakPlain
	case "home-finished-badge":
		// Bring an ongoing series to its last episode.
		for _, e := range a.Lib.Entries {
			if e.Status == core.Ongoing && e.Type.TracksProgress() && !e.ReachedEnd() {
				if done, total, _ := e.Progress(); total > 0 {
					n := total - done
					e.LogSession(core.SessionInput{Episodes: &n, FromPage: &done, ToPage: &total}, t0)
					break
				}
			}
		}
	case "swy":
		a.Push(newPosterPage(pageStillWithYou))
	case "rf":
		a.Push(newPosterPage(pageRecentlyFinished))
	case "ae":
		a.Push(newPosterPage(pageAllEntries))
	case "swy-search":
		p := newPosterPage(pageStillWithYou)
		a.Push(p)
		p.search.Toggle(a)
		p.search.input.Editor.SetText("entry 1")
	default:
		if strings.HasPrefix(screen, "detail") || strings.HasPrefix(screen, "log-") || strings.HasPrefix(screen, "reached-end") {
			return a.detailForSnapshot(screen)
		}
		if strings.HasPrefix(screen, "form-") {
			return a.formForSnapshot(screen)
		}
		if strings.HasPrefix(screen, "lib-") {
			return a.libraryForSnapshot(screen)
		}
		return fmt.Errorf("unknown screen %q; want one of %v", screen, SnapshotScreens)
	}
	return nil
}

// libraryForSnapshot sets up the Library states, using the second
// category (with its folders) as the prototype screenshots do.
func (a *App) libraryForSnapshot(screen string) error {
	cats := a.Lib.Categories
	if len(cats) < 4 {
		return fmt.Errorf("%s needs a library with categories (use -archive)", screen)
	}
	a.Go(TabLibrary)
	root := a.roots[TabLibrary].(*libraryRoot)
	series, films, empty := cats[1], cats[0], cats[3]
	openCat := func(c *core.Category) *categoryPage {
		p := &categoryPage{catID: c.ID}
		a.Push(p)
		return p
	}
	switch screen {
	case "lib-menu":
		m := &root.card(films.ID).menu
		m.open, m.openedAt = true, a.Now()
	case "lib-new-category":
		newCategoryDialog(a)
	case "lib-picker":
		newCategoryDialog(a)
		a.dialog.picker.open, a.dialog.picker.openedAt = true, a.Now()
	case "lib-rename":
		renameCategoryDialog(a, series.ID)
	case "lib-delete":
		deleteCategoryDialog(a, series.ID)
	case "lib-search":
		root.search.Toggle(a)
		root.search.input.Editor.SetText("entry 1")
	case "lib-category":
		openCat(series)
	case "lib-category-films":
		openCat(films)
	case "lib-category-empty":
		openCat(empty)
	case "lib-new-folder":
		openCat(series)
		newFolderDialog(a, series.ID)
	case "lib-delete-folder":
		openCat(series)
		deleteFolderDialog(a, series.ID, series.Folders[0])
	case "lib-folder", "lib-folder-search":
		openCat(series)
		p := &folderPage{catID: series.ID, folder: series.Folders[0]}
		a.Push(p)
		if screen == "lib-folder-search" {
			p.search.Toggle(a)
			p.search.input.Editor.SetText("x")
		}
	case "lib-folder-empty":
		openCat(series)
		a.Push(&folderPage{catID: series.ID, folder: series.Folders[1]})
	default:
		return fmt.Errorf("unknown screen %q", screen)
	}
	return nil
}

// formForSnapshot sets up the entry form states.
func (a *App) formForSnapshot(screen string) error {
	cats := a.Lib.Categories
	if len(cats) < 4 || len(a.Lib.Entries) == 0 {
		return fmt.Errorf("%s needs a library (use -archive)", screen)
	}
	a.Go(TabNew)
	f := a.roots[TabNew].(*entryForm)
	switch screen {
	case "form-new":
	case "form-series":
		f.catID = cats[1].ID
	case "form-podcast":
		f.catID = cats[3].ID
	case "form-edit":
		a.Go(TabHome)
		a.Push(editEntryPage(a, a.Lib.Entries[0]))
	case "form-calendar":
		openCalendar(a, "Date watched", core.Date{Year: 2026, Month: 6, Day: 16}, func(core.Date) {})
	default:
		return fmt.Errorf("unknown screen %q", screen)
	}
	return nil
}

// detailForSnapshot shows entry pages and the logging sheet, using the
// archive's entries in the prototype screenshots' order.
func (a *App) detailForSnapshot(screen string) error {
	es := a.Lib.Entries
	if len(es) < 3 {
		return fmt.Errorf("%s needs a library (use -archive)", screen)
	}
	find := func(keep func(*core.Entry) bool) *core.Entry {
		for _, e := range es {
			if keep(e) {
				return e
			}
		}
		return es[0]
	}
	e := es[0]
	switch screen {
	case "detail-noposter":
		e = find(func(e *core.Entry) bool { return e.Poster == "" })
		if e.Poster != "" {
			e.Poster = ""
		}
	case "detail-book", "log-book":
		e = find(func(e *core.Entry) bool { return e.Type == core.Book })
	case "detail-film":
		e = find(func(e *core.Entry) bool { return e.Type == core.Film })
	case "log-rewatch", "log-rewatch-partial":
		e = find(func(e *core.Entry) bool { return e.Type == core.Series && e.Status == core.Finished })
	}
	if screen == "detail-hero-prototype" {
		detailHero = heroPrototype
	}
	a.Push(newEntryDetail(e.ID))
	switch screen {
	case "log-session", "log-book":
		openLogSheet(a, e.ID, false, "")
	case "log-rewatch", "log-rewatch-partial":
		l := openLogSheet(a, e.ID, true, "")
		l.full = screen != "log-rewatch-partial"
	case "log-edit":
		openLogSheet(a, e.ID, false, e.Sessions[0].ID)
	case "reached-end", "reached-end-accent":
		markFinishedRed = screen == "reached-end"
		reachedEndDialog(a, e.ID)
	}
	return nil
}

// favoritesForSnapshot shows the Favorites tab in the prototype
// screenshots' states: a category's tab, a fuller Top 5, an empty one,
// the "Rank it" prompt and no categories at all.
func (a *App) favoritesForSnapshot(screen string) error {
	lib := a.Lib
	if screen == "fav-nocats" {
		lib.Categories, lib.Entries = nil, nil
	}
	a.Go(TabFavorites)
	p := a.roots[TabFavorites].(*favoritesPage)
	pick := func(i int) {
		if i < len(lib.Categories) {
			p.catID = lib.Categories[i].ID
		}
	}
	switch screen {
	case "fav-series":
		pick(1)
	case "fav-top3":
		pick(1)
		for _, e := range lib.Entries { // in archive order, as the prototype shot does
			if e.Favorite && e.CategoryID == p.catID && len(lib.TopFive(p.catID)) < 3 {
				lib.AddTopFive(e.ID)
			}
		}
	case "fav-podcasts":
		pick(3)
	case "fav-rank":
		for _, e := range lib.Entries {
			if len(lib.Categories) > 0 && !e.Favorite && e.CategoryID == lib.Categories[0].ID {
				lib.ToggleFavorite(e.ID)
				a.Push(newEntryDetail(e.ID))
				openRankPrompt(a, e.ID)
				break
			}
		}
	}
	return nil
}
