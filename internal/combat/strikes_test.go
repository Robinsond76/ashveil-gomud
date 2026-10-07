package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 62: the strikes a round records are the engine's own numbers.

func TestStrikeRecordsAMiss(t *testing.T) {
	defenseSpecs(t)
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.ToHitMin, gameplay.Combat.ToHitMax = 0, 0
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	r := strikeAt(armed(edgeSwordID), armed(0))
	require.Len(t, r.Strikes, 1)
	st := r.Strikes[0]
	assert.False(t, st.Hit)
	assert.Equal(t, 0, st.Chance)
	assert.GreaterOrEqual(t, st.Roll, st.Chance, "a miss rolled at or over its chance")
	assert.Zero(t, st.Damage)
	assert.Contains(t, st.Explain()[0], "it missed")
}

func TestStrikeRecordsADefense(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 100, 0, 0)
	bearer := armed(edgeSwordID)
	bearer.Equipment.Offhand = items.New(defShieldID)
	r := strikeAt(armed(edgeSwordID), bearer)
	require.Len(t, r.Strikes, 1)
	st := r.Strikes[0]
	assert.True(t, st.Hit, "the strike landed before the shield met it")
	assert.Equal(t, DefenseBlocked, st.Defense)
	assert.Equal(t, 100, st.DefenseChance)
	assert.Equal(t, []string{DefenseBlocked}, r.Defenses)
	assert.Contains(t, st.Explain()[1], "blocked, a 100 in 100 chance")
}

func TestStrikeRecordsEachStrikeOfARound(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 0, 0, 100)
	const twoStrikes = 99411
	items.SetTestItemSpec(&items.ItemSpec{ItemId: twoStrikes, Name: "test flail", Type: items.Weapon, Subtype: items.Bludgeoning, Hands: 1,
		Damage: items.Damage{DiceRoll: "2@1d1", Attacks: 2, DiceCount: 1, SideCount: 1}})
	t.Cleanup(func() { items.RemoveTestItemSpec(twoStrikes) })
	r := strikeAt(armed(twoStrikes), armed(0))
	require.Len(t, r.Strikes, 2)
	for _, st := range r.Strikes {
		assert.Equal(t, DefenseDodged, st.Defense)
		assert.Equal(t, 100, st.DefenseChance)
	}
}

func TestStrikeDamageMatchesTheRound(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 0, 0, 0)
	const heavy = 99412
	items.SetTestItemSpec(&items.ItemSpec{ItemId: heavy, Name: "test maul", Type: items.Weapon, Subtype: items.Bludgeoning, Hands: 1,
		Damage: items.Damage{DiceRoll: "3@1d1+9", Attacks: 3, DiceCount: 1, SideCount: 1, BonusDamage: 9}})
	t.Cleanup(func() { items.RemoveTestItemSpec(heavy) })
	r := strikeAt(armed(heavy), armed(0))
	require.Len(t, r.Strikes, 3)
	total, reduced := 0, 0
	for _, st := range r.Strikes {
		require.True(t, st.Hit)
		assert.Empty(t, st.Defense)
		assert.Equal(t, st.Raw-st.Reduced, st.Damage, "armor took exactly what it says it took")
		assert.Equal(t, st.Reduced, st.ArmorTook, "with no ward or aura, armor took it all")
		total += st.Damage
		reduced += st.Reduced
	}
	assert.Equal(t, r.DamageToTarget, total)
	assert.Equal(t, r.DamageToTargetReduction, reduced)
}
