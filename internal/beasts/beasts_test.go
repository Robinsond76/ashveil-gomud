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
	assert.Equal(t, 100, g.HPMult)
	assert.Equal(t, StandardPct(5), g.HPPctAt(5))
	assert.Less(t, g.HPPctAt(5), g.HPPctAt(20), "the beast grows with its Tamer")
	assert.Equal(t, BiteSides, g.Sides)
	assert.Equal(t, 8, StandardPct(1))
	assert.Equal(t, 19, StandardPct(10))
	assert.Equal(t, 37, StandardPct(20))
	assert.Equal(t, 6, BiteBonus(20))
	assert.False(t, g.Hobble)
	assert.Zero(t, g.Guards)
	assert.Zero(t, g.BreathEvery)

	g = GiftsFor(fxOf("houndmaster", 25))
	assert.Equal(t, Warhound, g.Kind)
	assert.Equal(t, WarhoundMobID, g.MobID)
	assert.Equal(t, 130, g.HPMult)
	assert.True(t, g.Hobble)
	assert.Equal(t, 2, g.Attack)
	assert.Equal(t, 1, g.Damage)

	g = GiftsFor(fxOf("bearward", 10))
	assert.Equal(t, Bear, g.Kind)
	assert.Equal(t, StandardPct(10)*120/100, g.HPPctAt(10), "120% of the standard beast")
	assert.Equal(t, 3, g.Guards)

	g = GiftsFor(fxOf("dragon-tamer", 10))
	assert.Equal(t, Drake, g.Kind)
	assert.Equal(t, BiteSides-1, g.Sides, "a smaller bite")
	assert.Equal(t, 3, g.BreathEvery)
	assert.Equal(t, 2, GiftsFor(fxOf("dragon-tamer", 25)).BreathEvery)

	// Thick Pelt adds percent points to the share.
	g = GiftsFor(fxOf("bearward", 45, "thick-pelt"))
	assert.Equal(t, 150, g.HPMult)
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
	probe := characters.Character{Level: 10, HPArchetype: "warrior", RaceId: RaceID}
	probe.RecalculateStats()
	assert.Less(t, wolf, probe.HealthLimit())
	assert.InDelta(t, float64(probe.HealthLimit()*StandardPct(10))/100, float64(wolf), 2)
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
