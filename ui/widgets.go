package ui

import (
	"image"
	"image/color"
	"math"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
)

// pressable draws w and dims it while pressed: the touch version of the
// prototype's hover opacity (".add-btn:hover{opacity:.88}").
func pressable(gtx layout.Context, c *widget.Clickable, w layout.Widget) layout.Dimensions {
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if c.Pressed() {
			defer paint.PushOpacity(gtx.Ops, 0.88).Pop()
		}
		return w(gtx)
	})
}

// rrect fills a rounded rectangle of size sz at the origin.
func rrect(gtx layout.Context, sz image.Point, radius int, c color.NRGBA) {
	paint.FillShape(gtx.Ops, c, clip.UniformRRect(image.Rectangle{Max: sz}, clampRadius(radius, sz)).Op(gtx.Ops))
}

// boxShadow approximates CSS box-shadow: 0 offsetY blur rgba(0,0,0,alpha)
// behind a rounded rectangle of size sz, over a page of colour bg. Gio has no blur, so we stack thin
// rounded rings whose opacities follow the Gaussian fall-off a CSS blur
// has: full strength well inside the edge, half at the edge, fading to
// nothing a blur's width outside it.
func boxShadow(gtx layout.Context, sz image.Point, radius int, offsetY, blur unit.Dp, alpha float32, bg color.NRGBA) {
	const layers = 24
	b := float64(gtx.Dp(blur))
	if b <= 0 {
		return
	}
	sigma := b / 2
	cover := func(d float64) float64 { // shadow strength at distance d outside the edge
		return 0.5 * math.Erfc(d/(sigma*math.Sqrt2))
	}
	oy := gtx.Dp(offsetY)
	// Draw from the outermost ring in. Each ring adds the strength gained
	// between its distance and the next one out, so a point is covered by
	// rings adding up to the strength at its own distance.
	for k := layers; k >= 1; k-- {
		d := -b + 2*b*float64(k)/layers
		dOut := -b + 2*b*float64(k+1)/layers
		a := alpha * float32(cover(d-b/layers)-cover(dOut-b/layers))
		if a <= 0 {
			continue
		}
		g := int(math.Round(d))
		r := image.Rectangle{Max: sz}.Inset(-g).Add(image.Pt(0, oy))
		if r.Dx() <= 0 || r.Dy() <= 0 {
			continue
		}
		paint.FillShape(gtx.Ops, shade(a, bg), clip.UniformRRect(r, clampRadius(radius+g, r.Size())).Op(gtx.Ops))
	}
}

// IconButton is the prototype's .ibtn: a 34dp round button with an 18dp
// accent line icon, shaded with the border colour while pressed.
type IconButton struct {
	Click widget.Clickable
}

func (b *IconButton) Layout(gtx layout.Context, th *Theme, ic *Icon) layout.Dimensions {
	return b.Click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		sz := gtx.Dp(34)
		if b.Click.Pressed() {
			paint.FillShape(gtx.Ops, th.Border, clip.Ellipse{Max: image.Pt(sz, sz)}.Op(gtx.Ops))
		}
		off := (sz - gtx.Dp(18)) / 2
		st := op.Offset(image.Pt(off, off)).Push(gtx.Ops)
		ic.Layout(gtx, 18, 1.6, th.Accent)
		st.Pop()
		return layout.Dimensions{Size: image.Pt(sz, sz)}
	})
}

// Button is a filled text button. Prototype styles: .rmodal-btn (accent,
// Playfair italic), .rmodal-btn.danger and .rmodal-cancel.
type Button struct {
	Click widget.Clickable
}

// ButtonStyle picks the look of a Button.
type ButtonStyle int

const (
	ButtonAccent ButtonStyle = iota // accent fill, Playfair italic
	ButtonDanger                    // red fill, Playfair italic
	ButtonQuiet                     // tag fill with border, DM Sans (Cancel)
)

// dangerRed is the prototype's .rmodal-btn.danger colour, the same in
// every theme.
var dangerRed = rgb(0xE05555)

func (b *Button) Layout(gtx layout.Context, th *Theme, style ButtonStyle, label string) layout.Dimensions {
	bg, fg, border := th.Accent, th.BtnText, color.NRGBA{}
	txt := Text{Font: font.Font{Typeface: Display, Style: font.Italic}, Size: 14}
	// The filled buttons have no border in the prototype, but card draws
	// a 1dp one (in the fill colour), so their padding is 1dp less than
	// the prototype's 12px.
	pad := layout.UniformInset(11)
	switch style {
	case ButtonDanger:
		bg, fg = dangerRed, rgb(0xFFFFFF)
	case ButtonQuiet:
		bg, fg, border = th.Tag, th.Muted, th.Border
		txt = Text{Font: font.Font{Typeface: Sans}, Size: 13}
		pad = layout.Inset{Top: 12, Bottom: 12, Left: 17, Right: 17}
	}
	txt.Color = fg
	return pressable(gtx, &b.Click, func(gtx layout.Context) layout.Dimensions {
		if border.A == 0 {
			border = bg
		}
		return card(gtx, bg, border, 12, pad, 0, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return txt.Layout(gtx, th, label)
			})
		})
	})
}

// Input is a text field: the prototype's .rmodal-input, .search-input,
// .finput and (multi-line) .ftarea — a filled box with a thin border that
// turns accent on focus.
type Input struct {
	Editor widget.Editor
}

// InputStyle sets an Input's look; the zero value is unusable.
type InputStyle struct {
	Bg          color.NRGBA
	Radius      unit.Dp
	Pad         layout.Inset
	Size        unit.Sp
	Font        font.Font // zero: DM Sans
	Placeholder string
	// PlaceholderAlpha is the placeholder's opacity (CSS ::placeholder).
	PlaceholderAlpha float32
	// Multiline makes a wrapping text area (.ftarea); MinHeight is its
	// smallest height and LineHeight its CSS line-height.
	Multiline  bool
	MinHeight  unit.Dp
	LineHeight float32
}

func (in *Input) Layout(gtx layout.Context, th *Theme, s InputStyle) layout.Dimensions {
	in.Editor.SingleLine = !s.Multiline
	f := s.Font
	if f.Typeface == "" {
		f.Typeface = Sans
	}
	if s.LineHeight > 0 {
		in.Editor.LineHeight = unit.Sp(float32(s.Size) * s.LineHeight)
		in.Editor.LineHeightScale = 1
	}
	border := th.Border
	if gtx.Focused(&in.Editor) {
		border = th.Accent
	}
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	if s.MinHeight > 0 {
		gtx.Constraints.Min.Y = max(gtx.Constraints.Min.Y, gtx.Dp(s.MinHeight))
	}
	return card(gtx, s.Bg, border, s.Radius, s.Pad, 0, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		if in.Editor.Len() == 0 && s.Placeholder != "" {
			rec := op.Record(gtx.Ops)
			pgtx := gtx
			pgtx.Constraints.Min = image.Point{}
			if s.Multiline {
				Paragraph{Font: f, Size: s.Size, LineHeight: s.LineHeight, Color: withAlpha(th.Muted, s.PlaceholderAlpha)}.Layout(pgtx, th, s.Placeholder)
			} else {
				Text{Font: f, Size: s.Size, MaxLines: 1, Color: withAlpha(th.Muted, s.PlaceholderAlpha)}.Layout(pgtx, th, s.Placeholder)
			}
			ph := rec.Stop()
			defer ph.Add(gtx.Ops)
		}
		return in.Editor.Layout(gtx, th.Shaper, f, s.Size, colorOp(gtx, th.Text), colorOp(gtx, withAlpha(th.Accent, 0.25)))
	})
}

// Focus moves keyboard focus (and the soft keyboard) to the input.
func (in *Input) Focus(gtx layout.Context) {
	gtx.Execute(key.FocusCmd{Tag: &in.Editor})
}
