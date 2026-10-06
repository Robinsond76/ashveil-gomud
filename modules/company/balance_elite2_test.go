package company

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// Phase 39i2: the Beast Tamer, Gryphon Rider, Alchemist and Arbalist elites
// against their advanced class in the even mirror. Opt-in
// (ASHVEIL_BALANCE=1); it reports and asserts nothing.
// ASHVEIL_BALANCE_FIGHTS sets the fights per cell and ASHVEIL_BALANCE_ONLY
// limits the run to cells whose label contains it.
//
// Garrick (the third slot) is swapped for the lineage and the company fights
// a five-foe group of its own level. The design asks each elite to beat its
// advanced route by about 10 points.
func TestPhase39i2Elites(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to measure the remaining neutral elite routes")
	}
	fights := 40
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && v > 0 {
		fights = v
	}
	only := os.Getenv("ASHVEIL_BALANCE_ONLY")
	lineages := map[string][]string{
		"beasttamer":    {"houndmaster", "packlord", "bearward", "beastlord", "dragon-tamer", "dragon-lord"},
		"gryphon-rider": {"gryphon-knight", "gryphon-lord", "skyscout", "falcon-marshal", "wyvern-rider", "wyvern-lord"},
		"alchemist":     {"apothecary", "panacean", "bombardier", "grenadier", "mutagenist", "transmuter"},
		"arbalist":      {"siegebreaker", "siege-master", "sharpshooter", "deadeye", "warden-of-the-wall", "bastion"},
	}
	for _, level := range []int{40, 60} {
		for _, lineage := range []string{"beasttamer", "gryphon-rider", "alchemist", "arbalist"} {
			for _, class := range lineages[lineage] {
				label := fmt.Sprintf("%s/L%d/%s", lineage, level, class)
				if only != "" && !strings.Contains(label, only) {
					continue
				}
				var results []balanceResult
				var lostPct float64
				for i := 0; i < fights; i++ {
					t.Run(fmt.Sprintf("%s/%d", label, i), func(t *testing.T) {
						f := newBalanceFightWithOptions(t, level, companyDefault, enemyDefault, balanceFightOptions{
							EnemyCount: 5, Coordination: 1, LegacyMirror: true, Boss: false,
							Classes: map[int]string{3: lineage},
						})
						if lineage == "arbalist" { // the tier-1 hunting crossbow
							f.brawl.companion(3).Character.Equipment.Weapon = items.New(10181)
						}
						f.brawl.companion(3).Character.SetClassState(class, nil)
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
				n := float64(len(results))
				t.Logf("ELITE2 | %-40s | wins %3d%% | median rounds %2d | HP lost %5.1f%%", label, wins, median, lostPct/n)
			}
		}
	}
}
