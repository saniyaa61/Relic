package core

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrTitleRequired    = errors.New("Please add a title.")
	ErrCategoryRequired = errors.New("Please choose a category.")
	ErrBadStatus        = errors.New("core: status must be ongoing or finished")
	ErrBadRating        = errors.New("core: rating must be 0–5 in half steps")
	ErrNegative         = errors.New("core: counts can't be negative")
	ErrNotFound         = errors.New("core: not found")
	ErrStartSession     = errors.New("Your first entry can't be deleted here — edit it instead.")
	ErrDateRequired     = errors.New("core: a date is required")
)

// EntryInput is what the New / Edit entry form collects.
type EntryInput struct {
	Folder string
	Title  string
	Poster string
	Rating float64
	Tags   []string
	Review string
	Status Status
	Fields Fields
	// "Watched so far" / "Pages read so far": how far the user already was
	// when they first logged the entry. Stored on the start session.
	StartEpisodes int
	StartPages    int
}

func (in EntryInput) validate() error {
	if strings.TrimSpace(in.Title) == "" {
		return ErrTitleRequired
	}
	if in.Status != Ongoing && in.Status != Finished {
		return ErrBadStatus
	}
	if !ValidRating(in.Rating) {
		return ErrBadRating
	}
	if in.StartEpisodes < 0 || in.StartPages < 0 {
		return ErrNegative
	}
	return nil
}

// NewEntry creates an entry in cat at time now (SPEC §4.1). It gets a start
// session when it has a review or its type tracks progress. The caller adds
// it to the library, which puts it at the top.
func NewEntry(cat *Category, in EntryInput, now time.Time) (*Entry, error) {
	if cat == nil {
		return nil, ErrCategoryRequired
	}
	if err := in.validate(); err != nil {
		return nil, err
	}
	now = now.UTC()
	e := &Entry{
		ID:         NewID(),
		CategoryID: cat.ID,
		Folder:     in.Folder,
		Type:       cat.Type,
		Title:      strings.TrimSpace(in.Title),
		Poster:     in.Poster,
		Rating:     in.Rating,
		Tags:       append([]string(nil), in.Tags...),
		Review:     in.Review,
		Status:     in.Status,
		CreatedAt:  now,
		Fields:     in.Fields,
	}
	if e.Status == Finished {
		e.FinishedAt = now
	}
	if strings.TrimSpace(e.Review) != "" || e.Type.TracksProgress() {
		s := Session{
			ID:              NewID(),
			At:              now,
			Note:            e.Review,
			IsStart:         true,
			StartedFinished: e.Status == Finished,
		}
		setStartProgress(&s, e.Type, in.StartEpisodes, in.StartPages)
		e.Sessions = append(e.Sessions, s)
	}
	return e, nil
}

func setStartProgress(s *Session, t EntryType, episodes, pages int) {
	s.Episodes, s.FromPage, s.ToPage = 0, 0, 0
	switch t.Progress() {
	case EpisodeProgress:
		s.Episodes = episodes
	case PageProgress:
		s.ToPage = pages
	}
}

// update applies an edit-form save to e (the entry half of Library.UpdateEntry).
func (e *Entry) update(cat *Category, in EntryInput, now time.Time) error {
	if cat == nil {
		return ErrCategoryRequired
	}
	if err := in.validate(); err != nil {
		return err
	}
	e.CategoryID = cat.ID
	e.Type = cat.Type
	e.Folder = in.Folder
	e.Title = strings.TrimSpace(in.Title)
	e.Poster = in.Poster
	e.Rating = in.Rating
	e.Tags = append([]string(nil), in.Tags...)
	e.Review = in.Review
	e.Fields = in.Fields
	e.SetStatus(in.Status, now)

	if s := e.StartSession(); s != nil {
		s.Note = e.Review
		setStartProgress(s, e.Type, in.StartEpisodes, in.StartPages)
	} else if strings.TrimSpace(e.Review) != "" || e.Type.TracksProgress() {
		s := Session{ID: NewID(), At: e.CreatedAt, Note: e.Review, IsStart: true,
			StartedFinished: e.Status == Finished}
		setStartProgress(&s, e.Type, in.StartEpisodes, in.StartPages)
		e.Sessions = append([]Session{s}, e.Sessions...)
	}
	return nil
}

// SetStatus is the one place status changes (SPEC §4.2). Becoming finished
// sets FinishedAt to at; leaving finished clears it; the same status changes
// nothing.
func (e *Entry) SetStatus(s Status, at time.Time) {
	if s == e.Status {
		return
	}
	e.Status = s
	if s == Finished {
		e.FinishedAt = at.UTC()
	} else {
		e.FinishedAt = time.Time{}
	}
}

// StartSession returns the entry's start session, or nil.
func (e *Entry) StartSession() *Session {
	for i := range e.Sessions {
		if e.Sessions[i].IsStart {
			return &e.Sessions[i]
		}
	}
	return nil
}

// StartEpisodes and StartPages are the "so far" values the edit form shows.
func (e *Entry) StartEpisodes() int {
	if s := e.StartSession(); s != nil {
		return s.Episodes
	}
	return 0
}

func (e *Entry) StartPages() int {
	if s := e.StartSession(); s != nil {
		return s.ToPage
	}
	return 0
}

// SessionInput is what the Log a session sheet collects. Nil numbers mean the
// field was left blank. When logging, blanks take defaults: 1 episode; from
// page = pages read so far; to page = from page. When editing, blanks keep
// the session's current value. Zero is always a real value.
type SessionInput struct {
	Episodes *int
	FromPage *int
	ToPage   *int
	Minutes  *int
	Note     string
}

func (in SessionInput) validate() error {
	for _, p := range []*int{in.Episodes, in.FromPage, in.ToPage, in.Minutes} {
		if p != nil && *p < 0 {
			return ErrNegative
		}
	}
	return nil
}

// LogSession adds a session at now. It reports whether to ask "Looks like you
// reached the end! Mark as finished?" (SPEC §4.4).
func (e *Entry) LogSession(in SessionInput, now time.Time) (prompt bool, err error) {
	if err := in.validate(); err != nil {
		return false, err
	}
	s := Session{ID: NewID(), At: now.UTC(), Note: in.Note}
	switch e.Type.Progress() {
	case EpisodeProgress:
		s.Episodes = 1
		if in.Episodes != nil {
			s.Episodes = *in.Episodes
		}
	case PageProgress:
		s.FromPage = e.PagesRead()
		if in.FromPage != nil {
			s.FromPage = *in.FromPage
		}
		s.ToPage = s.FromPage
		if in.ToPage != nil {
			s.ToPage = *in.ToPage
		}
	}
	if in.Minutes != nil && e.Type != Series && e.Type != Podcast {
		s.Minutes = Mins(*in.Minutes)
	}
	e.Sessions = append(e.Sessions, s)
	return e.ShouldPromptFinished(), nil
}

// EditSession changes a session (the start session included). Editing the
// start session's note also changes the entry's review. It reports whether to
// show the finished prompt.
func (e *Entry) EditSession(id string, in SessionInput) (prompt bool, err error) {
	if err := in.validate(); err != nil {
		return false, err
	}
	s := e.session(id)
	if s == nil {
		return false, ErrNotFound
	}
	if in.Episodes != nil {
		s.Episodes = *in.Episodes
	}
	if in.FromPage != nil {
		s.FromPage = *in.FromPage
	}
	if in.ToPage != nil {
		s.ToPage = *in.ToPage
	}
	if in.Minutes != nil {
		s.Minutes = Mins(*in.Minutes)
	}
	s.Note = in.Note
	if s.IsStart {
		e.Review = in.Note
	}
	return e.ShouldPromptFinished(), nil
}

// DeleteSession removes a session. The start session can't be deleted.
func (e *Entry) DeleteSession(id string) error {
	for i := range e.Sessions {
		if e.Sessions[i].ID == id {
			if e.Sessions[i].IsStart {
				return ErrStartSession
			}
			e.Sessions = append(e.Sessions[:i], e.Sessions[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (e *Entry) session(id string) *Session {
	for i := range e.Sessions {
		if e.Sessions[i].ID == id {
			return &e.Sessions[i]
		}
	}
	return nil
}

// RewatchInput is what the Log a rewatch sheet collects.
type RewatchInput struct {
	Date      Date // the day it happened; stored at local noon
	Note      string
	Rating    float64 // 0 = no new rating
	Full      *bool   // series/podcast; nil = true ("Yes, all of it")
	Episodes  int     // partial rewatch: "Episodes rewatched"
	StartDate Date    // books
	EndDate   Date    // books
}

func (in RewatchInput) rewatch(id string, loc *time.Location) (Rewatch, error) {
	if in.Date.IsZero() {
		return Rewatch{}, ErrDateRequired
	}
	if !ValidRating(in.Rating) {
		return Rewatch{}, ErrBadRating
	}
	if in.Episodes < 0 {
		return Rewatch{}, ErrNegative
	}
	r := Rewatch{ID: id, At: in.Date.Noon(loc).UTC(), Note: in.Note, Rating: in.Rating,
		Full: true, StartDate: in.StartDate, EndDate: in.EndDate}
	if in.Full != nil {
		r.Full = *in.Full
	}
	if !r.Full {
		r.Episodes = in.Episodes
	}
	return r, nil
}

// LogRewatch adds a rewatch on in.Date (local day in loc).
func (e *Entry) LogRewatch(in RewatchInput, loc *time.Location) error {
	r, err := in.rewatch(NewID(), loc)
	if err != nil {
		return err
	}
	e.Rewatches = append(e.Rewatches, r)
	return nil
}

// EditRewatch replaces the rewatch with the given id.
func (e *Entry) EditRewatch(id string, in RewatchInput, loc *time.Location) error {
	for i := range e.Rewatches {
		if e.Rewatches[i].ID == id {
			r, err := in.rewatch(id, loc)
			if err != nil {
				return err
			}
			e.Rewatches[i] = r
			return nil
		}
	}
	return ErrNotFound
}

// DeleteRewatch removes the rewatch with the given id.
func (e *Entry) DeleteRewatch(id string) error {
	for i := range e.Rewatches {
		if e.Rewatches[i].ID == id {
			e.Rewatches = append(e.Rewatches[:i], e.Rewatches[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}
