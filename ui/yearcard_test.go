package ui

import (
	"bytes"
	"image/png"
	"io"
	"testing"
	"time"
)

// waitAsync runs background results once they arrive.
func waitAsync(t *testing.T, a *App) {
	t.Helper()
	for i := 0; i < 400; i++ {
		a.asyncMu.Lock()
		n := len(a.async)
		a.asyncMu.Unlock()
		if n > 0 {
			a.runAsync()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("background work never finished")
}

type bufCloser struct {
	bytes.Buffer
	closed bool
}

func (b *bufCloser) Close() error { b.closed = true; return nil }

// "Share your year" makes the card, previews it, and "Save image" writes a
// 1080 × 1350 PNG; cancelling the save dialog does nothing.
func TestYearCardSave(t *testing.T) {
	a, _ := libApp(t)
	year := a.Now().In(a.Loc).Year()
	openYearCard(a, year-50)
	if !a.toast.isErr {
		t.Errorf("an empty year should say so, got %q", a.toast.msg)
	}

	// Pretend the archive's year is this year's digest.
	yc := a.Lib.YearCard(a.timeIndex(), 2026, a.Loc)
	if !yc.YearHasData() {
		t.Skip("fixture has no 2026 data")
	}
	openYearCard(a, 2026)
	if a.toast.msg != "Making your card…" {
		t.Errorf("toast %q", a.toast.msg)
	}
	waitAsync(t, a)
	d := a.dialog
	if d == nil || d.Confirm != "Save image" || d.Cancel != "Close" {
		t.Fatalf("preview dialog: %+v", d)
	}
	frame(a)

	var names []string
	cancel := true
	out := &bufCloser{}
	a.CreateFile = func(name string) (io.WriteCloser, error) {
		names = append(names, name)
		if cancel {
			return nil, ErrNoFile
		}
		return out, nil
	}
	a.toast.msg = ""
	d.OnConfirm(a, "")
	time.Sleep(50 * time.Millisecond)
	a.runAsync()
	if a.toast.msg != "" || a.dialog != d || d.closing() {
		t.Errorf("a cancelled save: toast %q, dialog closing %v", a.toast.msg, d.closing())
	}
	cancel = false
	d.OnConfirm(a, "")
	waitAsync(t, a)
	if a.toast.msg != "Saved ✦" || !out.closed {
		t.Fatalf("save: toast %q, closed %v", a.toast.msg, out.closed)
	}
	if names[1] != "relic-2026-in-review.png" {
		t.Errorf("file name %q", names[1])
	}
	img, err := png.Decode(&out.Buffer)
	if err != nil || img.Bounds().Dx() != 1080 || img.Bounds().Dy() != 1350 {
		t.Errorf("saved image: %v, %v", img.Bounds(), err)
	}

	a.CreateFile = nil
	d.OnConfirm(a, "")
	if !a.toast.isErr {
		t.Error("no save dialog on this platform should say so")
	}
}
