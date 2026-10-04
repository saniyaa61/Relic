// Command relic is the Relic app.
package main

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"

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

func run(w *app.Window) error {
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
