//go:build ios

package share

/*
#cgo CFLAGS: -Werror -xobjective-c -fmodules -fobjc-arc

#include <UIKit/UIKit.h>
#include <stdlib.h>

// shareImage writes the PNG to the app's temporary folder under name and
// presents the share sheet for it over the window's topmost controller.
static int shareImage(CFTypeRef ctrlRef, const void *bytes, int n, const char *name) {
	UIViewController *ctrl = (__bridge UIViewController *)ctrlRef;
	NSData *data = [NSData dataWithBytes:bytes length:n];
	NSString *path = [NSTemporaryDirectory() stringByAppendingPathComponent:@(name)];
	if (![data writeToFile:path atomically:YES]) {
		return 0;
	}
	NSURL *url = [NSURL fileURLWithPath:path];
	UIActivityViewController *vc = [[UIActivityViewController alloc] initWithActivityItems:@[url] applicationActivities:nil];
	while (ctrl.presentedViewController != nil) {
		ctrl = ctrl.presentedViewController;
	}
	// iPads show the sheet as a popover, which needs an anchor.
	UIPopoverPresentationController *pop = vc.popoverPresentationController;
	if (pop != nil) {
		pop.sourceView = ctrl.view;
		pop.sourceRect = CGRectMake(CGRectGetMidX(ctrl.view.bounds), CGRectGetMidY(ctrl.view.bounds), 0, 0);
		pop.permittedArrowDirections = 0;
	}
	[ctrl presentViewController:vc animated:YES completion:nil];
	return 1;
}
*/
import "C"

import (
	"errors"
	"unsafe"

	"gioui.org/app"
	"gioui.org/io/event"
)

const available = true

type sharer struct {
	ctrl uintptr
}

func (s *Sharer) listen(e event.Event) {
	if e, ok := e.(app.UIKitViewEvent); ok {
		s.ctrl = e.ViewController
	}
}

func (s *Sharer) image(name, _ string, png []byte) error {
	ctrl := s.ctrl
	if ctrl == 0 {
		return errors.New("share: the window isn't ready")
	}
	if len(png) == 0 {
		return errors.New("share: empty image")
	}
	ok := false
	s.w.Run(func() {
		cname := C.CString(name)
		defer C.free(unsafe.Pointer(cname))
		ok = C.shareImage(C.CFTypeRef(ctrl), unsafe.Pointer(&png[0]), C.int(len(png)), cname) != 0
	})
	if !ok {
		return errors.New("share: the picture couldn't be written")
	}
	return nil
}
