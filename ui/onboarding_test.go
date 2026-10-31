package ui

import (
	"path/filepath"
	"testing"

	"github.com/saniyaa61/relic/store"
)

// A fresh start shows the welcome page; a name is saved, then Home shows
// the welcome card until "Got it", which is saved too.
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
	h := a.roots[TabHome].(*homePage)
	h.gotIt.Click()
	frame(a)
	if a.Lib.Profile.ShowWelcome {
		t.Error("Got it should put the card away")
	}
	if lib, _ := db.Load(); lib.Profile.ShowWelcome {
		t.Error("Got it should be saved")
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
