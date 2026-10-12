package strategy

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/stretchr/testify/assert"
)

// The skill schedule (2026-10-12): a lineage's automatic abilities come at
// the level its base rank of the same name does, so the level-up line, the
// "Next:" pointer and the battle agree.
func TestAbilityLevelsMatchBaseRanks(t *testing.T) {
	checked := 0
	for _, spec := range Abilities {
		if spec.Archetype == "" {
			continue
		}
		for _, r := range classes.BaseRanks(spec.Archetype) {
			if r.Name != spec.Name {
				continue
			}
			checked++
			assert.Equal(t, r.Level, max(1, spec.MinLevel), "%s: ability level and base rank level", spec.Name)
		}
	}
	assert.GreaterOrEqual(t, checked, 11, "Tackle, Opening Strike, Aimed Shot, Sweep, Brace, Dive, Piercing Bolt, Guard String, Tangle, Emergency Splice, Sic, Rally all have base ranks")
}

// A level-1 character knows no automatic ability but the Doll Master's
// Puppet Strike, which is its plain attack: Tackle and the other
// signatures come at level 2 (owner, 2026-10-12).
func TestLevelOneKnowsNoSignatureAbility(t *testing.T) {
	for _, lineage := range []string{"warrior", "rogue", "ranger", "halberdier", "gryphon-rider", "arbalist", "beasttamer"} {
		assert.Empty(t, AtLevel(CompanionAbilities(lineage), 1), lineage)
		assert.NotEmpty(t, AtLevel(CompanionAbilities(lineage), 2), lineage)
	}
	assert.Equal(t, []Ability{PuppetStrike}, AtLevel(CompanionAbilities("dollmaster"), 1))
	assert.Equal(t, []Ability{Sweep}, AtLevel(CompanionAbilities("halberdier"), 3))
	assert.Equal(t, []Ability{Sweep, Brace}, AtLevel(CompanionAbilities("halberdier"), 4))
	all := func(string) int { return 1 }
	assert.NotContains(t, AtLevel(PlayerAbilities(all), 1), Tackle)
	assert.Contains(t, AtLevel(PlayerAbilities(all), 2), Tackle)
}
