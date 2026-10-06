package strategy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Phase 39e: a Beast Tamer's abilities are listed by archetype and skill, and
// each comes at its level.
func TestBeastTamerAbilityUnlocks(t *testing.T) {
	all := []Ability{Sic, Rally, PackSense}
	assert.Equal(t, all, CompanionAbilities("BeastTamer"))
	levels := map[string]int{"taming": 1}
	assert.Equal(t, all, PlayerAbilities(func(s string) int { return levels[s] }))
	assert.Equal(t, []Ability{Sic}, AtLevel(all, 2))
	assert.Equal(t, []Ability{Sic, Rally}, AtLevel(all, 3))
	assert.Equal(t, all, AtLevel(all, 8))
}

// The Tamer's abilities are resolved by the combat round, not chosen by the
// generic decision.
func TestBeastAbilitiesAreNotChosenByDecideAbility(t *testing.T) {
	ready := func(Ability) bool { return true }
	_, ok := DecideAbility(AbilitySituation{Known: []Ability{Sic, Rally, PackSense}, Ready: ready, Weapon: Melee, SweepFoes: 3, Struck: true})
	assert.False(t, ok)
}
