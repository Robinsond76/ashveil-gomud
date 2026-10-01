package combat

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 30g3 test items: armor heavy enough to burden anyone fully.
const burdenPlateID = 99420 // 40 kg

func burdenSpecs(t *testing.T) {
	t.Helper()
	defenseSpecs(t)
	items.SetTestItemSpec(&items.ItemSpec{ItemId: burdenPlateID, Name: "test lead plate", Type: items.Body, Weight: 40000})
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.AgilityBaseKg, gameplay.Combat.AgilityStrengthKg, gameplay.Combat.AgilityFreeLoad = 15, 0.5, 0.35
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	t.Cleanup(func() { items.RemoveTestItemSpec(burdenPlateID) })
}

// heavy wears the 40 kg plate: fully burdened at any Strength under 50.
func heavy(c *characters.Character) *characters.Character {
	c.Equipment.Body = items.New(burdenPlateID)
	return c
}

func TestBurdenedDodge(t *testing.T) {
	cases := []struct {
		dodge  int
		burden float64
		want   int
	}{
		{30, 0, 30},
		{30, 0.5, 21},
		{30, 1, 12},
		{100, 1, 40},
		{0, 1, 0},
		{30, 2, 12}, // held at full
		{30, -1, 30},
		{30, math.NaN(), 30},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, burdenedDodge(tc.dodge, tc.burden), "dodge %d, burden %v", tc.dodge, tc.burden)
	}
}

// Through the real strike loop: a certain dodge stays certain unburdened
// and falls to 40% fully burdened.
func TestBurdenLowersDodgeInTheStrikeLoop(t *testing.T) {
	burdenSpecs(t)
	defenseOdds(t, 0, 0, 100)

	light := armed(0)
	for i := 0; i < 50; i++ {
		require.Equal(t, []string{DefenseDodged}, strikeAt(armed(edgeSwordID), light).Defenses, "unburdened, every dodge certain")
	}

	burdened := heavy(armed(0))
	require.Equal(t, 1.0, burdened.Burden())
	const strikes = 400
	dodged := 0
	for i := 0; i < strikes; i++ {
		r := strikeAt(armed(edgeSwordID), burdened)
		switch {
		case len(r.Defenses) == 1 && r.Defenses[0] == DefenseDodged:
			dodged++
		default:
			require.True(t, r.Hit, "an undodged strike lands")
			require.Empty(t, r.Defenses)
		}
	}
	// 40% of 400 is 160; the bounds are more than five deviations wide.
	assert.Greater(t, dodged, 100)
	assert.Less(t, dodged, 220)
}

// Burden doesn't touch parry or block.
func TestBurdenLeavesParryAndBlock(t *testing.T) {
	burdenSpecs(t)

	defenseOdds(t, 0, 100, 0)
	parrier := heavy(armed(defAxeID))
	for i := 0; i < 50; i++ {
		require.Equal(t, []string{DefenseParried}, strikeAt(armed(edgeSwordID), parrier).Defenses, "a certain parry stays certain")
	}

	defenseOdds(t, 100, 0, 0)
	bearer := heavy(armed(edgeSwordID))
	bearer.Equipment.Offhand = items.New(defShieldID)
	for i := 0; i < 50; i++ {
		require.Equal(t, []string{DefenseBlocked}, strikeAt(armed(edgeSwordID), bearer).Defenses, "a certain block stays certain")
	}
}

// Decision 15 rolls the higher of parry and dodge: burden lowers the dodge
// first, so a burdened swordsman (dodge 100 → 40) parries (50) instead.
func TestBurdenMovesParryOrDodgeChoice(t *testing.T) {
	burdenSpecs(t)
	defenseOdds(t, 0, 50, 100)

	free := armed(defAxeID)
	for i := 0; i < 50; i++ {
		require.Equal(t, []string{DefenseDodged}, strikeAt(armed(edgeSwordID), free).Defenses, "unburdened: the certain dodge is rolled")
	}

	burdened := heavy(armed(defAxeID))
	parried := 0
	for i := 0; i < 200; i++ {
		r := strikeAt(armed(edgeSwordID), burdened)
		require.NotContains(t, r.Defenses, DefenseDodged, "burdened: the parry is higher, so no dodge is rolled")
		if len(r.Defenses) == 1 {
			parried++
		}
	}
	assert.Greater(t, parried, 50, "about half parried")
	assert.Less(t, parried, 150)
}

// The weapon rankings' estimate uses the same burdened dodge.
func TestExpectedDPSCountsBurden(t *testing.T) {
	burdenSpecs(t)
	defenseOdds(t, 0, 0, 50)
	attacker := armed(edgeSwordID)
	light, burdened := armed(0), heavy(armed(0))
	assert.Greater(t, expectedDPS(*attacker, *burdened), expectedDPS(*attacker, *light), "a burdened target dodges less, so takes more")
}
