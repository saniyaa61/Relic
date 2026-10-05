package store

import (
	"path/filepath"
	"testing"
)

func TestOpenMigrates(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "relic.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	v, err := db.SchemaVersion()
	if err != nil {
		t.Fatal(err)
	}
	if v != len(migrations) {
		t.Fatalf("schema version = %d, want %d", v, len(migrations))
	}
}

func TestMetaSurvivesReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "relic.db")

	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := db.Meta("theme"); err != nil || ok {
		t.Fatalf("Meta on empty db: ok=%v err=%v, want missing", ok, err)
	}
	if err := db.SetMeta("theme", "linen"); err != nil {
		t.Fatal(err)
	}
	if err := db.SetMeta("theme", "sage"); err != nil {
		t.Fatal(err)
	}
	// Empty string is a value, not "missing".
	if err := db.SetMeta("name", ""); err != nil {
		t.Fatal(err)
	}
	db.Close()

	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tests := []struct {
		key, want string
		ok        bool
	}{
		{"theme", "sage", true},
		{"name", "", true},
		{"missing", "", false},
	}
	for _, tt := range tests {
		got, ok, err := db.Meta(tt.key)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want || ok != tt.ok {
			t.Errorf("Meta(%q) = %q, %v; want %q, %v", tt.key, got, ok, tt.want, tt.ok)
		}
	}
}

func TestOpenRejectsNewerSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "relic.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.sql.Exec(`PRAGMA user_version = 999`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if db, err := Open(path); err == nil {
		db.Close()
		t.Fatal("Open accepted a database from a newer app")
	}
}
