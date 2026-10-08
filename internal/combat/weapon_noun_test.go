package combat

import "testing"

func TestWeaponNounIsTheKindOfWeapon(t *testing.T) {
	for in, want := range map[string]string{
		"acolyte's mace":                          "mace",
		"guardsman's broadsword":                  "broadsword",
		"crude cudgel":                            "cudgel",
		"sword of flame":                          "sword",
		`<ansi fg="item">iron-banded club</ansi>`: "club",
		"staff": "staff",
	} {
		if got := WeaponNoun(in); got != want {
			t.Errorf("WeaponNoun(%q) = %q, want %q", in, got, want)
		}
	}
}
