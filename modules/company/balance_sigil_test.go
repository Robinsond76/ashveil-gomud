package company

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/sigils"
)

// Phase 54: each sigil against no sigil. Opt-in (ASHVEIL_BALANCE=1); it
// reports and asserts nothing. ASHVEIL_BALANCE_FIGHTS sets the fights per
// cell and ASHVEIL_BALANCE_ONLY limits the run to cells whose label contains
// it. The numbers recorded in docs/PROJECT_STATUS.md came from this run.
//
// The company swaps Garrick (the third slot) for a Wizard, so the fire
// sigil has Shower of Sparks to strengthen, and meets five foes whose hedge
// priest heals (enemyHealer), so stillness has a chant to slow. The "+3"
// cells put the foes three levels above the company: no sigil may make a
// zone above the company's level easy.
func TestPhase54Sigils(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to measure the sigils")
	}
	fights := 40
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && v > 0 {
		fights = v
	}
	only := os.Getenv("ASHVEIL_BALANCE_ONLY")
	kinds := append([]sigils.Kind{sigils.None}, sigils.Kinds...)
	for _, level := range []int{5, 12, 20} {
		for _, above := range []int{0, 3} {
			for _, k := range kinds {
				name := string(k)
				if k == sigils.None {
					name = "none"
				}
				label := fmt.Sprintf("L%d+%d/%s", level, above, name)
				if only != "" && !strings.Contains(label, only) {
					continue
				}
				var results []balanceResult
				var lostPct float64
				for i := 0; i < fights; i++ {
					t.Run(fmt.Sprintf("%s/%d", label, i), func(t *testing.T) {
						f := newBalanceFightWithOptions(t, level, companyDefault, enemyHealer, balanceFightOptions{
							EnemyCount: 5, EnemyLevels: []int{level + above}, Coordination: 1, LegacyMirror: true,
							Classes: map[int]string{3: "wizard"}, Sigil: k,
						})
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
				t.Logf("SIGIL | %-18s | wins %3d%% | median rounds %2d | HP lost %5.1f%% | fallen %.2f", label, wins, median, lostPct/n, float64(fallen)/n)
			}
		}
	}
}
