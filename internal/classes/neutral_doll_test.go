package classes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39d: the Doll Master is a neutral lineage: three advanced routes,
// all open at any alignment, each naming an elite that opens later (39i).
func TestDollMasterHasThreeUngatedRoutes(t *testing.T) {
	adv := Advanced("dollmaster")
	require.Len(t, adv, 3)
	var ids []string
	for _, c := range adv {
		ids = append(ids, c.ID)
		assert.Equal(t, GateAny, c.Gate, c.ID)
		assert.NotEmpty(t, c.Ranks, c.ID)
		elite, ok := Elite(c.ID)
		require.True(t, ok, "%s names an elite", c.ID)
		assert.False(t, elite.Planned, "%s elite opened with 39i", elite.ID)
		assert.Len(t, elite.Ranks, 7, "%s elite has a rank every five levels, 30 to 60", elite.ID)
	}
	assert.Equal(t, []string{"puppeteer", "golemancer", "marionettist"}, ids)
	for _, align := range []int{-100, 0, 100} {
		for _, id := range ids {
			_, err := Check("dollmaster", "", id, 10, align)
			assert.NoError(t, err, "%s at alignment %d", id, align)
		}
	}
	_, err := Check("dollmaster", "", "puppeteer", 9, 0)
	assert.Error(t, err)
}

func TestDollMasterBaseRanksComeByLevel(t *testing.T) {
	assert.Zero(t, EffectsForLineage("dollmaster", "", 4, nil).Int(DollGuards))
	assert.Equal(t, 2, EffectsForLineage("dollmaster", "", 5, nil).Int(DollGuards))
	assert.Equal(t, 3, EffectsForLineage("dollmaster", "", 8, nil).Int(DollGuards))
	assert.False(t, EffectsForLineage("dollmaster", "", 11, nil).Has(DollTangle))
	assert.True(t, EffectsForLineage("dollmaster", "", 12, nil).Has(DollTangle))
	assert.False(t, EffectsForLineage("dollmaster", "", 17, nil).Has(Splice))
	assert.True(t, EffectsForLineage("dollmaster", "", 18, nil).Has(Splice))
}

func TestDollMasterRouteEffects(t *testing.T) {
	fx := EffectsForLineage("dollmaster", "puppeteer", 10, nil)
	assert.Equal(t, 1, fx.Int(DollCount))
	assert.Equal(t, 50, fx.Int(DollHPPct))
	assert.Equal(t, 60, EffectsForLineage("dollmaster", "puppeteer", 20, nil).Int(DollHPPct))
	assert.Equal(t, 4, EffectsForLineage("dollmaster", "puppeteer", 25, nil).Int(DollArmor))
	fx = EffectsForLineage("dollmaster", "golemancer", 10, nil)
	assert.Equal(t, 130, fx.Int(DollHPPct))
	assert.Equal(t, 15, fx.Int(DollArmor))
	assert.True(t, fx.Has(DollNoWear))
	assert.Equal(t, 4, fx.Int(DollGuards), "Golemancer's guard string works four times")
	fx = EffectsForLineage("dollmaster", "marionettist", 10, nil)
	assert.Equal(t, 2, fx.Int(TangleFoes))
	assert.Equal(t, 1, fx.Int(TangleCD))
	assert.Equal(t, 10, EffectsForLineage("dollmaster", "marionettist", 20, nil).Int(TanglePush))
}

func TestDollMasterTalents(t *testing.T) {
	assert.Len(t, TalentsFor("dollmaster"), 5)
	assert.NoError(t, CanPick("dollmaster", "", nil, 5, "stout-strings"))
	assert.ErrorIs(t, CanPick("dollmaster", "", []string{"stout-strings"}, 15, "stout-strings"), ErrTalentMaxed)
	assert.ErrorIs(t, CanPick("dollmaster", "", nil, 5, "sweep-drill"), ErrUnknownTalent)
	fx := EffectsForLineage("dollmaster", "", 45, []string{"stout-strings", "hardwood", "fine-carving", "steady-hand"})
	assert.Equal(t, 4, fx.Int(DollGuards), "3 from rank 8, 1 from the talent")
	assert.Equal(t, 10, fx.Int(DollHPBonus))
	assert.Equal(t, 1, fx.Int(DollDamage))
	assert.Equal(t, 2, fx.Int(DollAttack))
}
