package archetype

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 35b: a wizard learns Shower of Sparks when a real level-up
// reaches level 3, is told once, and keeps it when a death costs the
// level and it is earned again.
func TestWiringLevelUpTeachesLevelSpells(t *testing.T) {
	m := registered(t)
	lid := events.RegisterListener(events.LevelUp{}, m.onLevelUp)
	t.Cleanup(func() { events.UnregisterListener(events.LevelUp{}, lid) })
	u := trainee(t, 3501, 0)
	m.choose(u, "wizard", true)
	require.False(t, u.Character.HasSpell("sparks"), "level 1 has no level spells")

	reach := func(level int) string {
		return captureText(t, func() {
			for u.Character.Level < level {
				u.GrantXP(u.Character.XPTL(u.Character.Level)-u.Character.Experience+1, "test")
			}
		})
	}
	text := reach(2)
	assert.False(t, u.Character.HasSpell("sparks"))
	assert.NotContains(t, text, "You have learned a new spell")

	text = reach(3)
	assert.True(t, u.Character.HasSpell("sparks"))
	assert.Equal(t, 1, strings.Count(text, "You have learned a new spell"), text)

	// A death costs the level; earning it again teaches nothing new.
	u.Character.Level = 2
	u.Character.Experience = u.Character.XPTL(1)
	assert.True(t, u.Character.HasSpell("sparks"), "a lost level keeps the spell")
	text = reach(3)
	assert.True(t, u.Character.HasSpell("sparks"))
	assert.NotContains(t, text, "You have learned a new spell")
}
