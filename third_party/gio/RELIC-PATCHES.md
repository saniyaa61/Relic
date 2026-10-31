# Relic's copy of Gio

This is gioui.org v0.10.3 (tests and test data left out), used through a
`replace` in Relic's go.mod so Relic can carry small fixes. Each change is
marked "Relic patch" in the code. To move to a newer Gio, copy the new
release here and re-apply these:

1. `gesture/gesture.go`: `touchSlop` 3dp → 8dp. Scrolling took over after a
   3dp finger movement, so taps on buttons inside scrolling pages were often
   lost on phones (owner, 2026-10-31). Android and iOS use about 8dp.
