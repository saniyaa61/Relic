package ui

// Settings (SPEC §5 "Settings", §8; prototype renderSettingsPage), opened
// from the gear on Home: theme, light/dark, your name, the archive and
// clearing everything.

import (
	"errors"
	"image"
	"log"
	"strings"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"

	"github.com/saniyaa61/relic/archive"
	"github.com/saniyaa61/relic/core"
	"github.com/saniyaa61/relic/importer"
	"github.com/saniyaa61/relic/store"
)

var (
	IconSun      = IconSVG(`<circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/>`)
	IconMoon     = IconSVG(`<path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/>`)
	IconDownload = IconSVG(`<path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>`)
	IconUpload   = IconSVG(`<path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/>`)
	iconTrashBig = IconSVG(`<polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6"/><path d="M10 11v6"/><path d="M14 11v6"/><path d="M9 6V4h6v2"/>`)
)

// clearRed is the prototype's Danger Zone red (#c0392b) and its border
// (#f0d0d0), the same in every theme.
var (
	clearRed    = rgb(0xC0392B)
	clearBorder = rgb(0xF0D0D0)
)

var themeTitles = map[string]string{"linen": "Linen", "midnight": "Midnight", "blush": "Blush",
	"forest": "Forest", "rose": "Rose", "slate": "Slate", "custom": "Custom"}

type settingsPage struct {
	back    IconButton
	list    widget.List
	themes  [7]widget.Clickable
	modes   [2]widget.Clickable
	editing bool
	edit    widget.Clickable
	name    Input
	export  widget.Clickable
	imp     widget.Clickable
	clear   widget.Clickable
	custom  customPicker
}

func (p *settingsPage) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	prof := a.Lib.Profile
	if p.back.Click.Clicked(gtx) {
		a.Back()
	}
	names := append(append([]string{}, ThemeNames...), "custom")
	for i := range p.themes {
		if p.themes[i].Clicked(gtx) {
			base, accent := prof.CustomBase, prof.CustomAccent
			if names[i] == "custom" && (base == "" || accent == "") {
				base, accent = defaultCustomBase, defaultCustomAccent
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
	if p.edit.Clicked(gtx) {
		if p.editing {
			p.saveName(a)
		} else {
			p.editing = true
			p.name.Editor.SetText(prof.Name)
			p.name.Focus(gtx)
			p.name.Editor.SetCaret(p.name.Editor.Len(), p.name.Editor.Len())
		}
	}
	if p.editing {
		for {
			ev, ok := p.name.Editor.Update(gtx)
			if !ok {
				break
			}
			if _, ok := ev.(widget.SubmitEvent); ok {
				p.saveName(a)
			}
		}
	}
	if p.export.Clicked(gtx) {
		exportArchive(a)
	}
	if p.imp.Clicked(gtx) {
		chooseArchive(a)
	}
	if p.clear.Clicked(gtx) {
		confirmClearAll(a)
	}

	header := []layout.Widget{func(gtx layout.Context) layout.Dimensions {
		return logoBar(gtx, th, func(gtx layout.Context) layout.Dimensions { return p.back.Layout(gtx, th, IconBack) })
	}}
	current := prof.Theme
	if current == "" {
		current = "linen"
	}
	cards := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions {
			return p.themeCard(gtx, a, names, current, prof.Mode == "dark")
		},
		func(gtx layout.Context) layout.Dimensions { return p.identityCard(gtx, a) },
		func(gtx layout.Context) layout.Dimensions { return p.archiveCard(gtx, th) },
		func(gtx layout.Context) layout.Dimensions { return p.dangerCard(gtx, th) },
		func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 6, Bottom: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 11, Color: th.Muted}.Layout(gtx, th, "Relic · your stories, preserved")
				})
			})
		},
	}
	// .form-wrap: 12px 18px 0, cards 10px apart; then 24px at the end.
	rows := []layout.Widget{layout.Spacer{Height: 12}.Layout}
	for i, c := range cards {
		if i > 0 {
			rows = append(rows, layout.Spacer{Height: 10}.Layout)
		}
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: gutter, Right: gutter}.Layout(gtx, c)
		})
	}
	rows = append(rows, layout.Spacer{Height: 24}.Layout)
	return scrollPage(gtx, &p.list, header, rows)
}

func (p *settingsPage) saveName(a *App) {
	p.editing = false
	// Like the prototype, an empty field keeps the old name.
	if v := strings.TrimSpace(p.name.Editor.Text()); v != "" {
		prof := a.Lib.Profile
		prof.Name = string([]rune(v)[:min(len([]rune(v)), 30)])
		a.Lib.Profile = prof
		a.Update(func(w store.Writer) error { return w.SaveProfile(prof) })
	}
	a.Toast("Name saved.")
}

// settingsCard is one of the page's cards: card colour, 16px corners,
// a Playfair title.
func settingsCard(gtx layout.Context, th *Theme, pad layout.Inset, title string, titleGap float32, body layout.Widget) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return card(gtx, th.Card, th.Border, 16, pad, 0, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unitDp(titleGap)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 13, Color: th.Text}.Layout(gtx, th, title)
				})
			}),
			layout.Rigid(body))
	})
}

// themeCard is "Appearance theme": the tiles, the custom colours and the
// light / dark switch.
func (p *settingsPage) themeCard(gtx layout.Context, a *App, names []string, current string, dark bool) layout.Dimensions {
	th := a.Theme
	return settingsCard(gtx, th, layout.UniformInset(16), "Appearance theme", 4, func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: 14}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return Text{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 12, Color: th.Muted}.Layout(gtx, th, "Choose your reading atmosphere")
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				// 3 columns, 10px apart, rows 10px apart.
				gap := gtx.Dp(10)
				w := (gtx.Constraints.Max.X - 2*gap) / 3
				y := 0
				for r := 0; r*3 < len(names); r++ {
					h := 0
					for c := 0; c < 3 && r*3+c < len(names); c++ {
						i := r*3 + c
						st := op.Offset(image.Pt(c*(w+gap), y)).Push(gtx.Ops)
						cgtx := gtx
						cgtx.Constraints = layout.Exact(image.Pt(w, gtx.Dp(44)+gtx.Dp(28)))
						d := p.themes[i].Layout(cgtx, func(gtx layout.Context) layout.Dimensions {
							return themeTile(gtx, a, names[i], names[i] == current, dark)
						})
						st.Pop()
						h = max(h, d.Size.Y)
					}
					y += h + gap
				}
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, y-gap+gtx.Dp(16))}
			}),
		}
		if current == "custom" {
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: 16}.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return p.custom.Layout(gtx, a) })
			}))
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions { return p.modeRow(gtx, th, dark) }))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

// themeTile is one theme's swatch (44px of its background with a dot of
// its accent) and name, outlined in accent when chosen.
func themeTile(gtx layout.Context, a *App, name string, on, dark bool) layout.Dimensions {
	th := a.Theme
	size := gtx.Constraints.Min
	r, b := gtx.Dp(12), gtx.Dp(2)
	border := th.Border
	if on {
		border = th.Accent
	}
	rrect(gtx, size, r, border)
	inner := size.Sub(image.Pt(2*b, 2*b))
	st := op.Offset(image.Pt(b, b)).Push(gtx.Ops)
	defer st.Pop()
	rrect(gtx, inner, r-b, th.Card)
	top := image.Pt(inner.X, gtx.Dp(44))
	customTileTop(gtx, a, name, top, r-b, dark)
	// The name, 10px, 6px 4px padding, under the swatch.
	lst := op.Offset(image.Pt(0, top.Y)).Push(gtx.Ops)
	lgtx := gtx
	lgtx.Constraints = layout.Exact(image.Pt(inner.X, inner.Y-top.Y))
	layout.Center.Layout(lgtx, func(gtx layout.Context) layout.Dimensions {
		return Text{Font: font.Font{Typeface: Sans}, Size: 10, Color: th.Muted}.Layout(gtx, th, themeTitles[name])
	})
	lst.Pop()
	return layout.Dimensions{Size: size}
}

// modeRow is "Appearance mode" with the Light / Dark pill switch.
func (p *settingsPage) modeRow(gtx layout.Context, th *Theme, dark bool) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return card(gtx, th.Surface, th.Border, 12, layout.Inset{Top: 11, Bottom: 11, Left: 14, Right: 14}, 0, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return Text{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 13, Color: th.Text}.Layout(gtx, th, "Appearance mode")
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: 1}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return Text{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 11, Color: th.Muted}.Layout(gtx, th, "Light or dark surfaces")
						})
					}))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return card(gtx, th.Tag, th.Tag, 20, layout.UniformInset(2), 0, func(gtx layout.Context) layout.Dimensions {
					btn := func(i int, ic *Icon, label string, on bool) layout.Widget {
						return func(gtx layout.Context) layout.Dimensions {
							return p.modes[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								bg, fg, w := th.Tag, th.Muted, font.Normal
								if on {
									bg, fg, w = th.Accent, th.BtnText, font.Medium
								}
								return card(gtx, bg, bg, 20, layout.Inset{Top: 4, Bottom: 4, Left: 11, Right: 11}, 0, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions { return ic.Layout(gtx, 11, 2, fg) }),
										layout.Rigid(layout.Spacer{Width: 4}.Layout),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return Text{Font: font.Font{Typeface: Sans, Weight: w}, Size: 11, Color: fg}.Layout(gtx, th, label)
										}))
								})
							})
						}
					}
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(btn(0, IconSun, "Light", !dark)),
						layout.Rigid(layout.Spacer{Width: 2}.Layout),
						layout.Rigid(btn(1, IconMoon, "Dark", dark)))
				})
			}))
	})
}

// identityCard is "Identity": your name (with edit / save) and the date
// the archive started.
func (p *settingsPage) identityCard(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	name := a.Lib.Profile.Name
	if name == "" {
		name = "—"
	}
	started := "—"
	if len(a.Lib.Entries) > 0 {
		first := a.Lib.Entries[0].CreatedAt
		for _, e := range a.Lib.Entries {
			if e.CreatedAt.Before(first) {
				first = e.CreatedAt
			}
		}
		started = core.FormatDate(first, a.Loc)
	}
	label := func(s string) layout.FlexChild {
		return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Sans}, Size: 10, LineHeight: normalLineHeight[Sans], Color: th.Muted}.Layout(gtx, th, s)
			})
		})
	}
	return settingsCard(gtx, th, layout.Inset{Top: 12, Bottom: 12, Left: 16, Right: 16}, "Identity", 4, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				d := layout.Inset{Top: 7, Bottom: 7}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								label("Your name"),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									if !p.editing {
										return Text{Font: font.Font{Typeface: Display, Style: font.Italic}, Size: 13, LineHeight: normalLineHeight[Display], Color: th.Text}.Layout(gtx, th, name)
									}
									gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(160))
									gtx.Constraints.Min.X = gtx.Constraints.Max.X
									return layout.Inset{Top: 3}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										p.name.Editor.SingleLine, p.name.Editor.Submit, p.name.Editor.MaxLen = true, true, 30
										return p.name.Layout(gtx, th, InputStyle{Bg: th.Bg, Radius: 9, Size: 13,
											Font: font.Font{Typeface: Display, Style: font.Italic},
											Pad:  layout.Inset{Top: 4, Bottom: 4, Left: 8, Right: 8}, Placeholder: "Your name"})
									})
								}))
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							word := "edit"
							if p.editing {
								word = "save"
							}
							return p.edit.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return card(gtx, th.Surface, th.Border, 20, layout.Inset{Top: 3, Bottom: 3, Left: 10, Right: 10}, 0, func(gtx layout.Context) layout.Dimensions {
									return Text{Font: font.Font{Typeface: Sans}, Size: 11, Color: th.Muted}.Layout(gtx, th, word)
								})
							})
						}))
				})
				fillRect(gtx, image.Rect(0, d.Size.Y, gtx.Constraints.Max.X, d.Size.Y+gtx.Dp(1)), th.Border)
				d.Size.Y += gtx.Dp(1)
				return d
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 7}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						label("Archive started"),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return Text{Font: font.Font{Typeface: Display}, Size: 13, LineHeight: normalLineHeight[Display], Color: th.Text}.Layout(gtx, th, started)
						}))
				})
			}))
	})
}

// archiveCard is "Archive" with Export and Import side by side.
func (p *settingsPage) archiveCard(gtx layout.Context, th *Theme) layout.Dimensions {
	return settingsCard(gtx, th, layout.Inset{Top: 12, Bottom: 12, Left: 16, Right: 16}, "Archive", 10, func(gtx layout.Context) layout.Dimensions {
		btn := func(c *widget.Clickable, ic *Icon, label string) layout.Widget {
			return func(gtx layout.Context) layout.Dimensions {
				return pressable(gtx, c, func(gtx layout.Context) layout.Dimensions {
					border := th.Border
					if c.Pressed() {
						border = th.Accent
					}
					return card(gtx, th.Surface, border, 12, layout.Inset{Top: 10, Bottom: 8, Left: 8, Right: 8}, 0, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return iconChip(gtx, th, ic, 28, 14, 13, 1.6) }),
							layout.Rigid(layout.Spacer{Height: 6}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return Text{Font: font.Font{Typeface: Display, Style: font.Italic}, Size: 12, Color: th.Accent}.Layout(gtx, th, label)
							}))
					})
				})
			}
		}
		return gridRow(gtx, gtx.Dp(8), 0, []layout.Widget{btn(&p.export, IconDownload, "Export archive"), btn(&p.imp, IconUpload, "Import archive")})
	})
}

// dangerCard is "Danger Zone" with Clear all data.
func (p *settingsPage) dangerCard(gtx layout.Context, th *Theme) layout.Dimensions {
	return settingsCard(gtx, th, layout.UniformInset(16), "Danger Zone", 14, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return pressable(gtx, &p.clear, func(gtx layout.Context) layout.Dimensions {
			bg := th.Card
			if p.clear.Pressed() {
				bg = mixSRGB(clearRed, th.Card, 0.06)
			}
			return card(gtx, bg, clearBorder, 12, layout.Inset{Top: 12, Bottom: 12, Left: 14, Right: 14}, 0, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return iconTrashBig.Layout(gtx, 16, 1.6, clearRed) }),
					layout.Rigid(layout.Spacer{Width: 10}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return Text{Font: font.Font{Typeface: Sans}, Size: 13, FakeItalic: true, Color: clearRed}.Layout(gtx, th, "Clear all data")
					}))
			})
		})
	})
}

// ---------- Export, import, clear ----------

func exportArchive(a *App) {
	if a.CreateFile == nil {
		a.ToastError("Saving isn't available here yet.")
		return
	}
	lib, dir, now := a.Lib, a.PosterDir, a.Now()
	a.background(func() func(*App) {
		wc, err := a.CreateFile(archive.FileName)
		if errors.Is(err, ErrNoFile) {
			return nil
		}
		if err == nil {
			err = archive.Write(wc, lib, dir, now)
			if cerr := wc.Close(); err == nil {
				err = cerr
			}
		}
		if err != nil {
			log.Printf("relic: export: %v", err)
			return func(a *App) { a.ToastError("The archive couldn't be saved.") }
		}
		return func(a *App) { a.Toast("Archive exported.") }
	})
}

// chooseArchive picks a file, reads it, then asks before replacing
// everything (the prototype's order).
func chooseArchive(a *App) {
	if a.ChooseFile == nil {
		a.ToastError("Choosing a file isn't available here yet.")
		return
	}
	a.background(func() func(*App) {
		rc, err := a.ChooseFile(".json")
		if errors.Is(err, ErrNoFile) {
			return nil
		}
		var res *importer.Result
		if err == nil {
			res, err = archive.Read(rc, time.Now())
			rc.Close()
		}
		if err != nil {
			log.Printf("relic: import: %v", err)
			return func(a *App) { a.ToastError("Could not read that file.") }
		}
		return func(a *App) {
			a.ShowDialog(ConfirmDialog("Import this archive?", "Your current data will be replaced.",
				"Cancel", "Import", func(a *App) {
					if a.replaceLibrary(res) {
						a.Toast("Archive imported.")
					}
				}))
		}
	})
}

// replaceLibrary swaps in an imported library: its posters, its theme (but
// not light/dark, owner's decision), and every page starts afresh.
func (a *App) replaceLibrary(res *importer.Result) bool {
	res.KeepMode(a.Lib.Profile)
	if a.DB != nil {
		if a.PosterDir != "" {
			if err := store.WritePosters(a.PosterDir, res.Posters); err != nil {
				log.Printf("relic: import posters: %v", err)
				a.ToastError("Could not read that file.")
				return false
			}
		}
		if err := a.DB.ReplaceAll(res.Library); err != nil {
			log.Printf("relic: import: %v", err)
			a.ToastError("Could not read that file.")
			return false
		}
	}
	a.Lib = res.Library
	a.restart()
	if a.DB != nil && a.PosterDir != "" {
		if err := store.RemoveUnusedPosters(a.PosterDir, a.Lib); err != nil {
			log.Printf("relic: posters: %v", err)
		}
	}
	return true
}

// restart rebuilds every page for the current library and theme and goes
// Home.
func (a *App) restart() {
	a.Theme.Palette = PaletteFor(a.Lib.Profile)
	a.forgetPosters()
	a.roots = newRoots()
	a.stack, a.tab, a.sheet = nil, TabHome, nil
	a.shownAt = a.Now()
}

// confirmClearAll asks twice (SPEC §5) before deleting everything.
func confirmClearAll(a *App) {
	a.ShowDialog(ConfirmDialog("Clear all data?",
		"This will permanently delete all your entries and settings. Are you sure?",
		"Keep it", "Clear all data", func(a *App) {
			a.ShowDialog(ConfirmDialog("Really clear everything?",
				"There's no undo. Export your archive first if you might want it back.",
				"Keep it", "Yes, clear it all", func(a *App) {
					if a.clearAll() {
						a.Toast("All data cleared.")
					}
				}))
		}))
}

// clearAll deletes every category, entry, favourite and the name; the
// theme and light/dark stay.
func (a *App) clearAll() bool {
	p := a.Lib.Profile
	lib := &core.Library{Profile: core.Profile{Theme: p.Theme, Mode: p.Mode, CustomBase: p.CustomBase,
		CustomAccent: p.CustomAccent, FirstUsedAt: a.Now()}}
	if a.DB != nil {
		if err := a.DB.ReplaceAll(lib); err != nil {
			log.Printf("relic: clear: %v", err)
			a.ToastError("Couldn't clear your data.")
			return false
		}
		if a.PosterDir != "" {
			store.RemoveUnusedPosters(a.PosterDir, lib)
		}
	}
	a.Lib = lib
	a.restart()
	return true
}
