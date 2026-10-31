package ui

import (
	"image"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
)

// navItems are the bottom bar's destinations, labelled as in the prototype
// except the middle tab: "New" (SPEC §5) rather than the prototype's "Add"
// (owner's choice, 2026-10-06).
var navItems = [tabCount]struct {
	icon  *Icon
	label string
}{
	TabHome:      {IconHome, "Home"},
	TabLibrary:   {IconLibrary, "Library"},
	TabNew:       {IconAdd, "New"},
	TabFavorites: {IconFavorites, "Favorites"},
	TabDigest:    {IconDigest, "Digest"},
}

// navBar is the prototype's .bnav: a surface-coloured bar with a hairline
// on top and five evenly spread items.
type navBar struct {
	items [tabCount]widget.Clickable
}

// Layout draws the bar across the full width; bottomInset (px) is the
// system navigation area, which the bar's surface runs under. The
// prototype's 14px bottom padding was room for a phone's home indicator;
// on a phone the system area itself gives that room, so the bar keeps
// only 4dp there (owner: the bar sat too high above Android's buttons).
func (n *navBar) Layout(gtx layout.Context, a *App, bottomInset int) layout.Dimensions {
	th := a.Theme
	for i := range n.items {
		if n.items[i].Clicked(gtx) {
			a.Go(Tab(i))
		}
	}
	w := gtx.Constraints.Max.X
	rec := op.Record(gtx.Ops)
	pad := unit.Dp(14)
	if bottomInset > 0 {
		pad = 4
	}
	d := layout.Inset{Top: 8, Bottom: pad}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = w
		children := make([]layout.FlexChild, tabCount)
		for i := range n.items {
			children[i] = layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return n.item(gtx, th, Tab(i), Tab(i) == a.tab)
			})
		}
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx, children...)
	})
	content := rec.Stop()
	size := image.Pt(w, d.Size.Y+bottomInset)
	fillRect(gtx, image.Rectangle{Max: size}, th.Surface)
	fillRect(gtx, image.Rect(0, 0, w, gtx.Dp(1)), th.Border)
	content.Add(gtx.Ops)
	return layout.Dimensions{Size: size}
}

// navStyle is how the active tab stands out (owner: in Linen the
// prototype's accent-only colour change is too subtle). Options for the
// owner to choose from.
type navStyleKind int

const (
	navColour navStyleKind = iota // the prototype: accent colour only
	navPill                       // a soft pill behind the icon
	navDot                        // a small accent dot under the label
	navBold                       // heavier icon and label; others faded
)

var navStyle = navColour

func (n *navBar) item(gtx layout.Context, th *Theme, t Tab, active bool) layout.Dimensions {
	c := th.Muted
	on := active || n.items[t].Pressed()
	if on {
		c = th.Accent
	}
	stroke := float32(1.5)
	label := Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.04, Upper: true, Color: c}
	switch {
	case navStyle == navBold && on:
		stroke = 2
		label.Font.Weight = font.Medium
		label.FakeBold = true
	case navStyle == navBold:
		c = withAlpha(th.Muted, 0.6)
		label.Color = c
	}
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return n.items[t].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 4, Bottom: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle, Spacing: layout.SpaceSides}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if navStyle == navPill && on {
						pw, ph := gtx.Dp(52), gtx.Dp(28)
						iw := gtx.Dp(20)
						st := op.Offset(image.Pt((iw-pw)/2, (iw-ph)/2)).Push(gtx.Ops)
						rrect(gtx, image.Pt(pw, ph), ph/2, mixSRGB(th.Accent, th.Surface, 0.14))
						st.Pop()
					}
					return navItems[t].icon.Layout(gtx, 20, stroke, c)
				}),
				layout.Rigid(layout.Spacer{Height: 2}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return label.Layout(gtx, th, navItems[t].label)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if navStyle != navDot {
						return layout.Dimensions{}
					}
					d := gtx.Dp(4)
					sz := image.Pt(d, d+gtx.Dp(3))
					if on {
						st := op.Offset(image.Pt(0, gtx.Dp(3))).Push(gtx.Ops)
						rrect(gtx, image.Pt(d, d), d/2, th.Accent)
						st.Pop()
					}
					return layout.Dimensions{Size: sz}
				}),
			)
		})
	})
}
