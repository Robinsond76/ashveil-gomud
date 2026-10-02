package archetype

import (
	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProgressionShippedHPAndCreationChoice(t *testing.T) {
	m, _ := testModule(t)
	archetypes.SetProvider(m)
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	want := map[string]float64{"warrior": 6, "cleric": 5, "ranger": 5, "rogue": 4, "wizard": 3}
	assert.Equal(t, want, m.HealthArchetypes())
	for _, choice := range m.CreationChoices() {
		assert.Contains(t, choice.Description, "HP:")
		assert.Contains(t, choice.Description, "then 1 per level")
	}
	u := newUser(11)
	users.SetTestUser(u)
	t.Cleanup(func() { users.RemoveTestUser(11) })
	u.Character.Level = 10
	u.Character.Stats.Vitality.Training = 9
	u.Character.Health = 2
	m.choose(u, "wizard", true)
	cfg := configs.GetProgressionConfig()
	assert.Equal(t, 3.0, u.Character.HealthGainPerLevel())
	assert.Equal(t, cfg.HealthAtLevel(10, u.Character.Stats.Vitality.ValueAdj, 3), u.Character.HealthMax.Value)
	assert.Equal(t, 2, u.Character.Health)
	u.Character.HealthMax.Value = 9999
	m.onPlayerSpawn(events.PlayerSpawn{UserId: 11})
	assert.Equal(t, cfg.HealthAtLevel(10, u.Character.Stats.Vitality.ValueAdj, 3), u.Character.HealthMax.Value)
	hp, ok := m.HealthPerLevel("warrior")
	require.True(t, ok)
	assert.Equal(t, 6.0, hp)
	_, err := m.reset(11)
	require.NoError(t, err)
	assert.Equal(t, float64(cfg.DefaultHPPerLevel), u.Character.HealthGainPerLevel())
	assert.Equal(t, cfg.HealthAtLevel(10, u.Character.Stats.Vitality.ValueAdj, float64(cfg.DefaultHPPerLevel)), u.Character.HealthMax.Value)
	assert.Equal(t, 2, u.Character.Health, "reset cannot refill health")
}

func TestProgressionListShowsLiveHPRates(t *testing.T) {
	m, _ := testModule(t)
	text := m.list(12)
	assert.Contains(t, text, "Warrior")
	assert.Contains(t, text, "HP: 6 per level through level 20, then 1 per level.")
	assert.Contains(t, text, "HP: 3 per level through level 20, then 1 per level.")
	preview, _ := m.chooseResult(newUser(12), "rogue", false)
	assert.Contains(t, preview, "HP: 4 per level through level 20, then 1 per level.")
}
