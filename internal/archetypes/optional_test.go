package archetypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type optionalProvider struct {
	fakeProvider
	list []OptionalSkill
}

func (o optionalProvider) OptionalSkills() []OptionalSkill { return o.list }

func TestNewOptionalSkillsValidates(t *testing.T) {
	list, errs := NewOptionalSkills([]OptionalSkill{
		{Skill: " Cooking ", Archetypes: []string{"*"}, MaxRank: 4},
		{Skill: "scribe", Archetypes: []string{"Wizard", "cleric", "wizard"}, MaxRank: 4},
		{Skill: "cooking", Archetypes: []string{"*"}, MaxRank: 2}, // duplicate
		{Skill: "alchemy", MaxRank: 2},                            // nobody
		{Skill: "brewing", Archetypes: []string{"*"}, MaxRank: 5}, // too high
		{Skill: "Bad Id", Archetypes: []string{"*"}, MaxRank: 1},
		{Skill: "tanning", Archetypes: []string{"Not An Id"}, MaxRank: 1},
	})
	assert.Len(t, errs, 5)
	require.Len(t, list, 2)
	assert.Equal(t, "cooking", list[0].Skill, "sorted by skill")
	assert.Equal(t, []string{"wizard", "cleric"}, list[1].Archetypes, "normalized and deduplicated")
}

func TestOptionalSkillEligibility(t *testing.T) {
	SetProvider(optionalProvider{list: []OptionalSkill{
		{Skill: "cooking", Archetypes: []string{AnyArchetype}, MaxRank: 4},
		{Skill: "scribe", Archetypes: []string{"wizard", "cleric"}, MaxRank: 4},
	}})
	t.Cleanup(func() { SetProvider(nil) })

	assert.True(t, CanLearn("warrior", "cooking"))
	assert.True(t, CanLearn("", "cooking"), "anyone, even with no archetype yet")
	assert.True(t, CanLearn("Wizard", "SCRIBE"))
	assert.False(t, CanLearn("warrior", "scribe"))
	assert.False(t, CanLearn("", "scribe"))
	assert.False(t, CanLearn("warrior", "brawling"), "a class skill is never trained")
	assert.Equal(t, []string{"cooking", "scribe"}, LearnableSkills("cleric"))
	assert.Equal(t, []string{"cooking"}, LearnableSkills("ranger"))
	o, ok := Optional("scribe")
	require.True(t, ok)
	assert.Equal(t, 4, o.MaxRank)

	SetProvider(nil)
	assert.Nil(t, OptionalSkills())
	assert.False(t, CanLearn("warrior", "cooking"), "nothing is trainable without a provider")
}
