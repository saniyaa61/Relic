//go:build ios

package tz

/*
#cgo CFLAGS: -Werror -xobjective-c -fmodules -fobjc-arc
#include <Foundation/Foundation.h>
#include <stdlib.h>
#include <string.h>

static char *relicZoneName(void) {
	NSString *name = [[NSTimeZone localTimeZone] name];
	return name == nil ? NULL : strdup(name.UTF8String);
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

// systemZone asks [NSTimeZone localTimeZone].
func systemZone() (string, error) {
	c := C.relicZoneName()
	if c == nil {
		return "", errors.New("tz: no zone")
	}
	defer C.free(unsafe.Pointer(c))
	return C.GoString(c), nil
}
