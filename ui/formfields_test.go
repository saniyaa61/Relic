package ui

import (
	"testing"
	"time"

	"github.com/saniyaa61/relic/core"
)

// Every field on the form round-trips: what the form shows is what it
// saves, and every field of the type is on the form.
func TestFormFieldsRoundTrip(t *testing.T) {
	for _, typ := range core.EntryTypes {
		var in core.EntryInput
		n := 0
		for _, row := range rowsFor(typ) {
			if len(row) < 1 || len(row) > 2 {
				t.Errorf("%s: row of %d fields", typ, len(row))
			}
			for _, f := range row {
				n++
				if f.kind == dateFieldKind {
					setDate(&in, f.id, core.Date{Year: 2026, Month: 6, Day: n})
				} else {
					setField(&in, f.id, "1"+string(rune('0'+n%10)))
				}
			}
		}
		lib := &core.Library{}
		c, err := lib.AddCategory("c", typ, testNow)
		if err != nil {
			t.Fatal(err)
		}
		in.Title, in.Status = "x", core.Ongoing
		e, err := lib.AddEntry(c.ID, in, testNow)
		if err != nil {
			t.Fatal(err)
		}
		m := 0
		for _, row := range rowsFor(typ) {
			for _, f := range row {
				m++
				if f.kind == dateFieldKind {
					if got := fieldDate(e.Fields, f.id); got.Day != m {
						t.Errorf("%s %s: date %v", typ, f.id, got)
					}
					continue
				}
				if got, want := fieldText(e, f.id), "1"+string(rune('0'+m%10)); got != want {
					t.Errorf("%s %s = %q, want %q", typ, f.id, got, want)
				}
			}
		}
		// Every non-empty type field shows somewhere on the form.
		onForm := map[string]bool{}
		for _, row := range rowsFor(typ) {
			for _, f := range row {
				onForm[f.id] = true
			}
		}
		for _, fv := range core.FieldsFor(typ, e.Fields) {
			id := fv.ID
			if id == "duration" && typ == core.Other {
				id = "durationText"
			}
			if !onForm[id] {
				t.Errorf("%s: field %s isn't on the form", typ, fv.ID)
			}
		}
	}
	// Blank and junk numbers mean "not given".
	var in core.EntryInput
	setField(&in, "duration", " ")
	setField(&in, "totalPages", "abc")
	setField(&in, "tracks", "-3")
	if in.Fields.Duration != 0 || in.Fields.TotalPages != 0 || in.Fields.Tracks != 0 {
		t.Errorf("blank/junk numbers: %+v", in.Fields)
	}
}

var testNow = time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC)
