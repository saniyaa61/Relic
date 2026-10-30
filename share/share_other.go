//go:build !android && !ios

package share

import "gioui.org/io/event"

const available = false

type sharer struct{}

func (s *Sharer) listen(event.Event) {}

func (s *Sharer) image(string, string, []byte) error { return ErrNotAvailable }
