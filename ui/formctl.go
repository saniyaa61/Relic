package ui

// Controls for the entry form and the logging sheet, after the
// prototype's .fsec, .flbl, .status-btn, .sbtn, .tbub, .upload-box and
// .save-btn.

import (
	"image"

	"strings"

	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/saniyaa61/relic/core"
)

// formSection is a .fsec card with its Playfair title and an optional
// small note after the title: italic for "Details  Series / Drama", plain
// for "Feeling tags (pick up to 4)", as in the prototype.
func formSection(gtx layout.Context, th *Theme, title, note string, body layout.Widget) layout.Dimensions {
	italic := !strings.HasPrefix(note, "(")
	return layout.Inset{Bottom: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return card(gtx, th.Card, th.Border, 14, layout.UniformInset(14), 0, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if title == "" {
						return layout.Dimensions{}
					}
					return layout.Inset{Bottom: 11}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Alignment: layout.Baseline}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return Text{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 13, Color: th.Text}.Layout(gtx, th, title)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if note == "" {
									return layout.Dimensions{}
								}
								return layout.Inset{Left: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									f := font.Font{Typeface: Display}
									if italic {
										f.Style = font.Italic
									}
									return Text{Font: f, Size: 11, Color: th.Muted}.Layout(gtx, th, note)
								})
							}),
						)
					})
				}),
				layout.Rigid(body),
			)
		})
	})
}

// fieldLabel is the small letter-spaced label over a form field (.flbl).
func fieldLabel(gtx layout.Context, th *Theme, s string) layout.Dimensions {
	return layout.Inset{Bottom: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.06, Upper: true, Color: th.Muted}.Layout(gtx, th, s)
	})
}

// labelled stacks a label over a field, with the form's 10dp gap below.
func labelled(th *Theme, label string, field layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Bottom: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return fieldLabel(gtx, th, label) }),
				layout.Rigid(field))
		})
	}
}

// pairRow lays two widgets side by side with 8dp between (the form's
// two-field rows); weights set their shares.
func pairRow(gtx layout.Context, left, right layout.Widget, lw, rw float32) layout.Dimensions {
	return layout.Flex{Alignment: layout.Start}.Layout(gtx,
		layout.Flexed(lw, left),
		layout.Rigid(layout.Spacer{Width: 8}.Layout),
		layout.Flexed(rw, right))
}

// formInput is the form's text box style (.finput).
func formInput(placeholder string) InputStyle {
	return InputStyle{Radius: 9, Size: 13, Pad: layout.Inset{Top: 9, Bottom: 9, Left: 11, Right: 11},
		Placeholder: placeholder, PlaceholderAlpha: 1}
}

// toggleButton is a .status-btn / .tbtn: a quiet box that fills with the
// accent colour when on.
func toggleButton(gtx layout.Context, th *Theme, c *widget.Clickable, label string, on bool, padY unit.Dp) layout.Dimensions {
	bg, fg, border := th.Bg, th.Muted, th.Border
	if on {
		bg, fg, border = th.Accent, th.BtnText, th.Accent
	}
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return pressable(gtx, c, func(gtx layout.Context) layout.Dimensions {
		return card(gtx, bg, border, 9, layout.Inset{Top: padY, Bottom: padY, Left: 7, Right: 7}, 0, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Sans}, Size: 12, Color: fg}.Layout(gtx, th, label)
			})
		})
	})
}

// starInput is the rating row (.sbtn): five 24px stars; tapping the left
// half of a star gives a half star, as in the prototype.
type starInput struct {
	clicks [5]gesture.Click
}

// Layout draws the stars for rating and returns the rating after any tap.
func (s *starInput) Layout(gtx layout.Context, th *Theme, rating float64) float64 {
	children := make([]layout.FlexChild, 0, 9)
	for i := range 5 {
		v := float64(i + 1)
		for {
			e, ok := s.clicks[i].Update(gtx.Source)
			if !ok {
				break
			}
			if e.Kind == gesture.KindClick {
				w := gtx.Sp(24) + 2*gtx.Dp(1)
				if e.Position.X < w/2 {
					rating = v - 0.5
				} else {
					rating = v
				}
			}
		}
		glyph, c := "★", th.Border
		switch {
		case rating >= v:
			c = th.Accent2
		case rating >= v-0.5:
			glyph, c = "½", th.Accent2
		}
		if i > 0 {
			children = append(children, layout.Rigid(layout.Spacer{Width: 2}.Layout))
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(1).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				d := Text{Font: font.Font{Typeface: Sans}, Size: 24, LineHeight: 1, Color: c}.Layout(gtx, th, glyph)
				// Every star is as wide as a full one, so the halves line up.
				d.Size.X = gtx.Sp(24)
				defer clip.Rect{Max: d.Size}.Push(gtx.Ops).Pop()
				s.clicks[i].Add(gtx.Ops)
				return d
			})
		}))
	}
	layout.Flex{Alignment: layout.Middle}.Layout(gtx, children...)
	return rating
}

// tagPicker is the "Feeling tags" section: the preset bubbles, any custom
// tags, and the "Add your own tag…" box (up to 4 tags, as the prototype).
type tagPicker struct {
	tags   []string
	preset map[string]*widget.Clickable
	custom map[string]*widget.Clickable
	input  Input
	add    widget.Clickable
}

const maxTags = 4

func (t *tagPicker) toggle(tag string) {
	for i, x := range t.tags {
		if x == tag {
			t.tags = append(t.tags[:i], t.tags[i+1:]...)
			return
		}
	}
	if len(t.tags) < maxTags {
		t.tags = append(t.tags, tag)
	}
}

func (t *tagPicker) has(tag string) bool {
	for _, x := range t.tags {
		if x == tag {
			return true
		}
	}
	return false
}

func isPreset(tag string) bool {
	for _, p := range core.PresetTags {
		if p == tag {
			return true
		}
	}
	return false
}

func (t *tagPicker) Layout(gtx layout.Context, th *Theme) layout.Dimensions {
	if t.preset == nil {
		t.preset, t.custom = map[string]*widget.Clickable{}, map[string]*widget.Clickable{}
		t.input.Editor.Submit = true
		t.input.Editor.MaxLen = 20
	}
	clickFor := func(m map[string]*widget.Clickable, tag string) *widget.Clickable {
		if m[tag] == nil {
			m[tag] = &widget.Clickable{}
		}
		return m[tag]
	}
	addCustom := func() {
		v := strings.TrimSpace(t.input.Editor.Text())
		t.input.Editor.SetText("")
		if v != "" && len(t.tags) < maxTags && !t.has(v) {
			t.tags = append(t.tags, v)
		}
	}
	for {
		ev, ok := t.input.Editor.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			addCustom()
		}
	}
	if t.add.Clicked(gtx) {
		addCustom()
	}
	var bubbles []layout.Widget
	for _, tag := range core.PresetTags {
		c := clickFor(t.preset, tag)
		if c.Clicked(gtx) {
			t.toggle(tag)
		}
		bubbles = append(bubbles, func(gtx layout.Context) layout.Dimensions { return tagBubble(gtx, th, c, tag, t.has(tag)) })
	}
	for _, tag := range append([]string(nil), t.tags...) {
		if isPreset(tag) {
			continue
		}
		c := clickFor(t.custom, tag)
		if c.Clicked(gtx) {
			t.toggle(tag)
		}
		bubbles = append(bubbles, func(gtx layout.Context) layout.Dimensions { return tagBubble(gtx, th, c, tag, true) })
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return flow(gtx, gtx.Dp(5), bubbles)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return t.input.Layout(gtx, th, InputStyle{Bg: th.Bg, Radius: 9, Size: 12,
							Pad:         layout.Inset{Top: 6, Bottom: 6, Left: 10, Right: 10},
							Placeholder: "Add your own tag…", PlaceholderAlpha: 0.6})
					}),
					layout.Rigid(layout.Spacer{Width: 6}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return pressable(gtx, &t.add, func(gtx layout.Context) layout.Dimensions {
							return card(gtx, th.Accent, th.Accent, 9, layout.Inset{Top: 5, Bottom: 5, Left: 11, Right: 11}, 0, func(gtx layout.Context) layout.Dimensions {
								return Text{Font: font.Font{Typeface: Sans}, Size: 12, Color: th.BtnText}.Layout(gtx, th, "Add")
							})
						})
					}),
				)
			})
		}),
	)
}

// tagBubble is a .tbub feeling tag: accent outline, filled when picked.
func tagBubble(gtx layout.Context, th *Theme, c *widget.Clickable, tag string, on bool) layout.Dimensions {
	bg, fg := th.Tag, th.Accent
	if on {
		bg, fg = th.Accent, th.BtnText
	}
	return pressable(gtx, c, func(gtx layout.Context) layout.Dimensions {
		return card(gtx, bg, th.Accent, 20, layout.Inset{Top: 4, Bottom: 4, Left: 11, Right: 11}, 0, func(gtx layout.Context) layout.Dimensions {
			return Text{Font: font.Font{Typeface: Sans}, Size: 11, Color: fg}.Layout(gtx, th, tag)
		})
	})
}

// IconImage is the "Upload poster" picture glyph.
var IconImage = IconSVG(`<rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/>`)

// uploadBox is the dashed poster box (.upload-box): the poster once
// chosen, otherwise a picture glyph and "Upload poster".
func uploadBox(gtx layout.Context, a *App, c *widget.Clickable, poster string, height unit.Dp) layout.Dimensions {
	th := a.Theme
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		sz := image.Pt(gtx.Constraints.Max.X, gtx.Dp(height))
		rad := gtx.Dp(11)
		rrect(gtx, sz, rad, th.Bg)
		if !a.drawPoster(gtx, poster, sz, rad) {
			border := th.Border
			if c.Pressed() {
				border = th.Accent2
			}
			dashedRRect(gtx, sz, rad, 1.5, 3, 2, border)
			rec := op.Record(gtx.Ops)
			d := layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return IconImage.Layout(gtx, 24, 1.3, th.Muted) }),
				layout.Rigid(layout.Spacer{Height: 5}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Sans}, Size: 12, Color: th.Muted}.Layout(gtx, th, "Upload poster")
				}))
			call := rec.Stop()
			st := op.Offset(image.Pt((sz.X-d.Size.X)/2, (sz.Y-d.Size.Y)/2)).Push(gtx.Ops)
			call.Add(gtx.Ops)
			st.Pop()
		}
		return layout.Dimensions{Size: sz}
	})
}

// bigButton is the full-width accent .save-btn ("Preserve this memory →").
func bigButton(gtx layout.Context, th *Theme, c *widget.Clickable, label string) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return pressable(gtx, c, func(gtx layout.Context) layout.Dimensions {
		// card's 1dp border is part of the prototype's 14px padding.
		return card(gtx, th.Accent, th.Accent, 13, layout.UniformInset(13), 0, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Display, Style: font.Italic}, Size: 15, Tracking: 0, Color: th.BtnText}.Layout(gtx, th, label)
			})
		})
	})
}
