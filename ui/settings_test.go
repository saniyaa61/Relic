package ui

import (
	"bytes"
	"io"
	"testing"

	"github.com/saniyaa61/relic/core"
)

func settingsApp(t *testing.T) (*App, *settingsPage, func() *core.Library) {
	t.Helper()
	a, reload := libApp(t)
	p := &settingsPage{}
	a.Push(p)
	frame(a)
	return a, p, reload
}

func TestSettingsThemeAndName(t *testing.T) {
	a, p, reload := settingsApp(t)
	p.themes[1].Click() // Midnight
	frame(a)
	p.modes[1].Click() // Dark
	frame(a)
	if got := reload().Profile; got.Theme != "midnight" || got.Mode != "dark" {
		t.Errorf("saved theme %q mode %q", got.Theme, got.Mode)
	}
	if a.Theme.Bg != PaletteFor(core.Profile{Theme: "midnight", Mode: "dark"}).Bg {
		t.Error("the app's colours should follow")
	}

	// Edit, type, save. An empty name keeps the old one (prototype).
	p.edit.Click()
	frame(a)
	p.name.Editor.SetText("  Saniya ")
	p.edit.Click()
	frame(a)
	if got := reload().Profile.Name; got != "Saniya" || a.toast.msg != "Name saved." {
		t.Errorf("name %q, toast %q", got, a.toast.msg)
	}
	p.edit.Click()
	frame(a)
	p.name.Editor.SetText("   ")
	p.edit.Click()
	frame(a)
	if got := reload().Profile.Name; got != "Saniya" {
		t.Errorf("an empty name replaced it: %q", got)
	}

	// Custom: the picker's colours apply as the custom palette.
	p.themes[6].Click()
	frame(a)
	p.custom.accent = "#3A5A8A"
	p.custom.apply.Click()
	frame(a)
	if got := reload().Profile; got.Theme != "custom" || got.CustomAccent != "#3A5A8A" || got.CustomBase == "" {
		t.Errorf("custom: %+v", got)
	}
}

type memFile struct{ bytes.Buffer }

func (m *memFile) Close() error { return nil }

// Export writes an archive; importing it after a change brings the
// library back, with its theme but not its light/dark mode.
func TestSettingsExportImport(t *testing.T) {
	a, p, reload := settingsApp(t)
	before := len(a.Lib.Entries)
	a.SetTheme("forest", "light", "", "")
	out := &memFile{}
	a.CreateFile = func(name string) (io.WriteCloser, error) {
		if name != "relic-archive.json" {
			t.Errorf("file name %q", name)
		}
		return out, nil
	}
	p.export.Click()
	frame(a)
	waitAsync(t, a)
	if a.toast.msg != "Archive exported." || out.Len() == 0 {
		t.Fatalf("export: toast %q, %d bytes", a.toast.msg, out.Len())
	}

	// Change things, then import the archive back.
	a.SetTheme("rose", "dark", "", "")
	a.Lib.DeleteEntry(a.Lib.Entries[0].ID)
	data := out.Bytes()
	a.ChooseFile = func(ext ...string) (io.ReadCloser, error) {
		if len(ext) != 1 || ext[0] != ".json" {
			t.Errorf("extensions %v", ext)
		}
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	p.imp.Click()
	frame(a)
	waitAsync(t, a)
	if a.dialog == nil || a.dialog.Title != "Import this archive?" {
		t.Fatalf("no confirmation before replacing: %+v", a.dialog)
	}
	a.dialog.OnConfirm(a, "")
	lib := reload()
	if len(lib.Entries) != before || lib.Profile.Theme != "forest" || lib.Profile.Mode != "dark" {
		t.Errorf("after import: %d entries (want %d), theme %q mode %q", len(lib.Entries), before, lib.Profile.Theme, lib.Profile.Mode)
	}
	if a.toast.msg != "Archive imported." || a.Tab() != TabHome || len(a.stack) != 0 {
		t.Errorf("toast %q, tab %v, %d pages open", a.toast.msg, a.Tab(), len(a.stack))
	}
	frame(a)

	// A file that isn't an archive changes nothing.
	a.ChooseFile = func(...string) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader([]byte("not json"))), nil
	}
	chooseArchive(a)
	waitAsync(t, a)
	if !a.toast.isErr || a.toast.msg != "Could not read that file." {
		t.Errorf("bad file: %q", a.toast.msg)
	}
}

// Clear all data asks twice, then empties everything but the theme.
func TestSettingsClearAll(t *testing.T) {
	a, p, reload := settingsApp(t)
	a.SetTheme("slate", "dark", "", "")
	p.clear.Click()
	frame(a)
	first := a.dialog
	if first == nil {
		t.Fatal("no first confirmation")
	}
	first.OnConfirm(a, "")
	if a.dialog == first || a.dialog == nil {
		t.Fatal("no second confirmation")
	}
	if len(reload().Entries) == 0 {
		t.Fatal("cleared after one confirmation")
	}
	a.dialog.OnConfirm(a, "")
	lib := reload()
	if len(lib.Entries) != 0 || len(lib.Categories) != 0 || lib.Profile.Name != "" {
		t.Errorf("left %d entries, %d categories, name %q", len(lib.Entries), len(lib.Categories), lib.Profile.Name)
	}
	if lib.Profile.Theme != "slate" || lib.Profile.Mode != "dark" {
		t.Errorf("theme %q / %q should stay", lib.Profile.Theme, lib.Profile.Mode)
	}
	frame(a)
}
