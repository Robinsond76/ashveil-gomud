package company

import (
	"fmt"
	"os"
	"strconv"
	"testing"
)

// Phase 39d: the Doll Master against the Warrior in the even mirror. Opt-in
// (ASHVEIL_BALANCE=1); it reports and asserts nothing. The design (§13) asks
// the Doll Master (Doll Master vs warrior plus cleric support) to land
// within 5 points of the Warrior's win rate.
func TestPhase39dDollMasterAgainstWarrior(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to measure the Doll Master")
	}
	fights := 60
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && v > 0 {
		fights = v
	}
	for _, level := range []int{5, 10, 20} {
		for _, foes := range []int{2, 3, 5} {
			for _, class := range []string{"warrior", "dollmaster"} {
				var results []balanceResult
				for i := 0; i < fights; i++ {
					t.Run(fmt.Sprintf("L%d-%dfoes-%s-%d", level, foes, class, i), func(t *testing.T) {
						f := newBalanceFightWithOptions(t, level, companyDefault, enemyDefault, balanceFightOptions{
							EnemyCount: foes, Coordination: 1, LegacyMirror: true,
							Classes: map[int]string{3: class},
						})
						results = append(results, f.run())
					})
				}
				median, wins := balanceMedianAndWins(results)
				lost := balanceMean(balanceHPLost(results, sideCompany))
				t.Logf("DOLLMASTER | L%d | %d foes | third slot %-10s | wins %d%% | median rounds %d | company HP lost %.0f", level, foes, class, wins, median, lost)
			}
		}
	}
}
