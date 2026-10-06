// Package archive writes and reads relic-archive.json (SPEC §8): the whole
// library, profile and favourites included, with the poster files inside,
// so an export can be imported again with nothing lost. Read also accepts
// the prototype's export (through the importer), so either file works.
package archive

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/saniyaa61/relic/core"
	"github.com/saniyaa61/relic/importer"
)

// FileName is the export's name.
const FileName = "relic-archive.json"

// Format marks an archive written by this app; Version is bumped when the
// layout changes.
const (
	Format  = "relic-archive"
	Version = 2 // 1 is the prototype's export, which has no format field
)

// file is the JSON layout. The library's own types are stored as they are
// (their exported field names are the keys); Version says how to read them.
type file struct {
	Format     string            `json:"format"`
	Version    int               `json:"version"`
	ExportedAt time.Time         `json:"exportedAt"`
	Profile    core.Profile      `json:"profile"`
	Categories []*core.Category  `json:"categories"`
	Entries    []*core.Entry     `json:"entries"`
	Favourites core.Favourites   `json:"favourites"`
	Posters    map[string][]byte `json:"posters"` // file name → JPEG (base64 in JSON)
}

// Write writes lib as an archive, reading its posters from posterDir. A
// poster file that's missing is left out (the entry keeps its reference,
// which the app already treats as "no poster").
func Write(w io.Writer, lib *core.Library, posterDir string, now time.Time) error {
	f := file{Format: Format, Version: Version, ExportedAt: now.UTC(), Profile: lib.Profile,
		Categories: lib.Categories, Entries: lib.Entries, Favourites: lib.Favourites,
		Posters: map[string][]byte{}}
	if f.Categories == nil {
		f.Categories = []*core.Category{}
	}
	if f.Entries == nil {
		f.Entries = []*core.Entry{}
	}
	for _, e := range lib.Entries {
		if e.Poster == "" || posterDir == "" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(posterDir, e.Poster))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		f.Posters[e.Poster] = b
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")
	return enc.Encode(f)
}

// ErrNotArchive is returned for a file that isn't a Relic archive.
var ErrNotArchive = errors.New("not a Relic archive")

// Read reads an archive written by Write, or a prototype export. now is
// used by the prototype importer for missing dates.
func Read(r io.Reader, now time.Time) (*importer.Result, error) {
	data, err := io.ReadAll(io.LimitReader(r, 512<<20))
	if err != nil {
		return nil, err
	}
	var head struct {
		Format  string `json:"format"`
		Version int    `json:"version"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotArchive, err)
	}
	if head.Format == "" {
		return importer.Read(bytes.NewReader(data), now)
	}
	if head.Format != Format {
		return nil, ErrNotArchive
	}
	if head.Version > Version {
		return nil, fmt.Errorf("this archive is from a newer version of Relic (format %d)", head.Version)
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotArchive, err)
	}
	lib := &core.Library{Profile: f.Profile, Categories: f.Categories, Entries: f.Entries, Favourites: f.Favourites}
	res := &importer.Result{Library: lib, Posters: map[string][]byte{}}
	if err := check(lib); err != nil {
		return nil, err
	}
	for _, e := range lib.Entries {
		if b, ok := f.Posters[e.Poster]; ok && e.Poster != "" {
			res.Posters[e.Poster] = b
		}
	}
	rep := &res.Report
	rep.Categories, rep.Entries, rep.Posters = len(lib.Categories), len(lib.Entries), len(res.Posters)
	for _, e := range lib.Entries {
		rep.Sessions += len(e.Sessions)
		rep.Rewatches += len(e.Rewatches)
	}
	return res, nil
}

// check refuses an archive whose pieces don't fit together.
func check(lib *core.Library) error {
	cats := map[string]bool{}
	for _, c := range lib.Categories {
		if c == nil || c.ID == "" || cats[c.ID] {
			return fmt.Errorf("%w: a category is missing or repeated", ErrNotArchive)
		}
		cats[c.ID] = true
	}
	ids := map[string]bool{}
	for _, e := range lib.Entries {
		if e == nil || e.ID == "" || ids[e.ID] {
			return fmt.Errorf("%w: an entry is missing or repeated", ErrNotArchive)
		}
		if !cats[e.CategoryID] {
			return fmt.Errorf("%w: %q belongs to a category that isn't in the archive", ErrNotArchive, e.Title)
		}
		ids[e.ID] = true
	}
	return nil
}
