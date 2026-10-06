package ui

// The Library (SPEC §5, Phase 3 step 2): the categories grid, a
// category's page with its folders and Uncategorised entries, and a
// folder's page. Prototype: renderLibraryRoot, renderCategory,
// renderFolder.

import (
	"errors"
	"fmt"
	"image"
	"math"
	"strings"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"

	"github.com/saniyaa61/relic/core"
)

// ---------- Library root: the categories ----------

type libraryRoot struct {
	search  Search
	list    widget.List
	cards   map[string]*boxCard
	addCat  widget.Clickable
	results entryList
}

// boxCard is the click and ☰ menu state of one category or folder box.
type boxCard struct {
	click widget.Clickable
	menu  cardMenu
}

func newBoxCard() *boxCard { return &boxCard{menu: cardMenu{items: renameDeleteItems()}} }

func (p *libraryRoot) card(id string) *boxCard {
	if p.cards == nil {
		p.cards = map[string]*boxCard{}
	}
	if p.cards[id] == nil {
		p.cards[id] = newBoxCard()
	}
	return p.cards[id]
}

func (p *libraryRoot) WantsBack() bool { return p.search.WantsBack() || anyMenuOpen(p.cards) }
func (p *libraryRoot) Back(a *App) {
	if closeMenus(p.cards) {
		return
	}
	p.search.Back(a)
}

func anyMenuOpen(cards map[string]*boxCard) bool {
	for _, c := range cards {
		if c.menu.open {
			return true
		}
	}
	return false
}

func closeMenus(cards map[string]*boxCard) bool {
	closed := false
	for _, c := range cards {
		if c.menu.open {
			c.menu.open = false
			closed = true
		}
	}
	return closed
}

func (p *libraryRoot) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	lib := a.Lib
	if p.addCat.Clicked(gtx) {
		newCategoryDialog(a)
	}
	header := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions {
			return logoBar(gtx, th, func(gtx layout.Context) layout.Dimensions {
				if len(lib.Categories) == 0 {
					return layout.Dimensions{}
				}
				return p.search.Button(gtx, a)
			})
		},
		func(gtx layout.Context) layout.Dimensions { return p.search.Bar(gtx, a, "Search your library…") },
	}
	if q := p.search.Query(); strings.TrimSpace(q) != "" {
		return scrollPage(gtx, &p.list, header, searchResults(a, &p.results, lib.Entries, q, "", IconBook))
	}

	sub := "Nothing here yet. Build it exactly how your mind works — every category is yours to invent."
	if n := len(lib.Categories); n > 0 {
		sub = plural(n, "category", "categories") + " · " + plural(len(lib.Entries), "entry", "entries") + " preserved"
	}
	rows := []layout.Widget{func(gtx layout.Context) layout.Dimensions {
		return pageHead(gtx, th, "Your library", "Categories", sub)
	}}
	cats := lib.Categories
	for i := 0; i < len(cats); i += 2 {
		pair := cats[i:min(i+2, len(cats))]
		top := 2
		if i > 0 {
			top = 10
		}
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unitDp(float32(top)), Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				cells := make([]layout.Widget, len(pair))
				for j, c := range pair {
					cells[j] = func(gtx layout.Context) layout.Dimensions {
						return cardDrop(gtx, a.Now(), a.shownAt, i+j, func(gtx layout.Context) layout.Dimensions {
							return p.categoryBox(gtx, a, c)
						})
					}
				}
				return gridRow(gtx, gtx.Dp(10), 110, cells)
			})
		})
	}
	rows = append(rows, func(gtx layout.Context) layout.Dimensions {
		top := 2
		if len(cats) > 0 {
			top = 10
		}
		return layout.Inset{Top: unitDp(float32(top)), Bottom: 16, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return addTile(gtx, th, &p.addCat, "New category", false)
		})
	})
	return scrollPage(gtx, &p.list, header, rows)
}

// categoryBox is the prototype's .cat-card: icon chip, name, the fanned
// stack of the three newest posters, and the ☰ menu.
func (p *libraryRoot) categoryBox(gtx layout.Context, a *App, c *core.Category) layout.Dimensions {
	th := a.Theme
	bc := p.card(c.ID)
	if bc.click.Clicked(gtx) {
		p.search.closeNow()
		a.Push(&categoryPage{catID: c.ID})
	}
	covers := a.Lib.CoverEntries(c.ID, 3)
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	d := bc.click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		border := th.Border
		if bc.click.Pressed() {
			border = th.Accent2
		}
		return card(gtx, th.Card, border, 14, layout.UniformInset(14), 0, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.Y = max(gtx.Constraints.Min.Y, 0)
					return spreadColumn(gtx,
						func(gtx layout.Context) layout.Dimensions {
							return iconChip(gtx, th, typeIcon(c.Type), 36, 10, 17, 1.5)
						},
						func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Right: 18}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return Paragraph{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 14, LineHeight: 1.3, MaxLines: 2, Color: th.Text}.Layout(gtx, th, c.Name)
							})
						})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if len(covers) == 0 {
						return layout.Dimensions{}
					}
					return layout.Inset{Left: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return posterFan(gtx, a, covers)
					})
				}),
			)
		})
	})
	switch bc.menu.Layout(gtx, a, d.Size.X) {
	case 0:
		renameCategoryDialog(a, c.ID)
	case 1:
		deleteCategoryDialog(a, c.ID)
	}
	return d
}

// posterFan draws up to three posters as the prototype's .cat-cover-side:
// 38×52 tiles with a 2px card-coloured edge and a soft shadow, the newest
// flat in front, the next two dropped 6 and 12px and tilted 6° and 12°,
// each overlapping the one before by 18px, the whole stack nudged 3px
// left and 4px down.
func posterFan(gtx layout.Context, a *App, covers []*core.Entry) layout.Dimensions {
	th := a.Theme
	tile := image.Pt(gtx.Dp(38), gtx.Dp(52))
	step := gtx.Dp(38 - 18)
	w := tile.X + step*(len(covers)-1)
	size := image.Pt(w, tile.Y)
	defer op.Offset(image.Pt(-gtx.Dp(3), gtx.Dp(4))).Push(gtx.Ops).Pop()
	for i := len(covers) - 1; i >= 0; i-- { // back to front
		x := float32(step * i)
		y := float32(gtx.Dp(6) * i)
		angle := float32(6*i) * math.Pi / 180
		centre := f32.Pt(x+float32(tile.X)/2, y+float32(tile.Y)/2)
		tr := f32.AffineId().Offset(f32.Pt(x, y)).Rotate(centre, angle)
		st := op.Affine(tr).Push(gtx.Ops)
		boxShadow(gtx, tile, gtx.Dp(8), 2, 6, 0.2, th.Card)
		rrect(gtx, tile, gtx.Dp(8), th.Card)
		edge := gtx.Dp(2)
		inner := tile.Sub(image.Pt(2*edge, 2*edge))
		ist := op.Offset(image.Pt(edge, edge)).Push(gtx.Ops)
		if !a.drawPoster(gtx, covers[i].Poster, inner, gtx.Dp(6)) {
			rrect(gtx, inner, gtx.Dp(6), th.Tag)
		}
		ist.Pop()
		st.Pop()
	}
	return layout.Dimensions{Size: size}
}

// iconChip is a rounded tag-coloured square with an accent line icon
// (.cat-ico 36px / .fc-ico 32px).
func iconChip(gtx layout.Context, th *Theme, ic *Icon, box, radius, icon, stroke float32) layout.Dimensions {
	sz := gtx.Dp(unitDp(box))
	rrect(gtx, image.Pt(sz, sz), gtx.Dp(unitDp(radius)), th.Tag)
	off := (sz - gtx.Dp(unitDp(icon))) / 2
	st := op.Offset(image.Pt(off, off)).Push(gtx.Ops)
	ic.Layout(gtx, unitDp(icon), stroke, th.Accent)
	st.Pop()
	return layout.Dimensions{Size: image.Pt(sz, sz)}
}

// spreadColumn puts top at the top and bottom at the bottom of the
// available height (CSS justify-content: space-between).
func spreadColumn(gtx layout.Context, top, bottom layout.Widget) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceBetween}.Layout(gtx,
		layout.Rigid(top), layout.Rigid(bottom))
}

// gridRow lays out cells side by side in equal columns with gap px
// between, all as tall as the tallest and at least minH dp (a CSS grid
// row). Cells beyond the first may be missing: the row keeps its columns.
func gridRow(gtx layout.Context, gap, minH int, cells []layout.Widget) layout.Dimensions {
	const cols = 2
	w := (gtx.Constraints.Max.X - gap*(cols-1)) / cols
	h := gtx.Dp(unitDp(float32(minH)))
	// Measure first, so every cell can be laid out at the row's height.
	// Measuring draws into scratch ops, so nothing it adds (such as a
	// deferred menu) reaches the screen.
	var scratch op.Ops
	for _, c := range cells {
		cgtx := gtx
		cgtx.Ops = &scratch
		cgtx.Constraints = layout.Constraints{Min: image.Pt(w, h), Max: image.Pt(w, gtx.Constraints.Max.Y)}
		d := c(cgtx)
		h = max(h, d.Size.Y)
	}
	for i, c := range cells {
		cgtx := gtx
		cgtx.Constraints = layout.Exact(image.Pt(w, h))
		st := op.Offset(image.Pt(i*(w+gap), 0)).Push(gtx.Ops)
		c(cgtx)
		st.Pop()
	}
	return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
}

// ---------- Category page: folders and Uncategorised ----------

type categoryPage struct {
	catID     string
	back      IconButton
	search    Search
	list      widget.List
	folders   map[string]*boxCard
	addFolder widget.Clickable
	rows      entryList
	results   entryList
}

func (p *categoryPage) ShowsNav() bool  { return true }
func (p *categoryPage) WantsBack() bool { return p.search.WantsBack() || anyMenuOpen(p.folders) }
func (p *categoryPage) Back(a *App) {
	if closeMenus(p.folders) {
		return
	}
	p.search.Back(a)
}

func (p *categoryPage) folder(name string) *boxCard {
	if p.folders == nil {
		p.folders = map[string]*boxCard{}
	}
	if p.folders[name] == nil {
		p.folders[name] = newBoxCard()
	}
	return p.folders[name]
}

func (p *categoryPage) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	c := a.Lib.Category(p.catID)
	if c == nil { // deleted meanwhile
		a.Pop()
		return layout.Dimensions{}
	}
	if p.addFolder.Clicked(gtx) {
		newFolderDialog(a, c.ID)
	}
	header := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions { return subBar(gtx, a, &p.back, &p.search) },
		func(gtx layout.Context) layout.Dimensions { return p.search.Bar(gtx, a, "Search in "+c.Name+"…") },
	}
	if q := p.search.Query(); strings.TrimSpace(q) != "" {
		return scrollPage(gtx, &p.list, header, searchResults(a, &p.results, a.Lib.InCategory(c.ID), q, c.Name, IconBook))
	}

	all := a.Lib.InCategory(c.ID)
	loose := a.Lib.InFolder(c.ID, "")
	rows := []layout.Widget{func(gtx layout.Context) layout.Dimensions {
		return pageHead(gtx, th, "Category · "+c.Type.Label(), c.Name,
			plural(len(c.Folders), "folder", "folders")+" · "+plural(len(all), "entry", "entries"))
	}}
	if len(c.Folders) > 0 {
		cells := make([]layout.Widget, 0, len(c.Folders)+1)
		for i, f := range c.Folders {
			cells = append(cells, func(gtx layout.Context) layout.Dimensions {
				return cardDrop(gtx, a.Now(), a.shownAt, i, func(gtx layout.Context) layout.Dimensions {
					return p.folderBox(gtx, a, c, f)
				})
			})
		}
		cells = append(cells, func(gtx layout.Context) layout.Dimensions {
			return addTile(gtx, th, &p.addFolder, "New folder", true)
		})
		for i := 0; i < len(cells); i += 2 {
			pair := cells[i:min(i+2, len(cells))]
			top, bottom := 0, 0
			if i == 0 {
				top = 4
			} else {
				top = 9
			}
			if i+2 >= len(cells) {
				bottom = 12
			}
			rows = append(rows, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unitDp(float32(top)), Bottom: unitDp(float32(bottom)), Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return gridRow(gtx, gtx.Dp(9), 100, pair)
				})
			})
		}
	} else {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: 14, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return addTile(gtx, th, &p.addFolder, "Create your first folder", false)
			})
		})
	}
	switch {
	case len(loose) > 0:
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 6, Bottom: 10, Left: gutter, Right: gutter}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Text{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 16, Color: th.Text}.Layout(gtx, th, "Uncategorised")
			})
		})
		rows = append(rows, p.rows.widgets(a, loose)...)
	case len(c.Folders) == 0:
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return emptyState(gtx, th, IconFolder, "Nothing in "+c.Name+" yet. Add a folder above, or log an entry.")
			})
		})
	}
	rows = append(rows, layout.Spacer{Height: 10}.Layout)
	return scrollPage(gtx, &p.list, header, rows)
}

// folderBox is the prototype's .fc: folder icon chip, name at the bottom,
// ☰ menu.
func (p *categoryPage) folderBox(gtx layout.Context, a *App, c *core.Category, name string) layout.Dimensions {
	th := a.Theme
	bc := p.folder(name)
	if bc.click.Clicked(gtx) {
		p.search.closeNow()
		a.Push(&folderPage{catID: c.ID, folder: name})
	}
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	d := bc.click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		border := th.Border
		if bc.click.Pressed() {
			border = th.Accent2
		}
		return card(gtx, th.Card, border, 14, layout.UniformInset(14), 0, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return spreadColumn(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Bottom: 9}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return iconChip(gtx, th, IconFolder, 32, 9, 16, 1.4)
					})
				},
				func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Right: 20}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return Paragraph{Font: font.Font{Typeface: Display, Weight: font.Medium}, Size: 13, LineHeight: 1.3, MaxLines: 2, Color: th.Text}.Layout(gtx, th, name)
					})
				})
		})
	})
	switch bc.menu.Layout(gtx, a, d.Size.X) {
	case 0:
		renameFolderDialog(a, c.ID, name)
	case 1:
		deleteFolderDialog(a, c.ID, name)
	}
	return d
}

// ---------- Folder page ----------

type folderPage struct {
	catID, folder string
	back          IconButton
	search        Search
	list          widget.List
	rows          entryList
	results       entryList
}

func (p *folderPage) ShowsNav() bool  { return true }
func (p *folderPage) WantsBack() bool { return p.search.WantsBack() }
func (p *folderPage) Back(a *App)     { p.search.Back(a) }

func (p *folderPage) Layout(gtx layout.Context, a *App) layout.Dimensions {
	th := a.Theme
	c := a.Lib.Category(p.catID)
	if c == nil || folderGone(c, p.folder) {
		a.Pop()
		return layout.Dimensions{}
	}
	header := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions { return subBar(gtx, a, &p.back, &p.search) },
		func(gtx layout.Context) layout.Dimensions { return p.search.Bar(gtx, a, "Search in "+p.folder+"…") },
	}
	entries := a.Lib.InFolder(c.ID, p.folder)
	if q := p.search.Query(); strings.TrimSpace(q) != "" {
		return scrollPage(gtx, &p.list, header, searchResults(a, &p.results, entries, q, p.folder, IconBook))
	}
	rows := []layout.Widget{func(gtx layout.Context) layout.Dimensions {
		return pageHead(gtx, th, c.Name+" · Folder", p.folder, plural(len(entries), "entry", "entries"))
	}}
	if len(entries) == 0 {
		rows = append(rows, func(gtx layout.Context) layout.Dimensions {
			return emptyState(gtx, th, IconFolder, "Nothing in "+p.folder+" yet.")
		})
	} else {
		rows = append(rows, p.rows.widgets(a, entries)...)
	}
	rows = append(rows, layout.Spacer{Height: 10}.Layout)
	return scrollPage(gtx, &p.list, header, rows)
}

func folderGone(c *core.Category, name string) bool {
	for _, f := range c.Folders {
		if f == name {
			return false
		}
	}
	return true
}

// ---------- Dialogs: create, rename, delete (prototype wording) ----------

func newCategoryDialog(a *App) {
	t := core.Film
	var d *Dialog
	d = PromptDialog("New category",
		"Call it whatever makes sense to you — Films, Comfort Rewatches, Books for the train.",
		"Category name", "e.g. Films", "", "Create",
		func(a *App, name string) {
			if _, err := a.Lib.AddCategory(name, d.ChosenType(), a.Now()); err != nil {
				nameError(a, err, "You already have that category")
				return
			}
			if a.save(true, nil, false) {
				a.Toast("Category created")
			}
		})
	d.Type = &t
	a.ShowDialog(d)
}

func renameCategoryDialog(a *App, id string) {
	c := a.Lib.Category(id)
	if c == nil {
		return
	}
	t := c.Type
	var d *Dialog
	d = PromptDialog("Rename category", "", "Category name", "", c.Name, "Save",
		func(a *App, name string) {
			c := a.Lib.Category(id)
			if c == nil {
				return
			}
			for _, o := range a.Lib.Categories {
				if o.ID != id && strings.EqualFold(o.Name, name) {
					a.ToastError("You already have that category")
					return
				}
			}
			if err := a.Lib.RenameCategory(id, name); err != nil {
				nameError(a, err, "You already have that category")
				return
			}
			a.Lib.SetCategoryType(id, d.ChosenType())
			if a.save(true, nil, false) {
				a.Toast("Category updated")
			}
		})
	d.Type = &t
	a.ShowDialog(d)
}

func deleteCategoryDialog(a *App, id string) {
	c := a.Lib.Category(id)
	if c == nil {
		return
	}
	n := len(a.Lib.InCategory(id))
	sub := "The category and its folders will be removed. This can't be undone."
	if n > 0 {
		sub = fmt.Sprintf("This category holds %s. Deleting it removes them too — this can't be undone.", plural(n, "entry", "entries"))
	}
	a.ShowDialog(ConfirmDialog(fmt.Sprintf("Delete \"%s\"?", c.Name), sub, "", "Delete category", func(a *App) {
		if a.Lib.DeleteCategory(id) != nil {
			return
		}
		if a.save(true, nil, true) {
			a.removeUnusedPosters()
			a.Toast("Category deleted")
		}
	}))
}

func newFolderDialog(a *App, catID string) {
	c := a.Lib.Category(catID)
	if c == nil {
		return
	}
	a.ShowDialog(PromptDialog("New folder",
		"A shelf inside "+c.Name+" — by mood, year, whatever you like.",
		"Folder name", "e.g. Rainy Sundays", "", "Create",
		func(a *App, name string) {
			if err := a.Lib.AddFolder(catID, name); err != nil {
				nameError(a, err, "That folder already exists")
				return
			}
			if a.save(true, nil, false) {
				a.Toast("Folder created")
			}
		}))
}

func renameFolderDialog(a *App, catID, folder string) {
	a.ShowDialog(PromptDialog("Rename folder", "", "Folder name", "", folder, "Save",
		func(a *App, name string) {
			if name == folder {
				return
			}
			moved := a.Lib.InFolder(catID, folder)
			if err := a.Lib.RenameFolder(catID, folder, name); err != nil {
				nameError(a, err, "That folder already exists")
				return
			}
			if a.save(true, moved, false) {
				a.Toast("Folder renamed")
			}
		}))
}

func deleteFolderDialog(a *App, catID, folder string) {
	n := len(a.Lib.InFolder(catID, folder))
	sub := "This empty folder will be removed."
	if n > 0 {
		sub = plural(n, "entry", "entries") + " will move back to Uncategorised — nothing is lost."
	}
	a.ShowDialog(ConfirmDialog(fmt.Sprintf("Delete \"%s\"?", folder), sub, "", "Delete folder", func(a *App) {
		moved := a.Lib.InFolder(catID, folder)
		if a.Lib.DeleteFolder(catID, folder) != nil {
			return
		}
		if a.save(true, moved, false) {
			a.Toast("Folder deleted")
		}
	}))
}

// nameError shows taken for a clash, or the core's own message.
func nameError(a *App, err error, taken string) {
	if errors.Is(err, core.ErrNameTaken) {
		a.ToastError(taken)
		return
	}
	a.ToastError(err.Error())
}

// ---------- Entry placeholder (the real page is step 3) ----------

type entryPlaceholder struct {
	entry *core.Entry
	back  IconButton
	list  widget.List
}

func (p *entryPlaceholder) Layout(gtx layout.Context, a *App) layout.Dimensions {
	return scrollPage(gtx, &p.list, []layout.Widget{
		func(gtx layout.Context) layout.Dimensions { return subBar(gtx, a, &p.back, nil) },
	}, []layout.Widget{func(gtx layout.Context) layout.Dimensions {
		return pageHead(gtx, a.Theme, "Entry", p.entry.Title, "The entry page — your words, your journey — arrives in step 3.")
	}})
}
