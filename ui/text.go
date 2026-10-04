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
	// Ignored for tracked text.
	MaxLines int
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
	if t.Tracking != 0 {
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

// layoutTracked draws one line of s with letter-spacing. Gio has no
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
	outline := clip.Outline{Path: th.Shaper.Shape(glyphs)}.Op().Push(gtx.Ops)
	material.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	outline.Pop()
	st.Pop()
	return layout.Dimensions{Size: image.Pt(width, ascent+descent), Baseline: descent}
}
