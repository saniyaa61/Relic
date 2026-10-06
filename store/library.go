package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/saniyaa61/relic/core"
)

// Store is what the UI uses to keep the library. *DB implements it.
type Store interface {
	// Load reads the whole library.
	Load() (*core.Library, error)
	// ReplaceAll swaps everything for lib (import, clear all data).
	ReplaceAll(lib *core.Library) error
	// Update runs fn in one transaction: all its writes land, or none do.
	Update(fn func(w Writer) error) error
	Close() error
}

// Writer saves parts of the library inside an Update.
type Writer interface {
	SaveProfile(p core.Profile) error
	// SaveCategories stores the categories in this order. Categories not in
	// the list are deleted with their entries.
	SaveCategories(cats []*core.Category) error
	// SaveEntry inserts or replaces an entry with its sessions and rewatches.
	SaveEntry(e *core.Entry) error
	DeleteEntry(id string) error
	// SaveFavourites replaces all Top 5 lists and favourite orders.
	SaveFavourites(f core.Favourites) error
}

var _ Store = (*DB)(nil)

// Update runs fn in one transaction.
func (db *DB) Update(fn func(w Writer) error) error {
	tx, err := db.sql.Begin()
	if err != nil {
		return err
	}
	if err := fn(writer{tx}); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// ReplaceAll deletes everything and writes lib in its place.
func (db *DB) ReplaceAll(lib *core.Library) error {
	return db.Update(func(w Writer) error {
		tx := w.(writer).tx
		for _, t := range []string{"favourites", "rewatches", "sessions", "entries", "categories", "profile"} {
			if _, err := tx.Exec("DELETE FROM " + t); err != nil {
				return err
			}
		}
		if err := w.SaveProfile(lib.Profile); err != nil {
			return err
		}
		if err := w.SaveCategories(lib.Categories); err != nil {
			return err
		}
		for _, e := range lib.Entries {
			if err := w.SaveEntry(e); err != nil {
				return fmt.Errorf("store: entry %q: %w", e.Title, err)
			}
		}
		return w.SaveFavourites(lib.Favourites)
	})
}

type writer struct{ tx *sql.Tx }

// Times are stored as fixed-width RFC 3339 UTC with nanoseconds: they read
// back exactly, and sort as text in time order.
const timeLayout = "2006-01-02T15:04:05.000000000Z"

func timeText(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(timeLayout)
}

func parseTime(s sql.NullString) (time.Time, error) {
	if !s.Valid {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, s.String)
}

func jsonText(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (w writer) SaveProfile(p core.Profile) error {
	_, err := w.tx.Exec(`INSERT INTO profile (id, name, first_used_at, theme, mode, custom_base, custom_accent)
		VALUES (1, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET name = excluded.name, first_used_at = excluded.first_used_at,
			theme = excluded.theme, mode = excluded.mode, custom_base = excluded.custom_base,
			custom_accent = excluded.custom_accent`,
		p.Name, timeText(p.FirstUsedAt), p.Theme, p.Mode, p.CustomBase, p.CustomAccent)
	return err
}

func (w writer) SaveCategories(cats []*core.Category) error {
	keep := make([]any, 0, len(cats))
	for i, c := range cats {
		folders := c.Folders
		if folders == nil {
			folders = []string{}
		}
		f, err := jsonText(folders)
		if err != nil {
			return err
		}
		if _, err := w.tx.Exec(`INSERT INTO categories (id, position, name, type, folders, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET position = excluded.position, name = excluded.name,
				type = excluded.type, folders = excluded.folders, created_at = excluded.created_at`,
			c.ID, i, c.Name, string(c.Type), f, timeText(c.CreatedAt)); err != nil {
			return err
		}
		keep = append(keep, c.ID)
	}
	// Delete the rest (their entries go with them, by ON DELETE CASCADE).
	q := `DELETE FROM categories`
	if len(keep) > 0 {
		q += ` WHERE id NOT IN (?` + repeat(",?", len(keep)-1) + `)`
	}
	_, err := w.tx.Exec(q, keep...)
	return err
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

func (w writer) SaveEntry(e *core.Entry) error {
	tags := e.Tags
	if tags == nil {
		tags = []string{}
	}
	tagsText, err := jsonText(tags)
	if err != nil {
		return err
	}
	fields, err := jsonText(e.Fields)
	if err != nil {
		return err
	}
	// REPLACE would delete the row and cascade to its favourites, so upsert.
	if _, err := w.tx.Exec(`INSERT INTO entries (id, category_id, folder, type, title, poster, rating, tags,
			review, status, created_at, finished_at, favorite, fields)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET category_id = excluded.category_id, folder = excluded.folder,
			type = excluded.type, title = excluded.title, poster = excluded.poster, rating = excluded.rating,
			tags = excluded.tags, review = excluded.review, status = excluded.status,
			created_at = excluded.created_at, finished_at = excluded.finished_at,
			favorite = excluded.favorite, fields = excluded.fields`,
		e.ID, e.CategoryID, e.Folder, string(e.Type), e.Title, e.Poster, e.Rating, tagsText,
		e.Review, string(e.Status), timeText(e.CreatedAt), timeText(e.FinishedAt), boolInt(e.Favorite), fields); err != nil {
		return err
	}
	for _, q := range []string{`DELETE FROM sessions WHERE entry_id = ?`, `DELETE FROM rewatches WHERE entry_id = ?`} {
		if _, err := w.tx.Exec(q, e.ID); err != nil {
			return err
		}
	}
	for _, s := range e.Sessions {
		var mins any
		if s.Minutes != nil {
			mins = *s.Minutes
		}
		if _, err := w.tx.Exec(`INSERT INTO sessions (id, entry_id, at, note, is_start, started_finished,
				episodes, from_page, to_page, minutes) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			s.ID, e.ID, timeText(s.At), s.Note, boolInt(s.IsStart), boolInt(s.StartedFinished),
			s.Episodes, s.FromPage, s.ToPage, mins); err != nil {
			return err
		}
	}
	for _, r := range e.Rewatches {
		if _, err := w.tx.Exec(`INSERT INTO rewatches (id, entry_id, at, note, rating, full, episodes,
				start_date, end_date) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			r.ID, e.ID, timeText(r.At), r.Note, r.Rating, boolInt(r.Full), r.Episodes,
			dateText(r.StartDate), dateText(r.EndDate)); err != nil {
			return err
		}
	}
	return nil
}

func dateText(d core.Date) string {
	b, _ := d.MarshalText()
	return string(b)
}

func (w writer) DeleteEntry(id string) error {
	_, err := w.tx.Exec(`DELETE FROM entries WHERE id = ?`, id)
	return err
}

func (w writer) SaveFavourites(f core.Favourites) error {
	if _, err := w.tx.Exec(`DELETE FROM favourites`); err != nil {
		return err
	}
	for list, m := range map[string]map[string][]string{"top": f.TopFive, "order": f.Order} {
		for cat, ids := range m {
			for i, id := range ids {
				// Skip ids that no longer exist rather than fail the save.
				if _, err := w.tx.Exec(`INSERT INTO favourites (category_id, list, position, entry_id)
					SELECT ?, ?, ?, ? WHERE EXISTS (SELECT 1 FROM entries WHERE id = ?)
						AND EXISTS (SELECT 1 FROM categories WHERE id = ?)`,
					cat, list, i, id, id, cat); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Load reads the whole library. Entries come back newest first.
func (db *DB) Load() (*core.Library, error) {
	tx, err := db.sql.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	lib := &core.Library{Favourites: core.Favourites{TopFive: map[string][]string{}, Order: map[string][]string{}}}

	var first sql.NullString
	err = tx.QueryRow(`SELECT name, first_used_at, theme, mode, custom_base, custom_accent FROM profile WHERE id = 1`).
		Scan(&lib.Profile.Name, &first, &lib.Profile.Theme, &lib.Profile.Mode, &lib.Profile.CustomBase, &lib.Profile.CustomAccent)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("store: profile: %w", err)
	}
	if lib.Profile.FirstUsedAt, err = parseTime(first); err != nil {
		return nil, err
	}

	if err := loadCategories(tx, lib); err != nil {
		return nil, fmt.Errorf("store: categories: %w", err)
	}
	byID, err := loadEntries(tx, lib)
	if err != nil {
		return nil, fmt.Errorf("store: entries: %w", err)
	}
	if err := loadSessions(tx, byID); err != nil {
		return nil, fmt.Errorf("store: sessions: %w", err)
	}
	if err := loadRewatches(tx, byID); err != nil {
		return nil, fmt.Errorf("store: rewatches: %w", err)
	}
	if err := loadFavourites(tx, lib); err != nil {
		return nil, fmt.Errorf("store: favourites: %w", err)
	}
	return lib, nil
}

func loadCategories(tx *sql.Tx, lib *core.Library) error {
	rows, err := tx.Query(`SELECT id, name, type, folders, created_at FROM categories ORDER BY position`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		c := &core.Category{}
		var typ, folders string
		var created sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &typ, &folders, &created); err != nil {
			return err
		}
		c.Type = core.EntryType(typ)
		if err := json.Unmarshal([]byte(folders), &c.Folders); err != nil {
			return err
		}
		if c.CreatedAt, err = parseTime(created); err != nil {
			return err
		}
		lib.Categories = append(lib.Categories, c)
	}
	return rows.Err()
}

func loadEntries(tx *sql.Tx, lib *core.Library) (map[string]*core.Entry, error) {
	rows, err := tx.Query(`SELECT id, category_id, folder, type, title, poster, rating, tags, review, status,
		created_at, finished_at, favorite, fields FROM entries ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byID := map[string]*core.Entry{}
	for rows.Next() {
		e := &core.Entry{}
		var typ, status, tags, fields string
		var created, finished sql.NullString
		var fav int
		if err := rows.Scan(&e.ID, &e.CategoryID, &e.Folder, &typ, &e.Title, &e.Poster, &e.Rating, &tags,
			&e.Review, &status, &created, &finished, &fav, &fields); err != nil {
			return nil, err
		}
		e.Type, e.Status, e.Favorite = core.EntryType(typ), core.Status(status), fav != 0
		if err := json.Unmarshal([]byte(tags), &e.Tags); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(fields), &e.Fields); err != nil {
			return nil, err
		}
		if e.CreatedAt, err = parseTime(created); err != nil {
			return nil, err
		}
		if e.FinishedAt, err = parseTime(finished); err != nil {
			return nil, err
		}
		lib.Entries = append(lib.Entries, e)
		byID[e.ID] = e
	}
	return byID, rows.Err()
}

func loadSessions(tx *sql.Tx, byID map[string]*core.Entry) error {
	rows, err := tx.Query(`SELECT id, entry_id, at, note, is_start, started_finished, episodes, from_page,
		to_page, minutes FROM sessions ORDER BY at, is_start DESC, rowid`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var s core.Session
		var entryID string
		var at sql.NullString
		var start, startedFinished int
		var mins sql.NullInt64
		if err := rows.Scan(&s.ID, &entryID, &at, &s.Note, &start, &startedFinished, &s.Episodes,
			&s.FromPage, &s.ToPage, &mins); err != nil {
			return err
		}
		if s.At, err = parseTime(at); err != nil {
			return err
		}
		s.IsStart, s.StartedFinished = start != 0, startedFinished != 0
		if mins.Valid {
			s.Minutes = core.Mins(int(mins.Int64))
		}
		if e := byID[entryID]; e != nil {
			e.Sessions = append(e.Sessions, s)
		}
	}
	return rows.Err()
}

func loadRewatches(tx *sql.Tx, byID map[string]*core.Entry) error {
	rows, err := tx.Query(`SELECT id, entry_id, at, note, rating, full, episodes, start_date, end_date
		FROM rewatches ORDER BY at, rowid`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var r core.Rewatch
		var entryID, start, end string
		var at sql.NullString
		var full int
		if err := rows.Scan(&r.ID, &entryID, &at, &r.Note, &r.Rating, &full, &r.Episodes, &start, &end); err != nil {
			return err
		}
		if r.At, err = parseTime(at); err != nil {
			return err
		}
		r.Full = full != 0
		if err := r.StartDate.UnmarshalText([]byte(start)); err != nil {
			return err
		}
		if err := r.EndDate.UnmarshalText([]byte(end)); err != nil {
			return err
		}
		if e := byID[entryID]; e != nil {
			e.Rewatches = append(e.Rewatches, r)
		}
	}
	return rows.Err()
}

func loadFavourites(tx *sql.Tx, lib *core.Library) error {
	rows, err := tx.Query(`SELECT category_id, list, entry_id FROM favourites ORDER BY category_id, list, position`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cat, list, id string
		if err := rows.Scan(&cat, &list, &id); err != nil {
			return err
		}
		if list == "top" {
			lib.Favourites.TopFive[cat] = append(lib.Favourites.TopFive[cat], id)
		} else {
			lib.Favourites.Order[cat] = append(lib.Favourites.Order[cat], id)
		}
	}
	return rows.Err()
}
