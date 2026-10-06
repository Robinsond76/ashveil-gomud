package classes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39f: the Gryphon Rider is a neutral lineage: three advanced routes,
// all open at any alignment, each naming an elite that opens later (39i).
func TestGryphonRiderHasThreeUngatedRoutes(t *testing.T) {
	adv := Advanced("gryphon-rider")
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
	assert.Equal(t, []string{"gryphon-knight", "skyscout", "wyvern-rider"}, ids)
	for _, align := range []int{-100, 0, 100} {
		for _, id := range ids {
			_, err := Check("gryphon-rider", "", id, 10, align)
			assert.NoError(t, err, "%s at alignment %d", id, align)
		}
	}
	_, err := Check("gryphon-rider", "", "skyscout", 9, 0)
	assert.Error(t, err)
}

// The lineage's own ranks come by level, before any promotion: Dive at 1,
// Talons at 3, Power dive at 8; a route's ranks sit on top of them.
func TestGryphonRiderBaseRanksAndRoutes(t *testing.T) {
	assert.Equal(t, []string{"Dive"}, rankNames(BaseRanksReached("gryphon-rider", 2)))
	assert.Equal(t, []string{"Dive", "Talons"}, rankNames(BaseRanksReached("gryphon-rider", 7)))
	assert.Equal(t, []string{"Dive", "Talons", "Power dive"}, rankNames(BaseRanksReached("gryphon-rider", 8)))

	assert.Nil(t, EffectsForLineage("gryphon-rider", "", 2, nil), "Dive's numbers are in the ability; nothing to read yet")
	assert.True(t, EffectsForLineage("gryphon-rider", "", 3, nil).Has(Talons))
	assert.Zero(t, EffectsForLineage("gryphon-rider", "", 7, nil).Int(DiveDmg))
	assert.Equal(t, 25, EffectsForLineage("gryphon-rider", "", 8, nil).Int(DiveDmg))

	// Falling stone replaces Power dive's 25% with 50% in all.
	assert.Equal(t, 50, EffectsForLineage("gryphon-rider", "gryphon-knight", 20, nil).Int(DiveDmg))
	assert.Equal(t, 25, EffectsForLineage("gryphon-rider", "gryphon-knight", 19, nil).Int(DiveDmg))
	assert.True(t, EffectsForLineage("gryphon-rider", "gryphon-knight", 10, nil).Has(DiveDown))
	assert.True(t, EffectsForLineage("gryphon-rider", "gryphon-knight", 25, nil).Has(DiveSteady))
	assert.Equal(t, 6, EffectsForLineage("gryphon-rider", "skyscout", 10, nil).Int(SkyEye))
	assert.Equal(t, 12, EffectsForLineage("gryphon-rider", "skyscout", 25, nil).Int(SkyEye))
	assert.Equal(t, 1, EffectsForLineage("gryphon-rider", "skyscout", 15, nil).Int(DiveCD))
	assert.True(t, EffectsForLineage("gryphon-rider", "skyscout", 20, nil).Has(DiveExpo))
	assert.True(t, EffectsForLineage("gryphon-rider", "wyvern-rider", 10, nil).Has(DivePois))
	assert.Equal(t, 8, EffectsForLineage("gryphon-rider", "wyvern-rider", 15, nil).Int(HealthPct))

	// Other lineages have none of it.
	assert.False(t, EffectsForLineage("halberdier", "", 20, nil).Has(Talons))
}

func TestGryphonRiderTalents(t *testing.T) {
	assert.Len(t, TalentsFor("gryphon-rider"), 5)
	assert.NoError(t, CanPick("gryphon-rider", "", nil, 5, "wing-drill"))
	assert.ErrorIs(t, CanPick("gryphon-rider", "", []string{"wing-drill"}, 15, "wing-drill"), ErrTalentMaxed)
	assert.ErrorIs(t, CanPick("gryphon-rider", "", nil, 5, "sweep-drill"), ErrUnknownTalent)
	// Wing Drill stacks with Quick stoop; the cooldown floor is the hook's.
	assert.Equal(t, 2, EffectsForLineage("gryphon-rider", "skyscout", 15, []string{"wing-drill"}).Int(DiveCD))
}

// The level-up report names a base rank when it arrives.
func TestGryphonRiderLevelUpNamesRanks(t *testing.T) {
	lines := RankLines("gryphon-rider", "", 2, 3)
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "New rank: Talons")
	assert.Contains(t, MilestoneFor("gryphon-rider", "", 6), "Power dive")
}
