package archetype

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math"
	"testing"
)

func TestProgressionShippedHPAndCreationChoice(t *testing.T) {
	m, _ := testModule(t)
	archetypes.SetProvider(m)
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	// Phase 35a2: small growth from a higher floor.
	want := map[string]float64{"warrior": 1, "cleric": 0.6, "ranger": 0.8, "rogue": 0.7, "wizard": 0.5}
	assert.Equal(t, want, m.HealthArchetypes())
	for _, choice := range m.CreationChoices() {
		assert.Contains(t, choice.Description, "HP:")
		assert.Contains(t, choice.Description, "per level through level")
	}
	// 30g6: later gains keep each archetype's proportion of the middle rate.
	cfg0 := configs.GetProgressionConfig()
	assert.Greater(t, cfg0.HealthAfterFull(1), cfg0.HealthAfterFull(0.5), "a warrior out-gains a wizard after the full levels")
	assert.InDelta(t, float64(cfg0.HPAfterFull), cfg0.HealthAfterFull(float64(cfg0.DefaultHPPerLevel)), 1e-9)
	u := newUser(11)
	users.SetTestUser(u)
	t.Cleanup(func() { users.RemoveTestUser(11) })
	u.Character.Level = 10
	u.Character.Stats.Vitality.Training = 9
	u.Character.Health = 2
	m.choose(u, "wizard", true)
	cfg := configs.GetProgressionConfig()
	assert.Equal(t, 0.5, u.Character.HealthGainPerLevel())
	assert.Equal(t, cfg.HealthAtLevel(10, u.Character.Stats.Vitality.ValueAdj, 0.5, 0), u.Character.HealthMax.Value)
	assert.Equal(t, 2, u.Character.Health)
	u.Character.HealthMax.Value = 9999
	m.onPlayerSpawn(events.PlayerSpawn{UserId: 11})
	assert.Equal(t, cfg.HealthAtLevel(10, u.Character.Stats.Vitality.ValueAdj, 0.5, 0), u.Character.HealthMax.Value)
	hp, ok := m.HealthPerLevel("warrior")
	require.True(t, ok)
	assert.Equal(t, 1.0, hp)
	_, err := m.reset(11)
	require.NoError(t, err)
	assert.Equal(t, float64(cfg.DefaultHPPerLevel), u.Character.HealthGainPerLevel())
	assert.Equal(t, cfg.HealthAtLevel(10, u.Character.Stats.Vitality.ValueAdj, float64(cfg.DefaultHPPerLevel), 0), u.Character.HealthMax.Value)
	assert.Equal(t, 2, u.Character.Health, "reset cannot refill health")
}

func TestProgressionListShowsLiveHPRates(t *testing.T) {
	m, _ := testModule(t)
	text := m.list(12)
	assert.Contains(t, text, "Warrior")
	cfg := configs.GetProgressionConfig()
	line := func(start string, rate float64) string {
		return fmt.Sprintf("HP: %s%g per level through level %d, then %g per level.", start, rate, cfg.HPFullLevels, math.Round(cfg.HealthAfterFull(rate)*100)/100)
	}
	assert.Contains(t, text, line("10 extra to start, ", 1))
	assert.Contains(t, text, line("", 0.5))
	assert.Contains(t, text, "Attack 0.7 and Evasion 0.75 a level; trained for light armor.", "a caster's skill rates")
	preview, _ := m.chooseResult(newUser(12), "rogue", false)
	assert.Contains(t, preview, line("4 extra to start, ", 0.7))
}
