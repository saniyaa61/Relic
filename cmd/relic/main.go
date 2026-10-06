// Command relic is the Relic app.
package main

import (
	"image/color"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/saniyaa61/relic/store"
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

// openStore opens the app's database and counts this launch. The count is
// logged so the CI emulator run can confirm the data survived a restart.
func openStore() (*store.DB, error) {
	dir, err := app.DataDir()
	if err != nil {
		return nil, err
	}
	dir = filepath.Join(dir, "Relic")
	if err := os.MkdirAll(dir, 0o700); err != nil {
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
	a.OnWindowColors = func(status, navigation color.NRGBA) {
		w.Option(app.StatusColor(status), app.NavigationColor(navigation))
	}
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
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
