package classes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39a: the Halberdier is a neutral lineage: three advanced routes,
// all open at any alignment, each naming an elite that opens later (39i).
func TestHalberdierHasThreeUngatedRoutes(t *testing.T) {
	adv := Advanced("halberdier")
	require.Len(t, adv, 3)
	var ids []string
	for _, c := range adv {
		ids = append(ids, c.ID)
		assert.Equal(t, GateAny, c.Gate, c.ID)
		assert.NotEmpty(t, c.Ranks, c.ID)
		elite, ok := Elite(c.ID)
		require.True(t, ok, "%s names an elite", c.ID)
		assert.True(t, elite.Planned, "%s elite opens with 39i", elite.ID)
	}
	assert.Equal(t, []string{"sweeper", "vanguard", "valkyrie"}, ids)
	// Promotion needs level 10 only, at any alignment.
	for _, align := range []int{-100, 0, 100} {
		for _, id := range ids {
			_, err := Check("halberdier", "", id, 10, align)
			assert.NoError(t, err, "%s at alignment %d", id, align)
		}
	}
	_, err := Check("halberdier", "", "sweeper", 9, 0)
	assert.Error(t, err)
}

func TestHalberdierRouteEffects(t *testing.T) {
	assert.Equal(t, 100, EffectsFor("sweeper", 10, nil).Int(SweepPct))
	assert.Equal(t, 1, EffectsFor("sweeper", 15, nil).Int(SweepCD))
	assert.Zero(t, EffectsFor("sweeper", 14, nil).Int(SweepCD))
	fx := EffectsFor("vanguard", 10, nil)
	assert.True(t, fx.Has(BraceCol) && fx.Has(BraceDown))
	fx = EffectsFor("valkyrie", 10, nil)
	assert.Equal(t, 8, fx.Int(ChargedMana))
	assert.Equal(t, 6, fx.Int(ChargedDice))
	assert.Equal(t, 100, fx.Int(ManaPct))
	assert.Equal(t, 10, EffectsFor("valkyrie", 20, nil).Int(ChargedDice))
	assert.Equal(t, 150, EffectsFor("valkyrie", 25, nil).Int(ManaPct))
	// The Sweep Drill talent shortens the cooldown for any route, once.
	assert.Equal(t, 2, EffectsFor("sweeper", 15, []string{"sweep-drill"}).Int(SweepCD))
	assert.Equal(t, 1, EffectsFor("", 5, []string{"sweep-drill"}).Int(SweepCD))
}

func TestHalberdierOffersFiveTalents(t *testing.T) {
	ts := TalentsFor("halberdier")
	assert.Len(t, ts, 5)
	assert.NoError(t, CanPick("halberdier", "", nil, 5, "sweep-drill"))
	assert.ErrorIs(t, CanPick("halberdier", "", []string{"sweep-drill"}, 15, "sweep-drill"), ErrTalentMaxed)
	assert.ErrorIs(t, CanPick("halberdier", "", nil, 5, "deep-well"), ErrUnknownTalent)
}
