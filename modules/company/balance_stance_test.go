package company

import (
	"fmt"
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/stance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 69 balance cells: the same company, with and without a weapon
// stance, in the 5v5 mirror. A stance is a trade, so the cell reports win
// rate, health lost and rounds for both and checks neither a free win nor a
// trap. ASHVEIL_BALANCE=1 runs them; ASHVEIL_BALANCE_FIGHTS sets the fights
// per variant (default 100).

type stanceCell struct {
	name    string
	level   int
	members map[int]stanceKit // companion id -> gear
	stance  string
}

type stanceKit struct{ weapon, offhand int }

var stanceCells = []stanceCell{
	{"heavy blows: two warriors with glaives", 10, map[int]stanceKit{1: {10151, 0}, 3: {10151, 0}}, "heavy"},
	{"shield wall: two warriors with tower shields", 10, map[int]stanceKit{1: {10001, 20048}, 3: {10001, 20048}}, "wall"},
	{"quick draw: the ranger with a longbow", 10, map[int]stanceKit{4: {10172, 0}}, "quick"},
	{"keen edge: two warriors with daggers", 10, map[int]stanceKit{1: {10004, 0}, 3: {10004, 0}}, "keen"},
}

// stanceFight is one 5v5 fight with the cell's gear, in the stance or not.
func stanceFight(t *testing.T, cell stanceCell, inStance bool) balanceResult {
	return newStanceFight(t, cell, inStance).run()
}

// newStanceFight is the cell's fight, begun but not yet fought.
func newStanceFight(t *testing.T, cell stanceCell, inStance bool) *balanceFight {
	return newBalanceFightWithOptions(t, cell.level, companyDefault, enemyDefault, balanceFightOptions{EnemyCount: 5, Coordination: 1, LegacyMirror: true,
		Setup: func(b *brawl) {
			for id, kit := range cell.members {
				c := &b.companion(id).Character
				c.Equipment.Weapon, c.Equipment.Offhand = items.New(kit.weapon), items.Item{}
				if kit.offhand > 0 {
					c.Equipment.Offhand = items.New(kit.offhand)
				}
				c.RecalculateStats()
				if inStance {
					require.Contains(t, b.cmd("stance", fmt.Sprintf("#%d %s", id, cell.stance)), "stance")
				}
			}
		}})
}

// A fight with a stance set runs through the real round in the ordinary suite.
func TestBalanceHarnessRunsAStanceFight(t *testing.T) {
	f := newStanceFight(t, stanceCells[0], true)
	f.step()
	assert.Equal(t, stance.Heavy, f.companion(1).Character.Stance(), "the stance reached the fighter")
	assert.Positive(t, f.run().Rounds)
}

func TestBalanceStances(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to run the stance cells")
	}
	fights := balanceFights()
	for _, cell := range stanceCells {
		var plain, with []balanceResult
		for i := 0; i < fights; i++ {
			t.Run(fmt.Sprintf("%s-plain-%d", cell.stance, i), func(t *testing.T) { plain = append(plain, stanceFight(t, cell, false)) })
			t.Run(fmt.Sprintf("%s-stance-%d", cell.stance, i), func(t *testing.T) { with = append(with, stanceFight(t, cell, true)) })
		}
		_, plainWins := balanceMedianAndWins(plain)
		_, withWins := balanceMedianAndWins(with)
		plainLost, withLost := balanceHPLost(plain, sideCompany), balanceHPLost(with, sideCompany)
		plainRounds, withRounds := balanceMean(roundsOf(plain)), balanceMean(roundsOf(with))
		t.Logf("%s: wins %d%% -> %d%% (z %.1f); company health lost %.0f -> %.0f (z %.1f); rounds %.1f -> %.1f",
			cell.name, plainWins, withWins, balanceWelchZ(balanceWins(with), balanceWins(plain)),
			balanceMean(plainLost), balanceMean(withLost), balanceWelchZ(withLost, plainLost), plainRounds, withRounds)
		assert.LessOrEqual(t, withWins, plainWins+15, "%s is not a free win", cell.name)
		assert.GreaterOrEqual(t, withWins, plainWins-25, "%s is not a trap", cell.name)
	}
}

func roundsOf(results []balanceResult) []float64 {
	out := make([]float64, 0, len(results))
	for _, r := range results {
		out = append(out, float64(r.Rounds))
	}
	return out
}
