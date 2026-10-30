//go:build android

package share

import (
	"errors"

	"gioui.org/app"
	"gioui.org/io/event"
	"git.wow.st/gmp/jni"
)

// share_android.jar holds share_android.java, compiled (see that file).

const available = true

type sharer struct {
	view uintptr
}

func (s *Sharer) listen(e event.Event) {
	if e, ok := e.(app.AndroidViewEvent); ok {
		s.view = e.View
	}
}

func (s *Sharer) image(name, title string, png []byte) error {
	view := s.view
	if view == 0 {
		return errors.New("share: the window isn't ready")
	}
	done := make(chan error, 1)
	go s.w.Run(func() {
		done <- jni.Do(jni.JVMFor(app.JavaVM()), func(env jni.Env) error {
			loader := jni.ClassLoaderFor(env, jni.Object(app.AppContext()))
			cls, err := jni.LoadClass(env, loader, "org/saniyaa61/relic/share/share_android")
			if err != nil {
				return err
			}
			m := jni.GetStaticMethodID(env, cls, "share",
				"(Landroid/view/View;[BLjava/lang/String;Ljava/lang/String;)Ljava/lang/String;")
			res, err := jni.CallStaticObjectMethod(env, cls, m,
				jni.Value(view),
				jni.Value(jni.NewByteArray(env, png)),
				jni.Value(jni.JavaString(env, name)),
				jni.Value(jni.JavaString(env, title)))
			if err != nil {
				return err
			}
			if res != 0 {
				return errors.New("share: " + jni.GoString(env, jni.String(res)))
			}
			return nil
		})
	})
	return <-done
}
