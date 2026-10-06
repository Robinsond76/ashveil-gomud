package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39b: the Samurai's lineage reaches the blow formulas.

const (
	samuraiBladeID = 90244
	samuraiPlateID = 90245
)

// samurai is a Samurai (a companion's runtime archetype, so its base ranks
// apply from level 1) of a level and an optional route, with a sword that
// always rolls 20 and its battle state ready, as the aura pass leaves it.
func samurai(t *testing.T, level int, class string) *characters.Character {
	t.Helper()
	items.SetTestItemSpec(&items.ItemSpec{ItemId: samuraiBladeID, Name: "test blade", Type: items.Weapon, Subtype: items.Slashing, Hands: 1,
		Damage: items.Damage{DiceRoll: "20d1", Attacks: 1, DiceCount: 20, SideCount: 1}})
	t.Cleanup(func() { items.RemoveTestItemSpec(samuraiBladeID) })
	c := armed(samuraiBladeID)
	c.Level = level
	c.HPArchetype = "samurai"
	c.SetClassState(class, nil)
	c.RTState()
	return c
}

func TestIaijutsuIsTheFirstStrikeOfABattleOnly(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 0, 0, 0)
	s := samurai(t, 1, "")
	target := armed(0)
	assert.True(t, s.IaiReady())

	first := strikeAt(s, target)
	require.True(t, first.Hit)
	assert.Equal(t, 30, first.DamageToTarget, "20 and half again")
	assert.False(t, s.IaiReady(), "spent by the swing")
	assert.Equal(t, 20, strikeAt(s, target).DamageToTarget, "later blows are plain")

	s.EndFightRT()
	s.RTState()
	assert.True(t, s.IaiReady(), "a new battle renews it")
	assert.Equal(t, 30, strikeAt(s, target).DamageToTarget)
}

func TestIaijutsuIsSpentEvenWhenTheFirstSwingMisses(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 0, 0, 100)
	s := samurai(t, 1, "")
	r := strikeAt(s, armed(0))
	assert.False(t, r.Hit, "a certain dodge")
	assert.False(t, s.IaiReady(), "the opening is gone")
}

func TestNoIaijutsuWithoutTheLineageOrForAWindUp(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 0, 0, 0)
	plain := samurai(t, 1, "")
	plain.HPArchetype = "warrior"
	assert.False(t, plain.IaiReady())
	assert.Equal(t, 20, strikeAt(plain, armed(0)).DamageToTarget)

	s := samurai(t, 1, "")
	calculateCombatPower(*s, *armed(0), User, Mob, 0, 0, &Power{Multiplier: 2})
	assert.True(t, s.IaiReady(), "a wind-up's blow is not the first strike")
}

func TestIaijutsuRaisesTheCriticalChanceOnTheFirstStrike(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 0, 0, 0)
	firstCrits, laterCrits := 0, 0
	for i := 0; i < 1500; i++ {
		s := samurai(t, 1, "")
		if strikeAt(s, armed(0)).Crit {
			firstCrits++
		}
		if strikeAt(s, armed(0)).Crit {
			laterCrits++
		}
	}
	assert.Zero(t, laterCrits, "the odds are pinned to 0 for everyone else")
	assert.InDelta(t, 150, firstCrits, 60, "+10%% of 1500")
}

func TestFocusBuildsCriticalChanceFromQuietRoundsAndStopsAtItsCap(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 0, 0, 0)
	s := samurai(t, 3, "")
	assert.Zero(t, s.ClassCrit(), "nothing yet")
	s.RT.Quiet = 1
	assert.Equal(t, 5, s.ClassCrit())
	s.RT.Quiet = 2
	assert.Equal(t, 10, s.ClassCrit())
	s.RT.Quiet = 9
	assert.Equal(t, 15, s.ClassCrit(), "capped at +15")
	s.SetClassState("kensai", nil)
	s.Level = 20
	assert.Equal(t, 20, s.ClassCrit(), "Deep focus raises the cap")

	// In the formula: a Samurai that has been quiet crits more.
	s = samurai(t, 3, "")
	s.RT.IaiSpent = true
	s.RT.Quiet = 3
	crits := 0
	for i := 0; i < 1500; i++ {
		if strikeAt(s, armed(0)).Crit {
			crits++
		}
	}
	assert.InDelta(t, 225, crits, 70, "+15%% of 1500")
}

func TestSharpEyeAddsCriticalChanceToEveryBlow(t *testing.T) {
	s := samurai(t, 5, "")
	s.HPTalents = []string{"sharp-eye", "sharp-eye"}
	assert.Equal(t, 3, s.ClassCrit(), "one talent earned at level 5")
	s.Level = 15
	assert.Equal(t, 6, s.ClassCrit(), "two at level 15")
}

func TestKensaiIaijutsuIgnoresHalfTheTargetsArmor(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 0, 0, 0)
	items.SetTestItemSpec(&items.ItemSpec{ItemId: samuraiPlateID, Name: "test plate", Type: items.Body, DamageReduction: 60})
	t.Cleanup(func() { items.RemoveTestItemSpec(samuraiPlateID) })
	armored := func() *characters.Character {
		c := armed(0)
		c.Equipment.Body = items.New(samuraiPlateID)
		return c
	}
	require.Equal(t, 60, armored().GetDefense())

	total := func(class string, level int) int {
		sum := 0
		for i := 0; i < 400; i++ {
			sum += strikeAt(samurai(t, level, class), armored()).DamageToTarget
		}
		return sum
	}
	plain, kensai := total("", 10), total("kensai", 10)
	// 30 damage a first strike, armor takes up to 60% of it at random: an
	// average of ~30% off. Piercing leaves ~15%.
	assert.Greater(t, kensai, plain+400*2, "a Kensai's first strike gets through more armor")
	// Only the first strike pierces.
	s := samurai(t, 10, "kensai")
	strikeAt(s, armored())
	later := 0
	for i := 0; i < 400; i++ {
		later += strikeAt(s, armored()).DamageToTarget
	}
	assert.Less(t, later, 400*17, "later blows meet the whole armor")
}

func TestRoninVengeanceGrowsWithEachFallenAlly(t *testing.T) {
	defenseSpecs(t)
	foe := classed("", 12)
	ronin := samurai(t, 12, "ronin")
	assert.Equal(t, 100, classBlowDamage(ronin, foe, 100))
	ronin.Aura.Fallen = 2
	assert.Equal(t, 120, classBlowDamage(ronin, foe, 100), "+10%% a fallen ally")
	ronin.Level = 20
	assert.Equal(t, 130, classBlowDamage(ronin, foe, 100), "Deeper vengeance: 15%%, two fallen")

	kensai := samurai(t, 12, "kensai")
	kensai.Aura.Fallen = 4
	assert.Equal(t, 100, classBlowDamage(kensai, foe, 100), "only a Ronin")
}
