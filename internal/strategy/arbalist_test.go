package strategy

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Phase 39h: Piercing Bolt needs a shooting weapon and is ready on its own
// cooldown; the Arbalist's default aim is the most armored foe.
func TestDecidePiercingBolt(t *testing.T) {
	ready := func(Ability) bool { return true }
	known := []Ability{PiercingBolt}
	for _, tc := range []struct {
		name string
		s    AbilitySituation
		use  bool
	}{
		{"a crossbow", AbilitySituation{Known: known, Ready: ready, Weapon: Shooting}, true},
		{"a spear", AbilitySituation{Known: known, Ready: ready, Weapon: Melee}, false},
		{"unarmed", AbilitySituation{Known: known, Ready: ready, Weapon: Unarmed}, false},
		{"cooling", AbilitySituation{Known: known, Ready: func(Ability) bool { return false }, Weapon: Shooting}, false},
		{"abilities off", AbilitySituation{Known: known, Ready: ready, Weapon: Shooting, Off: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id, ok := DecideAbility(tc.s)
			assert.Equal(t, tc.use, ok)
			if tc.use {
				assert.Equal(t, PiercingBolt, id)
			}
		})
	}
}

func TestArbalistAbilityAndDefaults(t *testing.T) {
	assert.Equal(t, []Ability{PiercingBolt}, CompanionAbilities("arbalist"))
	assert.Equal(t, []Ability{PiercingBolt}, PlayerAbilities(func(s string) int {
		if s == "arbalestry" {
			return 1
		}
		return 0
	}))
	assert.Equal(t, Armored, DefaultRule("arbalist"))
	assert.Equal(t, Fighter, DefaultRole("arbalist"))
	assert.Equal(t, Armored, Default("Arbalist").Rule)
	assert.Contains(t, Rules, Armored)
	for _, alias := range []string{"armor", "armour", "tank", "tanks", "armored"} {
		r, ok := ParseRule(alias)
		assert.True(t, ok, alias)
		assert.Equal(t, Armored, r, alias)
	}
	assert.Contains(t, Armored.Describe(), "armored")
	// The ability text states the number the code uses.
	spec, ok := SpecOf(PiercingBolt)
	assert.True(t, ok)
	assert.Contains(t, spec.Does, "140%")
	assert.Equal(t, 140, BoltBlowPct)
	assert.True(t, strings.Contains(spec.Does, "winding the crossbow"))
}

func TestArmoredRulePicksTheMostArmoredReachableFoe(t *testing.T) {
	foes := []Foe{
		{ID: 1, HP: 50, MaxHP: 50, Row: 0, Col: 0, Reachable: true, Armor: 10},
		{ID: 2, HP: 40, MaxHP: 50, Row: 0, Col: 1, Reachable: true, Armor: 40},
		{ID: 3, HP: 20, MaxHP: 50, Row: 0, Col: 2, Reachable: true, Armor: 40},
		{ID: 4, HP: 50, MaxHP: 50, Row: 1, Col: 0, Reachable: false, Armor: 90},
	}
	got, ok := Pick(Armored, foes, 0, false)
	assert.True(t, ok)
	assert.Equal(t, 3, got, "the most armor among those it can reach; a tie goes to the weaker")
	foes[2].Armor = 0
	got, _ = Pick(Armored, foes, 0, false)
	assert.Equal(t, 2, got)
}
