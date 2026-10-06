package core

import "strconv"

// EntryType decides which fields an entry has (SPEC §3).
type EntryType string

const (
	Film    EntryType = "film"
	Series  EntryType = "series"
	Book    EntryType = "book"
	Podcast EntryType = "podcast"
	Short   EntryType = "short"
	Music   EntryType = "music"
	Other   EntryType = "other"
)

// EntryTypes lists every type in the order the New category dialog shows them.
var EntryTypes = []EntryType{Film, Series, Book, Podcast, Short, Music, Other}

// Valid reports whether t is one of EntryTypes.
func (t EntryType) Valid() bool {
	for _, x := range EntryTypes {
		if x == t {
			return true
		}
	}
	return false
}

// Label is the name shown when picking a type.
func (t EntryType) Label() string {
	switch t {
	case Film:
		return "Film"
	case Series:
		return "Series / Drama"
	case Book:
		return "Book"
	case Podcast:
		return "Podcast"
	case Short:
		return "Shorts / YouTube"
	case Music:
		return "Music / Album"
	}
	return "Other"
}

// ProgressUnit is what an entry's progress counts.
type ProgressUnit int

const (
	NoProgress ProgressUnit = iota
	EpisodeProgress
	PageProgress
)

// Progress reports what the type tracks: episodes (series, podcast), pages
// (book) or nothing.
func (t EntryType) Progress() ProgressUnit {
	switch t {
	case Series, Podcast:
		return EpisodeProgress
	case Book:
		return PageProgress
	}
	return NoProgress
}

// TracksProgress reports whether the type has episodes or pages to reach.
func (t EntryType) TracksProgress() bool { return t.Progress() != NoProgress }

// RewatchWord is the pill on a rewatch in the journey: Rewatch, Reread or Relisten.
func (t EntryType) RewatchWord() string {
	switch t {
	case Book:
		return "Reread"
	case Podcast, Music:
		return "Relisten"
	}
	return "Rewatch"
}

// PresetTags are the "How it felt" tags offered on the entry form (SPEC §3).
var PresetTags = []string{
	"Emotional", "Haunting", "Rewatch-worthy", "Quiet", "Profound", "Funny",
	"Disturbing", "Comforting", "Life-changing", "Beautiful", "Slow-burn",
	"Bittersweet", "Gothic", "Romantic", "Devastating",
}

// FieldValue is one type-specific field with its value as text.
type FieldValue struct {
	ID    string // stable key, e.g. "director", "totalEpisodes"
	Value string // "" when not given
}

// FieldsFor returns the type-specific fields of t in form order, with their
// values from f. Numbers that are 0 (not given) and unset dates come back as "".
// The derived progress fields (episodes watched, pages read) are not included.
func FieldsFor(t EntryType, f Fields) []FieldValue {
	num := func(n int) string {
		if n == 0 {
			return ""
		}
		return strconv.Itoa(n)
	}
	date := func(d Date) string {
		if d.IsZero() {
			return ""
		}
		return d.String()
	}
	switch t {
	case Film:
		return []FieldValue{{"director", f.Director}, {"language", f.Language}, {"cast", f.Cast},
			{"duration", num(f.Duration)}, {"watchedDate", date(f.WatchedDate)}}
	case Series:
		return []FieldValue{{"totalEpisodes", num(f.TotalEpisodes)}, {"episodeDuration", num(f.EpisodeDuration)},
			{"platform", f.Platform}, {"cast", f.Cast}}
	case Book:
		return []FieldValue{{"author", f.Author}, {"startDate", date(f.StartDate)},
			{"totalPages", num(f.TotalPages)}, {"publisher", f.Publisher}}
	case Podcast:
		return []FieldValue{{"host", f.Host}, {"platform", f.Platform}, {"totalEpisodes", num(f.TotalEpisodes)},
			{"episodeDuration", num(f.EpisodeDuration)}, {"language", f.Language}, {"watchedDate", date(f.WatchedDate)}}
	case Short:
		return []FieldValue{{"creator", f.Creator}, {"platform", f.Platform}, {"duration", num(f.Duration)},
			{"watchedDate", date(f.WatchedDate)}, {"url", f.URL}}
	case Music:
		return []FieldValue{{"artist", f.Artist}, {"genre", f.Genre}, {"tracks", num(f.Tracks)}, {"duration", num(f.Duration)},
			{"watchedDate", date(f.WatchedDate)}, {"platform", f.Platform}}
	}
	return []FieldValue{{"creator", f.Creator}, {"platform", f.Platform}, {"duration", f.DurationText},
		{"watchedDate", date(f.WatchedDate)}}
}
