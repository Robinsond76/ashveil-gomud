package hooks

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/classes"
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
			// Phase 35a2: the report shows the skill the levels brought.
			assert.Equal(t, [2]int{1, target}, [2]int{ev.AttackBefore, ev.AttackAfter})
			assert.Equal(t, [2]int{1, target}, [2]int{ev.EvasionBefore, ev.EvasionAfter})
			assert.Contains(t, messages, fmt.Sprintf("Attack 1 -> %d   Evasion 1 -> %d", target, target))
			// 38c1 shipped the level 5 talent, so only the level 2 report
			// still names an unshipped milestone next.
			assert.Contains(t, messages, " Next: ")
			if target == 2 {
				assert.Contains(t, messages, "(coming)")
			} else {
				assert.NotContains(t, messages, "level 5: talent (coming)")
			}
			// Phase 35b review: the report renders each scaling spell's growth.
			messages = ""
			ev.PowerLines = []string{"Magic Missile 8-13 -> 9-14", "Shower of Sparks 5-8"}
			SendLevelNotifications(ev)
			events.ProcessEvents()
			assert.Contains(t, messages, " Magic Missile 8-13 -> 9-14\n Shower of Sparks 5-8")
			// Phase 38c1: the class notes render in the report.
			messages = ""
			ev.ClassNotes = []string{"Elite promotion ready: Knight -> Paladin. Visit a camp or town and type class promote paladin."}
			SendLevelNotifications(ev)
			events.ProcessEvents()
			assert.Contains(t, messages, " Elite promotion ready: Knight -> Paladin. Visit a camp or town and type class promote paladin.\n")
			assert.Contains(t, messages, "stat train")
			assert.NotContains(t, messages, "stat step")
			assert.Equal(t, round, util.GetRoundCount())
		})
	}
}

// A rhythm changed while an unmigrated player is online is paid before the
// level-up: the report counts only the new levels' points, and a later load
// pays nothing twice.
func TestGrantXPPaysStatPointCatchUpBeforeLevelReport(t *testing.T) {
	dir := t.TempDir()
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
	t.Cleanup(configs.SetTestGamePlayConfig(g))
	u := users.NewUserRecord(35002, 0)
	u.Character.Level = 10
	u.Character.PeakLevel = 10
	u.Character.Experience = u.Character.XPTL(9)
	u.Character.StatPointRhythm = 0
	u.Character.StatPoints = 2
	u.Character.Validate()
	users.SetTestUser(u)
	t.Cleanup(func() { users.RemoveTestUser(u.UserId) })
	var ev events.LevelUp
	id := events.RegisterListener(events.LevelUp{}, func(e events.Event) events.ListenerReturn { ev = e.(events.LevelUp); return events.Continue })
	t.Cleanup(func() { events.UnregisterListener(events.LevelUp{}, id) })

	u.GrantXP(u.Character.XPTL(11)-u.Character.Experience, "test")
	events.ProcessEvents()
	require.Equal(t, 12, u.Character.Level)
	assert.Equal(t, 1, ev.StatPoints, "levels 11 and 12 pay one point")
	assert.Equal(t, 2+3+1, u.Character.StatPoints, "catch-up (5 - 2) at peak 10, then level 12")
	assert.Equal(t, 2, u.Character.StatPointRhythm)
	assert.False(t, u.Character.CatchUpStatPoints(), "never paid twice")
}

type fakeLineage struct{}

func (fakeLineage) CanTrain(int, string) (bool, string)      { return true, "" }
func (fakeLineage) CanLearnSpell(int, string) (bool, string) { return true, "" }
func (fakeLineage) Exists(string) bool                       { return true }
func (fakeLineage) ArchetypeName(id string) (string, bool)   { return id, true }
func (fakeLineage) PlayerArchetype(int) (string, bool)       { return "warrior", true }

type fakeClassState struct{ state classes.State }

func (f fakeClassState) PlayerClass(int) classes.State { return f.state }

// Phase 38c1: a player's real level-up at 30 reports an elite promotion that
// is ready, or waiting on its gate, and each rank a level earns.
func TestGrantXPLevelReportNamesElitePromotionAndRanks(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "users"), 0755))
	old := configs.Flatten(configs.GetOverrides())
	flat := configs.Flatten(configs.GetOverrides())
	flat["FilePaths.DataFiles"] = dir
	require.NoError(t, configs.RestoreOverrides(flat))
	t.Cleanup(func() { require.NoError(t, configs.RestoreOverrides(old)) })
	g := configs.GetGamePlayConfig()
	g.XPScale = 100
	t.Cleanup(configs.SetTestGamePlayConfig(g))
	archetypes.SetProvider(fakeLineage{})
	t.Cleanup(func() { archetypes.SetProvider(nil) })

	for _, tc := range []struct {
		name      string
		class     string
		from      int
		alignment int8
		want      string
	}{
		{"ready", "knight", 29, 41, "Elite promotion ready: Knight -> Paladin. Visit a camp or town and type class promote paladin."},
		{"waiting", "knight", 29, 22, "Paladin needs alignment +30 (yours: +22). You keep your Knight ranks and can promote once it rises."},
		{"warlord rank", "warlord", 34, 0, "Rank 35 Warlord: Battle Cry."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			classes.SetProvider(fakeClassState{classes.State{Class: tc.class}})
			t.Cleanup(func() { classes.SetProvider(nil) })
			u := users.NewUserRecord(35003, 0)
			u.Character.Level = tc.from
			u.Character.PeakLevel = tc.from
			u.Character.Experience = u.Character.XPTL(tc.from - 1)
			u.Character.Alignment = tc.alignment
			u.Character.SetUserId(u.UserId)
			u.Character.Validate()
			users.SetTestUser(u)
			t.Cleanup(func() { users.RemoveTestUser(u.UserId) })
			var ev events.LevelUp
			id := events.RegisterListener(events.LevelUp{}, func(e events.Event) events.ListenerReturn { ev = e.(events.LevelUp); return events.Continue })
			t.Cleanup(func() { events.UnregisterListener(events.LevelUp{}, id) })

			u.GrantXP(u.Character.XPTL(tc.from)-u.Character.Experience, "test")
			events.ProcessEvents()
			require.Equal(t, tc.from+1, u.Character.Level)
			require.NotEmpty(t, ev.ClassNotes)
			assert.Contains(t, ev.ClassNotes[len(ev.ClassNotes)-1], tc.want)
		})
	}
}
