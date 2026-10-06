package ui

// The colour dialog for the Custom palette, like the browser's own colour
// picker the prototype used: a shade square (saturation across, brightness
// down), a hue slider, and the hex code. Owner's choice, 2026-10-06.

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// hsv is a colour as hue (0–360), saturation and value (0–1).
type hsv struct{ h, s, v float64 }

func toHSV(c color.NRGBA) hsv {
	r, g, b := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	d := mx - mn
	var h float64
	switch {
	case d == 0:
		h = 0
	case mx == r:
		h = 60 * math.Mod((g-b)/d, 6)
	case mx == g:
		h = 60 * ((b-r)/d + 2)
	default:
		h = 60 * ((r-g)/d + 4)
	}
	if h < 0 {
		h += 360
	}
	s := 0.0
	if mx > 0 {
		s = d / mx
	}
	return hsv{h, s, mx}
}

func (c hsv) rgb() color.NRGBA {
	h := math.Mod(c.h, 360) / 60
	x := c.v * c.s * (1 - math.Abs(math.Mod(h, 2)-1))
	ch := c.v * c.s
	var r, g, b float64
	switch int(h) {
	case 0:
		r, g, b = ch, x, 0
	case 1:
		r, g, b = x, ch, 0
	case 2:
		r, g, b = 0, ch, x
	case 3:
		r, g, b = 0, x, ch
	case 4:
		r, g, b = x, 0, ch
	default:
		r, g, b = ch, 0, x
	}
	m := c.v - ch
	f := func(v float64) uint8 { return uint8(math.Round((v + m) * 255)) }
	return color.NRGBA{R: f(r), G: f(g), B: f(b), A: 0xFF}
}

func hexOf(c color.NRGBA) string { return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B) }

// colourPicker holds the dialog's colour while it's being chosen.
type colourPicker struct {
	col             hsv
	hex             Input
	lastHex         string
	square, hueBar  int // pointer tags
	squareImg       paint.ImageOp
	squareFor       hsv // hue and size the square was drawn for
	squareSize      image.Point
	hueImg          paint.ImageOp
	hueSize         image.Point
	dragSq, dragHue bool
}

func newColourPicker(hex string) *colourPicker {
	c, ok := ParseHex(hex)
	if !ok {
		c, _ = ParseHex(defaultCustomBase)
	}
	p := &colourPicker{col: toHSV(c)}
	p.setHex()
	return p
}

func (p *colourPicker) value() string { return hexOf(p.col.rgb()) }

func (p *colourPicker) setHex() {
	p.lastHex = p.value()
	p.hex.Editor.SetText(p.lastHex)
}

// Layout draws the square, the preview dot with the hue slider, and the
// hex field.
func (p *colourPicker) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	// Typing a valid code moves the picker to it.
	if t := strings.ToUpper(strings.TrimSpace(p.hex.Editor.Text())); t != p.lastHex {
		p.lastHex = t
		if !strings.HasPrefix(t, "#") {
			t = "#" + t
		}
		if c, ok := ParseHex(t); ok && len(t) == 7 {
			p.col = toHSV(c)
		}
	}
	w := gtx.Constraints.Max.X
	sq := image.Pt(w, gtx.Dp(150))
	p.handleSquare(gtx, sq)
	hueW := w - gtx.Dp(24+12)
	p.handleHue(gtx, hueW)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return p.drawSquare(gtx, sq) }),
		layout.Rigid(layout.Spacer{Height: 14}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					d := gtx.Dp(24)
					rrect(gtx, image.Pt(d, d), d/2, th.Border)
					st := op.Offset(image.Pt(gtx.Dp(1), gtx.Dp(1))).Push(gtx.Ops)
					rrect(gtx, image.Pt(d-2*gtx.Dp(1), d-2*gtx.Dp(1)), d/2-gtx.Dp(1), p.col.rgb())
					st.Pop()
					return layout.Dimensions{Size: image.Pt(d, d)}
				}),
				layout.Rigid(layout.Spacer{Width: 12}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return p.drawHue(gtx, hueW) }))
		}),
		layout.Rigid(layout.Spacer{Height: 14}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			p.hex.Editor.SingleLine, p.hex.Editor.MaxLen = true, 7
			return p.hex.Layout(gtx, th, InputStyle{Bg: th.Bg, Radius: 11, Size: 14, Center: true,
				Pad: layout.Inset{Top: 10, Bottom: 10, Left: 13, Right: 13}, Placeholder: "#7A5C3A", PlaceholderAlpha: 1})
		}))
}

// pointerPos returns where the pointer is (pressed or dragging) over tag.
func pointerPos(gtx layout.Context, tag event.Tag, dragging *bool) (f32.Point, bool) {
	var pos f32.Point
	moved := false
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: tag, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Kind {
		case pointer.Press:
			*dragging = true
			pos, moved = e.Position, true
		case pointer.Drag:
			if *dragging {
				pos, moved = e.Position, true
			}
		case pointer.Release, pointer.Cancel:
			*dragging = false
		}
	}
	return pos, moved
}

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

func (p *colourPicker) handleSquare(gtx layout.Context, size image.Point) {
	if pos, ok := pointerPos(gtx, &p.square, &p.dragSq); ok {
		p.col.s = clamp01(float64(pos.X) / float64(size.X))
		p.col.v = 1 - clamp01(float64(pos.Y)/float64(size.Y))
		p.setHex()
	}
}

func (p *colourPicker) handleHue(gtx layout.Context, w int) {
	if pos, ok := pointerPos(gtx, &p.hueBar, &p.dragHue); ok {
		p.col.h = clamp01(float64(pos.X)/float64(w)) * 359.999
		p.setHex()
	}
}

// drawSquare is the shade square for the current hue, with a white ring
// where the colour sits.
func (p *colourPicker) drawSquare(gtx layout.Context, size image.Point) layout.Dimensions {
	if p.squareSize != size || p.squareFor.h != p.col.h {
		img := image.NewNRGBA(image.Rectangle{Max: size})
		for y := 0; y < size.Y; y++ {
			v := 1 - float64(y)/float64(max(size.Y-1, 1))
			for x := 0; x < size.X; x++ {
				s := float64(x) / float64(max(size.X-1, 1))
				img.SetNRGBA(x, y, hsv{p.col.h, s, v}.rgb())
			}
		}
		p.squareImg, p.squareSize, p.squareFor = paint.NewImageOp(img), size, p.col
	}
	r := gtx.Dp(10)
	cl := clip.UniformRRect(image.Rectangle{Max: size}, r).Push(gtx.Ops)
	p.squareImg.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	event.Op(gtx.Ops, &p.square)
	cl.Pop()
	at := image.Pt(int(p.col.s*float64(size.X)), int((1-p.col.v)*float64(size.Y)))
	handle(gtx, at, p.col.rgb())
	return layout.Dimensions{Size: size}
}

// drawHue is the hue slider: a rounded rainbow bar with a ring handle.
func (p *colourPicker) drawHue(gtx layout.Context, w int) layout.Dimensions {
	h := gtx.Dp(12)
	size := image.Pt(w, h)
	if p.hueSize != size {
		img := image.NewNRGBA(image.Rectangle{Max: size})
		for x := 0; x < w; x++ {
			c := hsv{float64(x) / float64(max(w-1, 1)) * 360, 1, 1}.rgb()
			for y := 0; y < h; y++ {
				img.SetNRGBA(x, y, c)
			}
		}
		p.hueImg, p.hueSize = paint.NewImageOp(img), size
	}
	// A taller hit area so the thin bar is easy to grab.
	hit := gtx.Dp(12)
	area := clip.Rect(image.Rect(0, -hit, w, h+hit)).Push(gtx.Ops)
	event.Op(gtx.Ops, &p.hueBar)
	area.Pop()
	cl := clip.UniformRRect(image.Rectangle{Max: size}, h/2).Push(gtx.Ops)
	p.hueImg.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	cl.Pop()
	handle(gtx, image.Pt(int(p.col.h/360*float64(w)), h/2), hsv{p.col.h, 1, 1}.rgb())
	return layout.Dimensions{Size: size}
}

// handle is the white ring (with a soft dark edge) marking a choice.
func handle(gtx layout.Context, at image.Point, fill color.NRGBA) {
	r := gtx.Dp(8)
	st := op.Offset(at.Sub(image.Pt(r, r))).Push(gtx.Ops)
	defer st.Pop()
	d := 2 * r
	rrect(gtx, image.Pt(d, d), r, color.NRGBA{A: 0x55})
	b := gtx.Dp(1)
	inner := op.Offset(image.Pt(b, b)).Push(gtx.Ops)
	rrect(gtx, image.Pt(d-2*b, d-2*b), r-b, color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})
	inner.Pop()
	c := gtx.Dp(3)
	core := op.Offset(image.Pt(c, c)).Push(gtx.Ops)
	rrect(gtx, image.Pt(d-2*c, d-2*c), r-c, fill)
	core.Pop()
}

// openColourDialog lets the user pick a colour; "Use colour" hands it back.
func openColourDialog(a *App, title, current string, pick func(string)) {
	p := newColourPicker(current)
	d := &Dialog{Title: title, Cancel: "Cancel", Confirm: "Use colour",
		Body:      p.Layout,
		OnConfirm: func(a *App, _ string) { pick(p.value()) }}
	a.ShowDialog(d)
}
