// Command numbers prints every number the app shows for an archive, as
// JSON in the same shape as tools/protonumbers.js, so the two can be
// compared (tools/check-numbers.sh). Times are bucketed in UTC, as the
// prototype run is.
//
//	go run ./cmd/numbers -archive importer/testdata/scrubbed-archive.json -now 2026-09-29T09:00:00Z
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/saniyaa61/relic/archive"
	"github.com/saniyaa61/relic/core"
)

type period struct {
	New         int      `json:"new"`
	Finished    int      `json:"finished"`
	Minutes     float64  `json:"minutes"`
	Pages       int      `json:"pages"`
	Sessions    int      `json:"sessions"`
	TopCategory string   `json:"topCategory"`
	TopTag      string   `json:"topTag"`
	TopRated    []string `json:"topRated"`
}

type group struct {
	Name    string  `json:"name"`
	Minutes float64 `json:"minutes"`
	Folders []group `json:"folders,omitempty"`
}

type year struct {
	period
	Monthly []float64 `json:"monthly"`
	Card    struct {
		Minutes    float64  `json:"minutes"`
		Categories []group  `json:"categories"`
		Started    int      `json:"started"`
		Finished   int      `json:"finished"`
		TopRated   []string `json:"topRated"`
		TopFeeling string   `json:"topFeeling"`
	} `json:"card"`
}

func r(v float64) float64 { return math.Round(v*1e6) / 1e6 }

func ids(es []*core.Entry) []string {
	out := []string{}
	for _, e := range es {
		out = append(out, e.ID)
	}
	return out
}

func main() {
	path := flag.String("archive", "", "archive to read")
	nowFlag := flag.String("now", "", "the clock (RFC 3339)")
	flag.Parse()
	now, err := time.Parse(time.RFC3339, *nowFlag)
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.Open(*path)
	if err != nil {
		log.Fatal(err)
	}
	res, err := archive.Read(f, now)
	if err != nil {
		log.Fatal(err)
	}
	lib := res.Library
	loc := time.UTC
	x := core.NewTimeIndex(lib.Entries)
	catName := map[string]string{}
	for _, c := range lib.Categories {
		catName[c.ID] = c.Name
	}
	out := map[string]any{}

	entries := map[string]any{}
	for _, e := range lib.Entries {
		m := map[string]any{"minutes": r(x.Total(e.ID)), "watched": nil, "pages": nil}
		switch e.Type {
		case core.Series, core.Podcast:
			m["watched"] = e.WatchedEpisodes()
		case core.Book:
			m["pages"] = e.PagesRead()
		}
		entries[e.ID] = m
	}
	out["entries"] = entries

	total := x.Sum(lib.Entries)
	cur, longest := core.Streak(lib.Entries, now, loc)
	consumed := core.FormatDuration(total)
	if consumed == "" {
		consumed = "0 min"
	}
	finished, ongoing := 0, 0
	for _, e := range lib.Entries {
		switch e.Status {
		case core.Finished:
			finished++
		case core.Ongoing:
			ongoing++
		}
	}
	out["home"] = map[string]any{"total": len(lib.Entries), "finished": finished, "ongoing": ongoing,
		"minutes": r(total), "consumed": consumed,
		"streak": map[string]int{"current": cur, "longest": longest},
		// What Home's chips and rows actually use.
		"stillWithYou": len(lib.StillWithYou()), "recentlyFinished": len(lib.RecentlyFinished())}

	this := core.MonthOf(now, loc)
	var cats []group
	for _, ct := range lib.ConsumedByCategory(x) {
		g := group{Name: ct.Category.Name, Minutes: r(ct.Minutes), Folders: []group{}}
		for _, fo := range ct.Folders {
			g.Folders = append(g.Folders, group{Name: fo.Name, Minutes: r(fo.Minutes)})
		}
		cats = append(cats, g)
	}
	weekly := x.Weekly(lib.Entries, now, 8)
	for i := range weekly {
		weekly[i] = r(weekly[i])
	}
	out["consumed"] = map[string]any{
		"thisMonth":   r(x.SumInRange(lib.Entries, this.Start(loc), this.End(loc))),
		"perspective": core.Perspective(total),
		"weekly":      weekly,
		"categories":  cats,
	}

	digest := func(p core.Period) period {
		st := lib.Digest(x, p)
		return period{New: len(st.New), Finished: len(st.Finished), Minutes: r(st.Minutes), Pages: st.Pages,
			Sessions: st.Sessions, TopCategory: catName[st.TopCategory], TopTag: st.TopTag, TopRated: ids(st.TopRated)}
	}
	out["week"] = digest(core.LastWeek(now))
	b := lib.DigestBounds(now, loc)
	months := map[string]period{}
	for m := b.Floor; !b.Ceiling.Before(m); m = m.Add(1) {
		months[fmt.Sprintf("%d-%02d", m.Year, int(m.Month))] = digest(core.MonthOfDigest(m, loc))
	}
	out["months"] = months
	years := map[string]year{}
	for y := b.Floor.Year; y <= b.Ceiling.Year; y++ {
		var yr year
		yr.period = digest(core.YearOfDigest(y, loc))
		for _, v := range x.Monthly(lib.Entries, y, loc) {
			yr.Monthly = append(yr.Monthly, r(v))
		}
		yc := lib.YearCard(x, y, loc)
		yr.Card.Minutes = r(yc.Minutes)
		yr.Card.Categories = []group{}
		for _, c := range yc.Categories {
			yr.Card.Categories = append(yr.Card.Categories, group{Name: c.Category.Name, Minutes: r(c.Minutes)})
		}
		yr.Card.Started, yr.Card.Finished = len(yc.New), len(yc.Finished)
		yr.Card.TopRated, yr.Card.TopFeeling = ids(yc.TopRated), yc.TopTag
		years[fmt.Sprint(y)] = yr
	}
	out["years"] = years

	md := lib.MoodTrends(now, loc)
	var mm []string
	for _, m := range md.Months {
		mm = append(mm, fmt.Sprintf("%d-%02d", m.Year, int(m.Month)))
	}
	var tags []map[string]any
	for _, t := range md.Tags {
		tags = append(tags, map[string]any{"tag": t.Tag, "counts": t.Counts, "total": t.Total})
	}
	var ins []string
	for _, i := range md.Insights() {
		ins = append(ins, strings.Join([]string{i.Before, i.Tag, i.After}, ""))
	}
	out["moods"] = map[string]any{"months": mm, "tags": tags, "insights": ins}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", " ")
	if err := enc.Encode(out); err != nil {
		log.Fatal(err)
	}
}
