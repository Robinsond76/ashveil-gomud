package company

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hexes"
	"github.com/GoMudEngine/GoMud/internal/hooks"
)

// Phase 38a: the Witch against the Wizard in the even mirror. Opt-in
// (ASHVEIL_BALANCE=1); it reports and asserts nothing.
//
// One member of the shipped company is swapped for a wizard, then a witch,
// and the company fights a five-foe group of its own level through the real
// round with real dice. The numbers recorded in docs/PROJECT_STATUS.md came
// from this run.
func TestPhase38aWitchAgainstWizard(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to measure the Witch")
	}
	fights := 60
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && v > 0 {
		fights = v
	}
	for _, level := range []int{5, 10, 20} {
		for _, class := range []string{"warrior", "wizard", "witch"} {
			var results []balanceResult
			for i := 0; i < fights; i++ {
				t.Run(fmt.Sprintf("L%d-%s-%d", level, class, i), func(t *testing.T) {
					hexes.Default.Reset()
					t.Cleanup(hexes.Default.Reset)
					freshEvents(t)
					l := events.RegisterListener(events.MoraleCheck{}, hooks.DreadCheck)
					t.Cleanup(func() { events.UnregisterListener(events.MoraleCheck{}, l) })
					f := newBalanceFightWithOptions(t, level, companyDefault, enemyDefault, balanceFightOptions{
						EnemyCount: 5, Coordination: 1, LegacyMirror: true,
						Classes: map[int]string{3: class},
					})
					results = append(results, f.run())
				})
			}
			median, wins := balanceMedianAndWins(results)
			lost := balanceMean(balanceHPLost(results, sideCompany))
			var started, cast, broken float64
			for _, r := range results {
				started += float64(r.Tally.Casts[sideCompany])
				cast += float64(r.Tally.Cast[sideCompany])
				broken += float64(r.Tally.Broken[sideCompany])
			}
			n := float64(len(results))
			t.Logf("WITCH | L%d | third slot %-7s | wins %d%% | median rounds %d | company HP lost %.0f | casts begun/done/broken %.1f/%.1f/%.1f", level, class, wins, median, lost, started/n, cast/n, broken/n)
		}
	}
}
