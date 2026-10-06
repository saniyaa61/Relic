//go:build tour

package main

// The emulator tour (CI only): import the test archive, then show every
// screen in Linen light and one dark theme, so the run's screenshots cover
// the Phase 3 "Done when" check. CI copies the archive in before building:
//
//	cp importer/testdata/scrubbed-archive.json cmd/relic/tour-archive.json
//	gogio -tags tour ...

import (
	"bytes"
	_ "embed"
	"log"
	"time"

	"github.com/saniyaa61/relic/archive"
	"github.com/saniyaa61/relic/core"
	"github.com/saniyaa61/relic/store"
	"github.com/saniyaa61/relic/ui"
)

//go:embed tour-archive.json
var tourArchive []byte

var tourLight = []string{
	"onboarding", "home", "home-memory", "swy", "rf", "ae", "swy-search", "consumed", "consumed-folder",
	"library", "lib-category", "lib-folder", "lib-menu", "lib-new-category",
	"detail", "detail-book", "detail-film", "log-session", "log-rewatch", "reached-end",
	"form-new", "form-series", "form-edit", "form-calendar",
	"favorites", "fav-series", "fav-rank",
	"digest", "digest-week", "digest-year", "moods", "yearcard",
	"settings", "settings-custom", "settings-colour", "settings-clear",
}

var tourDark = []string{"home", "library", "detail", "favorites", "digest-year", "consumed", "settings"}

func init() {
	startTour = func(a *ui.App, db *store.DB, posters string) {
		res, err := archive.Read(bytes.NewReader(tourArchive), time.Now())
		if err != nil {
			log.Printf("relic: tour: %v", err)
			return
		}
		if err := store.WritePosters(posters, res.Posters); err != nil {
			log.Printf("relic: tour: %v", err)
			return
		}
		if err := db.ReplaceAll(res.Library); err != nil {
			log.Printf("relic: tour: %v", err)
			return
		}
		var steps []ui.TourStep
		for _, s := range tourLight {
			steps = append(steps, ui.TourStep{Screen: s, Theme: "linen", Mode: "light"})
		}
		for _, s := range tourDark {
			steps = append(steps, ui.TourStep{Screen: s, Theme: "midnight", Mode: "dark"})
		}
		now := time.Date(2026, 9, 29, 9, 0, 0, 0, time.UTC)
		a.Tour(steps, now, 4*time.Second, func() (*core.Library, error) { return db.Load() })
	}
}
