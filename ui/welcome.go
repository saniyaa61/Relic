package ui

// The welcome card: after onboarding, Home shows a short tour of the four
// tabs until "Got it" (owner's choice over slides or a New category
// pop-up, 2026-10-31). Profile.ShowWelcome keeps it across restarts.

import (
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/saniyaa61/relic/store"
)

var welcomeStops = []struct {
	icon        *Icon
	name, words string
}{
	{IconAdd, "New", "Log a film, a book, an album, whatever stayed with you."},
	{IconLibrary, "Library", "Your shelves. Make a category for each kind of story."},
	{IconFavorites, "Favorites", "The ones you'd keep forever, and your Top 5."},
	{IconDigest, "Digest", "Your week, month and year, gently gathered."},
}

// welcomeCard draws the tour and handles "Got it".
func welcomeCard(gtx layout.Context, a *App, gotIt *widget.Clickable) layout.Dimensions {
	th := a.Theme
	if gotIt.Clicked(gtx) {
		prof := a.Lib.Profile
		prof.ShowWelcome = false
		a.Lib.Profile = prof
		a.Update(func(w store.Writer) error { return w.SaveProfile(prof) })
		a.Toast("Enjoy your archive ✦")
	}
	return layout.Inset{Bottom: 20, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return card(gtx, th.Card, th.Border, 14, layout.UniformInset(16), 0, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			rows := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Sans}, Size: 10, Tracking: 0.12, Upper: true, Color: th.Muted}.Layout(gtx, th, "A quick tour")
				}),
				layout.Rigid(layout.Spacer{Height: 4}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 17, Color: th.Text}.Layout(gtx, th, "Welcome to your archive")
				}),
				layout.Rigid(layout.Spacer{Height: 12}.Layout),
			}
			for _, s := range welcomeStops {
				rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Bottom: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return iconChip(gtx, th, s.icon, 30, 9, 15, 1.5)
							}),
							layout.Rigid(layout.Spacer{Width: 12}.Layout),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return Text{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 13.5, Color: th.Text}.Layout(gtx, th, s.name)
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 11.5, LineHeight: 1.45, Color: th.Muted}.Layout(gtx, th, s.words)
									}))
							}))
					})
				}))
			}
			rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return pressable(gtx, gotIt, func(gtx layout.Context) layout.Dimensions {
							return card(gtx, th.Accent, th.Accent, 11, layout.Inset{Top: 8, Bottom: 8, Left: 18, Right: 18}, 0, func(gtx layout.Context) layout.Dimensions {
								return Text{Font: font.Font{Typeface: Display, Style: font.Italic}, Size: 13.5, Color: th.BtnText}.Layout(gtx, th, "Got it ✦")
							})
						})
					})
				})
			}))
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
		})
	})
}
