package ui

// The first launch (SPEC §5 "Onboarding"; prototype showOnboarding):
// "Welcome to Relic", an optional name, then Home, opening New category
// when there are none.

import (
	"image"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/saniyaa61/relic/store"
)

// StartOnboarding shows the welcome page if this is a fresh start: no
// categories, entries or name (prototype boot).
func (a *App) StartOnboarding() {
	l := a.Lib
	if len(l.Categories) == 0 && len(l.Entries) == 0 && l.Profile.Name == "" {
		a.Push(&onboardingPage{})
	}
}

type onboardingPage struct {
	list  widget.List
	name  Input
	begin widget.Clickable
	skip  widget.Clickable
	focus bool
}

func (p *onboardingPage) complete(a *App) {
	if v := strings.TrimSpace(p.name.Editor.Text()); v != "" {
		prof := a.Lib.Profile
		prof.Name = string([]rune(v)[:min(len([]rune(v)), 30)])
		a.Lib.Profile = prof
		a.Update(func(w store.Writer) error { return w.SaveProfile(prof) })
	}
	a.Go(TabHome)
	a.stack = nil
	if len(a.Lib.Categories) == 0 {
		a.Go(TabLibrary)
		newCategoryDialog(a)
	}
}

func (p *onboardingPage) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	if p.begin.Clicked(gtx) || p.skip.Clicked(gtx) {
		p.complete(a)
		return layout.Dimensions{}
	}
	for {
		ev, ok := p.name.Editor.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			p.complete(a)
			return layout.Dimensions{}
		}
	}
	p.name.Editor.Submit, p.name.Editor.MaxLen = true, 30
	centred := func(w layout.Widget) layout.FlexChild {
		return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Center.Layout(gtx, w)
		})
	}
	// .onboard-wrap: a centred column, 32px 28px padding.
	return layout.Inset{Top: 32, Bottom: 32, Left: 28, Right: 28}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceSides}.Layout(gtx,
			centred(func(gtx layout.Context) layout.Dimensions {
				// 100px wide: the logo is 200 × 341.
				return layout.Inset{Bottom: 16}.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return Logo(gtx, 170.5) })
			}),
			centred(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return richLine(gtx, th, 28, 1.2, []textRun{
						{s: "Welcome to ", font: font.Font{Typeface: Display, Weight: font.SemiBold}, color: th.Text},
						// 600 italic isn't loaded in the prototype either: the browser
						// thickens the 500 italic.
						{s: "Relic", font: font.Font{Typeface: Display, Weight: font.Medium, Style: font.Italic}, color: th.Accent, fakeBold: true},
					})
				})
			}),
			centred(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(280))
				return layout.Inset{Bottom: 30}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 14, LineHeight: 1.7, Alignment: textMiddle, Color: th.Muted}.Layout(gtx, th,
						"A private library for the films, series, and books that shape you. Everything here is yours — no algorithms, no noise, just your memories preserved.")
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: 14}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return p.name.Layout(gtx, th, InputStyle{Bg: th.Card, Radius: 12, Size: 15,
						Font: font.Font{Typeface: Display, Style: font.Italic}, Center: true,
						Pad: layout.Inset{Top: 13, Bottom: 13, Left: 16, Right: 16}, Placeholder: "What should we call you?", PlaceholderAlpha: 0.55})
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return bigButton(gtx, th, &p.begin, "Begin your library →")
			}),
			centred(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return p.skip.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return underlined(gtx, th, Text{Font: font.Font{Typeface: Sans}, Size: 12, Color: th.Muted}, "Skip for now")
					})
				})
			}))
	})
}

// underlined draws t with a thin underline 3px below the baseline
// (text-underline-offset: 3px).
func underlined(gtx layout.Context, th *Theme, t Text, s string) layout.Dimensions {
	d := t.Layout(gtx, th, s)
	y := d.Size.Y - d.Baseline + gtx.Dp(3)
	fillRect(gtx, image.Rect(0, y, d.Size.X, y+max(1, gtx.Dp(0.8))), t.Color)
	if y+gtx.Dp(1) > d.Size.Y {
		d.Size.Y = y + gtx.Dp(1)
	}
	return d
}
