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
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/saniyaa61/relic/archive"
	"github.com/saniyaa61/relic/core"
	"github.com/saniyaa61/relic/store"
	"github.com/saniyaa61/relic/ui"
)

//go:embed tour-archive.json
var tourArchive []byte

var tourLight = []string{
	"onboarding", "guide-0", "guide-1", "guide-6", "home", "home-memory", "swy", "rf", "ae", "swy-search", "consumed", "consumed-folder",
	"library", "lib-category", "lib-folder", "lib-menu", "lib-new-category",
	"detail", "detail-book", "detail-film", "log-session", "log-rewatch", "reached-end-accent",
	"form-new", "form-series", "form-edit", "form-calendar",
	"favorites", "fav-series", "fav-rank",
	"digest", "digest-week", "digest-year", "moods", "yearcard",
	"settings", "settings-custom", "settings-colour", "settings-clear",
}

var tourDark = []string{"home", "library", "detail", "favorites", "digest-year", "consumed", "settings",
	// Last: presses Share, and the system sheet stays up over the app.
	"yearcard-sheet"}

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
		a.Tour(steps, now, tourWait(), func() (*core.Library, error) { return db.Load() })
	}
}

// tourWait is how long each screen stays up. On Android, a fixed 4 s
// (screenshots there take well under a second). The iOS simulator's
// screenshots take anything from 2 to over 10 s on CI's Intel Macs, so
// there the tour waits for tools/ios-tour.sh to write the step number to
// a file (in the app's temporary folder, which the Mac can reach) once
// it has taken that screenshot, giving up after three minutes.
func tourWait() func(n int) {
	if runtime.GOOS != "ios" {
		return func(int) { time.Sleep(4 * time.Second) }
	}
	next := filepath.Join(os.TempDir(), "relic-tour-next")
	os.Remove(next)
	log.Printf("relic: tour waits on %s", next)
	return func(n int) {
		want := strconv.Itoa(n)
		for t := time.Now(); time.Since(t) < 3*time.Minute; time.Sleep(200 * time.Millisecond) {
			if b, err := os.ReadFile(next); err == nil && strings.TrimSpace(string(b)) == want {
				return
			}
		}
		log.Printf("relic: tour step-%d: no screenshot signal", n)
	}
}
