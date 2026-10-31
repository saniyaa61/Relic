//go:build android

package tz

import (
	"gioui.org/app"
	"git.wow.st/gmp/jni"
)

// systemZone asks java.util.TimeZone.getDefault().getID().
func systemZone() (string, error) {
	var name string
	err := jni.Do(jni.JVMFor(app.JavaVM()), func(env jni.Env) error {
		cls := jni.FindClass(env, "java/util/TimeZone")
		get := jni.GetStaticMethodID(env, cls, "getDefault", "()Ljava/util/TimeZone;")
		zone, err := jni.CallStaticObjectMethod(env, cls, get)
		if err != nil {
			return err
		}
		id := jni.GetMethodID(env, cls, "getID", "()Ljava/lang/String;")
		s, err := jni.CallObjectMethod(env, zone, id)
		if err != nil {
			return err
		}
		name = jni.GoString(env, jni.String(s))
		return nil
	})
	return name, err
}
