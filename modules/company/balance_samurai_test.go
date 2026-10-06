package company

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// Phase 39b: the Samurai against the Rogue (the design's nearest class) and
// the Warrior in the even mirror, then its three routes against the base
// class. Opt-in (ASHVEIL_BALANCE=1); it reports and asserts nothing.
// ASHVEIL_BALANCE_FIGHTS sets the fights per cell and ASHVEIL_BALANCE_ONLY
// limits the run to cells whose label contains it. The numbers recorded in
// docs/PROJECT_STATUS.md came from this run.
//
// One member (Garrick, the third slot) is swapped for the class and the
// company fights a five-foe group of its own level through the real round
// with real dice. The design asks the Samurai company to win within 5 points
// of the Rogue company.
func TestPhase39bSamurai(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to measure the Samurai")
	}
	fights := 60
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && v > 0 {
		fights = v
	}
	only := os.Getenv("ASHVEIL_BALANCE_ONLY")
	type cell struct {
		level          int
		archetype, cls string
	}
	var cells []cell
	for _, level := range []int{5, 10, 20} {
		for _, a := range []string{"warrior", "rogue", "samurai"} {
			cells = append(cells, cell{level, a, ""})
		}
	}
	for _, level := range []int{15, 25} {
		for _, c := range []string{"", "kensai", "hatamoto", "ronin"} {
			cells = append(cells, cell{level, "samurai", c})
		}
	}
	for _, c := range cells {
		route := c.cls
		if route == "" {
			route = "base"
		}
		label := fmt.Sprintf("L%d/%s/%s", c.level, c.archetype, route)
		if only != "" && !strings.Contains(label, only) {
			continue
		}
		var results []balanceResult
		var lostPct float64
		for i := 0; i < fights; i++ {
			t.Run(fmt.Sprintf("%s/%d", label, i), func(t *testing.T) {
				f := newBalanceFightWithOptions(t, c.level, companyDefault, enemyDefault, balanceFightOptions{
					EnemyCount: 5, Coordination: 1, LegacyMirror: true,
					Classes: map[int]string{3: c.archetype},
				})
				if c.cls != "" {
					f.brawl.companion(3).Character.SetClassState(c.cls, nil)
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
		t.Logf("SAMURAI | %-26s | wins %3d%% | median rounds %2d | HP lost %5.1f%% | fallen %.2f", label, wins, median, lostPct/n, float64(fallen)/n)
	}
}
