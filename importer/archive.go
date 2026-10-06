package importer

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

// The prototype's export (SPEC §9). Its values are loosely typed: numbers
// can arrive as strings, fields can be missing or null, so every field
// decodes leniently.

type archive struct {
	UserName      string              `json:"userName"`
	Theme         string              `json:"theme"`
	CustomBase    string              `json:"customBase"`
	CustomAccent  string              `json:"customAccent"`
	Sections      []section           `json:"sections"`
	Entries       []entry             `json:"entries"`
	TopFavorites  map[string][]string `json:"topFavorites"`
	FavoriteOrder map[string][]string `json:"favoriteOrder"`
	CreatedAt     str                 `json:"createdAt"`
}

type section struct {
	ID      str   `json:"id"`
	Name    str   `json:"name"`
	Type    str   `json:"type"`
	Folders []str `json:"folders"`
}

type entry struct {
	ID         str       `json:"id"`
	Type       str       `json:"type"`
	Title      str       `json:"title"`
	Poster     str       `json:"poster"`
	Rating     num       `json:"rating"`
	Tags       []str     `json:"tags"`
	Review     str       `json:"review"`
	Section    str       `json:"section"`
	Category   str       `json:"category"`
	Folder     str       `json:"folder"`
	Status     str       `json:"status"`
	CreatedAt  str       `json:"createdAt"`
	FinishedAt str       `json:"finishedAt"`
	Favorite   bool      `json:"favorite"`
	Sessions   []session `json:"sessions"`
	Rewatches  []rewatch `json:"rewatches"`

	Director        str `json:"director"`
	Language        str `json:"language"`
	Cast            str `json:"cast"`
	Platform        str `json:"platform"`
	Author          str `json:"author"`
	Publisher       str `json:"publisher"`
	Host            str `json:"host"`
	Creator         str `json:"creator"`
	Artist          str `json:"artist"`
	Genre           str `json:"genre"`
	URL             str `json:"url"`
	Duration        str `json:"duration"` // minutes, or free text for "other"
	EpisodeDuration num `json:"episodeDuration"`
	TotalEpisodes   num `json:"totalEpisodes"`
	WatchedEpisodes num `json:"watchedEpisodes"`
	TotalPages      num `json:"totalPages"` // also "Tracks" for music
	PagesRead       num `json:"pagesRead"`
	WatchedDate     str `json:"watchedDate"`
	StartDate       str `json:"startDate"`
}

type session struct {
	Date        str  `json:"date"`
	Note        str  `json:"note"`
	First       bool `json:"first"`
	DoneAtStart bool `json:"doneAtStart"`
	Eps         num  `json:"eps"`
	From        num  `json:"from"`
	To          num  `json:"to"`
	Mins        num  `json:"mins"`
}

type rewatch struct {
	Date   str   `json:"date"`
	Note   str   `json:"note"`
	Rating num   `json:"rating"`
	Full   *bool `json:"full"`
	Start  str   `json:"start"`
	End    str   `json:"end"`
}

// str accepts a string, number, bool or null and keeps it as text.
type str string

func (s *str) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	switch {
	case bytes.Equal(b, []byte("null")):
		*s = ""
	case len(b) > 0 && b[0] == '"':
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		*s = str(v)
	default:
		*s = str(b)
	}
	return nil
}

func (s str) String() string { return string(s) }
func (s str) trim() string   { return strings.TrimSpace(string(s)) }

// num accepts a number or a numeric string. ok is false when the value is
// missing, null, empty or not a number.
type num struct {
	v  float64
	ok bool
}

func (n *num) UnmarshalJSON(b []byte) error {
	var s str
	if err := s.UnmarshalJSON(b); err != nil {
		return err
	}
	f, err := strconv.ParseFloat(s.trim(), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		*n = num{}
		return nil
	}
	*n = num{f, true}
	return nil
}

// int rounds to the nearest whole number; missing is 0.
func (n num) int() int {
	if !n.ok {
		return 0
	}
	return int(math.Round(n.v))
}
