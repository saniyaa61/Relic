// Command import loads a prototype archive (relic-archive.json) into a Relic
// database, then reads it back and prints the counts, time totals and Top 5
// to compare with the prototype.
//
//	go run ./cmd/import -in relic-archive.json -db build/relic.db
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/saniyaa61/relic/importer"
	"github.com/saniyaa61/relic/store"
)

func main() {
	in := flag.String("in", "relic-archive.json", "prototype archive to import")
	dbPath := flag.String("db", "build/relic.db", "database to create or replace")
	flag.Parse()
	if err := run(*in, *dbPath); err != nil {
		fmt.Fprintln(os.Stderr, "import:", err)
		os.Exit(1)
	}
}

func run(in, dbPath string) error {
	f, err := os.Open(in)
	if err != nil {
		return err
	}
	defer f.Close()
	res, err := importer.Read(f, time.Now())
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return err
	}
	posters := filepath.Join(filepath.Dir(dbPath), "posters")
	if err := store.WritePosters(posters, res.Posters); err != nil {
		return err
	}
	db, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	// Keep the database's light/dark mode, as the app does on import.
	current, err := db.Load()
	if err != nil {
		return err
	}
	res.KeepMode(current.Profile)
	if err := db.ReplaceAll(res.Library); err != nil {
		return err
	}
	if err := store.RemoveUnusedPosters(posters, res.Library); err != nil {
		return err
	}

	r := res.Report
	fmt.Printf("Imported %d categories, %d entries, %d sessions, %d rewatches, %d posters into %s\n\n",
		r.Categories, r.Entries, r.Sessions, r.Rewatches, r.Posters, dbPath)
	if len(r.Warnings) > 0 {
		fmt.Println("Repairs and notes:")
		for _, w := range r.Warnings {
			fmt.Println("  -", w)
		}
		fmt.Println()
	}
	back, err := db.Load()
	if err != nil {
		return err
	}
	fmt.Println("Read back from the database:")
	fmt.Print(importer.Summarize(back))
	return nil
}
