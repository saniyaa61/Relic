package core

import (
	"errors"
	"testing"
	"time"
)

var t0 = time.Date(2026, 6, 16, 9, 0, 0, 0, time.UTC)

func TestNewEntryStartSession(t *testing.T) {
	tests := []struct {
		name      string
		typ       EntryType
		in        EntryInput
		wantStart bool
		wantEps   int
		wantTo    int
	}{
		{"film without review has no start", Film, EntryInput{Title: "Dune"}, false, 0, 0},
		{"film with review has start", Film, EntryInput{Title: "Dune", Review: "Vast."}, true, 0, 0},
		{"blank review doesn't count", Film, EntryInput{Title: "Dune", Review: "   "}, false, 0, 0},
		{"series always has start", Series, EntryInput{Title: "Signal", StartEpisodes: 3}, true, 3, 0},
		{"series with 0 watched", Series, EntryInput{Title: "Signal"}, true, 0, 0},
		{"podcast", Podcast, EntryInput{Title: "Pod", StartEpisodes: 12}, true, 12, 0},
		{"book holds pages read", Book, EntryInput{Title: "Stoner", StartPages: 80}, true, 0, 80},
		{"music without review", Music, EntryInput{Title: "Blue"}, false, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.in.Status = Ongoing
			e, err := NewEntry(&Category{ID: "c", Type: tt.typ}, tt.in, t0)
			if err != nil {
				t.Fatal(err)
			}
			s := e.StartSession()
			if (s != nil) != tt.wantStart {
				t.Fatalf("start session present = %v, want %v", s != nil, tt.wantStart)
			}
			if s == nil {
				return
			}
			if !s.At.Equal(t0) || s.Note != tt.in.Review || s.Episodes != tt.wantEps ||
				s.FromPage != 0 || s.ToPage != tt.wantTo || s.StartedFinished {
				t.Errorf("start session = %+v", *s)
			}
		})
	}
}

func TestNewEntryValidation(t *testing.T) {
	cat := &Category{ID: "c", Type: Film}
	tests := []struct {
		name string
		cat  *Category
		in   EntryInput
		want error
	}{
		{"title required", cat, EntryInput{Title: "  ", Status: Ongoing}, ErrTitleRequired},
		{"category required", nil, EntryInput{Title: "x", Status: Ongoing}, ErrCategoryRequired},
		{"bad status", cat, EntryInput{Title: "x", Status: "watching"}, ErrBadStatus},
		{"bad rating", cat, EntryInput{Title: "x", Status: Ongoing, Rating: 4.3}, ErrBadRating},
		{"rating too high", cat, EntryInput{Title: "x", Status: Ongoing, Rating: 5.5}, ErrBadRating},
		{"negative pages", cat, EntryInput{Title: "x", Status: Ongoing, StartPages: -1}, ErrNegative},
		{"ok half star", cat, EntryInput{Title: "x", Status: Ongoing, Rating: 4.5}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewEntry(tt.cat, tt.in, t0); !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestFinishedAtCreation(t *testing.T) {
	e, _ := NewEntry(&Category{ID: "c", Type: Series},
		EntryInput{Title: "Signal", Status: Finished, StartEpisodes: 16}, t0)
	if !e.FinishedAt.Equal(t0) {
		t.Errorf("FinishedAt = %v, want createdAt %v", e.FinishedAt, t0)
	}
	if !e.StartSession().StartedFinished {
		t.Error("start session should be marked startedFinished")
	}
	// Finished at creation → no "Started …" label.
	j := e.Journey("Dramas")
	if len(j) != 1 || j[0].Label != "" {
		t.Errorf("journey = %+v, want one item with no label", j)
	}
}

func TestSetStatus(t *testing.T) {
	t1 := t0.Add(48 * time.Hour)
	tests := []struct {
		name       string
		from       Status
		fromAt     time.Time
		to         Status
		wantFinish time.Time
	}{
		{"ongoing → finished sets date", Ongoing, time.Time{}, Finished, t1},
		{"finished → ongoing clears date", Finished, t0, Ongoing, time.Time{}},
		{"finished → finished keeps date", Finished, t0, Finished, t0},
		{"ongoing → ongoing stays clear", Ongoing, time.Time{}, Ongoing, time.Time{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Entry{Status: tt.from, FinishedAt: tt.fromAt}
			e.SetStatus(tt.to, t1)
			if e.Status != tt.to || !e.FinishedAt.Equal(tt.wantFinish) {
				t.Errorf("got %s %v, want %s %v", e.Status, e.FinishedAt, tt.to, tt.wantFinish)
			}
		})
	}
}

func TestReachedEndPrompt(t *testing.T) {
	type step struct {
		eps        int
		wantPrompt bool
	}
	tests := []struct {
		name    string
		total   int
		initial int
		steps   []step
	}{
		// 2-episode drama logged with 1 watched; one more session → reached end → prompt.
		{"2 eps, 1 watched, one more", 2, 1, []step{{1, true}}},
		// 8 episodes, 5 watched at creation, remaining 3 across two sessions → prompt only at 8.
		{"8 eps, 5 watched, 2 then 1", 8, 5, []step{{2, false}, {1, true}}},
		{"no total never prompts", 0, 5, []step{{10, false}}},
		{"overshooting still prompts", 4, 1, []step{{9, true}}},
		{"zero-episode session doesn't advance", 3, 2, []step{{0, false}, {1, true}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := NewEntry(&Category{ID: "c", Type: Series}, EntryInput{Title: "x", Status: Ongoing,
				StartEpisodes: tt.initial, Fields: Fields{TotalEpisodes: tt.total}}, t0)
			if e.ShouldPromptFinished() {
				t.Fatal("prompt before any session")
			}
			for i, s := range tt.steps {
				got, err := e.LogSession(SessionInput{Episodes: ip(s.eps)}, t0.Add(time.Duration(i+1)*time.Hour))
				if err != nil {
					t.Fatal(err)
				}
				if got != s.wantPrompt {
					t.Errorf("step %d (%d eps, watched now %d): prompt = %v, want %v",
						i, s.eps, e.WatchedEpisodes(), got, s.wantPrompt)
				}
			}
		})
	}
}

func TestFinishedEntryNeverPrompts(t *testing.T) {
	e, _ := NewEntry(&Category{ID: "c", Type: Series}, EntryInput{Title: "x", Status: Finished,
		StartEpisodes: 1, Fields: Fields{TotalEpisodes: 2}}, t0)
	if p, _ := e.LogSession(SessionInput{}, t0.Add(time.Hour)); p {
		t.Error("finished entry should not prompt")
	}
}

func TestLogSessionDefaults(t *testing.T) {
	tests := []struct {
		name    string
		typ     EntryType
		start   int
		in      SessionInput
		want    Session
		wantMin *int
	}{
		{"series defaults to 1 episode", Series, 0, SessionInput{}, Session{Episodes: 1}, nil},
		{"series 0 episodes is kept", Series, 0, SessionInput{Episodes: ip(0)}, Session{Episodes: 0}, nil},
		{"book from defaults to pages read", Book, 40, SessionInput{ToPage: ip(80)}, Session{FromPage: 40, ToPage: 80}, nil},
		{"book to defaults to from", Book, 40, SessionInput{}, Session{FromPage: 40, ToPage: 40}, nil},
		// Book session "from page 0" saves as 0.
		{"book from page 0 saves as 0", Book, 40, SessionInput{FromPage: ip(0), ToPage: ip(30)}, Session{FromPage: 0, ToPage: 30}, nil},
		{"book minutes kept", Book, 0, SessionInput{FromPage: ip(0), ToPage: ip(10), Minutes: ip(35)}, Session{ToPage: 10}, ip(35)},
		{"book 0 minutes kept as 0", Book, 0, SessionInput{FromPage: ip(0), ToPage: ip(10), Minutes: ip(0)}, Session{ToPage: 10}, ip(0)},
		{"other minutes", Other, 0, SessionInput{Minutes: ip(20)}, Session{}, ip(20)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := NewEntry(&Category{ID: "c", Type: tt.typ},
				EntryInput{Title: "x", Status: Ongoing, StartPages: tt.start}, t0)
			if _, err := e.LogSession(tt.in, t0.Add(time.Hour)); err != nil {
				t.Fatal(err)
			}
			s := e.Sessions[len(e.Sessions)-1]
			if s.Episodes != tt.want.Episodes || s.FromPage != tt.want.FromPage || s.ToPage != tt.want.ToPage {
				t.Errorf("session = eps %d pages %d–%d, want eps %d pages %d–%d",
					s.Episodes, s.FromPage, s.ToPage, tt.want.Episodes, tt.want.FromPage, tt.want.ToPage)
			}
			if (s.Minutes == nil) != (tt.wantMin == nil) || (s.Minutes != nil && *s.Minutes != *tt.wantMin) {
				t.Errorf("minutes = %v, want %v", s.Minutes, tt.wantMin)
			}
		})
	}
}

func TestLogSessionRejectsNegative(t *testing.T) {
	e, _ := NewEntry(&Category{ID: "c", Type: Book}, EntryInput{Title: "x", Status: Ongoing}, t0)
	if _, err := e.LogSession(SessionInput{ToPage: ip(-3)}, t0); !errors.Is(err, ErrNegative) {
		t.Errorf("err = %v, want ErrNegative", err)
	}
}

func TestEditAndDeleteSessions(t *testing.T) {
	e, _ := NewEntry(&Category{ID: "c", Type: Book}, EntryInput{Title: "x", Status: Ongoing,
		Review: "First.", StartPages: 20, Fields: Fields{TotalPages: 100}}, t0)
	e.LogSession(SessionInput{FromPage: ip(20), ToPage: ip(60)}, t0.Add(time.Hour))
	later := e.Sessions[1].ID
	start := e.StartSession().ID

	if got := e.PagesRead(); got != 60 {
		t.Fatalf("PagesRead = %d, want 60", got)
	}
	if err := e.DeleteSession(start); !errors.Is(err, ErrStartSession) {
		t.Errorf("deleting start: err = %v, want ErrStartSession", err)
	}
	// Editing recomputes progress, and can reach the end.
	prompt, err := e.EditSession(later, SessionInput{ToPage: ip(100), Note: "Done!"})
	if err != nil || !prompt {
		t.Errorf("edit to page 100: prompt=%v err=%v, want prompt", prompt, err)
	}
	if e.Sessions[1].FromPage != 20 {
		t.Errorf("blank from page on edit should keep 20, got %d", e.Sessions[1].FromPage)
	}
	// Editing the start note updates the review.
	if _, err := e.EditSession(start, SessionInput{Note: "Rewritten."}); err != nil {
		t.Fatal(err)
	}
	if e.Review != "Rewritten." {
		t.Errorf("review = %q, want start note", e.Review)
	}
	if err := e.DeleteSession(later); err != nil {
		t.Fatal(err)
	}
	if got := e.PagesRead(); got != 20 {
		t.Errorf("after delete PagesRead = %d, want 20", got)
	}
	if err := e.DeleteSession("nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestUpdateEntrySyncsReviewAndStart(t *testing.T) {
	l := newLib(t)
	dramas := mustCat(t, l, "Dramas", Series)
	e := mustEntry(t, l, dramas, EntryInput{Title: "Signal", StartEpisodes: 2,
		Fields: Fields{TotalEpisodes: 16}}, t0)
	e.LogSession(SessionInput{Episodes: ip(3)}, t0.Add(time.Hour))

	in := EntryInput{Title: "Signal", Status: Ongoing, Review: "Changed me.", StartEpisodes: 4,
		Fields: Fields{TotalEpisodes: 16}}
	if err := l.UpdateEntry(e.ID, dramas.ID, in, t0); err != nil {
		t.Fatal(err)
	}
	if s := e.StartSession(); s.Note != "Changed me." || s.Episodes != 4 {
		t.Errorf("start = %+v", *s)
	}
	if got := e.WatchedEpisodes(); got != 7 {
		t.Errorf("WatchedEpisodes = %d, want 4 + 3", got)
	}
	if e.StartEpisodes() != 4 {
		t.Errorf("StartEpisodes = %d", e.StartEpisodes())
	}
}

func TestUpdateEntryAddsStartWhenMissing(t *testing.T) {
	l := newLib(t)
	films := mustCat(t, l, "Films", Film)
	e := mustEntry(t, l, films, EntryInput{Title: "Dune"}, t0)
	if e.StartSession() != nil {
		t.Fatal("unexpected start session")
	}
	l.UpdateEntry(e.ID, films.ID, EntryInput{Title: "Dune", Status: Ongoing, Review: "Sand."}, t0.Add(time.Hour))
	s := e.StartSession()
	if s == nil || !s.At.Equal(t0) || s.Note != "Sand." {
		t.Errorf("start = %+v, want at createdAt with review", s)
	}
}

func TestRewatches(t *testing.T) {
	e, _ := NewEntry(&Category{ID: "c", Type: Series}, EntryInput{Title: "x", Status: Finished}, t0)
	if err := e.LogRewatch(RewatchInput{Note: "again"}, ist); !errors.Is(err, ErrDateRequired) {
		t.Errorf("err = %v, want ErrDateRequired", err)
	}
	d := Date{2026, 7, 1}
	if err := e.LogRewatch(RewatchInput{Date: d, Rating: 4}, ist); err != nil {
		t.Fatal(err)
	}
	r := e.Rewatches[0]
	if !r.Full || r.Rating != 4 || DateOf(r.At, ist) != d {
		t.Errorf("rewatch = %+v", r)
	}
	if e.Rating != 0 {
		t.Error("rewatch rating must not change the entry rating")
	}
	no := false
	if err := e.EditRewatch(r.ID, RewatchInput{Date: d, Full: &no}, ist); err != nil {
		t.Fatal(err)
	}
	if e.Rewatches[0].Full || e.Rewatches[0].ID != r.ID {
		t.Errorf("edited = %+v", e.Rewatches[0])
	}
	if err := e.DeleteRewatch(r.ID); err != nil || len(e.Rewatches) != 0 {
		t.Errorf("delete: %v, left %d", err, len(e.Rewatches))
	}
}
