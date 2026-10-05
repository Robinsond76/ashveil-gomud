package users

import (
	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

type warriorHP struct{}

func (warriorHP) CanTrain(int, string) (bool, string)      { return true, "" }
func (warriorHP) CanLearnSpell(int, string) (bool, string) { return true, "" }
func (warriorHP) Exists(string) bool                       { return true }
func (warriorHP) ArchetypeName(string) (string, bool)      { return "Warrior", true }
func (warriorHP) PlayerArchetype(int) (string, bool)       { return "warrior", true }
func (warriorHP) HealthPerLevel(string) (float64, bool)    { return 6, true }
func (warriorHP) HealthArchetypes() map[string]float64     { return map[string]float64{"warrior": 6} }

func TestProgressionCopyoverDefersVitalsUntilClassLoads(t *testing.T) {
	replayDataDir(t)
	archetypes.SetProvider(nil)
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	u := veteran()
	u.Character.Level = 20
	u.Character.Health = 110
	u.Character.Mana = 3
	require.NoError(t, SaveUser(*u))
	restored, err := loadUserById(7, true)
	require.NoError(t, err)
	assert.Equal(t, 110, restored.Character.Health, "boot must not clamp against uninitialized class tables")
	SetTestUser(restored)
	t.Cleanup(func() { RemoveTestUser(7) })
	archetypes.SetProvider(warriorHP{})
	ValidateActiveCharacters()
	cfg := configs.GetProgressionConfig()
	assert.Equal(t, cfg.HealthAtLevel(20, restored.Character.Stats.Vitality.ValueAdj, 6, 0), restored.Character.HealthMax.Value)
	assert.Equal(t, min(110, restored.Character.HealthMax.Value), restored.Character.Health)
	assert.Equal(t, min(3, restored.Character.ManaMax.Value), restored.Character.Mana)
}

// A player's character carries its user id from creation, so its HP uses the
// player's archetype rather than the enemy path (race/template overrides).
func TestProgressionNewPlayerCharactersUseTheirArchetype(t *testing.T) {
	replayDataDir(t)
	archetypes.SetProvider(warriorHP{})
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	assert.Equal(t, 6.0, NewUserRecord(7, 0).Character.HealthGainPerLevel())
	assert.Equal(t, float64(configs.GetProgressionConfig().DefaultHPPerLevel), NewUserRecord(0, 0).Character.HealthGainPerLevel(), "before an id, the default")

	u := NewUserRecord(8, 0)
	u.ReplaceCharacter(NewUserRecord(0, 0).Character)
	assert.Equal(t, 6.0, u.Character.HealthGainPerLevel(), "a replacement character joins its user")

	real := veteran()
	require.NoError(t, SaveUser(*real))
	replay, err := NewReplayUser(real)
	require.NoError(t, err)
	assert.Equal(t, 6.0, replay.Character.HealthGainPerLevel(), "a replay character keeps its own id")
}
