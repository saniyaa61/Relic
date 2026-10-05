package core

import (
	"sort"
	"time"
)

// TimeEvent is some time spent with an entry, at a moment (SPEC §4.5).
type TimeEvent struct {
	At      time.Time
	Minutes float64
}

// BookPageMinutes is the estimate for a page read when no time was logged.
const BookPageMinutes = 1.5

// TimeEvents lists the time the entry has taken, per SPEC §4.5. Every total
// and chart in the app is a sum over these.
func (e *Entry) TimeEvents() []TimeEvent {
	var ev []TimeEvent
	add := func(at time.Time, m float64) {
		if m > 0 {
			ev = append(ev, TimeEvent{at, m})
		}
	}
	f := e.Fields
	switch e.Type {
	case Series, Podcast:
		per := float64(f.EpisodeDuration)
		for _, s := range e.Sessions {
			add(s.At, float64(s.Episodes)*per)
		}
		for _, r := range e.Rewatches {
			if r.Full {
				add(r.At, float64(f.TotalEpisodes)*per)
			} else {
				add(r.At, float64(r.Episodes)*per)
			}
		}
	case Book:
		for _, s := range e.Sessions {
			add(s.At, bookSessionMinutes(s))
		}
	case Other:
		for _, s := range e.Sessions {
			if s.Minutes != nil {
				add(s.At, float64(*s.Minutes))
			}
		}
	case Film, Short:
		add(e.CreatedAt, float64(f.Duration))
		for _, r := range e.Rewatches {
			add(r.At, float64(f.Duration))
		}
	case Music:
		m := musicMinutes(f)
		add(e.CreatedAt, m)
		for _, r := range e.Rewatches {
			add(r.At, m)
		}
	}
	return ev
}

// TrackMinutes is the estimated length of one track when an album's length
// isn't given.
const TrackMinutes = 3.5

// musicMinutes is the album length, or tracks × 3.5 min when it's blank.
func musicMinutes(f Fields) float64 {
	if f.Duration > 0 {
		return float64(f.Duration)
	}
	return float64(f.Tracks) * TrackMinutes
}

// bookSessionMinutes is the logged time, or 1.5 minutes per page read.
func bookSessionMinutes(s Session) float64 {
	if s.Minutes != nil {
		return float64(*s.Minutes)
	}
	if s.ToPage > s.FromPage {
		return float64(s.ToPage-s.FromPage) * BookPageMinutes
	}
	return 0
}

// TimeIndex holds every entry's time events, worked out once per render and
// shared by all totals and charts on the page.
type TimeIndex struct {
	events map[string][]TimeEvent
	totals map[string]float64
}

// NewTimeIndex computes the time events of each entry.
func NewTimeIndex(entries []*Entry) *TimeIndex {
	x := &TimeIndex{events: make(map[string][]TimeEvent, len(entries)), totals: make(map[string]float64, len(entries))}
	for _, e := range entries {
		ev := e.TimeEvents()
		x.events[e.ID] = ev
		var t float64
		for _, v := range ev {
			t += v.Minutes
		}
		x.totals[e.ID] = t
	}
	return x
}

// Events returns an entry's time events.
func (x *TimeIndex) Events(id string) []TimeEvent { return x.events[id] }

// Total is an entry's all-time minutes.
func (x *TimeIndex) Total(id string) float64 { return x.totals[id] }

// Sum is the all-time minutes of the given entries.
func (x *TimeIndex) Sum(entries []*Entry) float64 {
	var t float64
	for _, e := range entries {
		t += x.totals[e.ID]
	}
	return t
}

// InRange is the entry's minutes with start <= At < end.
func (x *TimeIndex) InRange(id string, start, end time.Time) float64 {
	var t float64
	for _, v := range x.events[id] {
		if inRange(v.At, start, end) {
			t += v.Minutes
		}
	}
	return t
}

// SumInRange is InRange summed over the given entries.
func (x *TimeIndex) SumInRange(entries []*Entry, start, end time.Time) float64 {
	var t float64
	for _, e := range entries {
		t += x.InRange(e.ID, start, end)
	}
	return t
}

// Monthly returns minutes per local month of year (index 0 = January).
func (x *TimeIndex) Monthly(entries []*Entry, year int, loc *time.Location) [12]float64 {
	var out [12]float64
	for _, e := range entries {
		for _, v := range x.events[e.ID] {
			if t := v.At.In(loc); t.Year() == year {
				out[t.Month()-1] += v.Minutes
			}
		}
	}
	return out
}

// Weekly returns minutes in each of the last n rolling 7-day windows ending
// at now, oldest first (the Consumed sparkline uses n = 8).
func (x *TimeIndex) Weekly(entries []*Entry, now time.Time, n int) []float64 {
	out := make([]float64, n)
	week := 7 * 24 * time.Hour
	start := now.Add(-time.Duration(n) * week)
	for _, e := range entries {
		for _, v := range x.events[e.ID] {
			if inRange(v.At, start, now) {
				if i := int(v.At.Sub(start) / week); i >= 0 && i < n {
					out[i] += v.Minutes
				}
			}
		}
	}
	return out
}

// GroupTime is a category or folder with its time.
type GroupTime struct {
	Name    string // folder name; "" = Uncategorised
	Minutes float64
	Entries []*Entry
}

// CategoryTime is one Consumed section: a category with its folders.
type CategoryTime struct {
	Category *Category
	Minutes  float64
	Folders  []GroupTime // with time only, most time first; "" = Uncategorised
}

// ConsumedByCategory groups all-time minutes by category and folder for the
// Consumed page. Only groups with time appear, most time first. An entry
// whose folder no longer exists counts as Uncategorised, so category totals
// always equal the sum of their folders.
func (l *Library) ConsumedByCategory(x *TimeIndex) []CategoryTime {
	var out []CategoryTime
	for _, c := range l.Categories {
		groups := map[string]*GroupTime{}
		var order []string
		var total float64
		for _, e := range l.Entries {
			if e.CategoryID != c.ID {
				continue
			}
			f := e.Folder
			if folderIndex(c, f) < 0 {
				f = ""
			}
			g := groups[f]
			if g == nil {
				g = &GroupTime{Name: f}
				groups[f] = g
				order = append(order, f)
			}
			m := x.Total(e.ID)
			g.Minutes += m
			g.Entries = append(g.Entries, e)
			total += m
		}
		if total <= 0 {
			continue
		}
		ct := CategoryTime{Category: c, Minutes: total}
		for _, f := range order {
			if g := groups[f]; g.Minutes > 0 {
				ct.Folders = append(ct.Folders, *g)
			}
		}
		sort.SliceStable(ct.Folders, func(i, j int) bool { return ct.Folders[i].Minutes > ct.Folders[j].Minutes })
		out = append(out, ct)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Minutes > out[j].Minutes })
	return out
}

// ShareSlice is one segment of a share bar.
type ShareSlice struct {
	Name    string // category name, or "Other"
	Minutes float64
	Other   bool
}

// ShareBar is the top 3 categories plus "Other" for the rest (Consumed hero,
// Year card). cats must be sorted by minutes, most first.
func ShareBar(cats []CategoryTime) []ShareSlice {
	var out []ShareSlice
	var rest float64
	for i, c := range cats {
		if i < 3 {
			out = append(out, ShareSlice{Name: c.Category.Name, Minutes: c.Minutes})
		} else {
			rest += c.Minutes
		}
	}
	if rest > 0 {
		out = append(out, ShareSlice{Name: "Other", Minutes: rest, Other: true})
	}
	return out
}
