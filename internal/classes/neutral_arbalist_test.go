package classes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39h: the Arbalist is a neutral lineage: three advanced routes, all
// open at any alignment, each naming an elite that opens later (39i).
func TestArbalistHasThreeUngatedRoutes(t *testing.T) {
	adv := Advanced("arbalist")
	require.Len(t, adv, 3)
	var ids []string
	for _, c := range adv {
		ids = append(ids, c.ID)
		assert.Equal(t, GateAny, c.Gate, c.ID)
		assert.NotEmpty(t, c.Ranks, c.ID)
		elite, ok := Elite(c.ID)
		require.True(t, ok, "%s names an elite", c.ID)
		assert.False(t, elite.Planned, "%s elite opened with 39i2", elite.ID)
	}
	assert.Equal(t, []string{"siegebreaker", "sharpshooter", "warden-of-the-wall"}, ids)
	for _, align := range []int{-100, 0, 100} {
		for _, id := range ids {
			_, err := Check("arbalist", "", id, 10, align)
			assert.NoError(t, err, "%s at alignment %d", id, align)
		}
	}
	_, err := Check("arbalist", "", "sharpshooter", 9, 0)
	assert.Error(t, err)
}

// The lineage's own ranks come by level, before any promotion: Piercing Bolt
// at 1, Steady Aim at 3, Crippling Bolt at 6, Armor-breaker at 8.
func TestArbalistBaseRanksAndRoutes(t *testing.T) {
	assert.Equal(t, []string{"Piercing Bolt"}, rankNames(BaseRanksReached("arbalist", 2)))
	assert.Equal(t, []string{"Piercing Bolt", "Steady Aim"}, rankNames(BaseRanksReached("arbalist", 5)))
	assert.Equal(t, []string{"Piercing Bolt", "Steady Aim", "Crippling Bolt", "Armor-breaker"}, rankNames(BaseRanksReached("arbalist", 8)))

	assert.Equal(t, 50, EffectsForLineage("arbalist", "", 1, nil).Int(BoltPierce), "half the armor from level 1")
	assert.Equal(t, 100, EffectsForLineage("arbalist", "", 8, nil).Int(BoltPierce), "all of it from level 8")
	assert.Zero(t, EffectsForLineage("arbalist", "", 2, nil).Int(SteadyAim))
	assert.Equal(t, 10, EffectsForLineage("arbalist", "", 3, nil).Int(SteadyAim))
	assert.Zero(t, EffectsForLineage("arbalist", "", 5, nil).Int(BoltCripple))
	assert.Equal(t, 2, EffectsForLineage("arbalist", "", 6, nil).Int(BoltCripple))

	// Siegebreaker: each landed bolt wears armor down, deeper from 20.
	sb := func(level int) Effects { return EffectsForLineage("arbalist", "siegebreaker", level, nil) }
	assert.Equal(t, 10, sb(10).Int(Shred))
	assert.Equal(t, 30, sb(10).Int(ShredCap))
	assert.Equal(t, 20, sb(15).Int(BoltDmg))
	assert.Equal(t, 15, sb(20).Int(Shred))
	assert.Equal(t, 45, sb(20).Int(ShredCap))
	assert.Equal(t, 35, sb(25).Int(BoltDmg))

	// Sharpshooter.
	ss := func(level int) Effects { return EffectsForLineage("arbalist", "sharpshooter", level, nil) }
	assert.Equal(t, 15, ss(10).Int(RangedCrit))
	assert.True(t, ss(10).Has(NoBlockCrit))
	assert.Equal(t, 20, ss(15).Int(SteadyAim), "Steady hands replaces Steady Aim's 10")
	assert.True(t, ss(20).Has(FirstLoaded))
	assert.Equal(t, 4, ss(25).Int(RangedAttack))

	// Warden of the Wall.
	ww := func(level int) Effects { return EffectsForLineage("arbalist", "warden-of-the-wall", level, nil) }
	assert.Equal(t, 10, ww(10).Int(AuraResolv))
	assert.Equal(t, 4, ww(15).Int(Armor))
	assert.Equal(t, 15, ww(20).Int(AuraResolv))
	assert.Equal(t, 8, ww(25).Int(HealthPct))

	// Other lineages have none of it.
	assert.False(t, EffectsForLineage("ranger", "", 20, nil).Has(BoltPierce))
}

func TestArbalistTalents(t *testing.T) {
	assert.Len(t, TalentsFor("arbalist"), 5)
	assert.NoError(t, CanPick("arbalist", "", nil, 5, "heavy-bolts"))
	assert.ErrorIs(t, CanPick("arbalist", "", nil, 5, "wing-drill"), ErrUnknownTalent)
	// Heavy Bolts stacks with a route's.
	assert.Equal(t, 30, EffectsForLineage("arbalist", "siegebreaker", 15, []string{"heavy-bolts"}).Int(BoltDmg))
}

// The level-up report names a base rank when it arrives.
func TestArbalistLevelUpNamesRanks(t *testing.T) {
	lines := RankLines("arbalist", "", 5, 6)
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "New rank: Crippling Bolt")
	assert.Contains(t, MilestoneFor("arbalist", "", 6), "Armor-breaker")
}
