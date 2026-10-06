package ui

// Temporary pages for the app shell (Phase 3 step 1). Each is replaced by
// the real screen in its own step; nothing here is meant to ship.

import (
	"fmt"
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
)

// placeholder stands in for a tab whose screen isn't built yet.
type placeholder struct {
	eyebrow, title string
	step           int
	list           widget.List
}

func (p *placeholder) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	return scrollPage(gtx, &p.list, []layout.Widget{
		func(gtx layout.Context) layout.Dimensions { return logoBar(gtx, th, nil) },
	}, []layout.Widget{
		func(gtx layout.Context) layout.Dimensions {
			return pageHead(gtx, th, p.eyebrow, p.title, fmt.Sprintf("This screen arrives in step %d of Phase 3.", p.step))
		},
	})
}

// homePreview stands in for Home (step 4) with buttons that try each part
// of the shell: dialogs, toasts and a sub-page.
type homePreview struct {
	settings                                  IconButton
	prompt, confirm, toast, toastErr, subPage widget.Clickable
	list                                      widget.List
}

func (h *homePreview) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	if h.settings.Click.Clicked(gtx) || h.subPage.Clicked(gtx) {
		a.Push(&themePreview{})
	}
	if h.prompt.Clicked(gtx) {
		a.ShowDialog(PromptDialog("New category",
			"Call it whatever makes sense to you — Films, Comfort Rewatches, Books for the train.",
			"Category name", "e.g. Films", "", "Create",
			func(a *App, v string) { a.Toast("Preview only — “" + v + "” wasn't saved") }))
	}
	if h.confirm.Clicked(gtx) {
		a.ShowDialog(ConfirmDialog("Delete this entry?",
			"This piece of your journey will be removed. This can't be undone.", "", "Delete",
			func(a *App) { a.Toast("Preview only — nothing was deleted") }))
	}
	if h.toast.Clicked(gtx) {
		a.Toast("Session logged ✦")
	}
	if h.toastErr.Clicked(gtx) {
		a.ToastError("You already have that category")
	}
	return scrollPage(gtx, &h.list, []layout.Widget{
		func(gtx layout.Context) layout.Dimensions {
			return logoBar(gtx, th, func(gtx layout.Context) layout.Dimensions {
				return h.settings.Layout(gtx, th, IconSettings)
			})
		},
	}, []layout.Widget{
		func(gtx layout.Context) layout.Dimensions {
			return pageHead(gtx, th, "App shell preview", "Home",
				"The real Home screen arrives in step 4. Until then, try the parts every screen will share.")
		},
		func(gtx layout.Context) layout.Dimensions {
			return wideButton(gtx, th, &h.prompt, nil, "Show a dialog")
		},
		func(gtx layout.Context) layout.Dimensions {
			return wideButton(gtx, th, &h.confirm, nil, "Ask before deleting")
		},
		func(gtx layout.Context) layout.Dimensions { return wideButton(gtx, th, &h.toast, nil, "Show a toast") },
		func(gtx layout.Context) layout.Dimensions {
			return wideButton(gtx, th, &h.toastErr, nil, "Show an error toast")
		},
		func(gtx layout.Context) layout.Dimensions {
			return wideButton(gtx, th, &h.subPage, IconSettings, "Open a sub-page (theme picker)")
		},
	})
}

// themePreview is a sub-page for trying every theme and both modes. The
// choice is saved, so it is remembered next launch. The real picker is
// part of Settings (step 9).
type themePreview struct {
	back   IconButton
	search Search
	themes [7]widget.Clickable
	modes  [2]widget.Clickable
	list   widget.List
}

func (p *themePreview) WantsBack() bool { return p.search.WantsBack() }
func (p *themePreview) Back(a *App)     { p.search.Back(a) }

var themeLabels = []string{"Linen", "Midnight", "Blush", "Forest", "Rose", "Slate", "Custom"}

func (p *themePreview) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	prof := a.Lib.Profile
	names := append(append([]string{}, ThemeNames...), "custom")
	for i := range p.themes {
		if p.themes[i].Clicked(gtx) {
			base, accent := prof.CustomBase, prof.CustomAccent
			if names[i] == "custom" && (base == "" || accent == "") {
				base, accent = "#7A5C3A", "#C4956A"
			}
			a.SetTheme(names[i], prof.Mode, base, accent)
			prof = a.Lib.Profile
		}
	}
	for i, mode := range []string{"light", "dark"} {
		if p.modes[i].Clicked(gtx) {
			a.SetTheme(prof.Theme, mode, prof.CustomBase, prof.CustomAccent)
			prof = a.Lib.Profile
		}
	}
	current := prof.Theme
	if current == "" {
		current = "linen"
	}
	dark := prof.Mode == "dark"

	rows := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions {
			return pageHead(gtx, th, "Settings preview", "Theme",
				"Try every theme here; your choice is remembered next time. The real Settings screen arrives in step 9.")
		},
		p.section(th, "Theme", func(gtx layout.Context) layout.Dimensions {
			chips := make([]layout.Widget, len(p.themes))
			for i := range p.themes {
				chips[i] = func(gtx layout.Context) layout.Dimensions {
					return pill(gtx, th, &p.themes[i], themeLabels[i], names[i] == current)
				}
			}
			return flow(gtx, gtx.Dp(6), chips)
		}),
		p.section(th, "Mode", func(gtx layout.Context) layout.Dimensions {
			return flow(gtx, gtx.Dp(6), []layout.Widget{
				func(gtx layout.Context) layout.Dimensions { return pill(gtx, th, &p.modes[0], "Light", !dark) },
				func(gtx layout.Context) layout.Dimensions { return pill(gtx, th, &p.modes[1], "Dark", dark) },
			})
		}),
	}
	if q := strings.TrimSpace(p.search.Query()); q != "" {
		rows = []layout.Widget{func(gtx layout.Context) layout.Dimensions {
			return pageHead(gtx, th, "Search preview", "“"+q+"”",
				"Search results arrive with the Library in step 2. Press back once to close the search, and again to leave this page.")
		}}
	}
	return scrollPage(gtx, &p.list, []layout.Widget{
		func(gtx layout.Context) layout.Dimensions { return subBar(gtx, a, &p.back, &p.search) },
		func(gtx layout.Context) layout.Dimensions { return p.search.Bar(gtx, a, "Search…") },
	}, rows)
}

func (p *themePreview) section(th *Theme, label string, body layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: gutter, Right: gutter, Bottom: 16}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return eyebrow(gtx, th, label)
					})
				}),
				layout.Rigid(body),
			)
		})
	}
}
