//go:build ios

package photos

/*
#cgo CFLAGS: -Werror -xobjective-c -fmodules -fobjc-arc
#cgo LDFLAGS: -framework PhotosUI -framework UIKit

#include <CoreFoundation/CoreFoundation.h>

// In photos_ios.m.
extern int relicPickPhoto(CFTypeRef ctrl);
*/
import "C"

import (
	"errors"
	"unsafe"

	"gioui.org/app"
	"gioui.org/io/event"
)

const available = true

type picker struct {
	ctrl uintptr
}

// picked carries the Objective-C side's answer back to the waiting Pick.
var picked = make(chan result, 1)

func (p *Picker) listen(e event.Event) {
	if e, ok := e.(app.UIKitViewEvent); ok {
		p.ctrl = e.ViewController
	}
}

func (p *Picker) pick() ([]byte, error) {
	ctrl := p.ctrl
	if ctrl == 0 {
		return nil, errors.New("photos: the window isn't ready")
	}
	select {
	case <-picked: // a stale answer
	default:
	}
	ok := false
	p.w.Run(func() { ok = C.relicPickPhoto(C.CFTypeRef(ctrl)) != 0 })
	if !ok {
		return nil, errors.New("photos: the picker needs iOS 14 or newer")
	}
	r := <-picked
	return r.data, r.err
}

// relicPhotoPicked is called from photos_ios.m: status 1 with the JPEG,
// 0 if the user backed out, -1 if the picture couldn't be read.
//
//export relicPhotoPicked
func relicPhotoPicked(data unsafe.Pointer, n C.int, status C.int) {
	var r result
	switch status {
	case 1:
		r.data = C.GoBytes(data, n)
	case 0:
		r.err = ErrCancelled
	default:
		r.err = errors.New("photos: the picture couldn't be read")
	}
	select {
	case picked <- r:
	default:
	}
}
