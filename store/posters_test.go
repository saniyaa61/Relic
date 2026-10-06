package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saniyaa61/relic/core"
)

func TestPosters(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "posters")
	err := WritePosters(dir, map[string][]byte{"a.jpg": []byte("a"), "b.jpg": []byte("b"), "../evil.jpg": []byte("x")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "evil.jpg")); !os.IsNotExist(err) {
		t.Error("wrote outside the posters folder")
	}
	lib := &core.Library{Entries: []*core.Entry{{Poster: "a.jpg"}}}
	if err := RemoveUnusedPosters(dir, lib); err != nil {
		t.Fatal(err)
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 1 || files[0].Name() != "a.jpg" {
		t.Errorf("left %v", files)
	}
	if err := RemoveUnusedPosters(filepath.Join(root, "none"), lib); err != nil {
		t.Errorf("missing folder: %v", err)
	}
}
