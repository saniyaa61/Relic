// Command snapshot renders a Relic screen to a PNG without opening a window,
// for comparing against the prototype side by side.
//
//	go run ./cmd/snapshot -o spike.png -scale 2
package main

import (
	"flag"
	"image"
	"image/png"
	"log"
	"os"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/saniyaa61/relic/ui"
)

func main() {
	out := flag.String("o", "snapshot.png", "output PNG file")
	scale := flag.Float64("scale", 2, "pixels per dp")
	w := flag.Int("w", 400, "width in dp")
	h := flag.Int("h", 420, "height in dp")
	dark := flag.Bool("dark", false, "use Linen dark")
	flag.Parse()

	p := ui.LinenLight
	if *dark {
		p = ui.LinenDark
	}
	th, err := ui.NewTheme(p)
	if err != nil {
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
	ui.Spike(gtx, th, layout.Inset{})
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
