package core

import (
	"errors"
	"sort"
	"strings"
	"time"
)

var (
	ErrNameRequired = errors.New("Please give it a name.")
	ErrNameTaken    = errors.New("That name is already in use.")
	ErrBadType      = errors.New("core: unknown entry type")
)

// Library is everything the user has: profile, categories, entries and
// favourites. Entries are kept newest first.
type Library struct {
	Profile    Profile
	Categories []*Category
	Entries    []*Entry
	Favourites Favourites
}

// Category returns the category with the given id, or nil.
func (l *Library) Category(id string) *Category {
	for _, c := range l.Categories {
		if c.ID == id {
			return c
		}
	}
	return nil
}

// CategoryName returns the name of the category with the given id, or "".
func (l *Library) CategoryName(id string) string {
	if c := l.Category(id); c != nil {
		return c.Name
	}
	return ""
}

// Entry returns the entry with the given id, or nil.
func (l *Library) Entry(id string) *Entry {
	for _, e := range l.Entries {
		if e.ID == id {
			return e
		}
	}
	return nil
}

func sameName(a, b string) bool { return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b)) }

// AddCategory creates a category. Names are unique, ignoring case.
func (l *Library) AddCategory(name string, t EntryType, now time.Time) (*Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrNameRequired
	}
	if !t.Valid() {
		return nil, ErrBadType
	}
	for _, c := range l.Categories {
		if sameName(c.Name, name) {
			return nil, ErrNameTaken
		}
	}
	c := &Category{ID: NewID(), Name: name, Type: t, CreatedAt: now.UTC()}
	l.Categories = append(l.Categories, c)
	return c, nil
}

// RenameCategory renames a category. Entries and favourites refer to it by
// id, so nothing else changes.
func (l *Library) RenameCategory(id, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrNameRequired
	}
	c := l.Category(id)
	if c == nil {
		return ErrNotFound
	}
	for _, o := range l.Categories {
		if o.ID != id && sameName(o.Name, name) {
			return ErrNameTaken
		}
	}
	c.Name = name
	return nil
}

// DeleteCategory removes a category with all its entries and their Top 5 and
// favourite order (SPEC §4.16).
func (l *Library) DeleteCategory(id string) error {
	i := l.categoryIndex(id)
	if i < 0 {
		return ErrNotFound
	}
	l.Categories = append(l.Categories[:i], l.Categories[i+1:]...)
	kept := l.Entries[:0]
	for _, e := range l.Entries {
		if e.CategoryID != id {
			kept = append(kept, e)
		}
	}
	l.Entries = kept
	delete(l.Favourites.TopFive, id)
	delete(l.Favourites.Order, id)
	return nil
}

func (l *Library) categoryIndex(id string) int {
	for i, c := range l.Categories {
		if c.ID == id {
			return i
		}
	}
	return -1
}

func folderIndex(c *Category, name string) int {
	for i, f := range c.Folders {
		if sameName(f, name) {
			return i
		}
	}
	return -1
}

// AddFolder adds a folder to a category. Names are unique within the
// category, ignoring case.
func (l *Library) AddFolder(categoryID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrNameRequired
	}
	c := l.Category(categoryID)
	if c == nil {
		return ErrNotFound
	}
	if folderIndex(c, name) >= 0 {
		return ErrNameTaken
	}
	c.Folders = append(c.Folders, name)
	return nil
}

// RenameFolder renames a folder and moves its entries with it.
func (l *Library) RenameFolder(categoryID, old, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrNameRequired
	}
	c := l.Category(categoryID)
	if c == nil {
		return ErrNotFound
	}
	i := folderIndex(c, old)
	if i < 0 {
		return ErrNotFound
	}
	if j := folderIndex(c, name); j >= 0 && j != i {
		return ErrNameTaken
	}
	was := c.Folders[i]
	c.Folders[i] = name
	for _, e := range l.Entries {
		if e.CategoryID == categoryID && e.Folder == was {
			e.Folder = name
		}
	}
	return nil
}

// DeleteFolder removes a folder; its entries move to Uncategorised.
func (l *Library) DeleteFolder(categoryID, name string) error {
	c := l.Category(categoryID)
	if c == nil {
		return ErrNotFound
	}
	i := folderIndex(c, name)
	if i < 0 {
		return ErrNotFound
	}
	was := c.Folders[i]
	c.Folders = append(c.Folders[:i], c.Folders[i+1:]...)
	for _, e := range l.Entries {
		if e.CategoryID == categoryID && e.Folder == was {
			e.Folder = ""
		}
	}
	return nil
}

// AddEntry creates an entry (SPEC §4.1) and puts it at the top.
func (l *Library) AddEntry(categoryID string, in EntryInput, now time.Time) (*Entry, error) {
	e, err := NewEntry(l.Category(categoryID), in, now)
	if err != nil {
		return nil, err
	}
	l.Entries = append([]*Entry{e}, l.Entries...)
	return e, nil
}

// UpdateEntry saves the edit form. Moving to another category changes the
// entry's type and carries a favourite over to the new category's list.
func (l *Library) UpdateEntry(id, categoryID string, in EntryInput, now time.Time) error {
	e := l.Entry(id)
	if e == nil {
		return ErrNotFound
	}
	from := e.CategoryID
	if err := e.update(l.Category(categoryID), in, now); err != nil {
		return err
	}
	if from != e.CategoryID {
		l.Favourites.forget(from, id)
		if e.Favorite {
			l.Favourites.appendOrder(e.CategoryID, id)
		}
	}
	return nil
}

// DeleteEntry removes an entry and its favourite data.
func (l *Library) DeleteEntry(id string) error {
	for i, e := range l.Entries {
		if e.ID == id {
			l.Entries = append(l.Entries[:i], l.Entries[i+1:]...)
			l.Favourites.forget(e.CategoryID, id)
			return nil
		}
	}
	return ErrNotFound
}

// Scopes: the entry sets each page searches and lists (SPEC §4.9).

// InCategory returns the category's entries, newest first.
func (l *Library) InCategory(categoryID string) []*Entry {
	return l.filter(func(e *Entry) bool { return e.CategoryID == categoryID })
}

// InFolder returns a folder's entries; folder "" is Uncategorised.
func (l *Library) InFolder(categoryID, folder string) []*Entry {
	return l.filter(func(e *Entry) bool { return e.CategoryID == categoryID && e.Folder == folder })
}

// StillWithYou returns ongoing entries, newest first.
func (l *Library) StillWithYou() []*Entry {
	return l.filter(func(e *Entry) bool { return e.Status == Ongoing })
}

// RecentlyFinished returns finished entries, most recently finished first.
func (l *Library) RecentlyFinished() []*Entry {
	out := l.filter(func(e *Entry) bool { return e.Status == Finished })
	sort.SliceStable(out, func(i, j int) bool { return finishedOrCreated(out[i]).After(finishedOrCreated(out[j])) })
	return out
}

func finishedOrCreated(e *Entry) time.Time {
	if !e.FinishedAt.IsZero() {
		return e.FinishedAt
	}
	return e.CreatedAt
}

func (l *Library) filter(keep func(*Entry) bool) []*Entry {
	var out []*Entry
	for _, e := range l.Entries {
		if keep(e) {
			out = append(out, e)
		}
	}
	return out
}

// FirstActivity is min(firstUsedAt, earliest entry createdAt): where the
// Digest and Mood Trends start (SPEC §4.12). Zero if there is nothing.
func (l *Library) FirstActivity() time.Time {
	t := l.Profile.FirstUsedAt
	for _, e := range l.Entries {
		if t.IsZero() || e.CreatedAt.Before(t) {
			t = e.CreatedAt
		}
	}
	return t
}
