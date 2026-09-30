package windup

import "testing"

func TestGetCrushingBlow(t *testing.T) {
	a, ok := Get("crushing-blow")
	if !ok {
		t.Fatal("crushing-blow is registered")
	}
	if a.Name != "Crushing Blow" || a.Rounds != 1 || a.Multiplier != 2 || !a.KnockDown {
		t.Errorf("crushing-blow = %+v, want Crushing Blow, 1 round, x2, knocks down", a)
	}
	for _, line := range []string{a.Telegraph, a.Release, a.Broken, a.Wasted, a.Lost} {
		if line == "" {
			t.Errorf("crushing-blow is missing a line: %+v", a)
		}
	}
	if _, ok := Get("no-such-blow"); ok {
		t.Error("an unknown ability is not found")
	}
}

func TestRollStart(t *testing.T) {
	never := func(int) int { t.Fatal("a certain outcome needs no roll"); return 0 }
	if RollStart(0, never) || RollStart(-5, never) {
		t.Error("a chance of 0 or less never starts")
	}
	if !RollStart(100, never) {
		t.Error("a chance of 100 always starts")
	}
	if !RollStart(35, func(int) int { return 34 }) {
		t.Error("a roll under the chance starts")
	}
	if RollStart(35, func(int) int { return 35 }) {
		t.Error("a roll at the chance doesn't")
	}
}

func TestRender(t *testing.T) {
	got := Render("{actor} lifts {his} {weapon} at {target}.", "The ogre", "his", "great club", "you")
	if got != "The ogre lifts his great club at you." {
		t.Errorf("Render = %q", got)
	}
}
