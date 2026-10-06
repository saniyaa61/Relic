package importer

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/saniyaa61/relic/core"
)

var now = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

func read(t *testing.T, js string) *Result {
	t.Helper()
	res, err := Read(strings.NewReader(js), now)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestRejectsNonArchive(t *testing.T) {
	for _, js := range []string{`not json`, `{}`, `[1,2]`} {
		if _, err := Read(strings.NewReader(js), now); err == nil {
			t.Errorf("%q: expected an error", js)
		}
	}
}

func TestCategoriesAndTypes(t *testing.T) {
	res := read(t, `{
		"sections": [
			{"id": "s1780829186718", "name": "Movies", "folders": ["Bollywood"]},
			{"id": "s2", "name": "Dramas"},
			{"id": "s3", "name": "Reading list"},
			{"id": "s4", "name": "My podcasts"},
			{"id": "s5", "name": "Albums"},
			{"id": "s6", "name": "YouTube"},
			{"id": "s7", "name": "Games", "type": "other"},
			{"id": "s8", "name": "dramas"}
		],
		"entries": [
			{"id": "e1", "title": "Lost", "section": "Gone", "type": "book", "createdAt": "2026-06-01T10:00:00.000Z", "status": "ongoing"},
			{"id": "e2", "title": "Odd folder", "category": "Movies", "folder": "Hindi", "createdAt": "2026-06-01T10:00:00.000Z", "status": "finished"}
		]}`)
	lib := res.Library
	want := []struct {
		name string
		typ  core.EntryType
	}{{"Movies", core.Film}, {"Dramas", core.Series}, {"Reading list", core.Book}, {"My podcasts", core.Podcast},
		{"Albums", core.Music}, {"YouTube", core.Short}, {"Games", core.Other}, {"Gone", core.Book}}
	if len(lib.Categories) != len(want) {
		t.Fatalf("%d categories", len(lib.Categories))
	}
	for i, w := range want {
		if c := lib.Categories[i]; c.Name != w.name || c.Type != w.typ {
			t.Errorf("category %d = %s (%s), want %s (%s)", i, c.Name, c.Type, w.name, w.typ)
		}
	}
	if got := lib.Categories[0].CreatedAt; !got.Equal(time.UnixMilli(1780829186718)) {
		t.Errorf("category date from id = %v", got)
	}
	// Entry referenced by "category" only, with a folder that wasn't listed.
	e := lib.Entry("e2")
	if e.CategoryID != "s1780829186718" || e.Folder != "Hindi" {
		t.Errorf("e2 = %s / %q", e.CategoryID, e.Folder)
	}
	if f := lib.Categories[0].Folders; len(f) != 2 || f[1] != "Hindi" {
		t.Errorf("folders = %v", f)
	}
	if !hasWarning(res, "no longer existed") || !hasWarning(res, `Two categories are called "dramas"`) {
		t.Errorf("warnings = %q", res.Report.Warnings)
	}
}

func hasWarning(res *Result, s string) bool {
	for _, w := range res.Report.Warnings {
		if strings.Contains(w, s) {
			return true
		}
	}
	return false
}

func TestEntryRepairs(t *testing.T) {
	res := read(t, `{
		"userName": "Haya", "theme": "custom", "customBase": "#bb356f", "customAccent": "#6f75be",
		"sections": [{"id": "s1", "name": "Dramas", "type": "series"}, {"id": "s2", "name": "Books", "type": "book"},
			{"id": "s3", "name": "Albums", "type": "music"}, {"id": "s4", "name": "Misc", "type": "other"},
			{"id": "s5", "name": "Films", "type": "film"}],
		"entries": [
			{"id": "old", "type": "series", "title": "Old drama", "section": "Dramas", "status": "watching",
			 "createdAt": "2026-03-01T10:00:00.000Z", "review": "Lovely.", "totalEpisodes": "16", "watchedEpisodes": 9,
			 "episodeDuration": 60,
			 "sessions": [{"date": "2026-03-05T10:00:00.000Z", "eps": 3, "note": "more"}]},
			{"id": "book", "type": "book", "title": "Old book", "section": "Books", "status": "reading",
			 "createdAt": "2026-04-01T10:00:00.000Z", "pagesRead": 80, "totalPages": 300,
			 "sessions": [{"date": "2026-04-03T10:00:00.000Z", "from": 0, "to": 20, "mins": 0}]},
			{"id": "fin", "type": "series", "title": "Done", "section": "Dramas", "status": "finished",
			 "createdAt": "2026-05-01T10:00:00.000Z", "rating": 4.3, "totalEpisodes": 2, "watchedEpisodes": 2,
			 "sessions": [{"date": "2026-05-01T10:00:00.000Z", "first": true, "doneAtStart": true, "eps": 2}],
			 "rewatches": [{"date": "2026-06-18T09:00:00.000Z", "note": "again"},
			               {"date": "2026-06-20T09:00:00.000Z", "full": false}]},
			{"id": "alb", "type": "music", "title": "Album", "section": "Albums", "status": "finished",
			 "createdAt": "2026-05-02T10:00:00.000Z", "totalPages": 11},
			{"id": "oth", "type": "other", "title": "Thing", "section": "Misc", "status": "want-to",
			 "createdAt": "2026-05-03T10:00:00.000Z", "duration": "2 hours",
			 "sessions": [{"date": "2026-05-03T10:00:00.000Z", "first": true, "mins": 67},
			              {"date": "2026-05-04T10:00:00.000Z", "first": true, "mins": 20}]},
			{"id": "film", "type": "film", "title": "Film", "section": "Films", "status": "finished",
			 "createdAt": "2026-05-05T10:00:00.000Z", "duration": "120", "watchedDate": "2026-05-04"}
		]}`)
	lib := res.Library

	if p := lib.Profile; p.Name != "Haya" || p.Theme != "custom" || p.Mode != "light" || p.CustomAccent != "#6f75be" {
		t.Errorf("profile = %+v", p)
	}
	if !lib.Profile.FirstUsedAt.Equal(time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("first used = %v, want earliest createdAt", lib.Profile.FirstUsedAt)
	}

	// Missing start session: rebuilt at createdAt with the review and the
	// episodes not covered by later sessions (9 − 3 = 6).
	old := lib.Entry("old")
	s := old.StartSession()
	if old.Status != core.Ongoing || s == nil || s.Episodes != 6 || s.Note != "Lovely." || !s.At.Equal(old.CreatedAt) {
		t.Fatalf("old drama: status %s, start %+v", old.Status, s)
	}
	if old.WatchedEpisodes() != 9 || old.Fields.TotalEpisodes != 16 {
		t.Errorf("old drama progress %d/%d", old.WatchedEpisodes(), old.Fields.TotalEpisodes)
	}
	if old.Sessions[0] != *s {
		t.Error("start session should be first (oldest)")
	}

	// Book: start session to page = pagesRead; mins 0 means not logged.
	b := lib.Entry("book")
	if b.StartSession().ToPage != 80 || b.Sessions[1].Minutes != nil || b.PagesRead() != 80 {
		t.Errorf("book: start to %d, minutes %v", b.StartSession().ToPage, b.Sessions[1].Minutes)
	}

	// Finished without finishedAt → createdAt; rating rounds to half stars;
	// rewatch full defaults to true, explicit false is kept.
	f := lib.Entry("fin")
	if !f.FinishedAt.Equal(f.CreatedAt) || f.Rating != 4.5 {
		t.Errorf("fin: finishedAt %v rating %v", f.FinishedAt, f.Rating)
	}
	if !f.StartSession().StartedFinished || len(f.Rewatches) != 2 || !f.Rewatches[0].Full || f.Rewatches[1].Full {
		t.Errorf("fin sessions/rewatches = %+v %+v", f.Sessions, f.Rewatches)
	}

	// Music keeps tracks from totalPages and counts 11 × 3.5 min.
	if a := lib.Entry("alb"); a.Fields.Tracks != 11 || a.Fields.TotalPages != 0 || core.NewTimeIndex([]*core.Entry{a}).Total("alb") != 38.5 {
		t.Errorf("album fields = %+v", a.Fields)
	}

	// Other: free-text duration kept; unknown status → ongoing; two first sessions → one start.
	o := lib.Entry("oth")
	starts := 0
	for _, s := range o.Sessions {
		if s.IsStart {
			starts++
		}
	}
	if o.Fields.DurationText != "2 hours" || o.Status != core.Ongoing || starts != 1 {
		t.Errorf("other: %q %s starts=%d", o.Fields.DurationText, o.Status, starts)
	}

	// Numeric strings are read as numbers.
	if fl := lib.Entry("film"); fl.Fields.Duration != 120 || fl.Fields.WatchedDate != (core.Date{Year: 2026, Month: 5, Day: 4}) {
		t.Errorf("film fields = %+v", fl.Fields)
	}

	for _, w := range []string{`"Thing" had status "want-to"`, "more than one first session", "rating 4.3 rounded to 4.5"} {
		if !hasWarning(res, w) {
			t.Errorf("missing warning %q in %q", w, res.Report.Warnings)
		}
	}
	// Newest first.
	if lib.Entries[0].ID != "film" || lib.Entries[len(lib.Entries)-1].ID != "old" {
		t.Error("entries should be newest first")
	}
}

func TestProgressDriftIsReported(t *testing.T) {
	res := read(t, `{"sections": [{"id": "s1", "name": "Dramas", "type": "series"}],
		"entries": [{"id": "d", "type": "series", "title": "Drift", "section": "Dramas", "status": "ongoing",
			"createdAt": "2026-03-01T10:00:00.000Z", "watchedEpisodes": 5, "totalEpisodes": 10,
			"sessions": [{"date": "2026-03-01T10:00:00.000Z", "first": true, "eps": 3}]}]}`)
	if !hasWarning(res, "the prototype showed 5 episodes watched; its sessions add up to 3") {
		t.Errorf("warnings = %q", res.Report.Warnings)
	}
}

func TestFavourites(t *testing.T) {
	res := read(t, `{"sections": [{"id": "s1", "name": "Movies", "type": "film"}, {"id": "s2", "name": "Books", "type": "book"}],
		"entries": [
			{"id": "a", "title": "A", "section": "Movies", "status": "finished", "favorite": true, "createdAt": "2026-01-01T00:00:00Z"},
			{"id": "b", "title": "B", "section": "Movies", "status": "finished", "favorite": true, "createdAt": "2026-01-02T00:00:00Z"},
			{"id": "c", "title": "C", "section": "Movies", "status": "finished", "createdAt": "2026-01-03T00:00:00Z"},
			{"id": "d", "title": "D", "section": "Books", "status": "finished", "favorite": true, "createdAt": "2026-01-04T00:00:00Z"}],
		"topFavorites": {"Movies": ["b", "gone", "c"], "Nowhere": ["a"]},
		"favoriteOrder": {"Movies": ["b", "a"], "Books": ["d", "a"]}}`)
	lib := res.Library
	if got := lib.Favourites.TopFive["s1"]; len(got) != 1 || got[0] != "b" {
		t.Errorf("top five = %v", got)
	}
	if got := lib.Favourites.Order["s1"]; len(got) != 1 || got[0] != "a" {
		t.Errorf("order = %v (Top 5 entries aren't repeated)", got)
	}
	if got := lib.Favourites.Order["s2"]; len(got) != 1 || got[0] != "d" {
		t.Errorf("books order = %v", got)
	}
}

func dataURL(t *testing.T, mime string, w, h int) string {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{200, 50, 50, 255})
		}
	}
	var buf bytes.Buffer
	if mime == "image/png" {
		png.Encode(&buf, img)
	} else {
		jpeg.Encode(&buf, img, nil)
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestPosters(t *testing.T) {
	tests := []struct {
		name         string
		src          string
		wantW, wantH int
		ok           bool
	}{
		{"big portrait png shrinks", dataURL(t, "image/png", 800, 1200), 400, 600, true},
		{"big landscape jpeg shrinks", dataURL(t, "image/jpeg", 1500, 1000), 600, 400, true},
		{"small is not enlarged", dataURL(t, "image/jpeg", 200, 300), 200, 300, true},
		{"not a data url", "https://example.com/p.jpg", 0, 0, false},
		{"garbage", "data:image/png;base64,AAAA", 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := read(t, `{"sections": [{"id": "s1", "name": "Movies", "type": "film"}],
				"entries": [{"id": "p", "title": "P", "section": "Movies", "status": "finished",
				"createdAt": "2026-01-01T00:00:00Z", "poster": "`+tt.src+`"}]}`)
			e := res.Library.Entry("p")
			if !tt.ok {
				if e.Poster != "" || len(res.Posters) != 0 || !hasWarning(res, "poster couldn't be read") {
					t.Errorf("poster %q, warnings %q", e.Poster, res.Report.Warnings)
				}
				return
			}
			if e.Poster != "p.jpg" {
				t.Fatalf("poster ref = %q", e.Poster)
			}
			img, err := jpeg.Decode(bytes.NewReader(res.Posters["p.jpg"]))
			if err != nil {
				t.Fatal(err)
			}
			if b := img.Bounds(); b.Dx() != tt.wantW || b.Dy() != tt.wantH {
				t.Errorf("size = %dx%d, want %dx%d", b.Dx(), b.Dy(), tt.wantW, tt.wantH)
			}
		})
	}
}
