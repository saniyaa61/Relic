package importer

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/saniyaa61/relic/store"
)

// The Phase 2 check: import an archive, save it, read it back, and compare
// the counts, time totals and Top 5 with what the prototype computes from
// the same file. protoSummary below re-implements the prototype's own
// formulas (relic.html: renderHome, getEntryTimeEvents,
// getAllSessionsWithFirst, renderTop5Block) on the raw JSON, independently
// of the importer.
//
// It runs on the scrubbed copy in testdata, and on the real archive when
// RELIC_ARCHIVE points at it:
//
//	RELIC_ARCHIVE=../relic-archive.json go test ./importer -run Prototype -v

func TestMatchesPrototypeScrubbed(t *testing.T) {
	checkAgainstPrototype(t, filepath.Join("testdata", "scrubbed-archive.json"))
}

func TestMatchesPrototypeRealArchive(t *testing.T) {
	path := os.Getenv("RELIC_ARCHIVE")
	if path == "" {
		t.Skip("set RELIC_ARCHIVE to the real relic-archive.json to run this")
	}
	checkAgainstPrototype(t, path)
}

func checkAgainstPrototype(t *testing.T, path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Read(bytes.NewReader(raw), now)
	if err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(filepath.Join(t.TempDir(), "relic.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.ReplaceAll(res.Library); err != nil {
		t.Fatal(err)
	}
	back, err := db.Load()
	if err != nil {
		t.Fatal(err)
	}
	got := Summarize(back)
	want := protoSummary(t, raw)

	if got.Entries != want.Entries || got.Finished != want.Finished || got.Ongoing != want.Ongoing ||
		got.Favourites != want.Favourites {
		t.Errorf("counts: got total %d finished %d ongoing %d favourites %d; prototype %d %d %d %d",
			got.Entries, got.Finished, got.Ongoing, got.Favourites, want.Entries, want.Finished, want.Ongoing, want.Favourites)
	}
	if math.Abs(got.TotalMinutes-want.TotalMinutes) > 1e-6 {
		t.Errorf("all-time minutes: got %v, prototype %v", got.TotalMinutes, want.TotalMinutes)
	}
	if len(got.Categories) != len(want.Categories) {
		t.Fatalf("got %d categories, prototype %d", len(got.Categories), len(want.Categories))
	}
	for i, w := range want.Categories {
		g := got.Categories[i]
		if g.Name != w.Name || g.Entries != w.Entries || math.Abs(g.Minutes-w.Minutes) > 1e-6 {
			t.Errorf("category %d: got %s %d entries %v min; prototype %s %d entries %v min",
				i, g.Name, g.Entries, g.Minutes, w.Name, w.Entries, w.Minutes)
		}
		if !reflect.DeepEqual(nilIfEmpty(g.TopFive), nilIfEmpty(w.TopFive)) {
			t.Errorf("%s Top 5: got %q, prototype %q", w.Name, g.TopFive, w.TopFive)
		}
	}
	t.Logf("\n%s", got)
}

func nilIfEmpty(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return s
}

type obj = map[string]any

func f64(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case string:
		f, _ := strconv.ParseFloat(x, 64)
		return f
	}
	return 0
}

func s(v any) string { x, _ := v.(string); return x }

func protoSummary(t *testing.T, raw []byte) Summary {
	var a struct {
		Sections     []obj
		Entries      []obj
		TopFavorites map[string][]string
	}
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatal(err)
	}
	var sum Summary
	byName := map[string]*CategorySummary{}
	for _, sec := range a.Sections {
		sum.Categories = append(sum.Categories, CategorySummary{Name: s(sec["name"])})
	}
	for i := range sum.Categories {
		byName[sum.Categories[i].Name] = &sum.Categories[i]
	}
	titles := map[string]string{}
	for _, e := range a.Entries {
		sum.Entries++
		switch s(e["status"]) {
		case "finished":
			sum.Finished++
		case "ongoing", "watching", "reading":
			sum.Ongoing++
		}
		if e["favorite"] == true {
			sum.Favourites++
		}
		titles[s(e["id"])] = s(e["title"])
		m := protoMinutes(e)
		sum.TotalMinutes += m
		sec := s(e["section"])
		if sec == "" {
			sec = s(e["category"])
		}
		if c := byName[sec]; c != nil {
			c.Entries++
			c.Minutes += m
		}
	}
	for name, ids := range a.TopFavorites {
		if c := byName[name]; c != nil {
			for _, id := range ids {
				if title, ok := titles[id]; ok {
					c.TopFive = append(c.TopFive, title)
				}
			}
		}
	}
	return sum
}

// protoMinutes is relic.html's computeTotalMinutes, plus the two time rules
// the owner added after the prototype: partial rewatches count the episodes
// rewatched, and music counts its length (or tracks × 3.5 min).
func protoMinutes(e obj) float64 {
	list := func(k string) []obj {
		var out []obj
		l, _ := e[k].([]any)
		for _, x := range l {
			if o, ok := x.(obj); ok {
				out = append(out, o)
			}
		}
		return out
	}
	sessions, rewatches := list("sessions"), list("rewatches")
	hasFirst := false
	for _, x := range sessions {
		if x["first"] == true {
			hasFirst = true
		}
	}
	typ := s(e["type"])
	progress := typ == "series" || typ == "podcast" || typ == "book"
	// getAllSessionsWithFirst: synthesise the missing first session.
	if !hasFirst && (s(e["review"]) != "" || progress) {
		first := obj{}
		switch typ {
		case "series", "podcast":
			others := 0.0
			for _, x := range sessions {
				others += f64(x["eps"])
			}
			first["eps"] = math.Max(0, f64(e["watchedEpisodes"])-others)
		case "book":
			first["from"], first["to"] = 0.0, f64(e["pagesRead"])
		}
		sessions = append(sessions, first)
	}
	var total float64
	switch typ {
	case "series", "podcast":
		per := f64(e["episodeDuration"])
		for _, x := range sessions {
			total += f64(x["eps"]) * per
		}
		for _, r := range rewatches {
			if r["full"] != false {
				total += f64(e["totalEpisodes"]) * per
			} else {
				total += f64(r["episodes"]) * per // new rule; the prototype added nothing
			}
		}
	case "book":
		for _, x := range sessions {
			if m := f64(x["mins"]); m != 0 {
				total += m
			} else if _, ok := x["from"]; ok && f64(x["to"]) > f64(x["from"]) {
				total += (f64(x["to"]) - f64(x["from"])) * 1.5
			}
		}
	case "other":
		for _, x := range sessions {
			total += f64(x["mins"])
		}
	case "film", "short":
		d := f64(e["duration"])
		total += d * float64(1+len(rewatches))
	case "music": // new rule; the prototype counted nothing
		d := f64(e["duration"])
		if d == 0 {
			d = f64(e["totalPages"]) * 3.5
		}
		total += d * float64(1+len(rewatches))
	}
	return total
}
