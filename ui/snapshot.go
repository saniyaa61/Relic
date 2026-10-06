package ui

import (
	"fmt"
	"time"
)

// SnapshotScreens lists the states cmd/snapshot can render.
var SnapshotScreens = []string{"home", "library", "new", "favorites", "digest", "subpage", "search", "prompt", "confirm", "toast", "toast-error"}

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
	case "favorites":
		a.Go(TabFavorites)
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
	default:
		return fmt.Errorf("unknown screen %q; want one of %v", screen, SnapshotScreens)
	}
	return nil
}
