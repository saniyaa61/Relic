package ui

import (
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
)

func TestHSVRoundTrip(t *testing.T) {
	for _, hex := range []string{"#7A5C3A", "#C4956A", "#000000", "#FFFFFF", "#FF0000", "#00FF00", "#0000FF", "#3A5A8A", "#E090B0"} {
		c, _ := ParseHex(hex)
		if got := hexOf(toHSV(c).rgb()); got != hex {
			t.Errorf("%s → %s", hex, got)
		}
	}
	if h := toHSV(rgb(0x00FF00)); h.h != 120 || h.s != 1 || h.v != 1 {
		t.Errorf("green = %+v", h)
	}
}

// Typing a code moves the picker; "Use colour" hands the colour back.
func TestColourDialog(t *testing.T) {
	a, _ := testApp(t, nil)
	var got string
	openColourDialog(a, "Accent color", "#C4956A", func(v string) { got = v })
	d := a.dialog
	frame(a)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops}
	gtx.Constraints.Max.X = 600
	p := newColourPicker("#C4956A")
	p.hex.Editor.SetText("#3a5a8a")
	p.Layout(gtx, a)
	if p.value() != "#3A5A8A" {
		t.Errorf("typed code: picker at %s", p.value())
	}
	p.hex.Editor.SetText("#3a5") // half typed: stays put
	p.Layout(gtx, a)
	if p.value() != "#3A5A8A" {
		t.Errorf("half a code moved the picker to %s", p.value())
	}
	d.OnConfirm(a, "")
	if got != "#C4956A" {
		t.Errorf("unchanged dialog gave %q", got)
	}
}
