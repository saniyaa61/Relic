package ui

import (
	"image"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"
)

// navItems are the bottom bar's destinations, labelled as in the prototype.
var navItems = [tabCount]struct {
	icon  *Icon
	label string
}{
	TabHome:      {IconHome, "Home"},
	TabLibrary:   {IconLibrary, "Library"},
	TabNew:       {IconAdd, "Add"},
	TabFavorites: {IconFavorites, "Favorites"},
	TabDigest:    {IconDigest, "Digest"},
}

// navBar is the prototype's .bnav: a surface-coloured bar with a hairline
// on top and five evenly spread items.
type navBar struct {
	items [tabCount]widget.Clickable
}

// Layout draws the bar across the full width; bottomInset (px) is the
// system navigation area, added below the prototype's own padding.
func (n *navBar) Layout(gtx layout.Context, a *App, bottomInset int) layout.Dimensions {
	th := a.Theme
	for i := range n.items {
		if n.items[i].Clicked(gtx) {
			a.Go(Tab(i))
		}
	}
	w := gtx.Constraints.Max.X
	rec := op.Record(gtx.Ops)
	d := layout.Inset{Top: 8, Bottom: 14}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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

func (n *navBar) item(gtx layout.Context, th *Theme, t Tab, active bool) layout.Dimensions {
	c := th.Muted
	if active || n.items[t].Pressed() {
		c = th.Accent
	}
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return n.items[t].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 4, Bottom: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle, Spacing: layout.SpaceSides}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return navItems[t].icon.Layout(gtx, 20, 1.5, c)
				}),
				layout.Rigid(layout.Spacer{Height: 2}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.04, Upper: true, Color: c}.Layout(gtx, th, navItems[t].label)
				}),
			)
		})
	})
}
