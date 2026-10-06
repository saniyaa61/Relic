package ui

// The tour: steps through screens one after another, for checking every
// screen on the Android emulator (CI builds cmd/relic with -tags tour).

import (
	"log"
	"time"

	"github.com/saniyaa61/relic/core"
)

// TourStep is one screen of the tour (a SnapshotScreens name) in a theme.
type TourStep struct {
	Screen, Theme, Mode string
}

// Tour shows each step for hold, logging "relic: tour <n>/<total> <screen>
// <theme> <mode>" as it appears, then "relic: tour done". reload gives a
// fresh copy of the library for each step, so steps can't affect each other.
func (a *App) Tour(steps []TourStep, now time.Time, hold time.Duration, reload func() (*core.Library, error)) {
	defaults := [...]any{customTile, yearBars, homeStreak, detailHero, markFinishedRed}
	var show func(i int)
	show = func(i int) {
		if i == len(steps) {
			log.Printf("relic: tour done")
			return
		}
		s := steps[i]
		lib, err := reload()
		if err != nil {
			log.Printf("relic: tour: %v", err)
			return
		}
		customTile, yearBars, homeStreak = defaults[0].(customTileStyle), defaults[1].(yearBarStyle), defaults[2].(streakStyle)
		detailHero, markFinishedRed = defaults[3].(heroStyle), defaults[4].(bool)
		lib.Profile.Theme, lib.Profile.Mode = s.Theme, s.Mode
		a.Lib = lib
		a.dialog, a.toast = nil, toast{}
		a.restart()
		a.Now = func() time.Time { return now }
		if err := a.ShowForSnapshot(s.Screen); err != nil {
			log.Printf("relic: tour %s: %v", s.Screen, err)
		}
		log.Printf("relic: tour %d/%d %s %s %s", i+1, len(steps), s.Screen, s.Theme, s.Mode)
		a.background(func() func(*App) {
			time.Sleep(hold)
			return func(a *App) { show(i + 1) }
		})
	}
	show(0)
}
