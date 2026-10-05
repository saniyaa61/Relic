package core

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// NewID returns a random 128-bit id as 32 hex characters. Ids are stable for
// the life of a record so sync can match them later (SPEC §10).
func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("core: no randomness: " + err.Error())
	}
	return hex.EncodeToString(b[:])
}

// Status is where an entry stands. Only Ongoing and Finished are offered (SPEC §3).
type Status string

const (
	Ongoing  Status = "ongoing"
	Finished Status = "finished"
)

// Profile is the user's own settings (SPEC §2).
type Profile struct {
	Name         string
	FirstUsedAt  time.Time
	Theme        string // linen | midnight | blush | forest | rose | slate | custom
	Mode         string // light | dark
	CustomBase   string // hex colour, only when Theme is custom
	CustomAccent string
}

// Category is a user-created top-level shelf (SPEC §1).
type Category struct {
	ID        string
	Name      string
	Type      EntryType
	Folders   []string // ordered; unique within the category, case-insensitive
	CreatedAt time.Time
}

// Fields holds the type-specific values of an entry (SPEC §3). Which of them
// apply depends on the entry type; see FieldsFor.
//
// Totals and durations use 0 for "not given": a total of zero episodes or
// pages can't be reached, so it never counts as progress.
type Fields struct {
	Director  string
	Language  string
	Cast      string
	Platform  string
	Author    string
	Publisher string // "Publisher / Year"
	Host      string
	Creator   string
	Artist    string
	Genre     string
	URL       string

	Duration        int    // minutes (film, short)
	DurationText    string // free text (other; counts no time, see SPEC §11 #2)
	EpisodeDuration int    // minutes per episode (series, podcast)
	TotalEpisodes   int    // series, podcast
	TotalPages      int    // book
	Tracks          int    // music

	WatchedDate Date // film, podcast, short, music, other
	StartDate   Date // book
}

// Entry is one thing the user is watching, reading or listening to (SPEC §2).
// Progress (episodes watched, pages read) is never stored here; it is derived
// from Sessions.
type Entry struct {
	ID         string
	CategoryID string
	Folder     string // "" = Uncategorised
	Type       EntryType
	Title      string
	Poster     string  // reference to an image file, not the image itself
	Rating     float64 // 0–5 in 0.5 steps; 0 = unrated
	Tags       []string
	Review     string
	Status     Status
	CreatedAt  time.Time
	FinishedAt time.Time // zero unless Status is Finished
	Favorite   bool
	Fields     Fields
	Sessions   []Session // oldest first
	Rewatches  []Rewatch // oldest first
}

// Session is one sitting logged against an entry.
type Session struct {
	ID              string
	At              time.Time
	Note            string
	IsStart         bool // at most one per entry; can't be deleted
	StartedFinished bool // start session only: logged as already finished
	Episodes        int  // series, podcast
	FromPage        int  // book
	ToPage          int  // book
	Minutes         *int // book, other; nil = not logged (books then estimate)
}

// Rewatch is a later re-experience of an entry (reread, relisten).
type Rewatch struct {
	ID        string
	At        time.Time
	Note      string
	Rating    float64 // 0 = none; never changes the entry's rating
	Full      bool    // series/podcast: "Yes, all of it" vs "Partial rewatch"
	StartDate Date    // books, optional
	EndDate   Date    // books, optional
}

// Mins returns a pointer to n, for the optional Minutes fields.
func Mins(n int) *int { return &n }

// ValidRating reports whether r is 0–5 in half steps.
func ValidRating(r float64) bool {
	return r >= 0 && r <= 5 && r*2 == float64(int(r*2))
}
