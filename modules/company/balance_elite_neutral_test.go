package company

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// Phase 39i: the Halberdier, Samurai, Shaman and Doll Master elites in the
// even mirror. Opt-in (ASHVEIL_BALANCE=1); it reports and asserts nothing.
// ASHVEIL_BALANCE_FIGHTS sets the fights per cell and ASHVEIL_BALANCE_ONLY
// limits the run to cells whose label contains it.
//
// The company's third and fourth members (Garrick and Ysolde) are of the
// lineage: the base class, an advanced route or an elite (L40 and L60; the
// design asks each elite to beat its advanced route by about 10 points).
func TestPhase39iNeutralElites(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to measure the neutral elite routes")
	}
	fights := 40
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && v > 0 {
		fights = v
	}
	only := os.Getenv("ASHVEIL_BALANCE_ONLY")
	// ASHVEIL_BALANCE_SWAP=1 swaps only Garrick: a company of two casters
	// loses nearly every mirror fight, which hides what an elite adds.
	members := []int{3, 4}
	if os.Getenv("ASHVEIL_BALANCE_SWAP") == "1" {
		members = []int{3}
	}
	lineages := map[string][]string{
		"halberdier": {"", "sweeper", "reaper", "vanguard", "linebreaker", "valkyrie", "tempest-lancer"},
		"samurai":    {"", "kensai", "sword-saint", "hatamoto", "shogun", "ronin", "kenshi"},
		"shaman":     {"", "stormcaller", "tempest-lord", "mistweaver", "veil-mother", "earthspeaker", "mountain-speaker"},
		"dollmaster": {"", "puppeteer", "grand-puppeteer", "golemancer", "golem-lord", "marionettist", "string-sovereign"},
	}
	for _, level := range []int{40, 60} {
		for _, lineage := range []string{"halberdier", "samurai", "shaman", "dollmaster"} {
			for _, class := range lineages[lineage] {
				name := class
				if name == "" {
					name = "base"
				}
				label := fmt.Sprintf("%s/L%d/%s", lineage, level, name)
				if only != "" && !strings.Contains(label, only) {
					continue
				}
				var results []balanceResult
				var lostPct float64
				for i := 0; i < fights; i++ {
					t.Run(fmt.Sprintf("%s/%d", label, i), func(t *testing.T) {
						f := newBalanceFightWithOptions(t, level, companyDefault, enemyDefault, balanceFightOptions{
							EnemyCount: 5, Coordination: 1, LegacyMirror: true, Boss: false,
							Classes: classMap(members, lineage),
						})
						if class != "" {
							for _, member := range members {
								f.brawl.companion(member).Character.SetClassState(class, nil)
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
				n := float64(len(results))
				t.Logf("ELITE | %-34s | wins %3d%% | median rounds %2d | HP lost %5.1f%%", label, wins, median, lostPct/n)
			}
		}
	}
}

func classMap(members []int, lineage string) map[int]string {
	out := map[int]string{}
	for _, m := range members {
		out[m] = lineage
	}
	return out
}
