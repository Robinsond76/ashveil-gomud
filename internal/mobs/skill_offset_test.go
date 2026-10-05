package mobs

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMobSkillOffsets (Phase 35a2): a template's attackskill and evasion
// reach every spawn's Attack and Evasion, and stay within ±5.
func TestMobSkillOffsets(t *testing.T) {
	template := &Mob{MobId: 990035, Character: *characters.New(), AttackSkill: 3, EvasionSkill: -2}
	require.NoError(t, template.Validate())
	withTemplate(t, template)
	m := NewMobById(990035, 1, 20)
	require.NotNil(t, m)
	assert.Equal(t, 20, m.Character.Level)
	assert.Equal(t, m.Character.Level+3, m.Character.AttackSkill(), "a trained captain")
	assert.Equal(t, m.Character.Level-2, m.Character.Evasion(), "a lumbering brute")

	for _, bad := range [][2]int{{6, 0}, {0, -6}} {
		mob := &Mob{Character: *characters.New(), AttackSkill: bad[0], EvasionSkill: bad[1]}
		assert.Error(t, mob.Validate(), "%v is past ±5", bad)
	}
}
