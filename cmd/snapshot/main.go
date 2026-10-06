// Command snapshot renders a Relic screen to a PNG without opening a window,
// for comparing against the prototype side by side.
//
//	go run ./cmd/snapshot -o build/home.png -screen home -theme linen -dark
package main

import (
	"flag"
	"image"
	"image/png"
	"log"
	"os"
	"strings"
	"time"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/saniyaa61/relic/core"
	"github.com/saniyaa61/relic/importer"
	"github.com/saniyaa61/relic/store"
	"github.com/saniyaa61/relic/ui"
)

func main() {
	out := flag.String("o", "snapshot.png", "output PNG file")
	scale := flag.Float64("scale", 2, "pixels per dp")
	w := flag.Int("w", 400, "width in dp")
	h := flag.Int("h", 800, "height in dp")
	screen := flag.String("screen", "home", "what to show: "+strings.Join(ui.SnapshotScreens, ", "))
	theme := flag.String("theme", "linen", "theme: "+strings.Join(ui.ThemeNames, ", ")+", custom")
	dark := flag.Bool("dark", false, "dark mode")
	base := flag.String("base", "#7A5C3A", "custom theme base colour")
	accent := flag.String("accent", "#C4956A", "custom theme accent colour")
	top := flag.Float64("top", 0, "status bar height in dp, to check the safe area")
	bottom := flag.Float64("bottom", 0, "system navigation bar height in dp")
	archive := flag.String("archive", "", "prototype archive to show (e.g. importer/testdata/scrubbed-archive.json)")
	flag.Parse()

	mode := "light"
	if *dark {
		mode = "dark"
	}
	lib := &core.Library{}
	posterDir := ""
	if *archive != "" {
		lib, posterDir = loadArchive(*archive)
		defer os.RemoveAll(posterDir)
	}
	lib.Profile.Theme, lib.Profile.Mode = *theme, mode
	lib.Profile.CustomBase, lib.Profile.CustomAccent = *base, *accent
	th, err := ui.NewTheme(ui.LinenLight)
	if err != nil {
		log.Fatal(err)
	}
	a := ui.NewApp(th, lib, nil)
	a.PosterDir = posterDir
	if err := a.ShowForSnapshot(*screen); err != nil {
		log.Fatal(err)
	}

	s := float32(*scale)
	size := image.Pt(int(float32(*w)*s), int(float32(*h)*s))
	win, err := headless.NewWindow(size.X, size.Y)
	if err != nil {
		log.Fatal(err)
	}
	var ops op.Ops
	gtx := layout.Context{
		Ops:         &ops,
		Metric:      unit.Metric{PxPerDp: s, PxPerSp: s},
		Constraints: layout.Exact(size),
	}
	a.Layout(gtx, layout.Inset{Top: unit.Dp(*top), Bottom: unit.Dp(*bottom)})
	if err := win.Frame(&ops); err != nil {
		log.Fatal(err)
	}
	img := image.NewRGBA(image.Rectangle{Max: size})
	if err := win.Screenshot(img); err != nil {
		log.Fatal(err)
	}
	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
}

// loadArchive imports a prototype archive in memory, with its posters in
// a temporary folder.
func loadArchive(path string) (*core.Library, string) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	res, err := importer.Read(f, time.Now())
	if err != nil {
		log.Fatal(err)
	}
	dir, err := os.MkdirTemp("", "relic-posters-")
	if err != nil {
		log.Fatal(err)
	}
	if err := store.WritePosters(dir, res.Posters); err != nil {
		log.Fatal(err)
	}
	return res.Library, dir
}
