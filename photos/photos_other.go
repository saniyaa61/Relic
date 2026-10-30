//go:build !android && !ios

package photos

import "gioui.org/io/event"

const available = false

type picker struct{}

func (p *Picker) listen(event.Event) {}

func (p *Picker) pick() ([]byte, error) { return nil, ErrNotAvailable }
