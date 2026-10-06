package archive

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/saniyaa61/relic/core"
	"github.com/saniyaa61/relic/importer"
	"github.com/saniyaa61/relic/store"
)

func prototypeLibrary(t *testing.T) (*importer.Result, []byte) {
	t.Helper()
	raw, err := os.ReadFile("../importer/testdata/scrubbed-archive.json")
	if err != nil {
		t.Fatal(err)
	}
	res, err := Read(bytes.NewReader(raw), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return res, raw
}

func asJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Exporting and importing again gives back the same library and posters,
// through the real database.
func TestRoundTrip(t *testing.T) {
	res, _ := prototypeLibrary(t) // a prototype export is read through the importer
	if res.Report.Entries == 0 {
		t.Fatal("fixture: no entries")
	}
	// Details the prototype can't express, to be sure they survive.
	e := res.Library.Entries[0]
	e.Sessions = append(e.Sessions, core.Session{ID: core.NewID(), At: e.CreatedAt.Add(time.Hour), Minutes: core.Mins(0)})

	dir := t.TempDir()
	posters := filepath.Join(dir, "posters")
	if err := store.WritePosters(posters, res.Posters); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(filepath.Join(dir, "relic.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.ReplaceAll(res.Library); err != nil {
		t.Fatal(err)
	}
	before, err := db.Load()
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := Write(&buf, before, posters, time.Now()); err != nil {
		t.Fatal(err)
	}
	back, err := Read(bytes.NewReader(buf.Bytes()), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := db.ReplaceAll(back.Library); err != nil {
		t.Fatal(err)
	}
	after, err := db.Load()
	if err != nil {
		t.Fatal(err)
	}
	if asJSON(t, before) != asJSON(t, after) {
		t.Error("the library changed on the way through an archive")
	}
	if len(back.Posters) != len(res.Posters) {
		t.Errorf("posters: %d back, %d sent", len(back.Posters), len(res.Posters))
	}
	for name, b := range back.Posters {
		if !bytes.Equal(b, res.Posters[name]) {
			t.Errorf("poster %s changed", name)
		}
	}
	if !strings.Contains(buf.String(), `"format": "relic-archive"`) {
		t.Error("the archive should say what it is")
	}
}

func TestReadRejects(t *testing.T) {
	for name, in := range map[string]string{
		"not json":         "hello",
		"other format":     `{"format":"something-else"}`,
		"orphan entry":     `{"format":"relic-archive","version":2,"categories":[],"entries":[{"ID":"a","CategoryID":"x","Title":"Lost"}]}`,
		"repeated entries": `{"format":"relic-archive","version":2,"categories":[{"ID":"c"}],"entries":[{"ID":"a","CategoryID":"c"},{"ID":"a","CategoryID":"c"}]}`,
	} {
		if _, err := Read(strings.NewReader(in), time.Now()); !errors.Is(err, ErrNotArchive) {
			t.Errorf("%s: err = %v, want ErrNotArchive", name, err)
		}
	}
	if _, err := Read(strings.NewReader(`{"format":"relic-archive","version":99}`), time.Now()); err == nil || errors.Is(err, ErrNotArchive) {
		t.Errorf("newer version: %v", err)
	}
	// An empty library is a valid archive.
	var buf bytes.Buffer
	if err := Write(&buf, &core.Library{}, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	if res, err := Read(&buf, time.Now()); err != nil || len(res.Library.Entries) != 0 {
		t.Errorf("empty: %v", err)
	}
}
