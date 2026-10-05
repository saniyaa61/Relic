package core

import (
	"fmt"
	"strings"
)

// The one search implementation (SPEC §4.9). Each page passes its own entry
// set (see the Library scope methods) and a scope label.

// Matches reports whether query (case-insensitive substring) appears in the
// entry's title, review, tags, category name, folder or any type-specific
// field. An empty query matches everything.
func Matches(e *Entry, categoryName, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	has := func(s string) bool { return strings.Contains(strings.ToLower(s), q) }
	if has(e.Title) || has(e.Review) || has(categoryName) || has(e.Folder) {
		return true
	}
	for _, t := range e.Tags {
		if has(t) {
			return true
		}
	}
	for _, f := range FieldsFor(e.Type, e.Fields) {
		if has(f.Value) {
			return true
		}
	}
	return false
}

// Search filters entries to those matching query, keeping their order.
func (l *Library) Search(entries []*Entry, query string) []*Entry {
	names := make(map[string]string, len(l.Categories))
	for _, c := range l.Categories {
		names[c.ID] = c.Name
	}
	var out []*Entry
	for _, e := range entries {
		if Matches(e, names[e.CategoryID], query) {
			out = append(out, e)
		}
	}
	return out
}

// ResultsHeader is "3 results for “dune” in Movies", or "… across your
// library" when scope is "".
func ResultsHeader(n int, query, scope string) string {
	word := "results"
	if n == 1 {
		word = "result"
	}
	where := "across your library"
	if scope != "" {
		where = "in " + scope
	}
	return fmt.Sprintf("%d %s for “%s” %s", n, word, strings.TrimSpace(query), where)
}

// NoResults is the empty-state line for a search.
func NoResults(query string) string {
	return fmt.Sprintf("Nothing matches “%s”.", strings.TrimSpace(query))
}
