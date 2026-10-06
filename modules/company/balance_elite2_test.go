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
	// Review: the even mirror puts the Beast Tamer at the win ceiling and the
	// Alchemist at the floor, where no route can be ranked, so those two
	// meet foes a few levels above or below them (ASHVEIL_BALANCE_LEAD
	// overrides every lineage's lead for calibration).
	leads := map[string]int{"beasttamer": 0, "gryphon-rider": 0, "alchemist": 0, "arbalist": 0}
	override, overridden := os.LookupEnv("ASHVEIL_BALANCE_LEAD")
	if v, err := strconv.Atoi(override); overridden && err == nil {
		for k := range leads {
			leads[k] = v
		}
	}
	// Review: Garrick's broadsword is no lineage's weapon; each swapped
	// member wields a tier-one weapon of its own class.
	weapons := map[string]int{
		"beasttamer":    10131, // iron short spear, 1d6
		"gryphon-rider": 10141, // iron war spear, 1d8+1: a lance, for Lance charge and Thunder landing
		"alchemist":     10021, // ash quarterstaff, 1d6
		"arbalist":      10181, // hunting crossbow, 1d6+2
	}
	for _, level := range []int{40, 60} {
		for _, lineage := range []string{"beasttamer", "gryphon-rider", "alchemist", "arbalist"} {
			for _, class := range lineages[lineage] {
				lead := leads[lineage]
				label := fmt.Sprintf("%s/L%d%+d/%s", lineage, level, lead, class)
				if only != "" && !strings.Contains(label, only) {
					continue
				}
				var results []balanceResult
				var lostPct float64
				for i := 0; i < fights; i++ {
					t.Run(fmt.Sprintf("%s/%d", label, i), func(t *testing.T) {
						f := newBalanceFightWithOptions(t, level, companyDefault, enemyDefault, balanceFightOptions{
							EnemyCount: 5, EnemyLevels: []int{level + lead}, Coordination: 1, LegacyMirror: true, Boss: false,
							Classes: map[int]string{3: lineage},
						})
						f.brawl.companion(3).Character.Equipment.Weapon = items.New(weapons[lineage])
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
