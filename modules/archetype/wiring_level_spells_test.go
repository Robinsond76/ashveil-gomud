package archetype

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 35b: a wizard learns Shower of Sparks when a real level-up
// reaches level 3, is told once, sees it in the level-up report, and
// keeps it when a death costs the level and it is earned again.
func TestWiringLevelUpTeachesLevelSpells(t *testing.T) {
	m := registered(t)
	// GrantXP teaches through this module's grant; the module's own
	// LevelUp listener (registered by init) says so.
	t.Cleanup(users.RegisterLevelGrant(m.levelGrant))
	var report []string
	freshEvents(t)
	rid := events.RegisterListener(events.LevelUp{}, func(e events.Event) events.ListenerReturn {
		report = append(report, e.(events.LevelUp).PowerLines...)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.LevelUp{}, rid) })
	// GrantXP saves the user: into a scratch folder, not the shipped world.
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "users"), 0755))
	previous := configs.Flatten(configs.GetOverrides())
	flat := configs.Flatten(configs.GetOverrides())
	flat["FilePaths.DataFiles"] = dir
	require.NoError(t, configs.RestoreOverrides(flat))
	t.Cleanup(func() { require.NoError(t, configs.RestoreOverrides(previous)) })
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

	report = nil
	text = reach(3)
	assert.True(t, u.Character.HasSpell("sparks"))
	assert.Equal(t, 1, strings.Count(text, "You have learned a new spell"), text)
	assert.Contains(t, strings.Join(report, "\n"), "Shower of Sparks ", "the report names the new spell's size")

	// A death costs the level; earning it again teaches nothing new.
	u.Character.Level = 2
	u.Character.Experience = u.Character.XPTL(1)
	assert.True(t, u.Character.HasSpell("sparks"), "a lost level keeps the spell")
	text = reach(3)
	assert.True(t, u.Character.HasSpell("sparks"))
	assert.NotContains(t, text, "You have learned a new spell")
}
