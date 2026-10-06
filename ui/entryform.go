package ui

// The New / Edit entry form (SPEC §5; prototype renderNew, saveEntry,
// updateEntryFromForm).

import (
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget"

	"github.com/saniyaa61/relic/core"
)

type entryForm struct {
	editID string // "" for a new entry

	back     IconButton
	list     widget.List
	catDrop  dropdown
	foldDrop dropdown
	catID    string
	title    Input
	poster   string
	upload   widget.Clickable
	status   core.Status
	statusOn [2]widget.Clickable
	// fields holds the type-specific inputs for fieldsType; switching
	// category to another type starts them afresh, as the prototype did.
	fieldsType core.EntryType
	fields     map[string]*Input
	dates      map[string]*dateField
	rating     float64
	stars      starInput
	review     Input
	tags       tagPicker
	save       widget.Clickable
	newCat     widget.Clickable
	started    bool
}

// newEntryPage returns an empty New form (opening New always starts
// afresh, SPEC §5).
func newEntryPage() Screen { return &entryForm{status: core.Finished} }

// editEntryPage returns the form filled in from an entry.
func editEntryPage(a *App, e *core.Entry) *entryForm {
	f := &entryForm{editID: e.ID, catID: e.CategoryID, poster: e.Poster, status: e.Status, rating: e.Rating}
	f.title.Editor.SetText(e.Title)
	f.review.Editor.SetText(e.Review)
	f.tags.tags = append([]string(nil), e.Tags...)
	f.resetFields(e.Type, e)
	if c := a.Lib.Category(e.CategoryID); c != nil {
		for i, name := range c.Folders {
			if name == e.Folder {
				f.foldDrop.selected = i + 1
			}
		}
	}
	f.started = true
	return f
}

// resetFields makes empty (or, from e, filled) inputs for type t.
func (f *entryForm) resetFields(t core.EntryType, e *core.Entry) {
	f.fieldsType = t
	f.fields, f.dates = map[string]*Input{}, map[string]*dateField{}
	for _, row := range rowsFor(t) {
		for _, s := range row {
			switch s.kind {
			case dateFieldKind:
				d := &dateField{}
				if e != nil {
					d.value = fieldDate(e.Fields, s.id)
				}
				f.dates[s.id] = d
			default:
				in := &Input{}
				in.Editor.SingleLine = true
				if s.kind == numberField {
					in.Editor.Filter = "0123456789"
					in.Editor.InputHint = keyHintNumeric
				}
				if e != nil {
					in.Editor.SetText(fieldText(e, s.id))
				}
				f.fields[s.id] = in
			}
		}
	}
}

func (f *entryForm) category(a *App) *core.Category {
	if c := a.Lib.Category(f.catID); c != nil {
		return c
	}
	if len(a.Lib.Categories) > 0 {
		f.catID = a.Lib.Categories[0].ID
		return a.Lib.Categories[0]
	}
	return nil
}

// input collects the form into a core.EntryInput.
func (f *entryForm) input(a *App, c *core.Category) core.EntryInput {
	in := core.EntryInput{
		Title:  f.title.Editor.Text(),
		Poster: f.poster,
		Rating: f.rating,
		Tags:   append([]string(nil), f.tags.tags...),
		Review: f.review.Editor.Text(),
		Status: f.status,
	}
	if i := f.foldDrop.selected; i > 0 && i <= len(c.Folders) {
		in.Folder = c.Folders[i-1]
	}
	for id, inp := range f.fields {
		setField(&in, id, inp.Editor.Text())
	}
	for id, d := range f.dates {
		setDate(&in, id, d.value)
	}
	return in
}

func (f *entryForm) submit(a *App) {
	c := f.category(a)
	if c == nil {
		return
	}
	in := f.input(a, c)
	if strings.TrimSpace(in.Title) == "" {
		a.ToastError(core.ErrTitleRequired.Error())
		return
	}
	if f.editID == "" {
		e, err := a.Lib.AddEntry(c.ID, in, a.Now())
		if err != nil {
			a.ToastError(err.Error())
			return
		}
		if a.save(false, []*core.Entry{e}, false) {
			a.Go(TabHome)
			a.Toast("Entry preserved ✦")
		}
		return
	}
	if err := a.Lib.UpdateEntry(f.editID, c.ID, in, a.Now()); err != nil {
		a.ToastError(err.Error())
		return
	}
	if a.save(false, []*core.Entry{a.Lib.Entry(f.editID)}, true) {
		a.removeUnusedPosters()
		a.Pop()
		a.Toast("Entry updated ✦")
	}
}

func (f *entryForm) leave(a *App) {
	if f.editID != "" {
		a.Pop()
	} else {
		a.Go(TabHome)
	}
}

func (f *entryForm) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	if f.back.Click.Clicked(gtx) {
		f.leave(a)
	}
	if f.newCat.Clicked(gtx) {
		a.Go(TabLibrary)
		newCategoryDialog(a)
	}
	header := []layout.Widget{func(gtx layout.Context) layout.Dimensions {
		return logoBar(gtx, th, func(gtx layout.Context) layout.Dimensions { return f.back.Layout(gtx, th, IconBack) })
	}}
	c := f.category(a)
	if c == nil {
		return scrollPage(gtx, &f.list, header, []layout.Widget{f.pad(func(gtx layout.Context) layout.Dimensions {
			return formSection(gtx, th, "", "", func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: 6, Bottom: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Bottom: 14}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								gtx.Constraints.Min.X = gtx.Constraints.Max.X
								return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 14, LineHeight: 1.65, Alignment: text.Middle, Color: th.Muted}.Layout(gtx, th,
									"You need at least one section before adding entries. Set up your library first.")
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return bigButton(gtx, th, &f.newCat, "+ Create a category")
						}),
					)
				})
			})
		})})
	}
	if !f.started {
		f.catID = c.ID
		f.resetFields(c.Type, nil)
		f.started = true
	}
	if f.fieldsType != c.Type {
		f.resetFields(c.Type, nil)
	}
	if f.upload.Clicked(gtx) {
		a.pickPoster(func(name string) { f.poster = name })
	}
	for i, s := range []core.Status{core.Ongoing, core.Finished} {
		if f.statusOn[i].Clicked(gtx) {
			f.status = s
		}
	}
	if f.save.Clicked(gtx) {
		f.submit(a)
	}

	heading := "What are you preserving?"
	saveLabel := "Preserve this memory →"
	if f.editID != "" {
		heading, saveLabel = "Edit this entry", "Save changes →"
	}
	rows := []layout.Widget{
		// What and where.
		f.pad(func(gtx layout.Context) layout.Dimensions {
			return formSection(gtx, th, heading, "", func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return f.placeRow(gtx, a, c) }),
					layout.Rigid(labelled(th, "Title", func(gtx layout.Context) layout.Dimensions {
						return f.title.Layout(gtx, th, withBg(formInput("Title"), th.Bg))
					})),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return pairRow(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions { return fieldLabel(gtx, th, "Poster") }),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions { return uploadBox(gtx, a, &f.upload, f.poster, 106) }))
						}, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions { return fieldLabel(gtx, th, "Status") }),
								layout.Rigid(layout.Spacer{Height: 4}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return toggleButton(gtx, th, &f.statusOn[0], "Ongoing", f.status == core.Ongoing, 11)
								}),
								layout.Rigid(layout.Spacer{Height: 6}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return toggleButton(gtx, th, &f.statusOn[1], "Finished", f.status == core.Finished, 11)
								}))
						}, 1.2, 1)
					}),
				)
			})
		}),
		// Details for this type.
		f.pad(func(gtx layout.Context) layout.Dimensions {
			return formSection(gtx, th, "Details", c.Type.Label(), func(gtx layout.Context) layout.Dimensions {
				return f.detailRows(gtx, a)
			})
		}),
		f.pad(func(gtx layout.Context) layout.Dimensions {
			return formSection(gtx, th, "Your rating", "", func(gtx layout.Context) layout.Dimensions {
				f.rating = f.stars.Layout(gtx, th, f.rating)
				return layout.Dimensions{Size: gtxSize(gtx, gtx.Sp(24)+gtx.Dp(2))}
			})
		}),
		f.pad(func(gtx layout.Context) layout.Dimensions {
			return formSection(gtx, th, "Your words", "", func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return f.review.Layout(gtx, th, InputStyle{Bg: th.Bg, Radius: 9, Size: 13,
							Pad:  layout.Inset{Top: 9, Bottom: 9, Left: 11, Right: 11},
							Font: font.Font{Typeface: Serif, Style: font.Italic}, Multiline: true, MinHeight: 140, LineHeight: 1.6,
							Placeholder:      "How did it make you feel? What will stay with you? Write freely — this is just for you.",
							PlaceholderAlpha: 0.7})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						n := len(strings.Fields(f.review.Editor.Text()))
						if n == 0 {
							return layout.Dimensions{}
						}
						return layout.Inset{Top: 3, Right: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Constraints.Max.X
							return Paragraph{Font: font.Font{Typeface: Serif, Style: font.Italic}, Size: 11, Alignment: text.End, Color: th.Muted}.Layout(gtx, th, plural(n, "word", "words"))
						})
					}))
			})
		}),
		f.pad(func(gtx layout.Context) layout.Dimensions {
			return formSection(gtx, th, "Feeling tags", "(pick up to 4)", func(gtx layout.Context) layout.Dimensions {
				return f.tags.Layout(gtx, th)
			})
		}),
		f.pad(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 4, Bottom: 100}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return bigButton(gtx, th, &f.save, saveLabel)
			})
		}),
	}
	rows[0] = func(w layout.Widget) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 12}.Layout(gtx, w)
		}
	}(rows[0])
	return scrollPage(gtx, &f.list, header, rows)
}

// pad gives a form row the page's 18dp gutters (.form-wrap).
func (f *entryForm) pad(w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: gutter, Right: gutter}.Layout(gtx, w)
	}
}

// placeRow is the Category and Folder pickers; Folder only shows when the
// category has folders.
func (f *entryForm) placeRow(gtx layout.Context, a *App, c *core.Category) layout.Dimensions {
	th := a.Theme
	names := make([]string, len(a.Lib.Categories))
	for i, cat := range a.Lib.Categories {
		names[i] = cat.Name
		if cat.ID == c.ID {
			f.catDrop.selected = i
		}
	}
	catField := labelled(th, "Category", func(gtx layout.Context) layout.Dimensions {
		d, changed := f.catDrop.Layout(gtx, a, names, nil, formSelect)
		if changed {
			f.catID = a.Lib.Categories[f.catDrop.selected].ID
			f.foldDrop.selected = 0
		}
		return d
	})
	if len(c.Folders) == 0 {
		return catField(gtx)
	}
	folders := append([]string{"— No folder —"}, c.Folders...)
	return pairRow(gtx, catField, labelled(th, "Folder", func(gtx layout.Context) layout.Dimensions {
		d, _ := f.foldDrop.Layout(gtx, a, folders, nil, formSelect)
		return d
	}), 1, 1)
}

// detailRows lays out the type's fields, one or two to a row.
func (f *entryForm) detailRows(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	field := func(s fieldSpec) layout.Widget {
		return labelled(th, s.label, func(gtx layout.Context) layout.Dimensions {
			if s.kind == dateFieldKind {
				return f.dates[s.id].Layout(gtx, a, s.label)
			}
			return f.fields[s.id].Layout(gtx, th, withBg(formInput(s.placeholder), th.Bg))
		})
	}
	var rows []layout.FlexChild
	for _, row := range rowsFor(f.fieldsType) {
		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(row) == 1 {
				return field(row[0])(gtx)
			}
			return pairRow(gtx, field(row[0]), field(row[1]), 1, 1)
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
}

func withBg(s InputStyle, bg colorNRGBA) InputStyle { s.Bg = bg; return s }
