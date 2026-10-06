package ui

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/saniyaa61/relic/core"
)

// Every screen the emulator tour visits shows without an error, in light
// and dark, and the tour ends.
func TestTour(t *testing.T) {
	a, reload := libApp(t)
	var steps []TourStep
	for _, s := range SnapshotScreens {
		steps = append(steps, TourStep{s, "linen", "light"}, TourStep{s, "midnight", "dark"})
	}
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	a.Tour(steps, time.Date(2026, 9, 29, 9, 0, 0, 0, time.UTC), 0, func() (*core.Library, error) { return reload(), nil })
	for i := 0; i < 10*len(steps) && !strings.Contains(buf.String(), "relic: tour done"); i++ {
		frame(a)
		time.Sleep(time.Millisecond)
		a.runAsync()
	}
	out := buf.String()
	if !strings.Contains(out, "relic: tour done") {
		t.Fatalf("tour didn't finish:\n%s", out)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "relic: tour") && strings.Contains(line, ":") && !strings.Contains(line, "/") && !strings.Contains(line, "done") {
			t.Errorf("%s", line)
		}
	}
}
