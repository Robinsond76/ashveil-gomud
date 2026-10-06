package strategy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Phase 39d: a Doll Master's abilities are listed by archetype and skill,
// and each comes at its level.
func TestDollMasterAbilityUnlocks(t *testing.T) {
	all := []Ability{PuppetStrike, GuardString, Tangle, Splice}
	assert.Equal(t, all, CompanionAbilities("DollMaster"))
	levels := map[string]int{"puppetry": 1}
	assert.Equal(t, all, PlayerAbilities(func(s string) int { return levels[s] }))
	assert.Equal(t, []Ability{PuppetStrike}, AtLevel(all, 4))
	assert.Equal(t, []Ability{PuppetStrike, GuardString}, AtLevel(all, 5))
	assert.Equal(t, []Ability{PuppetStrike, GuardString, Tangle}, AtLevel(all, 12))
	assert.Equal(t, all, AtLevel(all, 18))
}

// The doll's abilities are resolved by the combat round, not chosen by the
// generic decision.
func TestDollAbilitiesAreNotChosenByDecideAbility(t *testing.T) {
	ready := func(Ability) bool { return true }
	_, ok := DecideAbility(AbilitySituation{Known: []Ability{PuppetStrike, GuardString, Tangle, Splice}, Ready: ready, Weapon: Melee, SweepFoes: 3, Struck: true})
	assert.False(t, ok)
}
