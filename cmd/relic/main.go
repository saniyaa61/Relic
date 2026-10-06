// Command relic is the Relic app.
package main

import (
	"errors"
	"flag"
	"fmt"
	"image/color"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/x/explorer"

	"github.com/saniyaa61/relic/importer"
	"github.com/saniyaa61/relic/store"
	"github.com/saniyaa61/relic/ui"
)

// importPath is a temporary, desktop-only way to load the prototype
// archive until Settings has Import (Phase 3 step 9):
//
//	go run ./cmd/relic -import C:/Users/saniy/Downloads/relic-archive.json
//
// It replaces everything in the app's database, keeping the light/dark mode.
var importPath = flag.String("import", "", "replace the app's data with this prototype archive (relic-archive.json)")

func main() {
	flag.Parse()
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

func run(w *app.Window) error {
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
	if *importPath != "" {
		if err := importArchive(db, *importPath, posters); err != nil {
			return fmt.Errorf("import %s: %w", *importPath, err)
		}
	}
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
	expl := explorer.NewExplorer(w)
	a.ChooseImage = func() (io.ReadCloser, error) {
		rc, err := expl.ChooseFile(".jpg", ".jpeg", ".png", ".webp", ".gif")
		if errors.Is(err, explorer.ErrUserDecline) {
			return nil, ui.ErrNoPicture
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
	a.Invalidate = w.Invalidate
	a.OnWindowColors = func(status, navigation color.NRGBA) {
		w.Option(app.StatusColor(status), app.NavigationColor(navigation))
	}
	var ops op.Ops
	for {
		e := w.Event()
		expl.ListenEvents(e)
		switch e := e.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			safe := layout.Inset{Top: e.Insets.Top, Bottom: e.Insets.Bottom, Left: e.Insets.Left, Right: e.Insets.Right}
			a.Layout(gtx, safe)
			e.Frame(gtx.Ops)
		}
	}
}

// importArchive replaces the database's contents with a prototype archive,
// keeping the current light/dark mode (owner's decision) and writing the
// posters beside the database.
func importArchive(db *store.DB, path, posters string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	res, err := importer.Read(f, time.Now())
	if err != nil {
		return err
	}
	current, err := db.Load()
	if err != nil {
		return err
	}
	res.KeepMode(current.Profile)
	if err := store.WritePosters(posters, res.Posters); err != nil {
		return err
	}
	if err := db.ReplaceAll(res.Library); err != nil {
		return err
	}
	if err := store.RemoveUnusedPosters(posters, res.Library); err != nil {
		return err
	}
	r := res.Report
	log.Printf("relic: imported %d categories, %d entries, %d posters from %s", r.Categories, r.Entries, r.Posters, path)
	for _, w := range r.Warnings {
		log.Printf("relic: import note: %s", w)
	}
	return nil
}
