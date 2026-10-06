package ui

import (
	"image"
	"image/color"
	"math"
	"strings"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"golang.org/x/image/math/fixed"
)

// Text is one styled run of text, mirroring a CSS text rule.
type Text struct {
	Font  font.Font
	Size  unit.Sp
	Color color.NRGBA
	// LineHeight is the CSS line-height as a multiple of Size, for single
	// lines. 0 means "normal": the font's own ascent plus descent.
	LineHeight float32
	// Tracking is CSS letter-spacing in em. Tracked text is always one line,
	// so use it for eyebrows and labels, not prose.
	Tracking float32
	Upper    bool
	// MaxLines truncates with "…" when exceeded; 0 means no limit.
	// Ignored for tracked and bold text.
	MaxLines int
	// FakeBold thickens the glyphs the way browsers do when CSS asks for
	// a weight (600) the loaded font lacks: the prototype's bold times
	// and labels are DM Sans / Lora Medium drawn this way. One line only.
	FakeBold bool
}

// Layout draws s and returns its size.
func (t Text) Layout(gtx layout.Context, th *Theme, s string) layout.Dimensions {
	if t.Upper {
		s = strings.ToUpper(s)
	}
	gtx.Constraints.Min = image.Point{}
	rec := op.Record(gtx.Ops)
	paint.ColorOp{Color: t.Color}.Add(gtx.Ops)
	material := rec.Stop()

	rec = op.Record(gtx.Ops)
	var dims layout.Dimensions
	if t.Tracking != 0 || t.FakeBold {
		dims = t.layoutTracked(gtx, th, s, material)
	} else {
		dims = widget.Label{MaxLines: t.MaxLines}.Layout(gtx, th.Shaper, t.Font, t.Size, s, material)
	}
	call := rec.Stop()

	if t.LineHeight == 0 {
		call.Add(gtx.Ops)
		return dims
	}
	// CSS centres the font's ascent+descent inside the line box, so any
	// difference ("leading") is split evenly above and below.
	box := int(math.Round(float64(gtx.Metric.PxPerSp * float32(t.Size) * t.LineHeight)))
	shift := (box - dims.Size.Y) / 2
	st := op.Offset(image.Pt(0, shift)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	st.Pop()
	return layout.Dimensions{
		Size:     image.Pt(dims.Size.X, box),
		Baseline: dims.Baseline + (box - dims.Size.Y - shift),
	}
}

// layoutTracked draws one line of s with letter-spacing (and fake bold).
// Gio has no
// letter-spacing, so we shape the line once and push each character right,
// keeping the font's own kerning. Like CSS, space follows every character,
// including the last.
func (t Text) layoutTracked(gtx layout.Context, th *Theme, s string, material op.CallOp) layout.Dimensions {
	th.Shaper.LayoutString(text.Parameters{
		Font:     t.Font,
		PxPerEm:  fixed.I(gtx.Sp(t.Size)),
		MaxLines: 1,
		MaxWidth: math.MaxInt32,
		Locale:   gtx.Locale,
	}, s)
	track := fixed.Int26_6(math.Round(float64(gtx.Metric.PxPerSp * float32(t.Size) * t.Tracking * 64)))
	var glyphs []text.Glyph
	var shift fixed.Int26_6
	for g, ok := th.Shaper.NextGlyph(); ok; g, ok = th.Shaper.NextGlyph() {
		g.X += shift
		glyphs = append(glyphs, g)
		if g.Flags&text.FlagClusterBreak != 0 {
			shift += track
		}
	}
	if len(glyphs) == 0 {
		return layout.Dimensions{}
	}
	first, last := glyphs[0], glyphs[len(glyphs)-1]
	width := (last.X + last.Advance + track - first.X).Ceil()
	ascent, descent := first.Ascent.Ceil(), first.Descent.Ceil()

	// Shape positions glyphs relative to the first one.
	st := op.Affine(f32.AffineId().Offset(f32.Pt(0, float32(ascent)))).Push(gtx.Ops)
	path := th.Shaper.Shape(glyphs)
	outline := clip.Outline{Path: path}.Op().Push(gtx.Ops)
	material.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	outline.Pop()
	if t.FakeBold {
		// Chrome (Skia) strokes the outline by size × a ratio that goes
		// from 1/24 at 9px to 1/32 at 36px; advances don't change.
		px := float32(t.Size)
		k := min(max((px-9)/27, 0), 1)
		ratio := 1.0/24 + k*(1.0/32-1.0/24)
		stroke := clip.Stroke{Path: path, Width: gtx.Metric.PxPerSp * px * ratio}.Op().Push(gtx.Ops)
		material.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		stroke.Pop()
	}
	st.Pop()
	return layout.Dimensions{Size: image.Pt(width, ascent+descent), Baseline: descent}
}

// Paragraph is wrapped prose, mirroring a CSS block with a line-height:
// lines sit LineHeight × Size apart and, as in CSS, half the extra space
// goes above the first line and half below the last.
type Paragraph struct {
	Font       font.Font
	Size       unit.Sp
	Color      color.NRGBA
	LineHeight float32 // multiple of Size; 0 means CSS "normal" for the font
	Alignment  text.Alignment
	MaxLines   int
}

func (p Paragraph) Layout(gtx layout.Context, th *Theme, s string) layout.Dimensions {
	gtx.Constraints.Min = image.Point{}
	factor := p.LineHeight
	if factor == 0 {
		factor = normalLineHeight[p.Font.Typeface]
	}
	lh := unit.Sp(float32(p.Size) * factor)
	// One line's natural height (ascent + descent) tells us the leading.
	rec := op.Record(gtx.Ops)
	one := widget.Label{MaxLines: 1}.Layout(gtx, th.Shaper, p.Font, p.Size, "Ag", op.CallOp{})
	rec.Stop()
	half := (gtx.Sp(lh) - one.Size.Y) / 2

	rec = op.Record(gtx.Ops)
	l := widget.Label{Alignment: p.Alignment, MaxLines: p.MaxLines, LineHeight: lh, LineHeightScale: 1}
	d := l.Layout(gtx, th.Shaper, p.Font, p.Size, s, colorOp(gtx, p.Color))
	call := rec.Stop()
	st := op.Offset(image.Pt(0, half)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	st.Pop()
	d.Size.Y += 2 * half
	d.Baseline += half
	return d
}

// colorOp records a paint colour, the material Gio's text widgets take.
func colorOp(gtx layout.Context, c color.NRGBA) op.CallOp {
	rec := op.Record(gtx.Ops)
	paint.ColorOp{Color: c}.Add(gtx.Ops)
	return rec.Stop()
}

// outlinedText draws one line of s filled with fill and outlined with a
// stroke of width px centred on the glyph edges, as SVG text with both
// fill and stroke draws.
func outlinedText(gtx layout.Context, th *Theme, f font.Font, size unit.Sp, s string, fill, stroke color.NRGBA, width float32) layout.Dimensions {
	th.Shaper.LayoutString(text.Parameters{
		Font:     f,
		PxPerEm:  fixed.I(gtx.Sp(size)),
		MaxLines: 1,
		MaxWidth: math.MaxInt32,
		Locale:   gtx.Locale,
	}, s)
	var glyphs []text.Glyph
	for g, ok := th.Shaper.NextGlyph(); ok; g, ok = th.Shaper.NextGlyph() {
		glyphs = append(glyphs, g)
	}
	if len(glyphs) == 0 {
		return layout.Dimensions{}
	}
	first, last := glyphs[0], glyphs[len(glyphs)-1]
	w := (last.X + last.Advance - first.X).Ceil()
	ascent, descent := first.Ascent.Ceil(), first.Descent.Ceil()
	defer op.Affine(f32.AffineId().Offset(f32.Pt(0, float32(ascent)))).Push(gtx.Ops).Pop()
	path := th.Shaper.Shape(glyphs)
	area := clip.Outline{Path: path}.Op().Push(gtx.Ops)
	paint.ColorOp{Color: fill}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	area.Pop()
	area = clip.Stroke{Path: path, Width: width}.Op().Push(gtx.Ops)
	paint.ColorOp{Color: stroke}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	area.Pop()
	return layout.Dimensions{Size: image.Pt(w, ascent+descent), Baseline: descent}
}
