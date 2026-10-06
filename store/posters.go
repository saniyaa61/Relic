package store

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/saniyaa61/relic/core"
)

// Posters are image files in a folder next to the database; an entry's
// Poster field holds the file name only (SPEC §11 #4).

// WritePosters saves each poster under its file name in dir.
func WritePosters(dir string, posters map[string][]byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for name, b := range posters {
		if name != filepath.Base(name) || strings.HasPrefix(name, ".") {
			continue // never write outside dir
		}
		tmp := filepath.Join(dir, name+".tmp")
		if err := os.WriteFile(tmp, b, 0o644); err != nil {
			return err
		}
		if err := os.Rename(tmp, filepath.Join(dir, name)); err != nil {
			return err
		}
	}
	return nil
}

// RemoveUnusedPosters deletes files in dir that no entry in lib refers to.
func RemoveUnusedPosters(dir string, lib *core.Library) error {
	used := map[string]bool{}
	for _, e := range lib.Entries {
		used[e.Poster] = true
	}
	files, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, f := range files {
		if !f.IsDir() && !used[f.Name()] {
			if err := os.Remove(filepath.Join(dir, f.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
