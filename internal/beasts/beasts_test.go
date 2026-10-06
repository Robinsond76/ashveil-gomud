package beasts

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/stretchr/testify/assert"
)

func fxOf(route string, level int, talents ...string) classes.Effects {
	return classes.EffectsForLineage(Lineage, route, level, talents)
}

func TestGiftsFollowTheRoute(t *testing.T) {
	g := GiftsFor(fxOf("", 5))
	assert.Equal(t, Wolf, g.Kind)
	assert.Equal(t, WolfMobID, g.MobID)
	assert.Equal(t, BaseHPPct, g.HPPct)
	assert.Equal(t, 8, g.Sides)
	assert.False(t, g.Hobble)
	assert.Zero(t, g.Guards)
	assert.Zero(t, g.BreathEvery)

	g = GiftsFor(fxOf("houndmaster", 25))
	assert.Equal(t, Warhound, g.Kind)
	assert.Equal(t, WarhoundMobID, g.MobID)
	assert.Equal(t, BaseHPPct*130/100, g.HPPct)
	assert.True(t, g.Hobble)
	assert.Equal(t, 2, g.Attack)
	assert.Equal(t, 1, g.Damage)

	g = GiftsFor(fxOf("bearward", 10))
	assert.Equal(t, Bear, g.Kind)
	assert.Equal(t, BaseHPPct*120/100, g.HPPct, "120% of the standard beast")
	assert.Equal(t, 3, g.Guards)

	g = GiftsFor(fxOf("dragon-tamer", 10))
	assert.Equal(t, Drake, g.Kind)
	assert.Equal(t, 6, g.Sides)
	assert.Equal(t, 3, g.BreathEvery)
	assert.Equal(t, 2, GiftsFor(fxOf("dragon-tamer", 25)).BreathEvery)

	// Thick Pelt adds percent points to the share.
	g = GiftsFor(fxOf("bearward", 45, "thick-pelt"))
	assert.Equal(t, BaseHPPct*(140+10)/100, g.HPPct)
}

func TestARecordMendsWithRest(t *testing.T) {
	c := &characters.Character{HPArchetype: Lineage, Level: 5}
	assert.False(t, NeedsRest(c))
	assert.False(t, Recover(c))
	c.EnsureBeast("Ash").Damage = 7
	assert.True(t, NeedsRest(c))
	assert.True(t, Recover(c))
	assert.False(t, NeedsRest(c))
	c.Beast.Wounded, c.Beast.Damage = true, 30
	assert.Equal(t, "wounded", Condition(*c.Beast, 30))
	assert.True(t, Recover(c))
	assert.False(t, c.Beast.Wounded)
	assert.Equal(t, "whole", Condition(*c.Beast, 30))
	assert.Equal(t, "bruised", Condition(characters.BeastState{Damage: 5}, 30))
	assert.Equal(t, "badly hurt", Condition(characters.BeastState{Damage: 20}, 30))
}

func TestABeastHasItsOwnShareOfAWarriorsHealth(t *testing.T) {
	c := &characters.Character{HPArchetype: Lineage, Level: 10}
	wolf := HealthLimit(c)
	probe := characters.Character{Level: 10, HPArchetype: "warrior"}
	probe.RecalculateStats()
	assert.Less(t, wolf, probe.HealthLimit())
	assert.InDelta(t, float64(probe.HealthLimit())*BaseHPPct/100, float64(wolf), 2)
}

func TestBreathScalesWithTheTamersLevel(t *testing.T) {
	assert.Equal(t, 1+10/BreathHalf, BreathDamage(10, 0, 1))
	assert.Equal(t, 6+10/BreathHalf+2, BreathDamage(10, 2, 6))
	assert.GreaterOrEqual(t, BreathDamage(0, 0, 0), 1)
}

func TestAClonedRecordIsIndependent(t *testing.T) {
	a := &characters.BeastState{Name: "Ash", Damage: 3}
	b := characters.CloneBeast(a)
	b.Damage = 9
	assert.Equal(t, 3, a.Damage)
	assert.Nil(t, characters.CloneBeast(nil))
}
