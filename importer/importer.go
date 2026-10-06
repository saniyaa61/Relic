package importer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/saniyaa61/relic/core"
)

// Result is a converted archive, ready for store.ReplaceAll.
type Result struct {
	Library *core.Library
	// Posters maps a poster file name (as set on Entry.Poster) to its
	// resized JPEG bytes.
	Posters map[string][]byte
	Report  Report
}

// Report says what the import found and what it had to repair.
type Report struct {
	Categories, Entries, Sessions, Rewatches, Posters int
	Warnings                                          []string
}

func (r *Report) warn(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

// Themes the app knows; anything else falls back to linen.
var themes = map[string]bool{"linen": true, "midnight": true, "blush": true, "forest": true,
	"rose": true, "slate": true, "custom": true}

// Read converts a prototype archive. now is used only when a date is missing
// everywhere.
func Read(r io.Reader, now time.Time) (*Result, error) {
	var a archive
	if err := json.NewDecoder(r).Decode(&a); err != nil {
		return nil, fmt.Errorf("importer: not a Relic archive: %w", err)
	}
	if a.Sections == nil && a.Entries == nil {
		return nil, errors.New("importer: not a Relic archive: no sections or entries")
	}
	im := &importer{res: &Result{Library: &core.Library{}, Posters: map[string][]byte{}}, now: now.UTC(),
		byName: map[string]*core.Category{}}
	im.fallback = im.parseTime(a.CreatedAt, now.UTC())
	im.profile(a)
	im.categories(a)
	for i := range a.Entries {
		im.entry(&a.Entries[i])
	}
	im.favourites(a)
	im.firstUse()

	lib := im.res.Library
	sort.SliceStable(lib.Entries, func(i, j int) bool { return lib.Entries[i].CreatedAt.After(lib.Entries[j].CreatedAt) })
	rep := &im.res.Report
	rep.Categories, rep.Entries, rep.Posters = len(lib.Categories), len(lib.Entries), len(im.res.Posters)
	for _, e := range lib.Entries {
		rep.Sessions += len(e.Sessions)
		rep.Rewatches += len(e.Rewatches)
	}
	return im.res, nil
}

type importer struct {
	res      *Result
	now      time.Time
	fallback time.Time // the archive's own date, for records with none
	byName   map[string]*core.Category
	ids      map[string]bool
}

func (im *importer) warn(format string, args ...any) { im.res.Report.warn(format, args...) }

func (im *importer) parseTime(s str, def time.Time) time.Time {
	if s.trim() == "" {
		return def
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000Z07:00", "2006-01-02"} {
		if t, err := time.Parse(layout, s.trim()); err == nil {
			return t.UTC()
		}
	}
	return def
}

func (im *importer) profile(a archive) {
	p := &im.res.Library.Profile
	p.Name = strings.TrimSpace(a.UserName)
	p.Theme = strings.ToLower(strings.TrimSpace(a.Theme))
	if !themes[p.Theme] {
		if p.Theme != "" {
			im.warn("Unknown theme %q; using Linen.", a.Theme)
		}
		p.Theme = "linen"
	}
	// The prototype didn't export light/dark.
	p.Mode = "light"
	p.CustomBase, p.CustomAccent = a.CustomBase, a.CustomAccent
	if p.Theme == "custom" && (p.CustomBase == "" || p.CustomAccent == "") {
		im.warn("Custom theme without both colours; using Linen.")
		p.Theme = "linen"
	}
}

// inferType mirrors the prototype's guess for sections saved without a type.
func inferType(name string) core.EntryType {
	n := strings.ToLower(name)
	has := func(words ...string) bool {
		for _, w := range words {
			if strings.Contains(n, w) {
				return true
			}
		}
		return false
	}
	switch {
	case has("podcast"):
		return core.Podcast
	case has("reel", "short", "youtube", "video", "clip"):
		return core.Short
	case has("music", "album", "song", "concert", "gig"):
		return core.Music
	case has("book", "novel", "read"):
		return core.Book
	case has("drama", "series", "show", "tv", "anime", "episode"):
		return core.Series
	}
	return core.Film
}

func (im *importer) categories(a archive) {
	lib := im.res.Library
	for _, s := range a.Sections {
		name := s.Name.trim()
		if name == "" {
			im.warn("Skipped a category with no name.")
			continue
		}
		if im.byName[strings.ToLower(name)] != nil {
			im.warn("Two categories are called %q; their entries are kept together.", name)
			continue
		}
		t := core.EntryType(s.Type.trim())
		if !t.Valid() {
			t = inferType(name)
		}
		c := &core.Category{ID: s.ID.trim(), Name: name, Type: t, CreatedAt: im.idTime(s.ID.trim(), "s")}
		if c.ID == "" || im.category(c.ID) != nil {
			c.ID = core.NewID()
		}
		for _, f := range s.Folders {
			im.addFolder(c, f.trim())
		}
		lib.Categories = append(lib.Categories, c)
		im.byName[strings.ToLower(name)] = c
	}
}

func (im *importer) category(id string) *core.Category { return im.res.Library.Category(id) }

// idTime reads the creation time the prototype put in its ids ("s1780829186718").
func (im *importer) idTime(id, prefix string) time.Time {
	ms, err := strconv.ParseInt(strings.TrimPrefix(id, prefix), 10, 64)
	if err != nil || ms <= 0 {
		return im.fallback
	}
	return time.UnixMilli(ms).UTC()
}

func (im *importer) addFolder(c *core.Category, f string) {
	if f == "" {
		return
	}
	for _, x := range c.Folders {
		if strings.EqualFold(x, f) {
			return
		}
	}
	c.Folders = append(c.Folders, f)
}

// categoryFor finds an entry's category by name, creating one if it's gone.
func (im *importer) categoryFor(e *entry) *core.Category {
	name := e.Section.trim()
	if name == "" {
		name = e.Category.trim()
	}
	if c := im.byName[strings.ToLower(name)]; c != nil {
		return c
	}
	if name == "" {
		name = "Imported"
	}
	if c := im.byName[strings.ToLower(name)]; c != nil {
		return c
	}
	t := core.EntryType(e.Type.trim())
	if !t.Valid() {
		t = inferType(name)
	}
	c := &core.Category{ID: core.NewID(), Name: name, Type: t, CreatedAt: im.fallback}
	im.res.Library.Categories = append(im.res.Library.Categories, c)
	im.byName[strings.ToLower(name)] = c
	im.warn("%q was in a category %q that no longer existed; it has been recreated.", e.Title.trim(), name)
	return c
}

func (im *importer) entry(a *entry) {
	title := a.Title.trim()
	if title == "" {
		title = "Untitled"
		im.warn("An entry had no title; it is called \"Untitled\".")
	}
	cat := im.categoryFor(a)
	t := core.EntryType(a.Type.trim())
	if !t.Valid() {
		t = cat.Type
	} else if t != cat.Type {
		im.warn("%q is a %s inside %q (a %s category); kept as %s.", title, t, cat.Name, cat.Type, t)
	}

	if im.ids == nil {
		im.ids = map[string]bool{}
	}
	id := a.ID.trim()
	if id == "" || im.ids[id] {
		id = core.NewID()
	}
	im.ids[id] = true

	created := im.parseTime(a.CreatedAt, time.Time{})
	if created.IsZero() {
		created = im.idTime(id, "e")
		im.warn("%q had no creation date; used %s.", title, created.Format("2 January 2006"))
	}

	e := &core.Entry{ID: id, CategoryID: cat.ID, Folder: a.Folder.trim(), Type: t, Title: title,
		Rating: rating(a.Rating), Review: a.Review.String(), CreatedAt: created, Favorite: a.Favorite}
	if a.Rating.ok && e.Rating != a.Rating.v {
		im.warn("%q: rating %v rounded to %v.", title, a.Rating.v, e.Rating)
	}
	im.addFolder(cat, e.Folder)
	for _, tg := range a.Tags {
		if s := tg.trim(); s != "" {
			e.Tags = append(e.Tags, s)
		}
	}

	switch strings.ToLower(a.Status.trim()) {
	case "finished":
		e.Status = core.Finished
		e.FinishedAt = im.parseTime(a.FinishedAt, created)
	case "ongoing", "watching", "reading":
		e.Status = core.Ongoing
	default:
		e.Status = core.Ongoing
		im.warn("%q had status %q; it is now Ongoing.", title, a.Status.trim())
	}

	e.Fields = im.fields(a, t, title)
	im.sessions(e, a)
	im.rewatches(e, a)
	im.poster(e, a)
	im.checkProgress(e, a)
	im.res.Library.Entries = append(im.res.Library.Entries, e)
}

// rating rounds to the nearest half star within 0–5.
func rating(n num) float64 {
	if !n.ok {
		return 0
	}
	return math.Max(0, math.Min(5, math.Round(n.v*2)/2))
}

func (im *importer) date(s str, title, field string) core.Date {
	d, err := core.ParseDate(s.trim())
	if err != nil {
		// Accept full timestamps too.
		if t := im.parseTime(s, time.Time{}); !t.IsZero() {
			return core.DateOf(t, time.UTC)
		}
		im.warn("%q: couldn't read %s %q; left blank.", title, field, s.trim())
		return core.Date{}
	}
	return d
}

func (im *importer) count(n num, title, field string) int {
	v := n.int()
	if v < 0 {
		im.warn("%q: %s was %d; set to 0.", title, field, v)
		return 0
	}
	return v
}

func (im *importer) fields(a *entry, t core.EntryType, title string) core.Fields {
	f := core.Fields{
		Director: a.Director.trim(), Language: a.Language.trim(), Cast: a.Cast.trim(),
		Platform: a.Platform.trim(), Author: a.Author.trim(), Publisher: a.Publisher.trim(),
		Host: a.Host.trim(), Creator: a.Creator.trim(), Artist: a.Artist.trim(), Genre: a.Genre.trim(),
		URL:             a.URL.trim(),
		EpisodeDuration: im.count(a.EpisodeDuration, title, "episode length"),
		TotalEpisodes:   im.count(a.TotalEpisodes, title, "total episodes"),
		WatchedDate:     im.date(a.WatchedDate, title, "date"),
		StartDate:       im.date(a.StartDate, title, "start date"),
	}
	switch t {
	case core.Music:
		// The prototype kept a music entry's track count in totalPages.
		f.Tracks = im.count(a.TotalPages, title, "tracks")
	default:
		f.TotalPages = im.count(a.TotalPages, title, "total pages")
	}
	if t == core.Other {
		f.DurationText = a.Duration.trim()
	} else if d := a.Duration.trim(); d != "" {
		var n num
		n.UnmarshalJSON([]byte(strconv.Quote(d)))
		if n.ok {
			f.Duration = im.count(n, title, "duration")
		} else {
			im.warn("%q: duration %q isn't a number of minutes; left blank.", title, d)
		}
	}
	return f
}

func (im *importer) sessions(e *core.Entry, a *entry) {
	others := 0
	for _, s := range a.Sessions {
		cs := core.Session{
			ID:              core.NewID(),
			At:              im.parseTime(s.Date, e.CreatedAt),
			Note:            s.Note.String(),
			IsStart:         s.First,
			StartedFinished: s.First && s.DoneAtStart,
			Episodes:        im.count(s.Eps, e.Title, "session episodes"),
			FromPage:        im.count(s.From, e.Title, "from page"),
			ToPage:          im.count(s.To, e.Title, "to page"),
		}
		// The prototype saved a blank minutes box as 0; 0 there means "not logged".
		if m := im.count(s.Mins, e.Title, "minutes"); m > 0 {
			cs.Minutes = core.Mins(m)
		}
		if cs.IsStart && e.StartSession() != nil {
			im.warn("%q had more than one first session; the extra one is now a normal session.", e.Title)
			cs.IsStart, cs.StartedFinished = false, false
		}
		if !cs.IsStart {
			others += cs.Episodes
		}
		e.Sessions = append(e.Sessions, cs)
	}

	// Repair: entries from before start sessions existed (SPEC §9).
	if e.StartSession() == nil && (strings.TrimSpace(e.Review) != "" || e.Type.TracksProgress()) {
		s := core.Session{ID: core.NewID(), At: e.CreatedAt, Note: e.Review, IsStart: true,
			StartedFinished: e.Status == core.Finished}
		switch e.Type.Progress() {
		case core.EpisodeProgress:
			s.Episodes = max(0, im.count(a.WatchedEpisodes, e.Title, "episodes watched")-others)
		case core.PageProgress:
			s.ToPage = im.count(a.PagesRead, e.Title, "pages read")
		}
		e.Sessions = append(e.Sessions, s)
	}

	sort.SliceStable(e.Sessions, func(i, j int) bool {
		a, b := e.Sessions[i], e.Sessions[j]
		if a.At.Equal(b.At) {
			return a.IsStart && !b.IsStart
		}
		return a.At.Before(b.At)
	})
}

func (im *importer) rewatches(e *core.Entry, a *entry) {
	for _, r := range a.Rewatches {
		cr := core.Rewatch{ID: core.NewID(), At: im.parseTime(r.Date, e.CreatedAt), Note: r.Note.String(),
			Rating: rating(r.Rating), Full: r.Full == nil || *r.Full,
			StartDate: im.date(r.Start, e.Title, "reread start"), EndDate: im.date(r.End, e.Title, "reread end")}
		e.Rewatches = append(e.Rewatches, cr)
	}
	sort.SliceStable(e.Rewatches, func(i, j int) bool { return e.Rewatches[i].At.Before(e.Rewatches[j].At) })
}

// checkProgress notes where the prototype's stored progress disagrees with
// its sessions: the new app derives progress, so the number shown may change.
func (im *importer) checkProgress(e *core.Entry, a *entry) {
	switch e.Type.Progress() {
	case core.EpisodeProgress:
		if a.WatchedEpisodes.ok && a.WatchedEpisodes.int() != e.WatchedEpisodes() {
			im.warn("%q: the prototype showed %d episodes watched; its sessions add up to %d, which is what Relic will show.",
				e.Title, a.WatchedEpisodes.int(), e.WatchedEpisodes())
		}
	case core.PageProgress:
		if a.PagesRead.ok && a.PagesRead.int() != e.PagesRead() {
			im.warn("%q: the prototype showed page %d; its sessions reach page %d, which is what Relic will show.",
				e.Title, a.PagesRead.int(), e.PagesRead())
		}
	}
}

func (im *importer) poster(e *core.Entry, a *entry) {
	src := a.Poster.trim()
	if src == "" {
		return
	}
	jpg, err := resizePoster(src)
	if err != nil {
		im.warn("%q: poster couldn't be read (%v); skipped.", e.Title, err)
		return
	}
	name := e.ID + ".jpg"
	e.Poster = name
	im.res.Posters[name] = jpg
}

// favourites maps the prototype's per-section-name lists to category ids,
// keeping only favourited entries that are still in that category.
func (im *importer) favourites(a archive) {
	lib := im.res.Library
	lib.Favourites = core.Favourites{TopFive: map[string][]string{}, Order: map[string][]string{}}
	valid := func(listName, section, id string, cat *core.Category) bool {
		e := lib.Entry(id)
		if e == nil || !e.Favorite || e.CategoryID != cat.ID {
			im.warn("%s for %q listed an entry that isn't a favourite there any more; dropped.", listName, section)
			return false
		}
		return true
	}
	for _, section := range sortedKeys(a.TopFavorites) {
		ids := a.TopFavorites[section]
		cat := im.byName[strings.ToLower(strings.TrimSpace(section))]
		if cat == nil {
			continue
		}
		for _, id := range ids {
			if !valid("Top 5", section, id, cat) || indexOf(lib.Favourites.TopFive[cat.ID], id) {
				continue
			}
			if len(lib.Favourites.TopFive[cat.ID]) == core.TopFiveMax {
				im.warn("Top 5 for %q had more than five; the extras moved to Favourites.", section)
				break
			}
			lib.Favourites.TopFive[cat.ID] = append(lib.Favourites.TopFive[cat.ID], id)
		}
	}
	for _, section := range sortedKeys(a.FavoriteOrder) {
		ids := a.FavoriteOrder[section]
		cat := im.byName[strings.ToLower(strings.TrimSpace(section))]
		if cat == nil {
			continue
		}
		for _, id := range ids {
			if !valid("Favourites order", section, id, cat) || indexOf(lib.Favourites.TopFive[cat.ID], id) ||
				indexOf(lib.Favourites.Order[cat.ID], id) {
				continue
			}
			lib.Favourites.Order[cat.ID] = append(lib.Favourites.Order[cat.ID], id)
		}
	}
}

func indexOf(list []string, id string) bool {
	for _, x := range list {
		if x == id {
			return true
		}
	}
	return false
}

// firstUse isn't in the export; the earliest entry stands in for it (SPEC §9).
func (im *importer) firstUse() {
	lib := im.res.Library
	var first time.Time
	for _, e := range lib.Entries {
		if first.IsZero() || e.CreatedAt.Before(first) {
			first = e.CreatedAt
		}
	}
	if first.IsZero() {
		first = im.fallback
	}
	lib.Profile.FirstUsedAt = first
}

// sortedKeys keeps the import (and its warnings) in a stable order.
func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
