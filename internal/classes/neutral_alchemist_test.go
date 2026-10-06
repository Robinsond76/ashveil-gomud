package classes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39g: the Alchemist is a neutral lineage: three advanced routes, all
// open at any alignment, each naming an elite that opens later (39i).
func TestAlchemistHasThreeUngatedRoutes(t *testing.T) {
	adv := Advanced("alchemist")
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
	assert.Equal(t, []string{"apothecary", "bombardier", "mutagenist"}, ids)
	for _, align := range []int{-100, 0, 100} {
		for _, id := range ids {
			_, err := Check("alchemist", "", id, 10, align)
			assert.NoError(t, err, "%s at alignment %d", id, align)
		}
	}
	_, err := Check("alchemist", "", "bombardier", 9, 0)
	assert.Error(t, err)
}

func TestAlchemistBaseRanksComeByLevel(t *testing.T) {
	assert.Equal(t, []string{"Healing Draught", "Brew"}, rankNames(BaseRanksReached("alchemist", 2)))
	assert.Equal(t, []string{"Healing Draught", "Brew", "Antidote", "Fire Flask"}, rankNames(BaseRanksReached("alchemist", 6)))
	assert.Contains(t, rankNames(BaseRanksReached("alchemist", 8)), "Bracing Tonic")
}

func TestAlchemistRouteEffectsGrowByRank(t *testing.T) {
	assert.Zero(t, EffectsForLineage("alchemist", "apothecary", 9, nil).Int(FlaskHeal))
	assert.Equal(t, 30, EffectsForLineage("alchemist", "apothecary", 10, nil).Int(FlaskHeal))
	assert.Equal(t, 4, EffectsForLineage("alchemist", "apothecary", 15, nil).Int(FlaskCap))
	assert.Equal(t, 30, EffectsForLineage("alchemist", "apothecary", 20, nil).Int(FlaskSplash))
	assert.True(t, EffectsForLineage("alchemist", "apothecary", 25, nil).Has(FlaskClean))

	assert.Equal(t, 43, EffectsForLineage("alchemist", "bombardier", 10, nil).Int(FlaskFire))
	assert.Zero(t, EffectsForLineage("alchemist", "bombardier", 14, nil).Int(FlaskReach))
	assert.Equal(t, 1, EffectsForLineage("alchemist", "bombardier", 15, nil).Int(FlaskReach))
	assert.True(t, EffectsForLineage("alchemist", "bombardier", 20, nil).Has(FlaskBurn))
	assert.Equal(t, 4, EffectsForLineage("alchemist", "bombardier", 25, nil).Int(FlaskCap))

	assert.Equal(t, 10, EffectsForLineage("alchemist", "mutagenist", 10, nil).Int(MutagenCost))
	assert.Equal(t, 5, EffectsForLineage("alchemist", "mutagenist", 15, nil).Int(MutagenCost), "Stable mutagen halves the cost")
	assert.Equal(t, 15, EffectsForLineage("alchemist", "mutagenist", 20, nil).Int(MutagenArmr))
	assert.Equal(t, 2, EffectsForLineage("alchemist", "mutagenist", 20, nil).Int(TonicLong))
	assert.True(t, EffectsForLineage("alchemist", "mutagenist", 25, nil).Has(MutagenFree))

	// Other lineages have none of it.
	assert.False(t, EffectsForLineage("halberdier", "", 25, nil).Has(FlaskClean))
}

func TestAlchemistTalents(t *testing.T) {
	assert.NoError(t, CanPick("alchemist", "", nil, 5, "deep-satchel"))
	assert.ErrorIs(t, CanPick("alchemist", "", nil, 5, "wing-drill"), ErrUnknownTalent)
	assert.Equal(t, 2, EffectsForLineage("alchemist", "", 5, []string{"deep-satchel"}).Int(FlaskCap))
	assert.Equal(t, 10, EffectsForLineage("alchemist", "bombardier", 10, []string{"hot-brew"}).Int(FlaskFire)-43)
}

func TestAlchemistLevelUpNamesRanks(t *testing.T) {
	lines := RankLines("alchemist", "", 2, 3)
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "New rank: Antidote")
	assert.Contains(t, MilestoneFor("alchemist", "", 5), "Fire Flask")
}
