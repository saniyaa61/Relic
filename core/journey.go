package core

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// Quote is the "Your words" sentence on the entry detail (SPEC §4.7).
type Quote struct {
	Text      string
	At        time.Time
	FromStart bool // label it "Started · <date>"
}

// Label is "Started · 16 June 2026" for the start session, else the date.
func (q Quote) Label(loc *time.Location) string {
	if q.FromStart {
		return "Started · " + FormatDate(q.At, loc)
	}
	return FormatDate(q.At, loc)
}

var (
	sentenceEnd = regexp.MustCompile(`[.!?]\s+`)
	capsWord    = regexp.MustCompile(`\b[A-Z]{3,}\b`)
)

// Sentences splits text after ".", "!" or "?" followed by space.
func Sentences(text string) []string {
	var out []string
	last := 0
	for _, m := range sentenceEnd.FindAllStringIndex(text, -1) {
		if s := strings.TrimSpace(text[last : m[0]+1]); s != "" {
			out = append(out, s)
		}
		last = m[1]
	}
	if s := strings.TrimSpace(text[last:]); s != "" {
		out = append(out, s)
	}
	return out
}

// SentenceScore is length in characters, +20 with a "!", +15 per ALL-CAPS
// word of 3+ letters.
func SentenceScore(s string) int {
	n := utf8.RuneCountInString(s)
	if strings.Contains(s, "!") {
		n += 20
	}
	return n + 15*len(capsWord.FindAllString(s, -1))
}

// YourWords picks the single most powerful sentence the user has written
// about the entry, across the review and every session note. On a tie the
// earliest wins. ok is false when nothing has been written.
func (e *Entry) YourWords() (q Quote, ok bool) {
	best := -1
	consider := func(text string, at time.Time, start bool) {
		for _, s := range Sentences(text) {
			if sc := SentenceScore(s); sc > best {
				best, q = sc, Quote{Text: s, At: at, FromStart: start}
			}
		}
	}
	sessions := e.sessionsOldestFirst()
	if e.StartSession() == nil {
		consider(e.Review, e.CreatedAt, false)
	}
	for _, s := range sessions {
		consider(s.Note, s.At, s.IsStart)
	}
	return q, best >= 0
}

func (e *Entry) sessionsOldestFirst() []Session {
	s := append([]Session(nil), e.Sessions...)
	sort.SliceStable(s, func(i, j int) bool {
		if s[i].At.Equal(s[j].At) {
			return s[i].IsStart && !s[j].IsStart
		}
		return s[i].At.Before(s[j].At)
	})
	return s
}

var verbWords = []struct {
	re   *regexp.Regexp
	verb string
}{
	{regexp.MustCompile(`listen|audio|music|podcast|song|radio`), "Started listening"},
	{regexp.MustCompile(`read|book|novel|article|write|comic|manga`), "Started reading"},
	{regexp.MustCompile(`watch|movie|film|show|series|video|drama|anime`), "Started watching"},
	{regexp.MustCompile(`play|game`), "Started playing"},
}

// StartVerb is how the start session reads, by type (SPEC §4.8). For
// "other" it's guessed from the category and folder names.
func StartVerb(t EntryType, categoryName, folder string) string {
	switch t {
	case Book:
		return "Started reading"
	case Podcast, Music:
		return "Started listening"
	case Film, Series, Short:
		return "Started watching"
	}
	name := strings.ToLower(categoryName + " " + folder)
	for _, w := range verbWords {
		if w.re.MatchString(name) {
			return w.verb
		}
	}
	return "Started"
}

// JourneyItem is one row of the "Your journey" timeline.
type JourneyItem struct {
	ID        string // session or rewatch id
	Rewatch   bool
	At        time.Time
	Label     string  // e.g. "Started watching · Ep 3", "Pages 40–80 · 35 min"; may be ""
	Pill      string  // "Rewatch" / "Reread" / "Relisten" on rewatches
	Rating    float64 // rewatch rating, 0 = none
	Note      string
	CanDelete bool // everything but the start session
}

// Journey is the entry's timeline, newest first (SPEC §4.8). categoryName is
// used to pick the start verb for "other" entries.
func (e *Entry) Journey(categoryName string) []JourneyItem {
	var out []JourneyItem
	for _, s := range e.sessionsOldestFirst() {
		out = append(out, JourneyItem{ID: s.ID, At: s.At, Label: e.sessionLabel(s, categoryName),
			Note: s.Note, CanDelete: !s.IsStart})
	}
	for _, r := range e.Rewatches {
		out = append(out, JourneyItem{ID: r.ID, Rewatch: true, At: r.At, Pill: e.Type.RewatchWord(),
			Rating: r.Rating, Note: r.Note, CanDelete: true})
	}
	// Newest first; at equal times the start session (the only undeletable
	// item) sorts last, as the oldest.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].At.Equal(out[j].At) {
			return out[i].CanDelete && !out[j].CanDelete
		}
		return out[i].At.After(out[j].At)
	})
	return out
}

func (e *Entry) sessionLabel(s Session, categoryName string) string {
	if s.IsStart {
		if s.StartedFinished {
			return ""
		}
		verb := StartVerb(e.Type, categoryName, e.Folder)
		switch {
		case e.Type.Progress() == EpisodeProgress && s.Episodes > 0:
			return fmt.Sprintf("%s · Ep %d", verb, s.Episodes)
		case e.Type == Book && s.ToPage > 0:
			return fmt.Sprintf("%s · p. %d", verb, s.ToPage)
		}
		return verb
	}
	switch e.Type {
	case Series, Podcast:
		verb := "Watched"
		if e.Type == Podcast {
			verb = "Listened to"
		}
		unit := "episodes"
		if s.Episodes == 1 {
			unit = "episode"
		}
		return fmt.Sprintf("%s %d %s", verb, s.Episodes, unit)
	case Book:
		l := fmt.Sprintf("Pages %d–%d", s.FromPage, s.ToPage)
		if s.Minutes != nil {
			l += fmt.Sprintf(" · %d min", *s.Minutes)
		}
		return l
	}
	if s.Minutes != nil {
		return fmt.Sprintf("%d min", *s.Minutes)
	}
	return ""
}
