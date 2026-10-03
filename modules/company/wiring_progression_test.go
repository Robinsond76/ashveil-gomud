package company

import (
	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"math"
	"os"
	"path/filepath"
	"testing"
)

type hpArchetypes struct{ fakeArchetypes }

func (hpArchetypes) HealthPerLevel(id string) (float64, bool) {
	rate, ok := map[string]float64{"warrior": 6, "cleric": 5, "ranger": 5, "rogue": 4, "wizard": 3}[id]
	return rate, ok
}
func (hpArchetypes) HealthArchetypes() map[string]float64 {
	return map[string]float64{"warrior": 6, "cleric": 5, "ranger": 5, "rogue": 4, "wizard": 3}
}

func TestProgressionCompanionSpawnAndSavedVitals(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("warrior")
	archetypes.SetProvider(hpArchetypes{fakeArchetypes{player: "warrior"}})
	turn, round := util.GetTurnCount(), util.GetRoundCount()
	b.respawn()
	cfg := configs.GetProgressionConfig()
	for id, rate := range map[int]float64{1: 6, 2: 5, 3: 6, 4: 5} {
		m := b.companion(id)
		assert.Equal(t, rate, m.Character.HealthGainPerLevel())
		assert.Equal(t, cfg.HealthAtLevel(m.Character.Level, m.Character.Stats.Vitality.ValueAdj, rate), m.Character.HealthMax.Value)
		m.Character.Level = 20
		require.True(t, module.RetrainCompanion(m.InstanceId))
		hp := m.Character.HealthMax.Value
		m.Character.Level = 21
		require.True(t, module.RetrainCompanion(m.InstanceId))
		assert.Equal(t, hp+1, m.Character.HealthMax.Value, "post-20 gain with no stat step")
		m.Character.Health = 3
		m.Character.Mana = 0
	}
	b.respawn()
	for id := 1; id <= 4; id++ {
		m := b.companion(id)
		assert.Equal(t, 21, m.Character.Level)
		assert.Equal(t, 3, m.Character.Health)
		assert.Zero(t, m.Character.Mana)
		assert.Equal(t, cfg.HealthAtLevel(21, m.Character.Stats.Vitality.ValueAdj, m.Character.HealthGainPerLevel()), m.Character.HealthMax.Value)
	}
	assert.Equal(t, turn, util.GetTurnCount())
	assert.Equal(t, round, util.GetRoundCount())
}

func TestProgressionEnemyHPTemplateRaceAndDefault(t *testing.T) {
	b := newBrawl(t)
	spec := mobs.GetMobSpec(9101)
	r := races.GetRace(spec.Character.RaceId)
	require.NotNil(t, r)
	old := r.HPPerLevel
	t.Cleanup(func() { r.HPPerLevel = old; spec.HPPerLevel = 0 })
	cfg := configs.GetProgressionConfig()
	for _, tc := range []struct{ template, race, want float64 }{{0, 0, float64(cfg.DefaultHPPerLevel)}, {0, 7, 7}, {8, 7, 8}} {
		r.HPPerLevel = tc.race
		path := filepath.Join(configs.GetFilePathsConfig().DataFiles.String(), "mobs", "brawl", "9101-bandit_cutthroat.yaml")
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		var authored mobs.Mob
		require.NoError(t, yaml.Unmarshal(data, &authored))
		authored.HPPerLevel = tc.template
		encoded, err := yaml.Marshal(authored)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, encoded, 0600))
		mobs.LoadDataFiles()
		m := mobs.NewMobByIdNoElite(9101, b.road.RoomId, 10)
		require.NotNil(t, m)
		assert.Equal(t, tc.want, m.Character.HealthGainPerLevel())
		assert.Equal(t, cfg.HealthAtLevel(10, m.Character.Stats.Vitality.ValueAdj, tc.want), m.Character.HealthMax.Value)
		assert.Equal(t, m.Character.HealthMax.Value, m.Character.Health)
		mobs.DestroyInstance(m.InstanceId)
	}
}

func TestProgressionPlayerFileLoadsClassHPAndKeepsTraining(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("wizard")
	archetypes.SetProvider(hpArchetypes{fakeArchetypes{player: "wizard"}})
	c := b.aria.Character
	c.Level, c.PeakLevel, c.Experience = 17, 20, 123456
	c.Stats.Vitality.Training = 11
	c.Stats.Speed.Training = 7
	c.StatPoints, c.TrainingPoints = 8, 9
	c.Health = 9999
	require.NoError(t, users.SaveUser(*b.aria))
	loaded, err := users.LoadUserFile(7)
	require.NoError(t, err)
	cfg := configs.GetProgressionConfig()
	assert.Equal(t, 17, loaded.Character.Level)
	assert.Equal(t, 20, loaded.Character.PeakLevel)
	assert.Equal(t, 123456, loaded.Character.Experience)
	assert.Equal(t, 11, loaded.Character.Stats.Vitality.Training)
	assert.Equal(t, 7, loaded.Character.Stats.Speed.Training)
	assert.Equal(t, 8, loaded.Character.StatPoints)
	assert.Equal(t, 9, loaded.Character.TrainingPoints)
	assert.Equal(t, cfg.HealthAtLevel(17, loaded.Character.Stats.Vitality.ValueAdj, 3), loaded.Character.HealthMax.Value)
	assert.Equal(t, loaded.Character.HealthMax.Value, loaded.Character.Health)
	c.Health = 2
	require.NoError(t, users.SaveUser(*b.aria))
	loaded, err = users.LoadUserFile(7)
	require.NoError(t, err)
	assert.Equal(t, 2, loaded.Character.Health)
}

func TestProgressionNewEnlistUsesConfiguredHP(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("warrior")
	archetypes.SetProvider(hpArchetypes{fakeArchetypes{player: "warrior"}})
	b.cmd("company", "dismiss all")
	_, err := module.summon(7, b.road.RoomId, "61")
	require.NoError(t, err)
	var m *mobs.Mob
	for _, instance := range module.instances[7] {
		m = mobs.GetInstance(instance)
	}
	require.NotNil(t, m)
	assert.Equal(t, "warrior", m.Character.HPArchetype)
	assert.Equal(t, 6.0, m.Character.HealthGainPerLevel())
}

// Calibration records the authored recruits at representative levels, using
// 33h1's old per-level training allocation for before and step allocation after.
func TestProgressionShippedCalibration(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("warrior")
	archetypes.SetProvider(hpArchetypes{fakeArchetypes{player: "warrior"}})
	b.respawn()
	cfg := configs.GetProgressionConfig()
	old := cfg
	old.StatStepLevels = 1
	oldStat := func(level, base, trained int) int {
		raw := old.RacialForLevel(level, base) + trained
		if raw >= int(old.StatCapThreshold) {
			return int(old.StatCapAnchor) + int(math.Round(math.Pow(float64(raw-int(old.StatCapAnchor)), float64(old.StatCapExponent))*float64(old.StatCapScale)))
		}
		return raw
	}
	t.Log("| recruit | level | HP before/after | Speed before/after | Vitality before/after |")
	for _, level := range []int{1, 5, 10, 20, 60} {
		for id := 1; id <= 4; id++ {
			m := b.companion(id)
			m.Character.Level = level
			require.True(t, module.RetrainCompanion(m.InstanceId))
			record, _ := module.registry.Get(7)
			var weights domain.GrowthWeights
			for _, c := range record.Companions {
				if c.ID == id {
					weights = growthWeightsOf(c)
				}
			}
			prior := domain.Deal(level, weights)
			base := trainingOf(&mobs.GetMobSpec(m.MobId).Character)
			oldSpeed := oldStat(level, m.Character.Stats.Speed.Base, base[1]+prior[1]+m.Character.Stats.Speed.Mods)
			oldVit := oldStat(level, m.Character.Stats.Vitality.Base, base[3]+prior[3]+m.Character.Stats.Vitality.Mods)
			oldHP := 5 + level + oldVit*4 + level/2 + m.Character.StatMod("healthmax")
			t.Logf("| %s | %d | %d/%d | %d/%d | %d/%d |", m.Character.Name, level, oldHP, m.Character.HealthMax.Value, oldSpeed, m.Character.Stats.Speed.ValueAdj, oldVit, m.Character.Stats.Vitality.ValueAdj)
		}
	}
	t.Log("| starter archetype | level | HP before/after |")
	for _, arch := range []string{"warrior", "cleric", "ranger", "rogue", "wizard"} {
		c := characters.New()
		c.RaceId = 1
		c.HPArchetype = arch
		c.Validate()
		t.Logf("| %s | 1 | 6/%d |", arch, c.HealthMax.Value)
	}
}

func (hpArchetypes) CompanionGrowth(id string) (map[string]int, bool) {
	weights := map[string]map[string]int{
		"warrior": {"strength": 4, "vitality": 3, "speed": 2, "perception": 1},
		"rogue":   {"speed": 4, "perception": 3, "strength": 2, "smarts": 1},
		"wizard":  {"mysticism": 4, "smarts": 3, "perception": 2, "speed": 1},
		"cleric":  {"mysticism": 3, "vitality": 3, "smarts": 2, "strength": 2},
		"ranger":  {"perception": 4, "speed": 3, "strength": 2, "vitality": 1},
	}
	w, ok := weights[id]
	return w, ok
}

func TestProgressionLiveCompanionChoiceRecalculatesHP(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypesFor("", nil)
	archetypes.SetProvider(hpArchetypes{})
	c := &b.companion(1).Character
	c.Level = 10
	require.True(t, module.RetrainCompanion(b.companion(1).InstanceId))
	c.Health = c.HealthMax.Value
	beforeHealth := c.Health
	assert.Contains(t, module.setArchetype(7, "1", "wizard"), "now a")
	assert.Equal(t, "wizard", c.HPArchetype)
	assert.Equal(t, 3.0, c.HealthGainPerLevel())
	assert.Equal(t, min(beforeHealth, c.HealthMax.Value), c.Health, "choice clamps without refilling")
}

func TestProgressionEquipmentClonePreservesHP(t *testing.T) {
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	b := equipmentBrawl(t)
	for _, class := range []string{"warrior", "wizard"} {
		archetypes.SetProvider(hpArchetypes{fakeArchetypes{player: class}})
		for _, companion := range []bool{false, true} {
			c := characters.New()
			c.Level = 10
			if companion {
				c.HPArchetype = class
			} else {
				c.SetUserId(b.aria.UserId)
			}
			require.NoError(t, c.Validate(true))
			clone, err := cloneCharacter(c)
			require.NoError(t, err)
			assert.Equal(t, c.HealthGainPerLevel(), clone.HealthGainPerLevel())
			assert.Equal(t, c.HealthMax.Value, clone.HealthMax.Value)
		}
	}
}

func TestProgressionEquipmentPreviewClassChangeAndApply(t *testing.T) {
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	b := equipmentBrawl(t)
	c := b.aria.Character
	c.Level = 10
	weapon := items.New(10004)
	c.Items = []items.Item{weapon}
	for _, class := range []string{"warrior", "wizard"} {
		archetypes.SetProvider(hpArchetypes{fakeArchetypes{player: class}})
		require.NoError(t, c.Validate(true))
		view := module.EquipmentViewFor(b.aria.UserId, "weapon")
		choice := editorChoice(t, editorSlot(t, view, "weapon"), weapon.ShorthandId())
		require.True(t, choice.Allowed, choice.Reason)
		require.NotNil(t, choice.After)
		assert.Equal(t, c.HealthMax.Value, choice.After.HealthMax, "class change invalidates cached previews")
	}
	view := module.EquipmentViewFor(b.aria.UserId, "weapon")
	choice := editorChoice(t, editorSlot(t, view, "weapon"), weapon.ShorthandId())
	require.Contains(t, applyEditorChoice(t, b, choice), "equipment updated")
	assert.Equal(t, c.HealthMax.Value, choice.After.HealthMax)
}
