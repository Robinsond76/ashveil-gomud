package classes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 38e: creature species have base ranks and no promotion or talents.
func TestCreatureSpeciesHaveRanksButNoRoutesOrTalents(t *testing.T) {
	assert.Equal(t, []string{"Run down"}, rankNames(BaseRanksReached("hound", 1)))
	assert.Equal(t, []string{"Run down", "Worry", "Fleet"}, rankNames(BaseRanksReached("hound", 10)))
	assert.Equal(t, []string{"Stone body", "Anchor"}, rankNames(BaseRanksReached("stone-golem", 1)))
	assert.Equal(t, []string{"Stone body", "Anchor", "Granite", "Bedrock"}, rankNames(BaseRanksReached("stone-golem", 20)))
	for _, species := range []string{"hound", "stone-golem"} {
		assert.Empty(t, Advanced(species), species)
		assert.Empty(t, TalentsFor(species), species)
		for level := 1; level <= 30; level++ {
			next := MilestoneFor(species, "", level)
			assert.NotContains(t, next, "promot", "%s at %d", species, level)
			assert.NotContains(t, next, "talent", "%s at %d", species, level)
		}
	}
}

func TestCreatureRanksCarryTheirEffects(t *testing.T) {
	hound := EffectsForLineage("hound", "", 20, nil)
	golem := EffectsForLineage("stone-golem", "", 20, nil)
	assert.Equal(t, 35, hound.Int(Pounce), "Savage pursuit replaces Run down's 20")
	assert.Equal(t, 15, golem.Int(Slow))
	assert.Equal(t, 25, golem.Int(SpellWeak))
	assert.Equal(t, 40, golem.Int(Armor), "Granite replaces Stone body's 30")
	assert.Equal(t, 15, golem.Int(AuraResolv), "Bedrock replaces Anchor's 10")
	require.True(t, golem.Int(AuraResolv) > 0)
	assert.Equal(t, 20, EffectsForLineage("hound", "", 1, nil).Int(Pounce))
	// 38e review: the golem's fist grows with Granite and Bedrock, so it
	// keeps pace with a warrior's blade late on.
	assert.Equal(t, 0, EffectsForLineage("stone-golem", "", 9, nil).Int(Damage))
	assert.Equal(t, 2, EffectsForLineage("stone-golem", "", 10, nil).Int(Damage))
	assert.Equal(t, 4, golem.Int(Damage))
}
