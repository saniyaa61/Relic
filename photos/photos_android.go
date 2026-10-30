//go:build android

package photos

/*
#include <jni.h>
*/
import "C"

import (
	"errors"
	"unsafe"

	"gioui.org/app"
	"gioui.org/io/event"
	"git.wow.st/gmp/jni"
)

// photos_android.jar holds photos_android.java, compiled (see that file).

const available = true

type picker struct {
	view uintptr
}

// picked carries the Java side's answer back to the waiting Pick.
var picked = make(chan result, 1)

func (p *Picker) listen(e event.Event) {
	if e, ok := e.(app.AndroidViewEvent); ok {
		p.view = e.View
	}
}

func (p *Picker) pick() ([]byte, error) {
	view := p.view
	if view == 0 {
		return nil, errors.New("photos: the window isn't ready")
	}
	select {
	case <-picked: // a stale answer
	default:
	}
	err := make(chan error, 1)
	go p.w.Run(func() {
		err <- jni.Do(jni.JVMFor(app.JavaVM()), func(env jni.Env) error {
			loader := jni.ClassLoaderFor(env, jni.Object(app.AppContext()))
			cls, err := jni.LoadClass(env, loader, "org/saniyaa61/relic/photos/photos_android")
			if err != nil {
				return err
			}
			m := jni.GetStaticMethodID(env, cls, "pick", "(Landroid/view/View;)V")
			return jni.CallStaticVoidMethod(env, cls, m, jni.Value(view))
		})
	})
	if e := <-err; e != nil {
		return nil, e
	}
	r := <-picked
	return r.data, r.err
}

//export Java_org_saniyaa61_relic_photos_photos_1android_onPicked
func Java_org_saniyaa61_relic_photos_photos_1android_onPicked(env *C.JNIEnv, _ C.jclass, data C.jbyteArray, msg C.jstring) {
	e := jni.EnvFor(uintptr(unsafe.Pointer(env)))
	var r result
	switch {
	case msg != 0:
		r.err = errors.New("photos: " + jni.GoString(e, jni.String(msg)))
	case data == 0:
		r.err = ErrCancelled
	default:
		r.data = jni.GetByteArrayElements(e, jni.ByteArray(data))
	}
	select {
	case picked <- r:
	default:
	}
}
