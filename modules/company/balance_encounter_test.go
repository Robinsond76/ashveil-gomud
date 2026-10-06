package company

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBalanceEncounterShapes (Phase 37) measures the shapes random
// encounters actually spawn, in the pilot zones' bands (Dark Forest 5-7,
// Catacombs 10-12): 2-3 foes from the band's low end to one under its top,
// four foes at the band's low, a group with a healer, a boss with two or
// three escorts, and (where the band allows) a company five levels under.
// ASHVEIL_BALANCE=1 runs it; ASHVEIL_BALANCE_FIGHTS sets fights per cell
// (default 40). It asserts the floors that held when it was tuned.
func TestBalanceEncounterShapes(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to measure encounter shapes")
	}
	fights := 40
	if n, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && n > 0 {
		fights = n
	}
	type cell struct {
		shape string
		level int
		cm    string
		em    string
		opts  balanceFightOptions
	}
	for _, band := range [][2]int{{5, 7}, {10, 12}} {
		low, high := band[0], band[1]
		middle := (low + high) / 2
		var cells []cell
		for _, level := range []int{low, middle} {
			for _, count := range []int{2, 3} {
				for enemy := low; enemy <= high-1; enemy++ {
					cells = append(cells, cell{fmt.Sprintf("ordinary L%d", level), level, companyDefault, enemyDefault, balanceFightOptions{EnemyCount: count, EnemyLevels: []int{enemy}}})
				}
			}
			cells = append(cells, cell{fmt.Sprintf("four L%d", level), level, companyDefault, enemyDefault, balanceFightOptions{EnemyCount: 4, EnemyLevels: []int{low}}})
			// A healer group is always three foes: the mirror's priest heals.
			for enemy := low; enemy <= high-1; enemy++ {
				cells = append(cells, cell{fmt.Sprintf("healer L%d", level), level, companyDefault, enemyHealer, balanceFightOptions{EnemyCount: 3, EnemyLevels: []int{enemy}}})
			}
		}
		for _, count := range []int{3, 4} {
			cells = append(cells,
				cell{fmt.Sprintf("boss+%d escorts L%d", count-1, high), high, companyDefault, enemyDefault, balanceFightOptions{EnemyCount: count, EnemyLevels: []int{low}, Boss: true, Coordination: 1}},
				cell{fmt.Sprintf("boss+%d escorts L%d tactics", count-1, high), high, companyTactics, enemyDefault, balanceFightOptions{EnemyCount: count, EnemyLevels: []int{low}, Boss: true, Coordination: 1}})
		}
		if under := low - 5; under >= 1 {
			for _, count := range []int{2, 3} {
				for enemy := low; enemy <= high-1; enemy++ {
					cells = append(cells, cell{fmt.Sprintf("under L%d", under), under, companyDefault, enemyDefault, balanceFightOptions{EnemyCount: count, EnemyLevels: []int{enemy}}})
				}
			}
		}

		rows := map[string][]balanceResult{}
		lost := map[string]float64{}
		order := []string{}
		for _, c := range cells {
			label := fmt.Sprintf("band%d-%d/%s/%dvL%d/%s", low, high, c.shape, c.opts.EnemyCount, c.opts.EnemyLevels[0], c.em)
			var results []balanceResult
			var hpLost float64
			for i := 0; i < fights; i++ {
				t.Run(fmt.Sprintf("%s/%d", label, i), func(t *testing.T) {
					f := newBalanceFightWithOptions(t, c.level, c.cm, c.em, c.opts)
					start := f.healthRemaining()[sideCompany]
					res := f.run()
					results = append(results, res)
					hpLost += 100 * float64(res.HPRemoved[sideCompany]) / float64(start)
				})
			}
			if len(results) == 0 {
				continue
			}
			if _, seen := rows[c.shape]; !seen {
				order = append(order, c.shape)
			}
			rows[c.shape] = append(rows[c.shape], results...)
			lost[c.shape] += hpLost
		}
		for _, shape := range order {
			results := rows[shape]
			median, wins := balanceMedianAndWins(results)
			clean, fallen := 0, 0
			for _, r := range results {
				fallen += r.Fallen[sideCompany]
				if r.Fallen[sideCompany] == 0 {
					clean++
				}
			}
			n := float64(len(results))
			t.Logf("ENCROW | %d-%d | %s | %d | %d%% | %.0f%% | %.2f | %.1f%% | %d | %ds |", low, high, shape, len(results), wins, 100*float64(clean)/n, float64(fallen)/n, lost[shape]/n, median, median*balanceSecondsPerRound())
			name := fmt.Sprintf("band %d-%d %s", low, high, shape)
			switch {
			case len(shape) > 8 && shape[:8] == "ordinary":
				assert.GreaterOrEqual(t, wins, 85, name)
			case len(shape) > 4 && shape[:4] == "four":
				assert.GreaterOrEqual(t, wins, 85, name)
			case len(shape) > 6 && shape[:6] == "healer":
				assert.GreaterOrEqual(t, wins, 60, name)
			}
		}
	}
}
