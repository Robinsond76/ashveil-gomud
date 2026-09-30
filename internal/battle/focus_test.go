package battle

import (
	"errors"
	"testing"
)

func TestFocusOverride(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	if err := SetFocus(1, "leader"); !errors.Is(err, ErrNoBattle) {
		t.Fatalf("no battle: %v", err)
	}
	Begin(1, 10, 5, "p", []int{100})
	if _, set := Focus(1); set {
		t.Error("a new battle has no override")
	}
	if !FocusReady(1) {
		t.Error("a new battle is ready for an order")
	}
	if err := SetFocus(1, "leader"); err != nil {
		t.Fatal(err)
	}
	if rule, set := Focus(1); !set || rule != "leader" {
		t.Errorf("override: %q %v", rule, set)
	}
	if FocusReady(1) {
		t.Error("pending until the next upkeep")
	}
	if err := SetFocus(1, "weakest"); !errors.Is(err, ErrFocusPending) {
		t.Errorf("a second order before the upkeep: %v", err)
	}
	if got := TakeRefocus(); len(got) != 1 || got[0] != 1 {
		t.Errorf("TakeRefocus: %v", got)
	}
	if got := TakeRefocus(); len(got) != 0 {
		t.Errorf("taken once: %v", got)
	}
	if !FocusReady(1) {
		t.Error("ready again after the upkeep")
	}
	// Back to the saved focus.
	if err := ClearFocus(1); err != nil {
		t.Fatal(err)
	}
	if _, set := Focus(1); set {
		t.Error("cleared")
	}
	if got := TakeRefocus(); len(got) != 1 {
		t.Errorf("a clear turns the company too: %v", got)
	}
	// A clone never shares the override.
	b, _ := Current(1)
	b.FocusSet = true
	if _, set := Focus(1); set {
		t.Error("Current is a copy")
	}
	// The next battle starts from the saved focus.
	SetFocus(1, "strongest")
	End(1)
	Begin(1, 10, 6, "q", []int{101})
	if _, set := Focus(1); set {
		t.Error("the override ends with its battle")
	}
	if got := TakeRefocus(); len(got) != 0 {
		t.Errorf("an ended battle's pending order is gone: %v", got)
	}
}
