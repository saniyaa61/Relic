package core

import (
	"reflect"
	"testing"
	"time"
)

func TestSentences(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"One. Two!  Three? Four", []string{"One.", "Two!", "Three?", "Four"}},
		{"No split...here", []string{"No split...here"}},
		{"  Trailing.  ", []string{"Trailing."}},
		{"Line one.\nLine two.", []string{"Line one.", "Line two."}},
	}
	for _, tt := range tests {
		if got := Sentences(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Sentences(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSentenceScore(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"quiet", 5},
		{"wow!", 4 + 20},
		{"I LOVED it", 10 + 15},
		{"OK fine", 7},                 // 2-letter caps don't count
		{"THE END!", 8 + 20 + 15 + 15}, // two caps words
		{"café", 4},                    // characters, not bytes
	}
	for _, tt := range tests {
		if got := SentenceScore(tt.in); got != tt.want {
			t.Errorf("SentenceScore(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestYourWords(t *testing.T) {
	e, _ := NewEntry(&Category{ID: "c", Type: Series}, EntryInput{Title: "x", Status: Ongoing,
		Review: "It began softly. Nothing prepared me."}, t0)
	q, ok := e.YourWords()
	if !ok || q.Text != "Nothing prepared me." || !q.FromStart {
		t.Fatalf("got %+v", q)
	}
	if got := q.Label(time.UTC); got != "Started · 16 June 2026" {
		t.Errorf("label = %q", got)
	}
	// A later note can win; re-evaluated when notes change.
	later := t0.Add(24 * time.Hour)
	e.LogSession(SessionInput{Note: "I CRIED for an hour!"}, later)
	q, _ = e.YourWords()
	if q.Text != "I CRIED for an hour!" || q.FromStart || !q.At.Equal(later) {
		t.Errorf("got %+v", q)
	}
	if got := q.Label(time.UTC); got != "17 June 2026" {
		t.Errorf("label = %q", got)
	}
	// Ties go to the earliest sentence.
	f, _ := NewEntry(&Category{ID: "c", Type: Film}, EntryInput{Title: "x", Status: Ongoing, Review: "abc. xyz."}, t0)
	if q, _ := f.YourWords(); q.Text != "abc." {
		t.Errorf("tie: %q", q.Text)
	}
	g, _ := NewEntry(&Category{ID: "c", Type: Film}, EntryInput{Title: "x", Status: Ongoing}, t0)
	if _, ok := g.YourWords(); ok {
		t.Error("nothing written should give no quote")
	}
}

func TestStartVerb(t *testing.T) {
	tests := []struct {
		typ         EntryType
		cat, folder string
		want        string
	}{
		{Book, "", "", "Started reading"},
		{Podcast, "", "", "Started listening"},
		{Music, "", "", "Started listening"},
		{Film, "", "", "Started watching"},
		{Series, "", "", "Started watching"},
		{Short, "", "", "Started watching"},
		{Other, "Audiobooks", "", "Started listening"}, // listen beats read
		{Other, "Comics", "", "Started reading"},
		{Other, "Stuff", "Manga", "Started reading"},
		{Other, "ANIME", "", "Started watching"},
		{Other, "Board Games", "", "Started playing"},
		{Other, "Misc", "", "Started"},
	}
	for _, tt := range tests {
		if got := StartVerb(tt.typ, tt.cat, tt.folder); got != tt.want {
			t.Errorf("StartVerb(%s, %q, %q) = %q, want %q", tt.typ, tt.cat, tt.folder, got, tt.want)
		}
	}
}

func TestJourneyLabels(t *testing.T) {
	tests := []struct {
		name  string
		typ   EntryType
		in    EntryInput
		log   SessionInput
		start string
		later string
	}{
		{"series", Series, EntryInput{StartEpisodes: 3}, SessionInput{Episodes: ip(2)}, "Started watching · Ep 3", "Watched 2 episodes"},
		{"series one ep", Series, EntryInput{}, SessionInput{}, "Started watching", "Watched 1 episode"},
		{"podcast", Podcast, EntryInput{StartEpisodes: 1}, SessionInput{Episodes: ip(2)}, "Started listening · Ep 1", "Listened to 2 episodes"},
		{"book", Book, EntryInput{StartPages: 42}, SessionInput{FromPage: ip(40), ToPage: ip(80), Minutes: ip(35)}, "Started reading · p. 42", "Pages 40–80 · 35 min"},
		{"book from page 0", Book, EntryInput{}, SessionInput{FromPage: ip(0), ToPage: ip(12)}, "Started reading", "Pages 0–12"},
		{"other minutes", Other, EntryInput{Review: "Hi."}, SessionInput{Minutes: ip(35)}, "Started playing", "35 min"},
		{"film no minutes", Film, EntryInput{Review: "Hi."}, SessionInput{Note: "again"}, "Started watching", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.in.Title, tt.in.Status = "x", Ongoing
			e, _ := NewEntry(&Category{ID: "c", Type: tt.typ}, tt.in, t0)
			e.LogSession(tt.log, t0.Add(time.Hour))
			j := e.Journey("Games")
			if len(j) != 2 {
				t.Fatalf("journey has %d items", len(j))
			}
			if j[0].Label != tt.later || !j[0].CanDelete {
				t.Errorf("later = %q (deletable %v), want %q", j[0].Label, j[0].CanDelete, tt.later)
			}
			if j[1].Label != tt.start || j[1].CanDelete {
				t.Errorf("start = %q (deletable %v), want %q", j[1].Label, j[1].CanDelete, tt.start)
			}
		})
	}
}

func TestJourneyOrderAndRewatchPill(t *testing.T) {
	e, _ := NewEntry(&Category{ID: "c", Type: Book}, EntryInput{Title: "x", Status: Finished, StartPages: 300}, t0)
	e.LogRewatch(RewatchInput{Date: Date{2026, 8, 1}, Rating: 5}, time.UTC)
	e.LogSession(SessionInput{FromPage: ip(0), ToPage: ip(10)}, t0) // same instant as the start
	j := e.Journey("Books")
	if len(j) != 3 || !j[0].Rewatch || j[0].Pill != "Reread" || j[0].Rating != 5 {
		t.Fatalf("journey[0] = %+v", j[0])
	}
	if j[2].CanDelete {
		t.Error("start session should sort last on a tie")
	}
	for _, c := range []struct {
		t    EntryType
		want string
	}{{Film, "Rewatch"}, {Series, "Rewatch"}, {Podcast, "Relisten"}, {Music, "Relisten"}, {Book, "Reread"}} {
		if got := c.t.RewatchWord(); got != c.want {
			t.Errorf("%s: %q", c.t, got)
		}
	}
}
