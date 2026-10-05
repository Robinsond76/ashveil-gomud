package hooks

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestGrantXPLevelReport(t *testing.T) {
	root, err := filepath.Abs("../..")
	require.NoError(t, err)

	for _, target := range []int{2, 4} {
		t.Run(fmt.Sprint(target), func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "templates/character"), 0755))
			tpl, err := os.ReadFile(filepath.Join(root, "_datafiles/world/default/templates/character/levelup.template"))
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "templates/character/levelup.template"), tpl, 0600))
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "users"), 0755))
			old := configs.Flatten(configs.GetOverrides())
			flat := configs.Flatten(configs.GetOverrides())
			flat["FilePaths.DataFiles"] = dir
			require.NoError(t, configs.RestoreOverrides(flat))
			t.Cleanup(func() { require.NoError(t, configs.RestoreOverrides(old)) })
			g := configs.GetGamePlayConfig()
			g.XPScale = 100
			g.Progression.StatPointsEveryNLevels = 2
			g.Progression.StatPointsPerLevel = 1
			g.Progression.TrainingPointsPerLevel = 1
			g.Progression.TrainingPointsEveryNLevels = 1
			g.Progression.SmoothStatGrowth = true
			t.Cleanup(configs.SetTestGamePlayConfig(g))
			u := users.NewUserRecord(35001, 0)
			u.Character.Level = 1
			u.Character.Stats.Strength.Base = 10
			u.Character.Validate()
			users.SetTestUser(u)
			t.Cleanup(func() { users.RemoveTestUser(u.UserId) })
			before := *u.Character
			round := util.GetRoundCount()
			var ev events.LevelUp
			var messages string
			id := events.RegisterListener(events.LevelUp{}, func(e events.Event) events.ListenerReturn { ev = e.(events.LevelUp); return SendLevelNotifications(e) })
			mid := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
				m := e.(events.Message)
				if m.UserId == u.UserId {
					messages += m.Text
				}
				return events.Continue
			})
			t.Cleanup(func() {
				events.UnregisterListener(events.LevelUp{}, id)
				events.UnregisterListener(events.Message{}, mid)
			})
			u.GrantXP(u.Character.XPTL(target-1), "test")
			events.ProcessEvents()
			require.Equal(t, target, u.Character.Level)
			assert.Equal(t, target-1, ev.LevelsGained)
			assert.Equal(t, target-1, ev.TrainingPoints)
			assert.Equal(t, target/2, ev.StatPoints)
			assert.Equal(t, before.HealthMax.Value, ev.HealthMaxBefore)
			assert.Equal(t, u.Character.HealthMax.Value, ev.HealthMaxAfter)
			assert.Equal(t, before.ManaMax.Value, ev.ManaMaxBefore)
			assert.Equal(t, u.Character.ManaMax.Value, ev.ManaMaxAfter)
			assert.Equal(t, before.Stats, ev.StatsBefore)
			assert.Equal(t, u.Character.Stats, ev.StatsAfter)
			require.NotEqual(t, ev.StatsBefore.Strength.ValueAdj, ev.StatsAfter.Strength.ValueAdj)
			assert.Contains(t, messages, fmt.Sprintf("Strength %d -> %d", ev.StatsBefore.Strength.ValueAdj, ev.StatsAfter.Strength.ValueAdj))
			assert.Contains(t, messages, fmt.Sprintf("You reach level %d", target))
			assert.Contains(t, messages, fmt.Sprintf("Health %d -> %d", ev.HealthMaxBefore, ev.HealthMaxAfter))
			assert.Contains(t, messages, fmt.Sprintf("Mana %d -> %d", ev.ManaMaxBefore, ev.ManaMaxAfter))
			assert.Contains(t, messages, "(coming)")
			assert.Contains(t, messages, "stat train")
			assert.NotContains(t, messages, "stat step")
			assert.Equal(t, round, util.GetRoundCount())
		})
	}
}
