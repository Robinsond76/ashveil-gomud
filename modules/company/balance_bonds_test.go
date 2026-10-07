package company

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// Phase 65: what bonds do to a fight. Opt-in (ASHVEIL_BALANCE=1); it reports
// and asserts nothing. The company fights the even mirror with every pair
// bonded alike: none (the baseline), friends (+40: one bond step each for a
// friend at half health), kin (+90: two) and rivals (-60: no guard for
// each other). ASHVEIL_BALANCE_FIGHTS sets the fights per cell,
// ASHVEIL_BALANCE_ONLY limits the cells by label, ASHVEIL_BALANCE_FOES the
// enemy count (3 by default: a fair zone).
func TestPhase65BondsInTheMirror(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to measure bonds")
	}
	fights := 60
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && v > 0 {
		fights = v
	}
	foes := 3
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FOES")); err == nil && v >= 2 && v <= 5 {
		foes = v
	}
	only := os.Getenv("ASHVEIL_BALANCE_ONLY")
	for _, level := range []int{5, 15} {
		for _, cell := range []struct {
			name  string
			value int
		}{{"none", 0}, {"friends", 40}, {"kin", 90}, {"rivals", -60}} {
			label := fmt.Sprintf("bonds/L%d/%s/%dfoes", level, cell.name, foes)
			if only != "" && !strings.Contains(label, only) {
				continue
			}
			var results []balanceResult
			var lostPct float64
			for i := 0; i < fights; i++ {
				t.Run(fmt.Sprintf("%s/%d", label, i), func(t *testing.T) {
					f := newBalanceFightWithOptions(t, level, companyDefault, enemyDefault, balanceFightOptions{
						EnemyCount: foes, Coordination: 1, LegacyMirror: true,
					})
					if cell.value != 0 {
						for a := 1; a <= 3; a++ {
							for b := a + 1; b <= 4; b++ {
								setBond(t, a, b, cell.value)
							}
						}
					}
					start := f.healthRemaining()[sideCompany]
					r := f.run()
					results = append(results, r)
					lostPct += 100 * float64(r.HPRemoved[sideCompany]) / float64(start)
				})
			}
			if len(results) == 0 {
				continue
			}
			median, wins := balanceMedianAndWins(results)
			fallen := 0
			for _, r := range results {
				fallen += r.Fallen[sideCompany]
			}
			n := float64(len(results))
			t.Logf("BONDS | %-30s | wins %3d%% | median rounds %2d | HP lost %5.1f%% | fallen %.2f", label, wins, median, lostPct/n, float64(fallen)/n)
		}
	}
}
