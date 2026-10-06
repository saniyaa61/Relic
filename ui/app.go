package ui

import (
	"errors"
	"image"
	"image/color"
	"io"
	"log"
	"sync"
	"time"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/saniyaa61/relic/core"
	"github.com/saniyaa61/relic/store"
)

// Tab is one of the five bottom-navigation destinations (SPEC §5).
type Tab int

const (
	TabHome Tab = iota
	TabLibrary
	TabNew
	TabFavorites
	TabDigest
	tabCount
)

// Screen is one full page: either a tab's root page or a sub-page pushed
// on top of it. Sub-pages hide the bottom bar.
type Screen interface {
	Layout(gtx layout.Context, a *App) layout.Dimensions
}

// BackHandler is a screen that can use the back button itself, such as a
// page with an open search (SPEC §4.9).
type BackHandler interface {
	// WantsBack reports whether the page has something of its own to close.
	WantsBack() bool
	// Back closes it.
	Back(a *App)
}

// App is the whole Relic window: the current tab, any sub-pages, the open
// dialog and the toast. It owns the library and saves through the store.
type App struct {
	Theme *Theme
	Lib   *core.Library
	// DB is where changes are saved; nil (snapshots, tests) saves nothing.
	DB *store.DB
	// Now and Loc are the clock and the user's timezone, replaceable for
	// snapshots and tests.
	Now func() time.Time
	Loc *time.Location
	// ChooseImage, if set, asks the system for a picture file (the
	// platform's file picker; it blocks until the user chooses or
	// cancels). Invalidate asks for a new frame from another goroutine.
	ChooseImage func() (io.ReadCloser, error)
	Invalidate  func()
	// PosterDir is the folder of poster JPEGs beside the database; ""
	// means posters aren't shown.
	PosterDir string
	// OnWindowColors, if set, is called when the system bar colours should
	// change (theme switch, or the bottom bar shown or hidden).
	OnWindowColors func(status, navigation color.NRGBA)

	tab     Tab
	roots   [tabCount]Screen
	stack   []Screen
	shownAt time.Time // when the current page appeared, for its fade-in
	nav     navBar
	dialog  *Dialog
	sheet   *Sheet
	toast   toast

	lastStatus, lastNav color.NRGBA
	posters             posterCache

	timeIdx *core.TimeIndex // built once per frame, see timeIndex

	asyncMu sync.Mutex
	async   []func(*App)
}

// NewApp starts on Home with the theme the profile asks for.
func NewApp(th *Theme, lib *core.Library, db *store.DB) *App {
	a := &App{Theme: th, Lib: lib, DB: db, Now: time.Now, Loc: time.Local}
	th.Palette = PaletteFor(lib.Profile)
	a.roots = [tabCount]Screen{
		TabHome:      &homePage{},
		TabLibrary:   &libraryRoot{},
		TabNew:       newEntryPage(),
		TabFavorites: &placeholder{eyebrow: "Favorites", title: "The ones you love", step: 5},
		TabDigest:    &placeholder{eyebrow: "Digest", title: "Your story so far", step: 6},
	}
	return a
}

// Tab returns the current tab.
func (a *App) Tab() Tab { return a.tab }

// Go switches to a tab's root page, closing any sub-pages.
func (a *App) Go(t Tab) {
	if t == TabNew {
		// Opening New always starts from an empty form (SPEC §5), even
		// when New is already open.
		a.roots[TabNew] = newEntryPage()
	} else if t == a.tab && len(a.stack) == 0 {
		return
	}
	a.tab = t
	a.stack = nil
	a.shownAt = a.Now()
}

// Push opens a sub-page on top of the current page.
func (a *App) Push(s Screen) {
	a.stack = append(a.stack, s)
	a.shownAt = a.Now()
}

// Pop closes the top sub-page.
func (a *App) Pop() {
	if len(a.stack) > 0 {
		a.stack = a.stack[:len(a.stack)-1]
		a.shownAt = a.Now()
	}
}

// top is the screen being shown.
func (a *App) top() Screen {
	if n := len(a.stack); n > 0 {
		return a.stack[n-1]
	}
	return a.roots[a.tab]
}

// Back does what the system back button does, in this order: close the
// dialog; let the page close its own search; close the sub-page; return
// from another tab to Home. On Home with nothing open it returns false,
// and the system handles back (Android leaves the app).
func (a *App) Back() bool {
	if a.dialog != nil && !a.dialog.closing() {
		a.dialog.close(a)
		return true
	}
	if a.sheet != nil && !a.sheet.closing() {
		a.CloseSheet()
		return true
	}
	if b, ok := a.top().(BackHandler); ok && b.WantsBack() {
		b.Back(a)
		return true
	}
	if len(a.stack) > 0 {
		a.Pop()
		return true
	}
	if a.tab != TabHome {
		a.Go(TabHome)
		return true
	}
	return false
}

// canGoBack reports whether Back would do anything, without doing it.
func (a *App) canGoBack() bool {
	if a.dialog != nil || a.sheet != nil || len(a.stack) > 0 || a.tab != TabHome {
		return true
	}
	b, ok := a.top().(BackHandler)
	return ok && b.WantsBack()
}

// Update runs fn in one store transaction, if there is a store. A failed
// save is logged and shown as an error toast.
func (a *App) Update(fn func(w store.Writer) error) bool {
	if a.DB == nil {
		return true
	}
	if err := a.DB.Update(fn); err != nil {
		log.Printf("relic: save failed: %v", err)
		a.ToastError("Couldn't save that. Please try again.")
		return false
	}
	return true
}

// save stores the given parts of the library in one transaction:
// categories (which also deletes entries of removed categories), the
// listed entries, and favourites. If saving fails, the library is read
// back from the store so the screen never shows unsaved changes.
func (a *App) save(categories bool, entries []*core.Entry, favourites bool) bool {
	ok := a.Update(func(w store.Writer) error {
		if categories {
			if err := w.SaveCategories(a.Lib.Categories); err != nil {
				return err
			}
		}
		for _, e := range entries {
			if err := w.SaveEntry(e); err != nil {
				return err
			}
		}
		if favourites {
			return w.SaveFavourites(a.Lib.Favourites)
		}
		return nil
	})
	if !ok && a.DB != nil {
		if lib, err := a.DB.Load(); err == nil {
			a.Lib = lib
		}
	}
	return ok
}

// removeUnusedPosters deletes poster files no entry uses any more.
func (a *App) removeUnusedPosters() {
	if a.PosterDir == "" || a.DB == nil {
		return
	}
	if err := store.RemoveUnusedPosters(a.PosterDir, a.Lib); err != nil {
		log.Printf("relic: tidying posters: %v", err)
	}
}

// background runs work off the UI goroutine (a file picker, say), then
// runs the function it returns on the next frame.
func (a *App) background(work func() func(*App)) {
	go func() {
		done := work()
		if done == nil {
			return
		}
		a.asyncMu.Lock()
		a.async = append(a.async, done)
		a.asyncMu.Unlock()
		if a.Invalidate != nil {
			a.Invalidate()
		}
	}()
}

// runAsync runs results delivered by background, on the UI goroutine.
func (a *App) runAsync() {
	a.asyncMu.Lock()
	jobs := a.async
	a.async = nil
	a.asyncMu.Unlock()
	for _, j := range jobs {
		j(a)
	}
}

// ErrNoPicture is what ChooseImage returns when the user cancels.
var ErrNoPicture = errors.New("no picture chosen")

// pickPoster lets the user choose a picture, resizes it to a stored
// poster file and calls done with its file name.
func (a *App) pickPoster(done func(name string)) {
	if a.ChooseImage == nil || a.PosterDir == "" {
		a.ToastError("Choosing a picture isn't available here yet.")
		return
	}
	dir := a.PosterDir
	a.background(func() func(*App) {
		rc, err := a.ChooseImage()
		if errors.Is(err, ErrNoPicture) {
			return nil
		}
		fail := func(err error) func(*App) {
			log.Printf("relic: poster: %v", err)
			return func(a *App) { a.ToastError("That picture couldn't be opened.") }
		}
		if err != nil {
			return fail(err)
		}
		defer rc.Close()
		raw, err := io.ReadAll(io.LimitReader(rc, 40<<20))
		if err != nil {
			return fail(err)
		}
		jpg, err := store.ResizePoster(raw)
		if err != nil {
			return fail(err)
		}
		name := core.NewID() + ".jpg"
		if err := store.WritePosters(dir, map[string][]byte{name: jpg}); err != nil {
			return fail(err)
		}
		return func(a *App) { done(name) }
	})
}

// timeIndex is every entry's time events, computed once per frame and
// shared by everything that shows time (CLAUDE.md: compute time events
// once per render).
func (a *App) timeIndex() *core.TimeIndex {
	if a.timeIdx == nil {
		a.timeIdx = core.NewTimeIndex(a.Lib.Entries)
	}
	return a.timeIdx
}

// navShown reports whether the bottom bar shows: on the tab pages, and on
// sub-pages that keep it (the Library's category and folder pages, as in
// the prototype).
func (a *App) navShown() bool {
	if len(a.stack) == 0 {
		return true
	}
	s, ok := a.top().(interface{ ShowsNav() bool })
	return ok && s.ShowsNav()
}

// SetTheme switches theme and light/dark mode, saving them to the profile
// so they're remembered next launch. base and accent are only used for the
// custom theme.
func (a *App) SetTheme(theme, mode, base, accent string) {
	p := a.Lib.Profile
	p.Theme, p.Mode = theme, mode
	if theme == "custom" {
		p.CustomBase, p.CustomAccent = base, accent
	}
	a.Lib.Profile = p
	a.Theme.Palette = PaletteFor(p)
	a.Update(func(w store.Writer) error { return w.SaveProfile(p) })
}

// Layout draws the whole window. safe is the system-bar inset.
func (a *App) Layout(gtx layout.Context, safe layout.Inset) layout.Dimensions {
	if a.shownAt.IsZero() {
		a.shownAt = a.Now()
	}
	a.runAsync()
	a.timeIdx = nil
	a.handleBack(gtx)
	th := a.Theme
	paint.Fill(gtx.Ops, th.Bg)
	size := gtx.Constraints.Max
	showNav := a.navShown()

	// The bottom bar takes its height from the bottom of the window; the
	// page gets the rest.
	pageH := size.Y
	if showNav {
		rec := op.Record(gtx.Ops)
		ngtx := gtx
		ngtx.Constraints = layout.Constraints{Min: image.Pt(size.X, 0), Max: size}
		nd := a.nav.Layout(ngtx, a, gtx.Dp(safe.Bottom))
		call := rec.Stop()
		pageH = size.Y - nd.Size.Y
		st := op.Offset(image.Pt(0, pageH)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		st.Pop()
	}
	pgtx := gtx
	pgtx.Constraints = layout.Exact(image.Pt(size.X, pageH))
	cl := clip.Rect{Max: pgtx.Constraints.Max}.Push(gtx.Ops)
	pageSafe := safe
	if showNav {
		pageSafe.Bottom = 0
	}
	a.layoutPage(pgtx, pageSafe)
	cl.Pop()

	if a.sheet != nil {
		if a.sheet.Layout(gtx, a, gtx.Dp(safe.Bottom)) {
			a.sheet = nil
		}
	}
	if a.dialog != nil {
		if a.dialog.Layout(gtx, a) {
			a.dialog = nil
		}
	}
	a.toast.Layout(gtx, a, gtx.Dp(safe.Top))

	navColor := th.Bg
	if showNav {
		navColor = th.Surface
	}
	if a.OnWindowColors != nil && (th.Bg != a.lastStatus || navColor != a.lastNav) {
		a.lastStatus, a.lastNav = th.Bg, navColor
		a.OnWindowColors(th.Bg, navColor)
	}
	return layout.Dimensions{Size: size}
}

// layoutPage draws the current screen, fading it up into place when it
// appears (the prototype's .page animation: 0.26s, 6px rise).
func (a *App) layoutPage(gtx layout.Context, safe layout.Inset) {
	t := easeCSS(progress(gtx, a.Now(), a.shownAt, 260*time.Millisecond))
	defer paint.PushOpacity(gtx.Ops, t).Pop()
	defer op.Offset(image.Pt(0, roundi(float32(gtx.Dp(6))*(1-t)))).Push(gtx.Ops).Pop()
	safe.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return a.top().Layout(gtx, a)
	})
}

// handleBack takes the Android back button, and Escape on desktop, but
// only while back has something to do; otherwise the system gets it.
func (a *App) handleBack(gtx layout.Context) {
	if !a.canGoBack() {
		return
	}
	for {
		e, ok := gtx.Event(key.Filter{Name: key.NameBack}, key.Filter{Name: key.NameEscape})
		if !ok {
			return
		}
		if e, ok := e.(key.Event); ok && e.State == key.Press {
			a.Back()
		}
	}
}

// ShowDialog opens d over the current page.
func (a *App) ShowDialog(d *Dialog) {
	d.openedAt = a.Now()
	if d.Type != nil {
		d.picker.value = *d.Type
	}
	a.dialog = d
}

// Toast shows a short confirmation at the top of the screen.
func (a *App) Toast(msg string) { a.toast.show(a.Now(), msg, false) }

// ToastError shows a short error at the top of the screen.
func (a *App) ToastError(msg string) { a.toast.show(a.Now(), msg, true) }

// gutter is the prototype's 18px page margin.
const gutter unit.Dp = 18
