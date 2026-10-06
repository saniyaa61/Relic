package ui

import "testing"

// A fresh start shows the welcome page; a name is saved, then the Library
// opens with New category.
func TestOnboarding(t *testing.T) {
	a, _ := testApp(t, nil)
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
	if len(a.stack) != 0 || a.Tab() != TabLibrary || a.dialog == nil || a.dialog.Title != "New category" {
		t.Errorf("after: tab %v, %d pages, dialog %v", a.Tab(), len(a.stack), a.dialog)
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
