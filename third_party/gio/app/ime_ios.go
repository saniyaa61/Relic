// SPDX-License-Identifier: Unlicense OR MIT

//go:build ios

package app

// Relic patch: the Go side of UITextInput (os_ios.m), so iOS can offer
// autocorrect, predictive text and dictation. iOS counts text in UTF-16
// units from the start of the field; the editor state counts runes, so
// every crossing converts. The editor is asked for its whole text (a
// snippet from 0 to "everything"), since UITextInput reads the document
// freely.

/*
#include <CoreFoundation/CoreFoundation.h>
#include <stdint.h>

__attribute__ ((visibility ("hidden"))) void gio_textInputChanged(CFTypeRef viewRef, int text, int selection);
__attribute__ ((visibility ("hidden"))) void gio_requestSnippetLater(CFTypeRef viewRef);
*/
import "C"

import (
	"gioui.org/f32"
	"gioui.org/io/key"
)

const imeWholeText = 1 << 30

// imeState is the editor state, asking once per field (and after the app
// changes the text) for the whole text; the keyboard queries many times a
// keystroke, and each ask is an event.
//
// The ask is posted to the main queue rather than made here: UIKit calls
// these queries from inside its own work (even from inside Gio showing the
// keyboard), and asking is an event Gio must not handle re-entrantly.
func (w *window) imeState() editorState {
	if !w.imeAsked {
		w.imeAsked = true
		C.gio_requestSnippetLater(w.view)
	}
	return w.w.EditorState()
}

//export gio_imeRequestSnippet
func gio_imeRequestSnippet(h C.uintptr_t) {
	viewFor(h).w.SetEditorSnippet(key.Range{Start: 0, End: imeWholeText})
}

//export gio_imeText
func gio_imeText(h C.uintptr_t, selStart, selEnd, compStart, compEnd *C.int) C.CFTypeRef {
	w := viewFor(h)
	st := w.imeState()
	sel := normRange(st.Selection.Range)
	*selStart, *selEnd = C.int(st.UTF16Index(sel.Start)), C.int(st.UTF16Index(sel.End))
	*compStart, *compEnd = -1, -1
	if c := normRange(st.compose); c.Start != -1 {
		*compStart, *compEnd = C.int(st.UTF16Index(c.Start)), C.int(st.UTF16Index(c.End))
	}
	return stringToNSString(st.Snippet.Text)
}

func (w *window) imeRunes(start, end C.int) key.Range {
	st := w.imeState()
	return key.Range{Start: st.RunesIndex(int(start)), End: st.RunesIndex(int(end))}
}

//export gio_imeReplace
func gio_imeReplace(h C.uintptr_t, start, end C.int, text C.CFTypeRef) {
	w := viewFor(h)
	w.w.EditorReplace(w.imeRunes(start, end), nsstringToString(text))
}

//export gio_imeSetSelection
func gio_imeSetSelection(h C.uintptr_t, start, end C.int) {
	w := viewFor(h)
	w.w.SetEditorSelection(w.imeRunes(start, end))
}

//export gio_imeSetComposing
func gio_imeSetComposing(h C.uintptr_t, start, end C.int) {
	w := viewFor(h)
	if start < 0 {
		w.w.SetComposingRegion(key.Range{Start: -1, End: -1})
		return
	}
	w.w.SetComposingRegion(w.imeRunes(start, end))
}

// gio_imeCaret gives the caret's top and height in window pixels.
//
//export gio_imeCaret
func gio_imeCaret(h C.uintptr_t, x, y, height *C.float) {
	st := viewFor(h).w.EditorState()
	c := st.Selection.Caret
	tr := st.Selection.Transform
	top := tr.Transform(f32.Pt(c.Pos.X, c.Pos.Y-c.Ascent))
	bottom := tr.Transform(f32.Pt(c.Pos.X, c.Pos.Y+c.Descent))
	*x, *y, *height = C.float(top.X), C.float(top.Y), C.float(bottom.Y-top.Y)
}

// EditorStateChanged tells the keyboard about changes it didn't make
// itself (a new field, the app setting text, a tap moving the caret);
// its own edits already match the editor state, so they don't show here.
func (w *window) EditorStateChanged(old, new editorState) {
	text := old.Snippet != new.Snippet
	sel := old.Selection.Range != new.Selection.Range || old.compose != new.compose
	if !text && !sel {
		return
	}
	if text {
		w.imeAsked = false
	}
	b := func(v bool) C.int {
		if v {
			return 1
		}
		return 0
	}
	C.gio_textInputChanged(w.view, b(text), b(sel))
}
