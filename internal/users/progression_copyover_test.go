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
	assert.Equal(t, cfg.HealthAtLevel(20, restored.Character.Stats.Vitality.ValueAdj, 6), restored.Character.HealthMax.Value)
	assert.Equal(t, min(110, restored.Character.HealthMax.Value), restored.Character.Health)
	assert.Equal(t, min(3, restored.Character.ManaMax.Value), restored.Character.Mana)
}
