// Package share opens the system share sheet for an image: Android's
// chooser and iOS's share sheet. On other systems (Windows, desktop Linux)
// it isn't available and the app offers only Save image.
package share

import (
	"errors"

	"gioui.org/app"
	"gioui.org/io/event"
)

// ErrNotAvailable is returned where there's no share sheet.
var ErrNotAvailable = errors.New("share: not available here")

// Sharer shares from one window. Pass it every window event with
// ListenEvents, as for x/explorer.
type Sharer struct {
	w *app.Window
	sharer
}

// New returns a Sharer for w.
func New(w *app.Window) *Sharer { return &Sharer{w: w} }

// Available reports whether this system has a share sheet.
func Available() bool { return available }

// ListenEvents keeps track of the native view the sheet opens over.
func (s *Sharer) ListenEvents(e event.Event) { s.listen(e) }

// Image opens the share sheet for a PNG called name, with title as the
// sheet's heading where the system shows one. It returns once the sheet
// is on its way; what the user then does (share or cancel) isn't
// reported. Call it off the window's event loop.
func (s *Sharer) Image(name, title string, png []byte) error {
	return s.image(name, title, png)
}
