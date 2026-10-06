package ui

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/saniyaa61/relic/core"
)

func TestNewEntryForm(t *testing.T) {
	a, reload := libApp(t)
	a.Go(TabNew)
	frame(a)
	f := a.roots[TabNew].(*entryForm)
	series := a.Lib.Categories[1]

	// No title: refused with the prototype's words, nothing saved.
	n := len(a.Lib.Entries)
	f.submit(a)
	if a.toast.msg != "Please add a title." || !a.toast.isErr || len(reload().Entries) != n {
		t.Fatalf("toast %q, %d entries", a.toast.msg, len(reload().Entries))
	}

	// Choose the series category: its fields replace the film ones.
	f.catID = series.ID
	frame(a)
	if f.fieldsType != core.Series || f.fields["totalEpisodes"] == nil || f.fields["director"] != nil {
		t.Fatalf("fields after switching to a series category: type %s", f.fieldsType)
	}
	f.foldDrop.selected = 2 // "— No folder —", then the folders
	f.title.Editor.SetText("  Pachinko ")
	f.fields["totalEpisodes"].Editor.SetText("8")
	f.fields["startEpisodes"].Editor.SetText("5")
	f.fields["episodeDuration"].Editor.SetText("60")
	f.status = core.Ongoing
	f.rating = 4.5
	f.review.Editor.SetText("It stays with you.")
	for _, tag := range []string{"Quiet", "Haunting", "Beautiful", "Profound", "Funny"} {
		f.tags.toggle(tag)
	}
	if len(f.tags.tags) != 4 {
		t.Errorf("%d tags, want at most 4", len(f.tags.tags))
	}
	f.submit(a)
	if a.Tab() != TabHome || a.toast.msg != "Entry preserved ✦" {
		t.Errorf("after save: tab %v, toast %q", a.Tab(), a.toast.msg)
	}
	lib := reload()
	e := lib.Entries[0]
	if e.Title != "Pachinko" || e.CategoryID != series.ID || e.Folder != series.Folders[1] || e.Type != core.Series {
		t.Fatalf("saved %+v", e)
	}
	if e.Fields.TotalEpisodes != 8 || e.Fields.EpisodeDuration != 60 || e.WatchedEpisodes() != 5 || e.Rating != 4.5 || len(e.Tags) != 4 {
		t.Errorf("saved fields %+v, watched %d, rating %v, tags %v", e.Fields, e.WatchedEpisodes(), e.Rating, e.Tags)
	}
	if s := e.StartSession(); s == nil || s.Note != "It stays with you." {
		t.Error("start session should hold the review")
	}

	// Opening New again starts from an empty form.
	a.Go(TabNew)
	frame(a)
	if g := a.roots[TabNew].(*entryForm); g.title.Editor.Text() != "" || g.status != core.Finished {
		t.Error("New should start empty, as finished")
	}
}

func TestEditEntryForm(t *testing.T) {
	a, reload := libApp(t)
	e := a.Lib.Entries[0]
	films := a.Lib.Categories[0]
	a.Push(newEntryDetail(e.ID))
	f := editEntryPage(a, e)
	a.Push(f)
	frame(a)
	if f.title.Editor.Text() != e.Title || f.fields["totalEpisodes"].Editor.Text() == "" {
		t.Fatalf("edit form not filled in: title %q", f.title.Editor.Text())
	}
	f.title.Editor.SetText("Renamed")
	f.catID = films.ID // moving category changes the type
	frame(a)
	f.submit(a)
	if _, ok := a.top().(*entryDetail); !ok || a.toast.msg != "Entry updated ✦" {
		t.Errorf("after saving an edit: top %T, toast %q", a.top(), a.toast.msg)
	}
	got := reload().Entry(e.ID)
	if got.Title != "Renamed" || got.CategoryID != films.ID || got.Type != core.Film {
		t.Errorf("edit saved %q in %s as %s", got.Title, got.CategoryID, got.Type)
	}
}

// Tapping the left half of a star gives a half star.
func TestStarInputHalves(t *testing.T) {
	th, _ := NewTheme(LinenLight)
	var s starInput
	var r input.Router
	rating := 0.0
	run := func() {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Constraints{Max: image.Pt(400, 100)}, Source: r.Source()}
		rating = s.Layout(gtx, th, rating)
		r.Frame(gtx.Ops)
	}
	tap := func(x float32) {
		r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Position: f32.Pt(x, 12)},
			pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: f32.Pt(x, 12)})
		run()
	}
	run()
	// Each star is 1+24+1 wide with 2 between: the third starts at 56.
	tap(56 + 5)
	if rating != 2.5 {
		t.Errorf("left half of star 3 = %v, want 2.5", rating)
	}
	tap(56 + 20)
	if rating != 3 {
		t.Errorf("right half of star 3 = %v, want 3", rating)
	}
}
