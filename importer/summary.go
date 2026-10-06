package importer

import (
	"fmt"
	"strings"

	"github.com/saniyaa61/relic/core"
)

// Summary is what to compare against the prototype after an import: the
// Home counts, time totals and each category's Top 5.
type Summary struct {
	Entries, Finished, Ongoing, Favourites int
	TotalMinutes                           float64
	Categories                             []CategorySummary
}

// CategorySummary is one category's line in a Summary.
type CategorySummary struct {
	Name    string
	Entries int
	Minutes float64
	TopFive []string // titles, in order
}

// Summarize computes a Summary from a library.
func Summarize(lib *core.Library) Summary {
	x := core.NewTimeIndex(lib.Entries)
	s := Summary{Entries: len(lib.Entries), Ongoing: len(lib.StillWithYou()),
		Finished: len(lib.RecentlyFinished()), TotalMinutes: x.Sum(lib.Entries)}
	for _, e := range lib.Entries {
		if e.Favorite {
			s.Favourites++
		}
	}
	for _, c := range lib.Categories {
		in := lib.InCategory(c.ID)
		cs := CategorySummary{Name: c.Name, Entries: len(in), Minutes: x.Sum(in)}
		for _, e := range lib.TopFive(c.ID) {
			cs.TopFive = append(cs.TopFive, e.Title)
		}
		s.Categories = append(s.Categories, cs)
	}
	return s
}

// String lays the summary out for the import command.
func (s Summary) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Total %d · Finished %d · Ongoing %d · Favourites %d\n", s.Entries, s.Finished, s.Ongoing, s.Favourites)
	fmt.Fprintf(&b, "Consumed, all time: %s (%.1f min)\n", core.FormatDurationOrZero(s.TotalMinutes), s.TotalMinutes)
	for _, c := range s.Categories {
		fmt.Fprintf(&b, "  %-16s %3d entries  %s\n", c.Name, c.Entries, core.FormatDurationOrZero(c.Minutes))
		for i, t := range c.TopFive {
			fmt.Fprintf(&b, "      #%d %s\n", i+1, t)
		}
	}
	return b.String()
}
