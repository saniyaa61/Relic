// Package tz finds the phone's time zone. Go on Android has no local zone
// of its own (time.Local is UTC there), and on iOS it can't always read
// one, so the app would count days, weeks and months in UTC. SetLocal asks
// the system and sets time.Local.
package tz

import (
	"time"
	_ "time/tzdata" // zone rules built in, in case the system's can't be read
)

// SetLocal sets time.Local to the system's zone where Go doesn't know it
// (Android, iOS) and returns its name, or "" if it left time.Local alone.
func SetLocal() string {
	name, err := systemZone()
	if err != nil || name == "" {
		return ""
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return ""
	}
	time.Local = loc
	return name
}
