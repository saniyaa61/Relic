package ui

import (
	"testing"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/saniyaa61/relic/core"
)

func TestHomeChipsOpenTheirPages(t *testing.T) {
	a, _ := libApp(t)
	h := a.roots[TabHome].(*homePage)
	frame(a)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops}
	for i, want := range []any{pageAllEntries, "Consumed", pageRecentlyFinished, pageStillWithYou} {
		a.stack = nil
		h.chips[i].Click()
		h.Layout(gtx, a)
		switch p := a.top().(type) {
		case *posterPage:
			if p.kind != want {
				t.Errorf("chip %d opened %v, want %v", i, p.kind, want)
			}
		case *comingSoon:
			if want != "Consumed" {
				t.Errorf("chip %d opened the Consumed placeholder", i)
			}
		default:
			t.Errorf("chip %d opened %T", i, p)
		}
	}
}

func TestPosterPagesShowTheirEntries(t *testing.T) {
	a, _ := libApp(t)
	for _, tc := range []struct {
		kind posterPageKind
		want []*core.Entry
	}{
		{pageStillWithYou, a.Lib.StillWithYou()},
		{pageRecentlyFinished, a.Lib.RecentlyFinished()},
		{pageAllEntries, a.Lib.Entries},
	} {
		p := newPosterPage(tc.kind)
		got := p.entries(a)
		if len(got) != len(tc.want) || len(got) == 0 {
			t.Errorf("page %v: %d entries, want %d", tc.kind, len(got), len(tc.want))
		}
		// Still with you puts 3 cards a row, the others 4 (SPEC §5); rows
		// plus the leading spacer.
		per := 4
		if tc.kind == pageStillWithYou {
			per = 3
		}
		if rows := p.cardRows(a, got); len(rows) != 1+(len(got)+per-1)/per {
			t.Errorf("page %v: %d rows for %d entries", tc.kind, len(rows)-1, len(got))
		}
	}
	// Recently finished is most recently finished first.
	rf := a.Lib.RecentlyFinished()
	for i := 1; i < len(rf); i++ {
		if rf[i].FinishedAt.After(rf[i-1].FinishedAt) {
			t.Fatal("recently finished out of order")
		}
	}
}

// The "✓ Finished?" badge asks, then marks the entry finished.
func TestFinishedBadge(t *testing.T) {
	a, reload := libApp(t)
	e := findEntry(a.Lib, func(e *core.Entry) bool {
		return e.Status == core.Ongoing && e.Type.Progress() == core.EpisodeProgress && e.Fields.TotalEpisodes > 0
	})
	n := e.Fields.TotalEpisodes - e.WatchedEpisodes()
	e.LogSession(core.SessionInput{Episodes: &n}, a.Now())
	if !e.ShouldPromptFinished() {
		t.Fatal("setup: entry should have reached the end")
	}
	confirmMarkFinished(a, e.ID)
	if a.dialog.Title != "Mark as finished?" || a.dialog.Danger {
		t.Errorf("dialog %q danger %v; want the accent button (owner's choice)", a.dialog.Title, a.dialog.Danger)
	}
	a.dialog.OnConfirm(a, "")
	if got := reload().Entry(e.ID); got.Status != core.Finished {
		t.Errorf("status %s", got.Status)
	}
}

func TestHomeEmptyAndMemory(t *testing.T) {
	a, _ := testApp(t, nil)
	frame(a) // empty library: no panic, the welcome state
	h := a.roots[TabHome].(*homePage)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops}
	h.setup.Click()
	h.Layout(gtx, a)
	if a.Tab() != TabLibrary || a.dialog == nil || a.dialog.Title != "New category" {
		t.Errorf("\"Set up your library categories\" should open New category in the Library")
	}

	b, _ := libApp(t)
	var fin *core.Entry
	for _, e := range b.Lib.Entries {
		if e.Status == core.Finished {
			fin = e
			break
		}
	}
	fin.CreatedAt = b.Now().AddDate(-1, 0, 1)
	if got := b.Lib.OneYearAgo(b.Now(), b.Loc); got != fin {
		t.Fatal("memory card should pick the entry from a year ago")
	}
	frame(b)
}

func TestStarString(t *testing.T) {
	for r, want := range map[float64]string{0: "", 3: "★★★", 3.5: "★★★½", 0.5: "½"} {
		if got := starString(r); got != want {
			t.Errorf("starString(%v) = %q, want %q", r, got, want)
		}
	}
}
