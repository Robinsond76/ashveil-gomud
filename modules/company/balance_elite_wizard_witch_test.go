package company

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// Phase 38c3: the wizard and witch routes in the even mirror. Opt-in
// (ASHVEIL_BALANCE=1); it reports and asserts nothing. ASHVEIL_BALANCE_FIGHTS
// sets the fights per cell and ASHVEIL_BALANCE_ONLY limits the run to cells
// whose label contains it. The numbers recorded in docs/PROJECT_STATUS.md
// came from this run.
//
// The company's third and fourth members (Garrick and Ysolde) are wizards or
// witches of the base class, an advanced class or an elite class (L40, L50 in the even mirror; the
// design asks each elite to win 10-20 points more often
// than its advanced route, and its siblings to stay within 5 points).
func TestPhase38c3WizardWitchRoutes(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to measure the wizard and witch routes")
	}
	fights := 60
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && v > 0 {
		fights = v
	}
	only := os.Getenv("ASHVEIL_BALANCE_ONLY")
	type cell struct {
		lineage, class string
		level          int
	}
	var cells []cell
	for _, level := range []int{40, 50} {
		for _, class := range []string{"", "theurgist", "archon", "arcanist", "archmage", "warlock", "necromancer", "sorcerer", "high-sorcerer"} {
			cells = append(cells, cell{"wizard", class, level})
		}
		for _, class := range []string{"", "hedge-witch", "wise-one", "coven-sage", "coven-mother", "hag", "crone-of-ash"} {
			cells = append(cells, cell{"witch", class, level})
		}
	}
	for _, c := range cells {
		class := c.class
		if class == "" {
			class = "base"
		}
		label := fmt.Sprintf("%s/L%d/%s", c.lineage, c.level, class)
		if only != "" && !strings.Contains(label, only) {
			continue
		}
		var results []balanceResult
		var lostPct float64
		for i := 0; i < fights; i++ {
			t.Run(fmt.Sprintf("%s/%d", label, i), func(t *testing.T) {
				f := newBalanceFightWithOptions(t, c.level, companyDefault, enemyDefault, balanceFightOptions{
					EnemyCount: 5, Coordination: 1, LegacyMirror: true, Boss: false,
					Classes: map[int]string{3: c.lineage, 4: c.lineage},
				})
				if c.class != "" {
					for _, member := range []int{3, 4} {
						f.brawl.companion(member).Character.SetClassState(c.class, nil)
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
		t.Logf("CLASS | %-30s | wins %3d%% | median rounds %2d | HP lost %5.1f%% | fallen %.2f", label, wins, median, lostPct/n, float64(fallen)/n)
	}
}
