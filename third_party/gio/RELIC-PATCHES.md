# Relic's copy of Gio

This is gioui.org v0.10.3 (tests and test data left out), used through a
`replace` in Relic's go.mod so Relic can carry small fixes. Each change is
marked "Relic patch" in the code. To move to a newer Gio, copy the new
release here and re-apply these:

1. `gesture/gesture.go`: `touchSlop` 3dp → 8dp. Scrolling took over after a
   3dp finger movement, so taps on buttons inside scrolling pages were often
   lost on phones (owner, 2026-10-31). Android and iOS use about 8dp.
2. iOS keyboard (`app/os_ios.m`, `app/os_ios.go`, new `app/ime_ios.go`):
   `GioView` adopts `UITextInput` instead of only `UIKeyInput`, with
   `GioTextPosition` / `GioTextRange` (UTF-16 offsets), marked text, and
   keyboard traits (sentence caps, autocorrect, spell check), so iOS
   offers autocorrect, predictive text and dictation (owner, 2026-10-31).
   The Go side reads and edits the editor state like the Android IME code
   does, asking for the whole text once per field; `EditorStateChanged`
   tells the keyboard's `inputDelegate` about changes it didn't make.
3. iOS keyboard, round 2 (owner, 2026-11-01): after choosing a suggestion
   the caret now goes after the new word (`replaceRange:withText:` sets it;
   Gio left it at the old offset, inside a longer word). The app froze when
   a sheet with a focused field closed: Gio shows and hides the keyboard in
   the middle of its own event handling, UIKit calls the UITextInput methods
   straight back, and they called into Gio re-entrantly. `showTextInput` /
   `hideTextInput` now set `gio_textInputBusy`, the edit methods wait for
   the main queue while it's set (`gioLater`), and the whole-text request
   always goes through the main queue.
