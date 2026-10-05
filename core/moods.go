package core

import (
	"fmt"
	"sort"
	"time"
)

// MoodTrendTags is how many feelings Mood Trends shows.
const MoodTrendTags = 8

// TagTrend is one feeling's count per month in the window.
type TagTrend struct {
	Tag    string
	Counts []int // one per MoodData.Months
	Total  int
}

// Peak is the tag's largest single month count.
func (t TagTrend) Peak() int {
	p := 0
	for _, c := range t.Counts {
		if c > p {
			p = c
		}
	}
	return p
}

// MoodData is the Mood Trends page (SPEC §4.13).
type MoodData struct {
	Months []Month    // oldest first, at most 12, never before the Digest floor
	Tags   []TagTrend // every tag in the window, most frequent first (then A–Z)
}

// MoodTrends counts each entry's tags in the local month it was created,
// over the last 12 months ending this month but not before the floor month.
func (l *Library) MoodTrends(now time.Time, loc *time.Location) MoodData {
	floor := l.DigestBounds(now, loc).Floor
	this := MonthOf(now, loc)
	var d MoodData
	for i := 11; i >= 0; i-- {
		if m := this.Add(-i); !m.Before(floor) {
			d.Months = append(d.Months, m)
		}
	}
	idx := map[Month]int{}
	for i, m := range d.Months {
		idx[m] = i
	}
	byTag := map[string]*TagTrend{}
	for _, e := range l.Entries {
		i, ok := idx[MonthOf(e.CreatedAt, loc)]
		if !ok {
			continue
		}
		for _, t := range e.Tags {
			tt := byTag[t]
			if tt == nil {
				tt = &TagTrend{Tag: t, Counts: make([]int, len(d.Months))}
				byTag[t] = tt
			}
			tt.Counts[i]++
			tt.Total++
		}
	}
	for _, tt := range byTag {
		d.Tags = append(d.Tags, *tt)
	}
	sort.Slice(d.Tags, func(i, j int) bool {
		if d.Tags[i].Total != d.Tags[j].Total {
			return d.Tags[i].Total > d.Tags[j].Total
		}
		return d.Tags[i].Tag < d.Tags[j].Tag
	})
	return d
}

// EnoughMonths reports whether there are at least 2 months to show; if not,
// the page says "Trends take shape after a couple of months of logging."
func (d MoodData) EnoughMonths() bool { return len(d.Months) >= 2 }

// Shown is the top MoodTrendTags feelings.
func (d MoodData) Shown() []TagTrend {
	if len(d.Tags) > MoodTrendTags {
		return d.Tags[:MoodTrendTags]
	}
	return d.Tags
}

// MaxCount is the largest single month count across the shown tags: the
// height every bar is measured against.
func (d MoodData) MaxCount() int {
	m := 0
	for _, t := range d.Shown() {
		if p := t.Peak(); p > m {
			m = p
		}
	}
	return m
}

// MonthTop returns month i's top n feelings, most frequent that month
// first; ties go to the more frequent feeling overall.
func (d MoodData) MonthTop(i, n int) []string {
	type tc struct {
		tag string
		n   int
	}
	var list []tc
	for _, t := range d.Tags {
		if c := t.Counts[i]; c > 0 {
			list = append(list, tc{t.Tag, c})
		}
	}
	sort.SliceStable(list, func(a, b int) bool { return list[a].n > list[b].n })
	var out []string
	for j := 0; j < len(list) && j < n; j++ {
		out = append(out, list[j].tag)
	}
	return out
}

// Insight is one "What stands out" line. The tag is shown in italics between
// Before and After.
type Insight struct {
	Before, Tag, After string
}

// Insights are the "What stands out" lines (SPEC §4.13).
func (d MoodData) Insights() []Insight {
	if len(d.Tags) == 0 {
		return nil
	}
	top := d.Tags[0]
	unit := "entries"
	if top.Total == 1 {
		unit = "entry"
	}
	out := []Insight{{"Your most frequent feeling: ", top.Tag, fmt.Sprintf(" (%d %s).", top.Total, unit)}}
	n := len(d.Months)
	if n < 4 {
		return out
	}
	recentN := min(3, n/2)
	earlierN := n - recentN
	type delta struct {
		tag string
		d   float64
	}
	var ds []delta
	for _, t := range d.Tags {
		var r, e int
		for i, c := range t.Counts {
			if i >= earlierN {
				r += c
			} else {
				e += c
			}
		}
		ds = append(ds, delta{t.Tag, float64(r)/float64(recentN) - float64(e)/float64(earlierN)})
	}
	sort.SliceStable(ds, func(i, j int) bool { return ds[i].d > ds[j].d })
	up, down := ds[0], ds[len(ds)-1]
	if up.d >= 0.3 {
		out = append(out, Insight{"Lately you're leaning toward ", up.tag, "."})
	}
	if down.d <= -0.3 && down.tag != up.tag {
		out = append(out, Insight{"", down.tag, " has faded from your recent entries."})
	}
	return out
}
