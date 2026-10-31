// Command relic is the Relic app.
package main

import (
	"errors"
	"image"
	"image/color"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/x/explorer"

	"github.com/saniyaa61/relic/photos"
	"github.com/saniyaa61/relic/share"
	"github.com/saniyaa61/relic/store"
	"github.com/saniyaa61/relic/tz"
	"github.com/saniyaa61/relic/ui"
)

func main() {
	go func() {
		w := new(app.Window)
		w.Option(app.Title("Relic"), app.Size(400, 800))
		if err := run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

// dataDir is the app's folder: the database and the posters folder.
func dataDir() (string, error) {
	dir, err := app.DataDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "Relic")
	return dir, os.MkdirAll(dir, 0o700)
}

// openStore opens the app's database and counts this launch. The count is
// logged so the CI emulator run can confirm the data survived a restart.
func openStore() (*store.DB, error) {
	dir, err := dataDir()
	if err != nil {
		return nil, err
	}
	db, err := store.Open(filepath.Join(dir, "relic.db"))
	if err != nil {
		return nil, err
	}
	n := 0
	if v, ok, err := db.Meta("launches"); err != nil {
		db.Close()
		return nil, err
	} else if ok {
		n, _ = strconv.Atoi(v)
	}
	n++
	if err := db.SetMeta("launches", strconv.Itoa(n)); err != nil {
		db.Close()
		return nil, err
	}
	log.Printf("relic: store ok, launch %d", n)
	return db, nil
}

// startTour is set in tour builds (tour.go) to step through every screen.
var startTour func(a *ui.App, db *store.DB, posters string)

func run(w *app.Window) error {
	// Days, weeks and months follow the phone's time zone; Go doesn't know
	// it on Android (it uses UTC) and not always on iOS.
	if name := tz.SetLocal(); name != "" {
		log.Printf("relic: time zone %s", name)
	}
	db, err := openStore()
	if err != nil {
		return err
	}
	defer db.Close()
	dir, err := dataDir()
	if err != nil {
		return err
	}
	posters := filepath.Join(dir, "posters")
	lib, err := db.Load()
	if err != nil {
		return err
	}
	// The archive starts on first launch (SPEC §2 firstUsedAt).
	if lib.Profile.FirstUsedAt.IsZero() {
		lib.Profile.FirstUsedAt = time.Now().UTC()
		if err := db.Update(func(w store.Writer) error { return w.SaveProfile(lib.Profile) }); err != nil {
			return err
		}
	}
	th, err := ui.NewTheme(ui.LinenLight)
	if err != nil {
		return err
	}
	a := ui.NewApp(th, lib, db)
	a.PosterDir = posters
	a.StartOnboarding()
	expl := explorer.NewExplorer(w)
	a.ChooseFile = func(extensions ...string) (io.ReadCloser, error) {
		rc, err := expl.ChooseFile(extensions...)
		if errors.Is(err, explorer.ErrUserDecline) {
			return nil, ui.ErrNoFile
		}
		return rc, err
	}
	a.CreateFile = func(name string) (io.WriteCloser, error) {
		wc, err := expl.CreateFile(name)
		if errors.Is(err, explorer.ErrUserDecline) {
			return nil, ui.ErrNoFile
		}
		return wc, err
	}
	ph := photos.New(w)
	if photos.Available() {
		a.PickPhoto = func() ([]byte, error) {
			b, err := ph.Pick()
			if errors.Is(err, photos.ErrCancelled) {
				return nil, ui.ErrNoFile
			}
			return b, err
		}
	}
	sh := share.New(w)
	if share.Available() {
		a.ShareImage = sh.Image
	}
	if runtime.GOOS == "ios" {
		a.TopTrim = 13
	}
	a.Invalidate = w.Invalidate
	if startTour != nil {
		startTour(a, db, posters)
	} else {
		a.ShowSplash()
	}
	a.OnWindowColors = func(status, navigation color.NRGBA) {
		w.Option(app.StatusColor(status), app.NavigationColor(navigation))
	}
	var ops op.Ops
	var kb keyboardWatch
	for {
		e := w.Event()
		expl.ListenEvents(e)
		sh.ListenEvents(e)
		ph.ListenEvents(e)
		switch e := e.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			// The whole window, system bars included: app.NewContext would
			// already move and shrink the drawing clear of the bars, and
			// App.Layout keeps clear of them itself (it draws the page and
			// bottom-bar backgrounds under them), so the bars' space was
			// counted twice on phones.
			ops.Reset()
			a.KeyboardOpen = kb.open(e.Size, e.Metric.Dp(e.Insets.Bottom), e.Metric.PxPerDp)
			m, k := fitWidth(runtime.GOOS, e.Metric, e.Size.X)
			gtx := layout.Context{Ops: &ops, Now: e.Now, Source: e.Source, Metric: m,
				Constraints: layout.Exact(e.Size)}
			// The insets are in the system's dp; in the scaled dp they're
			// 1/k as many, so they cover the same pixels.
			safe := layout.Inset{Top: e.Insets.Top / unit.Dp(k), Bottom: e.Insets.Bottom / unit.Dp(k),
				Left: e.Insets.Left / unit.Dp(k), Right: e.Insets.Right / unit.Dp(k)}
			a.Layout(gtx, safe)
			e.Frame(gtx.Ops)
		}
	}
}

// designWidth is the phone width, in dp, the layout is drawn for (a
// standard ~390pt iPhone; the prototype's own column is up to 400px).
const designWidth = 390

// fitWidth scales a phone's dp and sp so the layout fills its width as it
// would a designWidth phone: wider phones zoom in, narrower ones zoom out
// (owner's choice, 2026-10-31). The scale stays within 0.85–1.2 (tablets
// and folds don't blow up), and desktops keep their own size. It returns
// the metric and the scale.
func fitWidth(goos string, m unit.Metric, widthPx int) (unit.Metric, float32) {
	if goos != "android" && goos != "ios" || m.PxPerDp <= 0 || widthPx <= 0 {
		return m, 1
	}
	k := float32(widthPx) / m.PxPerDp / designWidth
	k = max(0.85, min(1.2, k))
	m.PxPerDp *= k
	m.PxPerSp *= k
	return m, k
}

// keyboardWatch tells when the on-screen keyboard is up. Gio doesn't say,
// but the keyboard takes its height from the bottom of the window: iOS
// shrinks the window, and Android either shrinks it or reports the
// keyboard as bottom inset. Either way, the space left at the bottom
// shrinks by far more than any system bar is tall.
type keyboardWatch struct {
	width, tallest int // the tallest free height seen at this width
}

// open takes a frame's size and bottom inset (px) and reports whether the
// keyboard is up.
func (k *keyboardWatch) open(size image.Point, bottomInset int, pxPerDp float32) bool {
	if size.X != k.width { // rotated or resized: start again
		k.width, k.tallest = size.X, 0
	}
	free := size.Y - bottomInset
	if free > k.tallest {
		k.tallest = free
	}
	return float32(k.tallest-free) > 120*pxPerDp
}
