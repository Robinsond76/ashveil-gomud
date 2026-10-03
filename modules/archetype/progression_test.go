package archetype

import (
	"fmt"
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
	want := map[string]float64{"warrior": 3, "cleric": 2.5, "ranger": 2.5, "rogue": 2, "wizard": 1.5}
	assert.Equal(t, want, m.HealthArchetypes())
	for _, choice := range m.CreationChoices() {
		assert.Contains(t, choice.Description, "HP:")
		assert.Contains(t, choice.Description, fmt.Sprintf("then %g per level", float64(configs.GetProgressionConfig().HPAfterFull)))
	}
	u := newUser(11)
	users.SetTestUser(u)
	t.Cleanup(func() { users.RemoveTestUser(11) })
	u.Character.Level = 10
	u.Character.Stats.Vitality.Training = 9
	u.Character.Health = 2
	m.choose(u, "wizard", true)
	cfg := configs.GetProgressionConfig()
	assert.Equal(t, 1.5, u.Character.HealthGainPerLevel())
	assert.Equal(t, cfg.HealthAtLevel(10, u.Character.Stats.Vitality.ValueAdj, 1.5), u.Character.HealthMax.Value)
	assert.Equal(t, 2, u.Character.Health)
	u.Character.HealthMax.Value = 9999
	m.onPlayerSpawn(events.PlayerSpawn{UserId: 11})
	assert.Equal(t, cfg.HealthAtLevel(10, u.Character.Stats.Vitality.ValueAdj, 1.5), u.Character.HealthMax.Value)
	hp, ok := m.HealthPerLevel("warrior")
	require.True(t, ok)
	assert.Equal(t, 3.0, hp)
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
	assert.Contains(t, text, fmt.Sprintf("HP: 3 per level through level %d, then %g per level.", configs.GetProgressionConfig().HPFullLevels, float64(configs.GetProgressionConfig().HPAfterFull)))
	assert.Contains(t, text, fmt.Sprintf("HP: 1.5 per level through level %d, then %g per level.", configs.GetProgressionConfig().HPFullLevels, float64(configs.GetProgressionConfig().HPAfterFull)))
	preview, _ := m.chooseResult(newUser(12), "rogue", false)
	assert.Contains(t, preview, fmt.Sprintf("HP: 2 per level through level %d, then %g per level.", configs.GetProgressionConfig().HPFullLevels, float64(configs.GetProgressionConfig().HPAfterFull)))
}
