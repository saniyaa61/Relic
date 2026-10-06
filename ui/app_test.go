package ui

import (
	"image"
	"image/color"
	"path/filepath"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/saniyaa61/relic/core"
	"github.com/saniyaa61/relic/store"
)

// testApp returns an app on a fake clock that tests move with advance.
func testApp(t *testing.T, db *store.DB) (*App, func(time.Duration)) {
	t.Helper()
	th, err := NewTheme(LinenLight)
	if err != nil {
		t.Fatal(err)
	}
	lib := &core.Library{}
	if db != nil {
		if lib, err = db.Load(); err != nil {
			t.Fatal(err)
		}
	}
	a := NewApp(th, lib, db)
	now := time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC)
	a.Now = func() time.Time { return now }
	return a, func(d time.Duration) { now = now.Add(d) }
}

// frame lays the app out once in a phone-sized window.
func frame(a *App) {
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}, Constraints: layout.Exact(image.Pt(800, 1600))}
	a.Layout(gtx, layout.Inset{})
}

func TestBackOrder(t *testing.T) {
	a, advance := testApp(t, nil)
	frame(a)
	if a.canGoBack() || a.Back() {
		t.Fatal("Home with nothing open should leave back to the system")
	}

	// Another tab: back returns to Home.
	a.Go(TabDigest)
	if !a.canGoBack() || !a.Back() || a.Tab() != TabHome {
		t.Fatalf("back from Digest: tab = %v, want Home", a.Tab())
	}

	// A sub-page with its search open: the first back closes the search
	// and stays on the page (SPEC §4.9), the second leaves the page.
	p := newPosterPage(pageAllEntries)
	a.Push(p)
	p.search.Toggle(a)
	p.search.input.Editor.SetText("dune")
	if p.search.Query() != "dune" {
		t.Fatalf("query = %q", p.search.Query())
	}
	a.Back()
	if p.search.IsOpen() || len(a.stack) != 1 {
		t.Fatalf("first back: search open %v, %d sub-pages; want closed search, still on the page", p.search.IsOpen(), len(a.stack))
	}
	if p.search.Query() != "" || p.search.input.Editor.Text() != "" {
		t.Error("closing search should clear the query")
	}
	a.Back()
	if len(a.stack) != 0 || a.Tab() != TabHome {
		t.Fatalf("second back: %d sub-pages, tab %v", len(a.stack), a.Tab())
	}

	// A dialog closes before anything else.
	a.Push(newPosterPage(pageAllEntries))
	a.ShowDialog(ConfirmDialog("Delete this entry?", "", "", "", func(*App) {}))
	a.Back()
	if len(a.stack) != 1 {
		t.Fatal("back with a dialog open should close only the dialog")
	}
	advance(300 * time.Millisecond)
	frame(a)
	if a.dialog != nil {
		t.Fatal("dialog should be gone once its closing animation ends")
	}
	a.Back()
	if len(a.stack) != 0 {
		t.Fatal("back after the dialog should leave the sub-page")
	}
}

func TestGoClearsSubPages(t *testing.T) {
	a, _ := testApp(t, nil)
	a.Push(newPosterPage(pageAllEntries))
	a.Go(TabLibrary)
	if len(a.stack) != 0 || a.Tab() != TabLibrary {
		t.Fatalf("Go(Library): %d sub-pages, tab %v", len(a.stack), a.Tab())
	}
	// Opening New always starts from a fresh page (SPEC §5).
	a.Go(TabNew)
	first := a.roots[TabNew]
	a.Go(TabHome)
	a.Go(TabNew)
	if a.roots[TabNew] == first {
		t.Error("opening New should start from an empty form")
	}
	second := a.roots[TabNew]
	a.Go(TabNew)
	if a.roots[TabNew] == second {
		t.Error("tapping New while on it should also start afresh")
	}
}

func TestPromptDialog(t *testing.T) {
	a, advance := testApp(t, nil)
	var got []string
	d := PromptDialog("New category", "", "Category name", "e.g. Films", "", "Create",
		func(a *App, v string) { got = append(got, v) })
	a.ShowDialog(d)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops}

	// An empty or blank name doesn't confirm.
	d.input.Editor.SetText("   ")
	d.submit(gtx, a)
	if len(got) != 0 || d.closing() {
		t.Fatalf("blank name confirmed: %v", got)
	}
	d.input.Editor.SetText("  Comfort Rewatches ")
	d.submit(gtx, a)
	if len(got) != 1 || got[0] != "Comfort Rewatches" || !d.closing() {
		t.Fatalf("got %q, closing %v; want the trimmed name and the dialog closing", got, d.closing())
	}
	advance(time.Second)
	frame(a)
	if a.dialog != nil {
		t.Error("dialog still open")
	}
}

func TestConfirmDialogDefaults(t *testing.T) {
	d := ConfirmDialog("Delete \"Films\"?", "", "", "", func(*App) {})
	if d.Cancel != "Keep it" || d.Confirm != "Delete" || !d.Danger || d.Label != "" {
		t.Errorf("confirm dialog = %+v", d)
	}
	p := PromptDialog("Rename", "", "", "", "Films", "", nil)
	if p.Label != "Name" || p.Confirm != "Save" || p.Cancel != "Cancel" || p.input.Editor.Text() != "Films" {
		t.Errorf("prompt dialog = %+v", p)
	}
}

func TestToastTiming(t *testing.T) {
	var tt toast
	t0 := time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC)
	tt.show(t0, "Session logged ✦", false)
	if !tt.visible(t0.Add(2400 * time.Millisecond)) {
		t.Error("toast should still be fading out at 2.4s")
	}
	if tt.visible(t0.Add(2500 * time.Millisecond)) {
		t.Error("toast should be gone after 2.45s")
	}
	// A second toast while one is showing replaces it and restarts the
	// timer, without fading in again.
	t1 := t0.Add(2 * time.Second)
	tt.show(t1, "Entry deleted", false)
	if tt.msg != "Entry deleted" || !tt.visible(t1.Add(2*time.Second)) {
		t.Error("second toast should restart the timer")
	}
	if got := t1.Sub(tt.shownAt); got != toastFade {
		t.Errorf("second toast should skip its fade-in; shownAt is %v before now", got)
	}
}

// The theme and light/dark mode are saved, so they are remembered next
// launch.
func TestThemeIsRemembered(t *testing.T) {
	path := filepath.Join(t.TempDir(), "relic.db")
	db, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := testApp(t, db)
	a.SetTheme("forest", "dark", "", "")
	if a.Theme.Palette != PaletteFor(core.Profile{Theme: "forest", Mode: "dark"}) {
		t.Error("palette didn't switch")
	}
	db.Close()

	db, err = store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	a2, _ := testApp(t, db)
	if p := a2.Lib.Profile; p.Theme != "forest" || p.Mode != "dark" {
		t.Errorf("after restart profile = %+v, want forest dark", p)
	}
	if a2.Theme.Palette != PaletteFor(core.Profile{Theme: "forest", Mode: "dark"}) {
		t.Error("app didn't start in the saved theme")
	}
}

func TestWindowColorsFollowTheme(t *testing.T) {
	a, _ := testApp(t, nil)
	var status, nav []string
	a.OnWindowColors = func(s, n color.NRGBA) { status = append(status, hex(s)); nav = append(nav, hex(n)) }
	frame(a)
	frame(a) // unchanged: no second call
	a.Push(newPosterPage(pageAllEntries))
	frame(a)
	a.SetTheme("linen", "dark", "", "")
	frame(a)
	want := []string{"#F7F3EC", "#F7F3EC", "#18140D"}
	wantNav := []string{"#FDFAF5", "#F7F3EC", "#18140D"}
	if len(status) != len(want) {
		t.Fatalf("status colours %v, want %v", status, want)
	}
	for i := range want {
		if status[i] != want[i] || nav[i] != wantNav[i] {
			t.Errorf("call %d: status %s nav %s, want %s %s", i, status[i], nav[i], want[i], wantNav[i])
		}
	}
}

// Taps on the dialog card's empty space must not close it; taps on the
// dimmed page around it must.
func TestDialogTaps(t *testing.T) {
	a, advance := testApp(t, nil)
	a.ShowDialog(ConfirmDialog("Delete this entry?", "This can't be undone.", "", "", func(*App) {}))
	var r input.Router
	run := func() {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2},
			Constraints: layout.Exact(image.Pt(800, 1600)), Source: r.Source()}
		a.Layout(gtx, layout.Inset{})
		r.Frame(gtx.Ops)
	}
	tap := func(x, y float32) {
		r.Queue(
			pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Position: f32.Pt(x, y)},
			pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: f32.Pt(x, y)},
		)
		run()
	}
	advance(time.Second)
	run()
	tap(70, 800) // inside the card's left padding (card starts at x=58)
	if a.dialog == nil || a.dialog.closing() {
		t.Fatal("a tap on the card itself closed the dialog")
	}
	tap(20, 100) // the scrim
	if a.dialog == nil || !a.dialog.closing() {
		t.Fatal("a tap on the dimmed page should close the dialog")
	}
}

// The Android back key arrives as a key event; Gio tells Android whether
// the app took it. On Home with nothing open the app must not take it, so
// Android does its usual thing (leaves the app).
//
// Gio also reports "handled" while an animation has asked for the next
// frame (for the 0.26s page fade, say), so each check lets animations
// finish first, as they would on a phone between presses.
func TestSystemBackKey(t *testing.T) {
	a, advance := testApp(t, nil)
	var r input.Router
	run := func() {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2},
			Constraints: layout.Exact(image.Pt(800, 1600)), Source: r.Source()}
		a.Layout(gtx, layout.Inset{})
		r.Frame(gtx.Ops)
	}
	back := func() bool {
		advance(time.Second)
		run()
		r.WakeupTime() // clear the animation's last wakeup
		// The same check the Android window makes to answer onBack.
		r.Queue(key.Event{Name: key.NameBack})
		_, handled := r.WakeupTime()
		run()
		return handled
	}
	run()
	if back() {
		t.Error("the app took back on Home with nothing open")
	}
	a.Go(TabFavorites)
	run()
	if !back() || a.Tab() != TabHome {
		t.Errorf("back on Favorites: tab %v, want Home", a.Tab())
	}
	run()
	if back() {
		t.Error("after returning Home, back should go to the system again")
	}
	// Escape does the same on desktop.
	a.Push(newPosterPage(pageAllEntries))
	run()
	r.Queue(key.Event{Name: key.NameEscape})
	run()
	if len(a.stack) != 0 {
		t.Error("Escape should close the sub-page")
	}
}
