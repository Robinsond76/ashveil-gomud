package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/wounds"
	"gopkg.in/yaml.v3"
)

// Phase 30b: a companion's wounds are cloned deep and survive the store's
// YAML round trip.
func TestMemberStateWoundsCloneAndRoundTrip(t *testing.T) {
	s := MemberState{Level: 3, Wounds: []wounds.Wound{{Kind: wounds.Cut, Place: "arm", Points: 2}}}
	c := s.Clone()
	c.Wounds[0].Points = 9
	if s.Wounds[0].Points != 2 {
		t.Fatal("Clone shares the wounds")
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var back MemberState
	if err := yaml.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if len(back.Wounds) != 1 || back.Wounds[0] != s.Wounds[0] {
		t.Fatalf("round trip: %+v", back.Wounds)
	}
}
