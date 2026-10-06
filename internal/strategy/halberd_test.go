package strategy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Phase 39a: the Halberdier's abilities, rule and numbers.

func TestHalberdierAbilityUnlocks(t *testing.T) {
	assert.Equal(t, []Ability{Sweep, Brace}, CompanionAbilities("Halberdier"))
	levels := map[string]int{"polearm": 1}
	assert.Equal(t, []Ability{Sweep, Brace}, PlayerAbilities(func(s string) int { return levels[s] }))
	spec, ok := SpecOf(Brace)
	assert.True(t, ok)
	assert.Equal(t, 3, spec.MinLevel, "Brace comes at level 3")
	spec, _ = SpecOf(Sweep)
	assert.Zero(t, spec.MinLevel)
}

func TestDecideSweepAndBrace(t *testing.T) {
	ready := func(Ability) bool { return true }
	rest := func(id Ability) bool { return id != Sweep }
	cases := []struct {
		name string
		s    AbilitySituation
		want Ability
	}{
		{"sweep a crowded row", AbilitySituation{Known: []Ability{Sweep, Brace}, Ready: ready, Weapon: Melee, SweepFoes: 2}, Sweep},
		{"no sweep of a lone foe", AbilitySituation{Known: []Ability{Sweep, Brace}, Ready: ready, Weapon: Melee, SweepFoes: 1}, ""},
		{"no sweep with a bow", AbilitySituation{Known: []Ability{Sweep}, Ready: ready, Weapon: Shooting, SweepFoes: 3}, ""},
		{"brace when the sweep rests and a foe strikes", AbilitySituation{Known: []Ability{Sweep, Brace}, Ready: rest, Weapon: Melee, SweepFoes: 3, Struck: true}, Brace},
		{"brace when the row is a lone foe", AbilitySituation{Known: []Ability{Sweep, Brace}, Ready: ready, Weapon: Melee, SweepFoes: 1, Struck: true}, Brace},
		{"no brace when nobody strikes", AbilitySituation{Known: []Ability{Brace}, Ready: ready, Weapon: Melee}, ""},
		{"sweep before brace", AbilitySituation{Known: []Ability{Sweep, Brace}, Ready: ready, Weapon: Melee, SweepFoes: 2, Struck: true}, Sweep},
		{"abilities off", AbilitySituation{Known: []Ability{Sweep}, Ready: ready, Weapon: Melee, SweepFoes: 2, Off: true}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := DecideAbility(c.s)
			assert.Equal(t, c.want != "", ok)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestHookAndSweepNumbers(t *testing.T) {
	assert.Zero(t, HookChance(5))
	assert.Equal(t, 20, HookChance(6))
	assert.Equal(t, 20, HookChance(19))
	assert.Equal(t, 40, HookChance(20))
	assert.False(t, SweepWide(7))
	assert.True(t, SweepWide(8))
}

func TestCrowdedRulePicksTheWeakestInTheFullestRow(t *testing.T) {
	foes := []Foe{
		{ID: 1, HP: 50, MaxHP: 50, Row: 0, Col: 0, Reachable: true},
		{ID: 2, HP: 20, MaxHP: 50, Row: 0, Col: 1, Reachable: true},
		{ID: 3, HP: 50, MaxHP: 50, Row: 0, Col: 2, Reachable: true},
		{ID: 4, HP: 5, MaxHP: 50, Row: 1, Col: 0, Reachable: true},
		{ID: 5, HP: 50, MaxHP: 50, Row: 1, Col: 1, Reachable: true},
	}
	got, ok := Pick(Crowded, foes, 0, false)
	assert.True(t, ok)
	assert.Equal(t, 2, got, "row 0 holds three; its weakest")
	// A row shrinks as its foes fall: two and two, the front row first.
	two := []Foe{foes[0], foes[3], foes[4]}
	got, _ = Pick(Crowded, two, 0, false)
	assert.Equal(t, 4, got, "row 1 now holds two, row 0 one")
	// Unreachable foes don't count.
	foes[0].Reachable, foes[1].Reachable = false, false
	got, _ = Pick(Crowded, foes, 0, false)
	assert.Equal(t, 4, got)
}

func TestCrowdedIsTheHalberdiersDefaultRule(t *testing.T) {
	assert.Equal(t, Crowded, Default("halberdier").Rule)
	assert.Equal(t, Fighter, Default("halberdier").Role)
	assert.Equal(t, Weakest, Default("warrior").Rule)
	r, ok := ParseRule("rows")
	assert.True(t, ok)
	assert.Equal(t, Crowded, r)
	assert.Contains(t, Rules, Crowded)
}
