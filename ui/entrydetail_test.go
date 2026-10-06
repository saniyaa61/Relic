package ui

import (
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/saniyaa61/relic/core"
)

func submitSheet(t *testing.T, a *App, l *logSheet) {
	t.Helper()
	e := a.Lib.Entry(l.entryID)
	if e == nil {
		t.Fatal("entry gone")
	}
	l.submit(a, e)
}

func findEntry(lib *core.Library, keep func(*core.Entry) bool) *core.Entry {
	for _, e := range lib.Entries {
		if keep(e) {
			return e
		}
	}
	return nil
}

// Logging episodes up to the total asks "Looks like you reached the end!"
// and marking it finished is saved.
func TestLogSessionReachesEnd(t *testing.T) {
	a, reload := libApp(t)
	e := findEntry(a.Lib, func(e *core.Entry) bool {
		return e.Type == core.Series && e.Status == core.Ongoing && e.Fields.TotalEpisodes > e.WatchedEpisodes()
	})
	if e == nil {
		t.Skip("no ongoing series in the fixture")
	}
	a.Push(newEntryDetail(e.ID))
	frame(a)

	// A blank episodes field logs one episode, and doesn't prompt early.
	l := openLogSheet(a, e.ID, false, "")
	before := e.WatchedEpisodes()
	submitSheet(t, a, l)
	if got := reload().Entry(e.ID).WatchedEpisodes(); got != before+1 || a.toast.msg != "Session logged ✦" {
		t.Fatalf("watched %d (want %d), toast %q", got, before+1, a.toast.msg)
	}
	if a.dialog != nil && !e.ReachedEnd() {
		t.Fatal("prompted before the end")
	}

	// The rest of the episodes: now it asks.
	l = openLogSheet(a, e.ID, false, "")
	l.eps.Editor.SetText(itoa(e.Fields.TotalEpisodes - e.WatchedEpisodes()))
	submitSheet(t, a, l)
	if a.dialog == nil || a.dialog.Title != "Looks like you reached the end!" || a.dialog.Cancel != "Not yet" {
		t.Fatalf("no reached-the-end prompt: %+v", a.dialog)
	}
	a.dialog.OnConfirm(a, "")
	got := reload().Entry(e.ID)
	if got.Status != core.Finished || got.FinishedAt.IsZero() || a.toast.msg != "Marked as finished ✦" {
		t.Errorf("after confirming: %s at %v, toast %q", got.Status, got.FinishedAt, a.toast.msg)
	}
}

func TestLogBookSessionDefaults(t *testing.T) {
	a, reload := libApp(t)
	e := findEntry(a.Lib, func(e *core.Entry) bool { return e.Type == core.Book })
	read := e.PagesRead()
	l := openLogSheet(a, e.ID, false, "")
	l.to.Editor.SetText(itoa(read + 20))
	l.mins.Editor.SetText("0") // zero is a value
	submitSheet(t, a, l)
	s := reload().Entry(e.ID).Sessions
	last := s[len(s)-1]
	if last.FromPage != read || last.ToPage != read+20 || last.Minutes == nil || *last.Minutes != 0 {
		t.Errorf("book session %+v (minutes %v); want from %d to %d, 0 min", last, last.Minutes, read, read+20)
	}
}

func TestPartialRewatch(t *testing.T) {
	a, reload := libApp(t)
	e := findEntry(a.Lib, func(e *core.Entry) bool { return e.Type == core.Series })
	rating := e.Rating
	l := openLogSheet(a, e.ID, true, "")
	l.full = false
	l.partial.Editor.SetText("4")
	l.rating = 2.5
	l.note.Editor.SetText("Different now.")
	submitSheet(t, a, l)
	got := reload().Entry(e.ID)
	r := got.Rewatches[len(got.Rewatches)-1]
	if r.Full || r.Episodes != 4 || r.Rating != 2.5 || r.Note != "Different now." {
		t.Errorf("rewatch %+v", r)
	}
	if got.Rating != rating {
		t.Errorf("rewatch rating changed the entry rating to %v (owner's decision: never)", got.Rating)
	}
	if a.sheet == nil || !a.sheet.closing() {
		t.Error("sheet should close after saving")
	}
}

// Editing the start session's note changes the review too (SPEC §4.3),
// and a session can be deleted after asking; the start session can't.
func TestEditAndDeleteJourney(t *testing.T) {
	a, reload := libApp(t)
	e := a.Lib.Entries[0]
	start := e.StartSession()
	l := openLogSheet(a, e.ID, false, start.ID)
	if l.note.Editor.Text() != start.Note {
		t.Fatal("edit sheet should start with the session's note")
	}
	l.note.Editor.SetText("A new first thought.")
	submitSheet(t, a, l)
	if got := reload().Entry(e.ID); got.Review != "A new first thought." || a.toast.msg != "Session updated ✦" {
		t.Errorf("review %q, toast %q", got.Review, a.toast.msg)
	}

	l = openLogSheet(a, e.ID, false, "")
	submitSheet(t, a, l)
	added := e.Sessions[len(e.Sessions)-1].ID
	confirmDeleteJourneyItem(a, e.ID, added)
	if a.dialog.Title != "Delete this entry?" {
		t.Errorf("dialog %q", a.dialog.Title)
	}
	a.dialog.OnConfirm(a, "")
	for _, s := range reload().Entry(e.ID).Sessions {
		if s.ID == added {
			t.Error("deleted session still stored")
		}
	}
	confirmDeleteJourneyItem(a, e.ID, start.ID)
	a.dialog.OnConfirm(a, "")
	if a.toast.msg != core.ErrStartSession.Error() || reload().Entry(e.ID).StartSession() == nil {
		t.Errorf("start session deletion: toast %q", a.toast.msg)
	}
}

func TestFavouriteAndTopFive(t *testing.T) {
	a, reload := libApp(t)
	e := findEntry(a.Lib, func(e *core.Entry) bool { return !e.Favorite })
	p := newEntryDetail(e.ID)
	a.Push(p)
	frame(a)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops}
	p.fav.Click()
	p.handle(gtx, a, e)
	if !reload().Entry(e.ID).Favorite {
		t.Fatal("favourite not saved")
	}
	p.topFive.Click()
	p.handle(gtx, a, e)
	if _, ok := reload().TopFiveRank(e.ID); !ok {
		t.Errorf("not added to the Top 5 (toast %q)", a.toast.msg)
	}
	p.topFive.Click()
	p.handle(gtx, a, e)
	if _, ok := reload().TopFiveRank(e.ID); ok {
		t.Error("tapping #N All-Time ✕ should remove it")
	}
}

// "Delete this entry" needs a second tap within 3 seconds.
func TestDeleteEntryTwoTaps(t *testing.T) {
	a, reload := libApp(t)
	e := a.Lib.Entries[2]
	p := newEntryDetail(e.ID)
	a.Push(p)
	frame(a)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops}
	now := a.Now()
	a.Now = func() time.Time { return now }
	p.del.Click()
	p.handle(gtx, a, e)
	if reload().Entry(e.ID) == nil {
		t.Fatal("deleted on the first tap")
	}
	now = now.Add(4 * time.Second) // too late: arms again instead
	p.del.Click()
	p.handle(gtx, a, e)
	if reload().Entry(e.ID) == nil {
		t.Fatal("deleted after the 3s window")
	}
	now = now.Add(time.Second)
	p.del.Click()
	p.handle(gtx, a, e)
	if reload().Entry(e.ID) != nil || a.toast.msg != "Entry removed from your library" {
		t.Errorf("not deleted on the second tap; toast %q", a.toast.msg)
	}
	if a.top() == Screen(p) {
		t.Error("should leave the deleted entry's page")
	}
}
