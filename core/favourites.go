package core

import "errors"

var (
	ErrTopFiveFull = errors.New("Your Top 5 is full — remove one to add this.")
	ErrNotFavorite = errors.New("core: only favourites can join the Top 5")
)

// TopFiveMax is the size of each category's All-Time Top 5.
const TopFiveMax = 5

// Favourites holds the user's orderings, keyed by category id (SPEC §4.14).
type Favourites struct {
	TopFive map[string][]string // ordered entry ids, at most TopFiveMax
	Order   map[string][]string // custom order of the remaining favourites
}

func (f *Favourites) init() {
	if f.TopFive == nil {
		f.TopFive = map[string][]string{}
	}
	if f.Order == nil {
		f.Order = map[string][]string{}
	}
}

func (f *Favourites) forget(categoryID, id string) {
	f.init()
	f.TopFive[categoryID] = without(f.TopFive[categoryID], id)
	f.Order[categoryID] = without(f.Order[categoryID], id)
}

func (f *Favourites) appendOrder(categoryID, id string) {
	f.init()
	if indexOf(f.Order[categoryID], id) < 0 && indexOf(f.TopFive[categoryID], id) < 0 {
		f.Order[categoryID] = append(f.Order[categoryID], id)
	}
}

func without(list []string, id string) []string {
	i := indexOf(list, id)
	if i < 0 {
		return list
	}
	return append(list[:i:i], list[i+1:]...)
}

func indexOf(list []string, id string) int {
	for i, x := range list {
		if x == id {
			return i
		}
	}
	return -1
}

// ToggleFavorite flips an entry's favourite flag. Unfavouriting also takes it
// out of the Top 5.
func (l *Library) ToggleFavorite(id string) (favorite bool, err error) {
	e := l.Entry(id)
	if e == nil {
		return false, ErrNotFound
	}
	e.Favorite = !e.Favorite
	if e.Favorite {
		l.Favourites.appendOrder(e.CategoryID, id)
	} else {
		l.Favourites.forget(e.CategoryID, id)
	}
	return e.Favorite, nil
}

// TopFiveRank returns the entry's 1-based place in its category's Top 5.
func (l *Library) TopFiveRank(id string) (rank int, ok bool) {
	e := l.Entry(id)
	if e == nil {
		return 0, false
	}
	i := indexOf(l.Favourites.TopFive[e.CategoryID], id)
	return i + 1, i >= 0
}

// AddTopFive puts a favourite at the end of its category's Top 5.
func (l *Library) AddTopFive(id string) error {
	e := l.Entry(id)
	if e == nil {
		return ErrNotFound
	}
	if !e.Favorite {
		return ErrNotFavorite
	}
	list := l.Favourites.TopFive[e.CategoryID]
	if indexOf(list, id) >= 0 {
		return nil
	}
	if len(list) >= TopFiveMax {
		return ErrTopFiveFull
	}
	l.Favourites.init()
	l.Favourites.TopFive[e.CategoryID] = append(list, id)
	l.Favourites.Order[e.CategoryID] = without(l.Favourites.Order[e.CategoryID], id)
	return nil
}

// RemoveTopFive takes an entry out of the Top 5; it drops back to the bottom
// of the favourites list.
func (l *Library) RemoveTopFive(id string) error {
	e := l.Entry(id)
	if e == nil {
		return ErrNotFound
	}
	list := l.Favourites.TopFive[e.CategoryID]
	if indexOf(list, id) < 0 {
		return ErrNotFound
	}
	l.Favourites.TopFive[e.CategoryID] = without(list, id)
	l.Favourites.appendOrder(e.CategoryID, id)
	return nil
}

// MoveTopFive moves an entry one place left (dir -1) or right (+1). Moving
// past either end does nothing.
func (l *Library) MoveTopFive(id string, dir int) {
	e := l.Entry(id)
	if e == nil {
		return
	}
	list := l.Favourites.TopFive[e.CategoryID]
	i := indexOf(list, id)
	j := i + dir
	if i < 0 || j < 0 || j >= len(list) {
		return
	}
	list[i], list[j] = list[j], list[i]
}

// TopFive returns a category's Top 5 entries in order.
func (l *Library) TopFive(categoryID string) []*Entry {
	var out []*Entry
	for _, id := range l.Favourites.TopFive[categoryID] {
		if e := l.Entry(id); e != nil && e.Favorite && e.CategoryID == categoryID {
			out = append(out, e)
		}
	}
	return out
}

// FavouritesIn returns a category's favourites that aren't in its Top 5: the
// user's custom order first, then any others, newest first.
func (l *Library) FavouritesIn(categoryID string) []*Entry {
	top := l.Favourites.TopFive[categoryID]
	seen := map[string]bool{}
	var out []*Entry
	add := func(e *Entry) {
		if e != nil && e.Favorite && e.CategoryID == categoryID && !seen[e.ID] && indexOf(top, e.ID) < 0 {
			seen[e.ID] = true
			out = append(out, e)
		}
	}
	for _, id := range l.Favourites.Order[categoryID] {
		add(l.Entry(id))
	}
	for _, e := range l.Entries {
		add(e)
	}
	return out
}

// FavouriteCategories returns, in library order, the categories that have
// at least one favourite: the Favorites page tabs.
func (l *Library) FavouriteCategories() []*Category {
	var out []*Category
	for _, c := range l.Categories {
		for _, e := range l.Entries {
			if e.Favorite && e.CategoryID == c.ID {
				out = append(out, c)
				break
			}
		}
	}
	return out
}
