package store

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/ncruces/go-sqlite3/driver" // SQLite translated to pure Go: no cgo, uses Go's own file I/O
)

// migrations[i] moves the schema from version i to i+1. Append only; never
// edit a migration that has shipped.
var migrations = []string{
	`CREATE TABLE meta (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`,
	// 2: the library. Times are RFC 3339 UTC text (NULL = not set); lists and
	// type-specific fields are JSON text.
	`CREATE TABLE profile (
		id            INTEGER PRIMARY KEY CHECK (id = 1),
		name          TEXT NOT NULL,
		first_used_at TEXT,
		theme         TEXT NOT NULL,
		mode          TEXT NOT NULL,
		custom_base   TEXT NOT NULL,
		custom_accent TEXT NOT NULL
	);
	CREATE TABLE categories (
		id         TEXT PRIMARY KEY,
		position   INTEGER NOT NULL,
		name       TEXT NOT NULL,
		type       TEXT NOT NULL,
		folders    TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	CREATE TABLE entries (
		id          TEXT PRIMARY KEY,
		category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
		folder      TEXT NOT NULL,
		type        TEXT NOT NULL,
		title       TEXT NOT NULL,
		poster      TEXT NOT NULL,
		rating      REAL NOT NULL,
		tags        TEXT NOT NULL,
		review      TEXT NOT NULL,
		status      TEXT NOT NULL,
		created_at  TEXT NOT NULL,
		finished_at TEXT,
		favorite    INTEGER NOT NULL,
		fields      TEXT NOT NULL
	);
	CREATE INDEX entries_category ON entries(category_id);
	CREATE TABLE sessions (
		id               TEXT PRIMARY KEY,
		entry_id         TEXT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
		at               TEXT NOT NULL,
		note             TEXT NOT NULL,
		is_start         INTEGER NOT NULL,
		started_finished INTEGER NOT NULL,
		episodes         INTEGER NOT NULL,
		from_page        INTEGER NOT NULL,
		to_page          INTEGER NOT NULL,
		minutes          INTEGER
	);
	CREATE INDEX sessions_entry ON sessions(entry_id);
	CREATE TABLE rewatches (
		id         TEXT PRIMARY KEY,
		entry_id   TEXT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
		at         TEXT NOT NULL,
		note       TEXT NOT NULL,
		rating     REAL NOT NULL,
		full       INTEGER NOT NULL,
		episodes   INTEGER NOT NULL,
		start_date TEXT NOT NULL,
		end_date   TEXT NOT NULL
	);
	CREATE INDEX rewatches_entry ON rewatches(entry_id);
	CREATE TABLE favourites (
		category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
		list        TEXT NOT NULL CHECK (list IN ('top', 'order')),
		position    INTEGER NOT NULL,
		entry_id    TEXT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
		PRIMARY KEY (category_id, list, position)
	)`,
}

// DB is Relic's local SQLite database.
type DB struct {
	sql *sql.DB
}

// Open opens (or creates) the database at path and brings its schema up to date.
func Open(path string) (*DB, error) {
	dsn := "file:" + path +
		"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	s, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	// One connection keeps writes ordered and avoids SQLITE_BUSY on a phone.
	s.SetMaxOpenConns(1)
	db := &DB{sql: s}
	if err := db.migrate(); err != nil {
		s.Close()
		return nil, err
	}
	return db, nil
}

// Close closes the database.
func (db *DB) Close() error { return db.sql.Close() }

// SchemaVersion reports how many migrations have been applied.
func (db *DB) SchemaVersion() (int, error) {
	var v int
	err := db.sql.QueryRow(`PRAGMA user_version`).Scan(&v)
	return v, err
}

func (db *DB) migrate() error {
	v, err := db.SchemaVersion()
	if err != nil {
		return fmt.Errorf("store: read schema version: %w", err)
	}
	if v > len(migrations) {
		return fmt.Errorf("store: database schema %d is newer than this app (%d)", v, len(migrations))
	}
	for ; v < len(migrations); v++ {
		tx, err := db.sql.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(migrations[v]); err != nil {
			tx.Rollback()
			return fmt.Errorf("store: migration %d: %w", v+1, err)
		}
		if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, v+1)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// Meta returns the value stored under key, and whether it exists.
func (db *DB) Meta(key string) (string, bool, error) {
	var v string
	err := db.sql.QueryRow(`SELECT value FROM meta WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

// SetMeta stores value under key, replacing any previous value.
func (db *DB) SetMeta(key, value string) error {
	_, err := db.sql.Exec(
		`INSERT INTO meta (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}
