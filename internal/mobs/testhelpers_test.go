package mobs

import "testing"

func TestResetHostility(t *testing.T) {
	t.Cleanup(ResetHostility)
	MakeHostile("bandits", 7, 50)
	if !IsHostile("bandits", 7) {
		t.Fatal("the grudge is held")
	}
	ResetHostility()
	if IsHostile("bandits", 7) {
		t.Fatal("the grudge is forgotten")
	}
}
