package classes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39e: the Beast Tamer is a neutral lineage: three advanced routes, all
// open at any alignment, each naming an elite that opens later (39i).
func TestBeastTamerHasThreeUngatedRoutes(t *testing.T) {
	adv := Advanced("beasttamer")
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
	assert.Equal(t, []string{"houndmaster", "bearward", "dragon-tamer"}, ids)
	for _, align := range []int{-100, 0, 100} {
		for _, id := range ids {
			_, err := Check("beasttamer", "", id, 10, align)
			assert.NoError(t, err, "%s at alignment %d", id, align)
		}
	}
	_, err := Check("beasttamer", "", "bearward", 9, 0)
	assert.Error(t, err)
}

func TestBeastTamerBaseRanksComeByLevel(t *testing.T) {
	assert.True(t, EffectsForLineage("beasttamer", "", 1, nil).Has(BeastSic))
	assert.Equal(t, 10, EffectsForLineage("beasttamer", "", 1, nil).Int(SicAttack))
	assert.Zero(t, EffectsForLineage("beasttamer", "", 2, nil).Int(Rally))
	assert.Equal(t, 2, EffectsForLineage("beasttamer", "", 3, nil).Int(Rally))
	assert.Zero(t, EffectsForLineage("beasttamer", "", 7, nil).Int(PackSense))
	assert.Equal(t, 5, EffectsForLineage("beasttamer", "", 8, nil).Int(PackSense))
}

func TestBeastTamerRoutesChooseTheBeast(t *testing.T) {
	assert.Zero(t, EffectsForLineage("beasttamer", "", 30, nil).Int(BeastKind), "a plain Tamer keeps its wolf")
	fx := EffectsForLineage("beasttamer", "houndmaster", 10, nil)
	assert.Equal(t, KindWarhound, fx.Int(BeastKind))
	assert.True(t, fx.Has(BeastHobble))
	assert.Equal(t, 110, fx.Int(BeastHPPct))
	assert.Equal(t, 130, EffectsForLineage("beasttamer", "houndmaster", 25, nil).Int(BeastHPPct))
	fx = EffectsForLineage("beasttamer", "bearward", 10, nil)
	assert.Equal(t, KindBear, fx.Int(BeastKind))
	assert.Equal(t, 120, fx.Int(BeastHPPct))
	assert.Equal(t, 3, fx.Int(BeastGuards))
	assert.Equal(t, 4, EffectsForLineage("beasttamer", "bearward", 25, nil).Int(BeastGuards))
	fx = EffectsForLineage("beasttamer", "dragon-tamer", 10, nil)
	assert.Equal(t, KindDrake, fx.Int(BeastKind))
	assert.Equal(t, 3, fx.Int(BeastBreath))
	assert.Equal(t, 1, EffectsForLineage("beasttamer", "dragon-tamer", 25, nil).Int(BeastBreathCut))
	assert.Equal(t, 2, EffectsForLineage("beasttamer", "houndmaster", 15, nil).Int(BeastAttack))
}

func TestBeastTamerTalents(t *testing.T) {
	assert.Len(t, TalentsFor("beasttamer"), 5)
	assert.NoError(t, CanPick("beasttamer", "", nil, 5, "steady-leash"))
	assert.ErrorIs(t, CanPick("beasttamer", "", []string{"steady-leash"}, 15, "steady-leash"), ErrTalentMaxed)
	assert.ErrorIs(t, CanPick("beasttamer", "", nil, 5, "hardwood"), ErrUnknownTalent)
	fx := EffectsForLineage("beasttamer", "", 45, []string{"steady-leash", "thick-pelt", "strong-jaws", "field-hand"})
	assert.Equal(t, 3, fx.Int(Rally), "2 from rank 3, 1 from the talent")
	assert.Equal(t, 10, fx.Int(BeastHPBonus))
	assert.Equal(t, 1, fx.Int(BeastDamage))
}
