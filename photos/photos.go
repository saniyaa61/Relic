// Package photos lets the user choose a picture from the phone's photo
// library: Android's photo picker (or the gallery before Android 13) and
// iOS's Photos picker. On other systems it isn't available and the app
// uses the file picker instead.
package photos

import (
	"errors"

	"gioui.org/app"
	"gioui.org/io/event"
)

var (
	// ErrNotAvailable is returned where there's no photo library.
	ErrNotAvailable = errors.New("photos: not available here")
	// ErrCancelled is returned when the user backs out of the picker.
	ErrCancelled = errors.New("photos: cancelled")
)

// Picker picks from one window. Pass it every window event with
// ListenEvents, as for x/explorer.
type Picker struct {
	w *app.Window
	picker
}

// New returns a Picker for w.
func New(w *app.Window) *Picker { return &Picker{w: w} }

// Available reports whether this system has a photo library picker.
func Available() bool { return available }

// ListenEvents keeps track of the native view the picker opens over.
func (p *Picker) ListenEvents(e event.Event) { p.listen(e) }

// Pick opens the picker and returns the chosen picture's bytes (JPEG,
// PNG, GIF or WebP; other formats such as HEIC come back as JPEG). It
// blocks until the user chooses or backs out, so call it off the
// window's event loop. One pick at a time.
func (p *Picker) Pick() ([]byte, error) { return p.pick() }

type result struct {
	data []byte
	err  error
}
