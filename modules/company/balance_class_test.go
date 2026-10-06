package company

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// Phase 38b: the faith routes in the even mirror. Opt-in (ASHVEIL_BALANCE=1);
// it reports and asserts nothing. ASHVEIL_BALANCE_FIGHTS sets the fights per
// cell and ASHVEIL_BALANCE_ONLY limits the run to cells whose label contains
// it. The numbers recorded in docs/PROJECT_STATUS.md came from this run.
//
// Fighting healers: the front warrior (Tamsin) unpromoted, then a Mercenary,
// Knight or Blackguard (L15, L25), then a Mercenary, Paladin or Dread Knight
// (L35, L45), against five foes of the company's level. The design asks the
// Paladin and Blackguard companies to lose 15-30% less health than the
// Mercenary company.
//
// Summoners: the cleric (Oswin) a Priest, Hierarch, Blood Priest or
// Demonologist (L35, L45) against the even mirror with a boss. The design
// asks the summoning company to win 10-20 points more often than its
// advanced route, and the Angel and Demon to stay within 5 points.
func TestPhase38bClassRoutes(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to measure the class routes")
	}
	fights := 60
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && v > 0 {
		fights = v
	}
	only := os.Getenv("ASHVEIL_BALANCE_ONLY")
	type cell struct {
		label  string
		level  int
		member int
		class  string
		boss   bool
	}
	var cells []cell
	for _, level := range []int{15, 25, 35, 45} {
		routes := []string{"", "mercenary", "knight", "blackguard"}
		if level >= 30 {
			routes = []string{"", "mercenary", "paladin", "dread-knight"}
		}
		for _, class := range routes {
			cells = append(cells, cell{"healer", level, 1, class, false})
		}
	}
	for _, level := range []int{35, 45} {
		for _, class := range []string{"", "priest", "hierarch", "blood-priest", "demonologist"} {
			cells = append(cells, cell{"summon", level, 2, class, true})
		}
	}
	for _, c := range cells {
		class := c.class
		if class == "" {
			class = "base"
		}
		label := fmt.Sprintf("%s/L%d/%s", c.label, c.level, class)
		if only != "" && !strings.Contains(label, only) {
			continue
		}
		var results []balanceResult
		var lostPct float64
		for i := 0; i < fights; i++ {
			t.Run(fmt.Sprintf("%s/%d", label, i), func(t *testing.T) {
				f := newBalanceFightWithOptions(t, c.level, companyDefault, enemyDefault, balanceFightOptions{
					EnemyCount: 5, Coordination: 1, LegacyMirror: true, Boss: c.boss,
				})
				if c.class != "" {
					f.brawl.companion(c.member).Character.SetClassState(c.class, nil)
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
		t.Logf("CLASS | %-28s | wins %3d%% | median rounds %2d | HP lost %5.1f%% | fallen %.2f", label, wins, median, lostPct/n, float64(fallen)/n)
	}
}
