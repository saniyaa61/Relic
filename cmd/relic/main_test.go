package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saniyaa61/relic/core"
	"github.com/saniyaa61/relic/store"
)

// The temporary -import flag loads an archive, keeps the current mode and
// writes the posters.
func TestImportArchive(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "relic.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Update(func(w store.Writer) error { return w.SaveProfile(core.Profile{Theme: "linen", Mode: "dark"}) }); err != nil {
		t.Fatal(err)
	}
	posters := filepath.Join(dir, "posters")
	if err := importArchive(db, "../../importer/testdata/scrubbed-archive.json", posters); err != nil {
		t.Fatal(err)
	}
	lib, err := db.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(lib.Categories) != 8 || len(lib.Entries) != 15 || lib.Profile.Mode != "dark" {
		t.Errorf("%d categories, %d entries, mode %q; want 8, 15, dark", len(lib.Categories), len(lib.Entries), lib.Profile.Mode)
	}
	files, _ := os.ReadDir(posters)
	if len(files) != 15 {
		t.Errorf("%d poster files, want 15", len(files))
	}
}
