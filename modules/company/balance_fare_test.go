package company

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/survival"
)

// Phase 50: a company's battle condition against a well-kept one. Opt-in
// (ASHVEIL_BALANCE=1); it reports and asserts nothing. ASHVEIL_BALANCE_FIGHTS
// sets the fights per cell and ASHVEIL_BALANCE_ONLY limits the run to cells
// whose label contains it. The numbers recorded in docs/PROJECT_STATUS.md
// came from this run. The company (a Wizard in slot 3, so a meal's mana
// matters) meets five even foes. Every member of the company is in the same condition;
// "fed" is the 37b baseline (needs full, no meal), which the condition code
// must leave exactly as it was.
func TestPhase50BattleCondition(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to measure the battle condition")
	}
	fights := 40
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && v > 0 {
		fights = v
	}
	only := os.Getenv("ASHVEIL_BALANCE_ONLY")
	meal := func(kind string) survival.Needs {
		n := survival.FullNeeds()
		n.Meal, n.MealBattles = kind, 3
		return n
	}
	states := []struct {
		name  string
		needs survival.Needs
	}{
		{"fed", survival.FullNeeds()},
		{"hungry", survival.Needs{Hunger: 40, Thirst: 100, Fatigue: 100}},
		{"thirsty", survival.Needs{Hunger: 100, Thirst: 40, Fatigue: 100}},
		{"tired", survival.Needs{Hunger: 100, Thirst: 100, Fatigue: 40}},
		{"hungry+thirsty+tired", survival.Needs{Hunger: 40, Thirst: 40, Fatigue: 40}},
		{"starving+parched", survival.Needs{Hunger: 10, Thirst: 10, Fatigue: 100}},
		{"starving+parched+exhausted", survival.Needs{Hunger: 10, Thirst: 10, Fatigue: 10}},
		{"strong", meal("seared")},
		{"steady", meal("roast")},
		{"hearty", meal("stew")},
		{"clear-headed", meal("fish")},
	}
	for _, level := range []int{5, 12, 20} {
		for _, st := range states {
			label := fmt.Sprintf("L%d/%s", level, st.name)
			if only != "" && !strings.Contains(label, only) {
				continue
			}
			var results []balanceResult
			var lostPct float64
			for i := 0; i < fights; i++ {
				t.Run(fmt.Sprintf("%s/%d", label, i), func(t *testing.T) {
					needs := map[survival.MemberKey]survival.Needs{survival.LeaderMemberKey: st.needs}
					for id := 1; id <= 4; id++ {
						needs[survival.CompanionMemberKey(id)] = st.needs
					}
					useFareNeeds(t, needs)
					f := newBalanceFightWithOptions(t, level, companyDefault, enemyDefault, balanceFightOptions{
						EnemyCount: 5, EnemyLevels: []int{level}, Coordination: 1, LegacyMirror: true,
						Classes: map[int]string{3: "wizard"},
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
			t.Logf("FARE | %-30s | wins %3d%% | median rounds %2d | HP lost %5.1f%% | fallen %.2f", label, wins, median, lostPct/n, float64(fallen)/n)
		}
	}
}
