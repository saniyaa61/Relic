package ui

// The Custom theme in Settings: its tile, and the base / accent colour
// pickers. The prototype used a rainbow-gradient tile and the browser's
// colour picker; neither exists here, so the looks are options for the
// owner (open question in STATUS).

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
)

// The prototype's starting colours for a custom palette.
const (
	defaultCustomBase   = "#7A5C3A"
	defaultCustomAccent = "#C4956A"
)

// customTileStyle is how the Custom tile's swatch looks.
type customTileStyle int

const (
	tileRainbow customTileStyle = iota // A: the prototype's four-colour gradient with 🎨
	tileOwn                            // B: your custom colours, like the other tiles
	tilePlain                          // C: flat tag colour with 🎨
)

var customTile = tileRainbow

// customTileTop paints a theme tile's 44px swatch: the theme's background
// with a dot of its accent, or the Custom tile's look.
func customTileTop(gtx layout.Context, a *App, name string, size image.Point, r int, dark bool) {
	th := a.Theme
	mode := "light"
	if dark {
		mode = "dark"
	}
	clipTop := clip.RRect{Rect: image.Rectangle{Max: size}, NW: r, NE: r}.Push(gtx.Ops)
	defer clipTop.Pop()
	dot := func(c color.NRGBA) {
		d := gtx.Dp(20)
		st := op.Offset(image.Pt((size.X-d)/2, (size.Y-d)/2)).Push(gtx.Ops)
		rrect(gtx, image.Pt(d, d), d/2, c)
		st.Pop()
	}
	palette := func(theme string) Palette {
		p := a.Lib.Profile
		p.Theme, p.Mode = theme, mode
		if theme == "custom" && (p.CustomBase == "" || p.CustomAccent == "") {
			p.CustomBase, p.CustomAccent = defaultCustomBase, defaultCustomAccent
		}
		return PaletteFor(p)
	}
	if name != "custom" {
		pl := palette(name)
		fillRect(gtx, image.Rectangle{Max: size}, pl.Bg)
		dot(pl.Accent)
		return
	}
	switch customTile {
	case tileRainbow:
		// linear-gradient(135deg, #f5c6a0, #a0c6f5, #c6f5a0, #f5a0c6),
		// drawn as three diagonal bands.
		stops := []color.NRGBA{rgb(0xF5C6A0), rgb(0xA0C6F5), rgb(0xC6F5A0), rgb(0xF5A0C6)}
		w, h := float32(size.X), float32(size.Y)
		for i := 0; i < 3; i++ {
			t0, t1 := float32(i)/3, float32(i+1)/3
			p0 := f32.Pt(w*t0, h*t0)
			p1 := f32.Pt(w*t1, h*t1)
			var path clip.Path
			path.Begin(gtx.Ops)
			// The band between two lines perpendicular to the diagonal.
			n := f32.Pt(h, -w) // along the band
			path.MoveTo(p0.Add(n))
			path.LineTo(p0.Sub(n))
			path.LineTo(p1.Sub(n))
			path.LineTo(p1.Add(n))
			path.Close()
			area := clip.Outline{Path: path.End()}.Op().Push(gtx.Ops)
			paint.LinearGradientOp{Stop1: p0, Color1: stops[i], Stop2: p1, Color2: stops[i+1]}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			area.Pop()
		}
	case tileOwn:
		pl := palette("custom")
		fillRect(gtx, image.Rectangle{Max: size}, pl.Bg)
		dot(pl.Accent)
		return
	default:
		fillRect(gtx, image.Rectangle{Max: size}, th.Tag)
	}
	e := gtx.Dp(18)
	st := op.Offset(image.Pt((size.X-e)/2, (size.Y-e)/2)).Push(gtx.Ops)
	Emoji(gtx, "1f3a8", 18)
	st.Pop()
}

// customPicker is the box under the tiles when Custom is chosen: base
// tone and accent colour, each opening a colour dialog, and Apply.
type customPicker struct {
	base, accent string
	baseBtn      widget.Clickable
	accentBtn    widget.Clickable
	apply        widget.Clickable
}

func (c *customPicker) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	prof := a.Lib.Profile
	if c.base == "" {
		c.base, c.accent = prof.CustomBase, prof.CustomAccent
		if c.base == "" || c.accent == "" {
			c.base, c.accent = defaultCustomBase, defaultCustomAccent
		}
	}
	if c.baseBtn.Clicked(gtx) {
		openColourDialog(a, "Base tone", c.base, func(v string) { c.base = v })
	}
	if c.accentBtn.Clicked(gtx) {
		openColourDialog(a, "Accent color", c.accent, func(v string) { c.accent = v })
	}
	if c.apply.Clicked(gtx) {
		a.SetTheme("custom", prof.Mode, c.base, c.accent)
		a.Toast("Custom palette applied ✦")
	}
	row := func(title, sub, hex string, btn *widget.Clickable) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return Text{Font: font.Font{Typeface: Sans}, Size: 12, Color: th.Text}.Layout(gtx, th, title)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return Text{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 11, Color: th.Muted}.Layout(gtx, th, sub)
						}))
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return colourSwatch(gtx, th, hex, 40, 32, false)
					})
				}))
		}
	}
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return card(gtx, th.Surface, th.Border, 12, layout.UniformInset(14), 0, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(row("Base tone", "Background & surfaces", c.base, &c.baseBtn)),
			layout.Rigid(layout.Spacer{Height: 12}.Layout),
			layout.Rigid(row("Accent color", "Highlights & buttons", c.accent, &c.accentBtn)),
			layout.Rigid(layout.Spacer{Height: 12}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return pressable(gtx, &c.apply, func(gtx layout.Context) layout.Dimensions {
					return card(gtx, th.Accent, th.Accent, 9, layout.UniformInset(7), 0, func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return Text{Font: font.Font{Typeface: Sans}, Size: 12, Color: th.BtnText}.Layout(gtx, th, "Apply palette")
						})
					})
				})
			}))
	})
}

// colourSwatch is a rounded square of a colour; ring marks the chosen one.
func colourSwatch(gtx layout.Context, th *Theme, hex string, w, h float32, ring bool) layout.Dimensions {
	c, ok := ParseHex(hex)
	if !ok {
		c = th.Tag
	}
	size := image.Pt(gtx.Dp(unitDp(w)), gtx.Dp(unitDp(h)))
	r := gtx.Dp(8)
	if ring {
		rrect(gtx, size, r, th.Text)
		b := gtx.Dp(2)
		st := op.Offset(image.Pt(b, b)).Push(gtx.Ops)
		rrect(gtx, size.Sub(image.Pt(2*b, 2*b)), r-b, th.Surface)
		st.Pop()
		b = gtx.Dp(4)
		st = op.Offset(image.Pt(b, b)).Push(gtx.Ops)
		rrect(gtx, size.Sub(image.Pt(2*b, 2*b)), r-b, c)
		st.Pop()
	} else {
		rrect(gtx, size, r, c)
	}
	return layout.Dimensions{Size: size}
}
