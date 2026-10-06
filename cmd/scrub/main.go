// Command scrub makes a test copy of a prototype archive with the personal
// parts replaced: titles, words, names, people and posters. Dates, numbers,
// ids, structure and the shape of each sentence (length, "!", ALL-CAPS
// words) are kept, so imports and totals behave exactly as on the original.
//
//	go run ./cmd/scrub -in relic-archive.json -out importer/testdata/scrubbed-archive.json
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
	"unicode"

	"github.com/saniyaa61/relic/core"
)

func main() {
	in := flag.String("in", "relic-archive.json", "archive to scrub")
	out := flag.String("out", "importer/testdata/scrubbed-archive.json", "where to write the scrubbed copy")
	flag.Parse()
	if err := run(*in, *out); err != nil {
		fmt.Fprintln(os.Stderr, "scrub:", err)
		os.Exit(1)
	}
}

// Text fields replaced letter by letter.
var textFields = []string{"review", "director", "language", "cast", "platform", "author", "publisher",
	"host", "creator", "artist", "genre", "url"}

func run(in, out string) error {
	b, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	var a map[string]any
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	if _, ok := a["userName"]; ok {
		a["userName"] = "Reader"
	}

	names := map[string]string{}   // old section name → new
	folders := map[string]string{} // old "section\x00folder" → new
	preset := map[string]bool{}
	for _, t := range core.PresetTags {
		preset[t] = true
	}
	plural := map[core.EntryType]string{core.Film: "Films", core.Series: "Series", core.Book: "Books",
		core.Podcast: "Podcasts", core.Short: "Shorts", core.Music: "Music", core.Other: "Things"}

	sections, _ := a["sections"].([]any)
	for i, s := range sections {
		sec, _ := s.(map[string]any)
		old, _ := sec["name"].(string)
		// Keep the name's type guess for sections saved without a type.
		t := core.EntryType(str(sec["type"]))
		if !t.Valid() {
			t = guess(old)
		}
		name := fmt.Sprintf("%s %d", plural[t], i+1)
		names[old] = name
		sec["name"] = name
		fs, _ := sec["folders"].([]any)
		for j, f := range fs {
			nf := fmt.Sprintf("Folder %d.%d", i+1, j+1)
			folders[old+"\x00"+str(f)] = nf
			fs[j] = nf
		}
	}

	customTags := map[string]string{}
	entries, _ := a["entries"].([]any)
	for i, x := range entries {
		e, _ := x.(map[string]any)
		sec := str(e["section"])
		if sec == "" {
			sec = str(e["category"])
		}
		for _, k := range []string{"section", "category"} {
			if v, ok := e[k].(string); ok {
				if n, ok := names[v]; ok {
					e[k] = n
				}
			}
		}
		if f := str(e["folder"]); f != "" {
			if n, ok := folders[sec+"\x00"+f]; ok {
				e["folder"] = n
			} else {
				e["folder"] = "Lost folder"
			}
		}
		e["title"] = fmt.Sprintf("Entry %d", i+1)
		for _, k := range textFields {
			if v, ok := e[k].(string); ok {
				e[k] = shape(v)
			}
		}
		if v, ok := e["duration"].(string); ok {
			e["duration"] = shape(v)
		}
		if tags, ok := e["tags"].([]any); ok {
			for j, t := range tags {
				s := str(t)
				if preset[s] {
					continue
				}
				if _, ok := customTags[s]; !ok {
					customTags[s] = fmt.Sprintf("Tag %d", len(customTags)+1)
				}
				tags[j] = customTags[s]
			}
		}
		for _, k := range []string{"sessions", "rewatches"} {
			list, _ := e[k].([]any)
			for _, y := range list {
				if m, ok := y.(map[string]any); ok {
					if n, ok := m["note"].(string); ok {
						m["note"] = shape(n)
					}
				}
			}
		}
		if p, ok := e["poster"].(string); ok && p != "" {
			e["poster"] = poster(p, i)
		}
	}
	for _, k := range []string{"topFavorites", "favoriteOrder"} {
		m, _ := a[k].(map[string]any)
		renamed := map[string]any{}
		for old, v := range m {
			if n, ok := names[old]; ok {
				renamed[n] = v
			} else {
				renamed[old] = v
			}
		}
		if m != nil {
			a[k] = renamed
		}
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(a); err != nil {
		return err
	}
	return os.WriteFile(out, buf.Bytes(), 0o644)
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

// shape replaces letters but keeps length, case, digits and punctuation.
func shape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsUpper(r):
			b.WriteRune('X')
		case unicode.IsLetter(r):
			b.WriteRune('x')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// guess mirrors the prototype's type guess from a section name.
func guess(name string) core.EntryType {
	n := strings.ToLower(name)
	for _, g := range []struct {
		t     core.EntryType
		words []string
	}{
		{core.Podcast, []string{"podcast"}},
		{core.Short, []string{"reel", "short", "youtube", "video", "clip"}},
		{core.Music, []string{"music", "album", "song", "concert", "gig"}},
		{core.Book, []string{"book", "novel", "read"}},
		{core.Series, []string{"drama", "series", "show", "tv", "anime", "episode"}},
	} {
		for _, w := range g.words {
			if strings.Contains(n, w) {
				return g.t
			}
		}
	}
	return core.Film
}

// poster is a small plain image in the original's format (WebP becomes PNG).
func poster(orig string, i int) string {
	img := image.NewRGBA(image.Rect(0, 0, 40, 60))
	c := color.RGBA{uint8(40 + 20*i), uint8(90 + 9*i), 140, 255}
	for y := 0; y < 60; y++ {
		for x := 0; x < 40; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	mime := "image/png"
	if strings.HasPrefix(orig, "data:image/jpeg") {
		mime = "image/jpeg"
		jpeg.Encode(&buf, img, nil)
	} else {
		png.Encode(&buf, img)
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}
