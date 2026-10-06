package company

import (
	"fmt"
	"os"
	"strconv"
	"testing"
)

// Phase 39a: the Halberdier against the Warrior in the even mirror. Opt-in
// (ASHVEIL_BALANCE=1); it reports and asserts nothing. The design (§13) asks
// the Halberdier to land within 5 points of the Warrior's win rate: it should
// beat a warrior against crowded rows and trail it against a lone foe.
func TestPhase39aHalberdierAgainstWarrior(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to measure the Halberdier")
	}
	fights := 60
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && v > 0 {
		fights = v
	}
	for _, level := range []int{5, 10, 20} {
		for _, foes := range []int{2, 3, 5} {
			for _, class := range []string{"warrior", "halberdier"} {
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
				t.Logf("HALBERDIER | L%d | %d foes | third slot %-10s | wins %d%% | median rounds %d | company HP lost %.0f", level, foes, class, wins, median, lost)
			}
		}
	}
}
