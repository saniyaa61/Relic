package ui

import (
	"path/filepath"
	"testing"

	"github.com/saniyaa61/relic/store"
)

// A fresh start shows the welcome page; a name is saved, then the guided
// tour walks through every step and is saved as done.
func TestOnboarding(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "relic.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	a, _ := testApp(t, db)
	a.StartOnboarding()
	p, ok := a.top().(*onboardingPage)
	if !ok {
		t.Fatalf("fresh start shows %T", a.top())
	}
	if a.navShown() {
		t.Error("the welcome page hides the bottom bar")
	}
	frame(a)
	p.name.Editor.SetText("  Saniya  ")
	p.begin.Click()
	frame(a)
	if a.Lib.Profile.Name != "Saniya" {
		t.Errorf("name %q", a.Lib.Profile.Name)
	}
	if len(a.stack) != 0 || a.Tab() != TabHome || a.dialog != nil || !a.Lib.Profile.ShowWelcome {
		t.Errorf("after: tab %v, %d pages, dialog %v, welcome %v", a.Tab(), len(a.stack), a.dialog, a.Lib.Profile.ShowWelcome)
	}
	if lib, err := db.Load(); err != nil || !lib.Profile.ShowWelcome {
		t.Errorf("the welcome card should be saved: %v", err)
	}
	// The guided tour starts on Home and walks each step, opening its tab.
	frame(a)
	if a.guide == nil {
		t.Fatal("the guided tour should start after onboarding")
	}
	for i := 1; i < len(guideSteps); i++ {
		a.guide.next.Click()
		frame(a)
		if a.guide == nil || a.guide.step != i || a.Tab() != guideSteps[i].tab {
			t.Fatalf("step %d: guide %+v, tab %v", i, a.guide, a.Tab())
		}
	}
	if r := a.guideRects.nav[TabLibrary]; r.Empty() || a.guideRects.gear.Empty() {
		t.Errorf("tour targets: library %v, gear %v", r, a.guideRects.gear)
	}
	a.guide.next.Click() // Begin
	frame(a)
	if a.guide != nil || a.Lib.Profile.ShowWelcome || a.Tab() != TabHome {
		t.Errorf("Begin should end the tour on Home: guide %v, pending %v", a.guide, a.Lib.Profile.ShowWelcome)
	}
	if lib, _ := db.Load(); lib.Profile.ShowWelcome {
		t.Error("the finished tour should be saved")
	}

	// Skip leaves the name empty; a library with data never sees it.
	b, _ := testApp(t, nil)
	b.StartOnboarding()
	b.top().(*onboardingPage).skip.Click()
	frame(b)
	if b.Lib.Profile.Name != "" || len(b.stack) != 0 {
		t.Error("skip")
	}
	c, _ := libApp(t)
	c.StartOnboarding()
	if len(c.stack) != 0 {
		t.Error("an existing library shouldn't be welcomed again")
	}
}

// Skipping (the link or the back button) ends the tour for good.
func TestGuideSkip(t *testing.T) {
	a, _ := libApp(t)
	a.Lib.Profile.ShowWelcome = true
	frame(a)
	if a.guide == nil {
		t.Fatal("a pending tour should start")
	}
	a.guide.next.Click()
	frame(a)
	if !a.Back() || a.guide != nil || a.Lib.Profile.ShowWelcome {
		t.Errorf("back should skip the tour: guide %v, pending %v", a.guide, a.Lib.Profile.ShowWelcome)
	}
	a.Lib.Profile.ShowWelcome = true
	frame(a)
	a.guide.skip.Click()
	frame(a)
	if a.guide != nil || a.Lib.Profile.ShowWelcome {
		t.Error("Skip tour should end it")
	}
}
