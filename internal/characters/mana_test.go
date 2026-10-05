package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// manaClasses serves the shipped Phase 35b mana pools: user 8 is a wizard,
// 9 a cleric, 10 a warrior.
type manaClasses struct{ skillClasses }

var manaProfiles = map[string]archetypes.Profile{
	"wizard":  {Name: "Wizard", ManaBase: 40, ManaPerLevel: 10},
	"cleric":  {Name: "Cleric", ManaBase: 36, ManaPerLevel: 9},
	"warrior": {Name: "Warrior"},
}

func (manaClasses) PlayerArchetype(userID int) (string, bool) {
	switch userID {
	case 8:
		return "wizard", true
	case 9:
		return "cleric", true
	case 10:
		return "warrior", true
	}
	return "", false
}
func (manaClasses) CombatProfile(id string) (archetypes.Profile, bool) {
	p, ok := manaProfiles[id]
	return p, ok
}

// Phase 35b: casters' mana pools come from their class; everyone else
// keeps the progression defaults, and an enemy template may override them.
func TestManaPoolsByClass(t *testing.T) {
	archetypes.SetProvider(manaClasses{})
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	cfg := configs.GetProgressionConfig()
	base, perLevel := int(cfg.ManaBase), float64(cfg.ManaPerLevel)

	for _, level := range []int{1, 10, 30} {
		pool := func(userID int, companionClass string) int {
			c := New()
			c.SetUserId(userID)
			c.HPArchetype = companionClass
			c.Level = level
			c.Validate()
			return c.ManaMax.Value
		}
		plain := pool(0, "")
		assert.Equal(t, plain+(40-base)+int(float64(level)*10)-int(float64(level)*perLevel), pool(8, ""), "wizard at %d", level)
		assert.Equal(t, plain+(36-base)+int(float64(level)*9)-int(float64(level)*perLevel), pool(9, ""), "cleric at %d", level)
		assert.Equal(t, plain, pool(10, ""), "a warrior keeps the default at %d", level)
		assert.Equal(t, pool(8, ""), pool(0, "wizard"), "a wizard companion's pool matches a player's at %d", level)
	}

	enemy := New()
	enemy.ManaBaseOverride, enemy.ManaPerLevelOverride = 30, 5
	b, pl := enemy.ManaRates()
	assert.Equal(t, 30, b)
	assert.Equal(t, 5.0, pl)
	b, pl = New().ManaRates()
	assert.Equal(t, base, b)
	assert.Equal(t, perLevel, pl)
}

// Phase 35b: a level-up raises the pool but never refills it.
func TestLevelUpDoesNotRefillMana(t *testing.T) {
	c := levelledCharacter(4)
	c.Mana = 3
	before := c.ManaMax.Value
	c.Experience = c.XPTNL()
	ok, _ := c.LevelUp()
	require.True(t, ok)
	assert.GreaterOrEqual(t, c.ManaMax.Value, before)
	assert.Equal(t, 3, c.Mana)
}

// Phase 35b: passive recovery stops at half of max health (and the wound
// limit) and restores no mana.
func TestTrickleHealStopsAtHalf(t *testing.T) {
	c := levelledCharacter(5)
	c.HealthMax.Value = 41
	c.Health, c.Mana = 1, 2
	for range 100 {
		c.TrickleHeal(3)
	}
	assert.Equal(t, 20, c.Health)
	assert.Equal(t, 2, c.Mana)
	c.Health = 30
	assert.Zero(t, c.TrickleHeal(3), "above half, nothing")
	assert.Equal(t, 30, c.Health)
	c.Health = 0
	assert.Zero(t, c.TrickleHeal(3), "the downed aren't raised")
}

// Phase 35b: a cast fails only above its chance, so 100% never fails.
func TestCastFails(t *testing.T) {
	assert.False(t, CastFails(100, 100))
	assert.False(t, CastFails(40, 40))
	assert.True(t, CastFails(41, 40))
	assert.True(t, CastFails(1, 0))
}
