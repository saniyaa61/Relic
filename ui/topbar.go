package ui

import (
	"image"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
)

// topBar is the prototype's .topbar: 13dp from the top, 18dp gutters, the
// left and right parts pushed to the edges and centred vertically.
func topBar(gtx layout.Context, left, right layout.Widget) layout.Dimensions {
	return layout.Inset{Top: 13, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
			layout.Rigid(left),
			layout.Rigid(right),
		)
	})
}

// logoBar is the top bar of the tab pages: the logo with "Relic" and
// "your stories, preserved", and right (which may be nil) at the end.
func logoBar(gtx layout.Context, th *Theme, right layout.Widget) layout.Dimensions {
	if right == nil {
		right = func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }
	}
	return topBar(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return Logo(gtx, 38) }),
			layout.Rigid(layout.Spacer{Width: 8}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return Text{Font: font.Font{Typeface: Display, Weight: font.SemiBold}, Size: 18, LineHeight: 1, Tracking: 0.04, Color: th.Accent}.Layout(gtx, th, "Relic")
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return Text{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 10, LineHeight: 1.2, Tracking: 0.05, Color: th.Muted}.Layout(gtx, th, "your stories, preserved")
					}),
				)
			}),
		)
	}, right)
}

// subBar is the top bar of sub-pages: a back arrow, and the search button
// when search is non-nil. The arrow does what the system back button does,
// so it closes an open search first (SPEC §4.9).
func subBar(gtx layout.Context, a *App, back *IconButton, search *Search) layout.Dimensions {
	if back.Click.Clicked(gtx) {
		a.Back()
	}
	return topBar(gtx, func(gtx layout.Context) layout.Dimensions {
		return back.Layout(gtx, a.Theme, IconBack)
	}, func(gtx layout.Context) layout.Dimensions {
		if search == nil {
			return layout.Dimensions{}
		}
		return search.Button(gtx, a)
	})
}

// Search is a page's search: the magnifier button and the bar that slides
// open under the top bar. Pages own one and ask it for the query; the
// shared search implementation (step 2) turns the query into results.
type Search struct {
	open      bool
	changedAt time.Time
	focus     bool
	btn       IconButton
	input     Input
}

// Query is the text typed, or "" when search is closed.
func (s *Search) Query() string {
	if !s.open {
		return ""
	}
	return s.input.Editor.Text()
}

// IsOpen reports whether the search bar is showing.
func (s *Search) IsOpen() bool { return s.open }

// Toggle opens or closes the search; closing clears the query, as in the
// prototype.
func (s *Search) Toggle(a *App) {
	s.open = !s.open
	s.changedAt = a.Now()
	s.focus = s.open
	if !s.open {
		s.input.Editor.SetText("")
	}
}

// WantsBack and Back let a page pass the back button to its search.
func (s *Search) WantsBack() bool { return s.open }
func (s *Search) Back(a *App)     { s.Toggle(a) }

// Button is the top bar's magnifier, which toggles the search.
func (s *Search) Button(gtx layout.Context, a *App) layout.Dimensions {
	if s.btn.Click.Clicked(gtx) {
		s.Toggle(a)
	}
	return s.btn.Layout(gtx, a.Theme, IconSearch)
}

// Bar draws the search field, sliding open and closed like the
// prototype's .search-bar (max-height 0 ↔ 60px over 0.28s, padding
// 0 → 10px top, 4px bottom).
func (s *Search) Bar(gtx layout.Context, a *App, placeholder string) layout.Dimensions {
	if s.changedAt.IsZero() && !s.open {
		return layout.Dimensions{}
	}
	t := easeCSS(progress(gtx, a.Now(), s.changedAt, 280*time.Millisecond))
	if !s.open {
		t = 1 - t
	}
	if t == 0 {
		return layout.Dimensions{}
	}
	if s.focus {
		s.input.Focus(gtx)
		s.focus = false
	}
	rec := op.Record(gtx.Ops)
	d := layout.Inset{Top: unit.Dp(10 * t), Bottom: unit.Dp(4 * t), Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return s.input.Layout(gtx, a.Theme, InputStyle{
			Bg: a.Theme.Card, Radius: 11, Size: 13,
			Pad:              layout.Inset{Top: 9, Bottom: 9, Left: 14, Right: 14},
			Placeholder:      placeholder,
			PlaceholderAlpha: 0.7,
		})
	})
	call := rec.Stop()
	h := min(d.Size.Y, roundi(float32(gtx.Dp(60))*t))
	defer clip.Rect{Max: image.Pt(d.Size.X, h)}.Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	return layout.Dimensions{Size: image.Pt(d.Size.X, h)}
}

// closeNow closes the search without animating, as when leaving the page.
func (s *Search) closeNow() {
	s.open = false
	s.changedAt = time.Time{}
	s.input.Editor.SetText("")
}
