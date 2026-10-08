package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestWeaponNounIsTheKindOfWeapon(t *testing.T) {
	for in, want := range map[string]string{
		"acolyte's mace":                          "mace",
		"guardsman's broadsword":                  "broadsword",
		"crude cudgel":                            "cudgel",
		"sword of flame":                          "sword",
		`<ansi fg="item">iron-banded club</ansi>`: "club",
		"staff": "staff",
		`iron mace <ansi fg="black-bold">(glowing)</ansi>`: "mace",
		`Dawnfang, a keen iron shortsword (rare)`:          "shortsword",
	} {
		if got := WeaponNoun(in); got != want {
			t.Errorf("WeaponNoun(%q) = %q, want %q", in, got, want)
		}
	}
}

// Phase 88 review: a rolled weapon's display name carries its quality, its
// own name and a "(rare)" label; combat lines name only the kind of weapon,
// and a relic keeps its name.
func TestWeaponNounOfAnItem(t *testing.T) {
	loadTestData(t)

	mace := items.New(10023) // acolyte's mace
	if got := WeaponNounOf(&mace); got != "mace" {
		t.Errorf("acolyte's mace: got %q, want mace", got)
	}

	rolled := items.New(10023)
	rolled.Loot = items.Rolled{Version: items.RollVersion, Quality: items.QualityStandard, Rarity: items.RarityRare, Tier: 2, Identified: true, Name: "Dawnfang"}
	if got := WeaponNounOf(&rolled); got != "mace" {
		t.Errorf("rolled %q: got %q, want mace", rolled.DisplayName(), got)
	}

	reaper := items.New(50001) // Ashen Reaper, a relic
	if got := WeaponNounOf(&reaper); got != "Ashen Reaper" {
		t.Errorf("relic: got %q, want Ashen Reaper", got)
	}

	var none items.Item
	if got := WeaponNounOf(&none); got != "" {
		t.Errorf("no weapon: got %q", got)
	}
}
