package ui

import (
	"strconv"
	"strings"

	"github.com/saniyaa61/relic/core"
)

// The type-specific fields on the entry form (SPEC §3), laid out in rows
// as the prototype's FIELD_SCHEMAS, plus the owner's additions: podcasts
// get "Ep. length (mins)" like series, music gets "Length (mins)" beside
// Tracks.

type fieldKind int

const (
	textField fieldKind = iota
	numberField
	dateFieldKind
)

type fieldSpec struct {
	id, label, placeholder string
	kind                   fieldKind
}

// formRows lists each type's fields, one or two per row.
var formRows = map[core.EntryType][][]fieldSpec{
	core.Film: {
		{{"director", "Director", "Director", textField}, {"language", "Language", "e.g. English", textField}},
		{{"cast", "Cast", "Lead actors", textField}},
		{{"duration", "Duration (mins)", "e.g. 105", numberField}, {"watchedDate", "Date watched", "", dateFieldKind}},
	},
	core.Series: {
		{{"totalEpisodes", "Total episodes", "e.g. 16", numberField}, {"startEpisodes", "Watched so far", "e.g. 4", numberField}},
		{{"episodeDuration", "Ep. length (mins)", "e.g. 60", numberField}, {"platform", "Platform", "e.g. Netflix", textField}},
		{{"cast", "Cast", "Lead actors", textField}},
	},
	core.Book: {
		{{"author", "Author", "Author", textField}, {"startDate", "Started reading", "", dateFieldKind}},
		{{"totalPages", "Total pages", "e.g. 321", numberField}, {"startPages", "Pages read so far", "e.g. 80", numberField}},
		{{"publisher", "Publisher / Year", "e.g. Penguin, 2019", textField}},
	},
	core.Podcast: {
		{{"host", "Host / Creator", "Host name", textField}, {"platform", "Platform", "e.g. Spotify", textField}},
		{{"totalEpisodes", "Total episodes", "e.g. 200", numberField}, {"startEpisodes", "Episodes listened", "e.g. 12", numberField}},
		{{"episodeDuration", "Ep. length (mins)", "e.g. 45", numberField}, {"language", "Language", "e.g. English", textField}},
		{{"watchedDate", "Date started", "", dateFieldKind}},
	},
	core.Short: {
		{{"creator", "Creator / Channel", "Channel name", textField}, {"platform", "Platform", "e.g. YouTube", textField}},
		{{"duration", "Duration (mins)", "e.g. 12", numberField}, {"watchedDate", "Date watched", "", dateFieldKind}},
		{{"url", "Link (optional)", "https://", textField}},
	},
	core.Music: {
		{{"artist", "Artist", "Artist", textField}, {"genre", "Genre", "e.g. Folk", textField}},
		{{"tracks", "Tracks", "e.g. 12", numberField}, {"duration", "Length (mins)", "e.g. 44", numberField}},
		{{"watchedDate", "Release / Listen date", "", dateFieldKind}, {"platform", "Platform", "e.g. Spotify", textField}},
	},
	core.Other: {
		{{"creator", "Creator / Author", "Creator", textField}, {"platform", "Where", "e.g. Online", textField}},
		{{"durationText", "Duration / Length", "e.g. 2 hours", textField}, {"watchedDate", "Date", "", dateFieldKind}},
	},
}

func rowsFor(t core.EntryType) [][]fieldSpec {
	if r, ok := formRows[t]; ok {
		return r
	}
	return formRows[core.Other]
}

// fieldText reads a field's value for the form; numbers of 0 ("not
// given") and the start progress when nothing was logged show blank.
func fieldText(e *core.Entry, id string) string {
	f := e.Fields
	num := func(n int) string {
		if n == 0 {
			return ""
		}
		return strconv.Itoa(n)
	}
	switch id {
	case "director":
		return f.Director
	case "language":
		return f.Language
	case "cast":
		return f.Cast
	case "platform":
		return f.Platform
	case "author":
		return f.Author
	case "publisher":
		return f.Publisher
	case "host":
		return f.Host
	case "creator":
		return f.Creator
	case "artist":
		return f.Artist
	case "genre":
		return f.Genre
	case "url":
		return f.URL
	case "durationText":
		return f.DurationText
	case "duration":
		return num(f.Duration)
	case "episodeDuration":
		return num(f.EpisodeDuration)
	case "totalEpisodes":
		return num(f.TotalEpisodes)
	case "totalPages":
		return num(f.TotalPages)
	case "tracks":
		return num(f.Tracks)
	case "startEpisodes":
		return num(e.StartEpisodes())
	case "startPages":
		return num(e.StartPages())
	}
	return ""
}

// fieldDate reads a date field.
func fieldDate(f core.Fields, id string) core.Date {
	if id == "startDate" {
		return f.StartDate
	}
	return f.WatchedDate
}

// setField stores a text or number field into in. Numbers that aren't
// whole numbers count as blank (0, "not given"), as parseFloat||0 did.
func setField(in *core.EntryInput, id, text string) {
	text = strings.TrimSpace(text)
	n, _ := strconv.Atoi(text)
	n = max(n, 0)
	f := &in.Fields
	switch id {
	case "director":
		f.Director = text
	case "language":
		f.Language = text
	case "cast":
		f.Cast = text
	case "platform":
		f.Platform = text
	case "author":
		f.Author = text
	case "publisher":
		f.Publisher = text
	case "host":
		f.Host = text
	case "creator":
		f.Creator = text
	case "artist":
		f.Artist = text
	case "genre":
		f.Genre = text
	case "url":
		f.URL = text
	case "durationText":
		f.DurationText = text
	case "duration":
		f.Duration = n
	case "episodeDuration":
		f.EpisodeDuration = n
	case "totalEpisodes":
		f.TotalEpisodes = n
	case "totalPages":
		f.TotalPages = n
	case "tracks":
		f.Tracks = n
	case "startEpisodes":
		in.StartEpisodes = n
	case "startPages":
		in.StartPages = n
	}
}

// setDate stores a date field into in.
func setDate(in *core.EntryInput, id string, d core.Date) {
	if id == "startDate" {
		in.Fields.StartDate = d
	} else {
		in.Fields.WatchedDate = d
	}
}
