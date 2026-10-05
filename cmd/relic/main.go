// Command relic is the Relic app.
package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/saniyaa61/relic/store"
	"github.com/saniyaa61/relic/ui"
)

func main() {
	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("Relic"),
			app.Size(400, 800),
			app.StatusColor(ui.LinenLight.Bg),
			app.NavigationColor(ui.LinenLight.Bg),
		)
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
	th, err := ui.NewTheme(ui.LinenLight)
	if err != nil {
		return err
	}
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			safe := layout.Inset{Top: e.Insets.Top, Bottom: e.Insets.Bottom, Left: e.Insets.Left, Right: e.Insets.Right}
			ui.Spike(gtx, th, safe)
			e.Frame(gtx.Ops)
		}
	}
}
