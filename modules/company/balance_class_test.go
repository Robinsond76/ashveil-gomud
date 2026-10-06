package company

import (
	"fmt"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/require"
	"os"
	"strconv"
	"strings"
	"testing"
)

// Phase 38b (38c1 adds the Warlord and Elder Druid cells): the faith routes in the even mirror. Opt-in (ASHVEIL_BALANCE=1);
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
			routes = []string{"", "mercenary", "paladin", "warlord", "dread-knight"}
		}
		for _, class := range routes {
			cells = append(cells, cell{"healer", level, 1, class, false})
		}
	}
	for _, level := range []int{35, 45} {
		for _, class := range []string{"", "priest", "hierarch", "druid", "elder-druid", "blood-priest", "demonologist"} {
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

// Phase 38c2: the rogue and ranger routes in the even mirror, opt-in as
// above. The rogue (Tamsin) or the ranger (Ysolde) is the base archetype,
// its advanced class, or its elite class (L25 advanced, L35 and L50 elite)
// against three foes of the company's level (a fair zone). The elite should beat its
// advanced class, and the three elites of a lineage stay within 5 points of
// one another.
func TestPhase38c2EliteRoutes(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to measure the rogue and ranger routes")
	}
	fights := 40
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && v > 0 {
		fights = v
	}
	only := os.Getenv("ASHVEIL_BALANCE_ONLY")
	enemies, lead := 3, 0 // ASHVEIL_BALANCE_FOES and ASHVEIL_BALANCE_LEAD make the fight harder
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FOES")); err == nil && v >= 2 && v <= 5 {
		enemies = v
	}
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_LEAD")); err == nil {
		lead = v
	}
	type cell struct {
		lineage string
		member  int
		level   int
		class   string
	}
	var cells []cell
	for _, level := range []int{25, 35, 50} {
		for _, class := range []string{"", "scout", "pathfinder", "duelist", "swordmaster", "assassin", "nightblade"} {
			cells = append(cells, cell{"rogue", 1, level, class})
		}
		for _, class := range []string{"", "warden", "sentinel", "hunter", "marksman", "stalker", "ravager"} {
			cells = append(cells, cell{"ranger", 4, level, class})
		}
	}
	for _, c := range cells {
		if c.level < 30 && strings.Contains("pathfinder swordmaster nightblade sentinel marksman ravager", c.class) && c.class != "" {
			continue
		}
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
					EnemyCount: enemies, EnemyLevels: []int{c.level + lead}, Classes: map[int]string{c.member: c.lineage},
				})
				if c.class != "" {
					f.brawl.companion(c.member).Character.SetClassState(c.class, nil)
				}
				// The shipped line: warriors in front, the healer in the middle,
				// the rogue or ranger where its lineage stands.
				for key, at := range map[domain.MemberKey][2]int{
					domain.CompanionMemberKey(1): {0, 0}, domain.LeaderMemberKey: {0, 1}, domain.CompanionMemberKey(3): {0, 2},
					domain.CompanionMemberKey(2): {1, 1}, domain.CompanionMemberKey(4): {2, 1},
				} {
					require.NoError(t, module.registry.PlaceMember(7, key, at[0], at[1]))
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
		t.Logf("ELITE | %-28s | wins %3d%% | median rounds %2d | HP lost %5.1f%% | fallen %.2f", label, wins, median, lostPct/n, float64(fallen)/n)
	}
}
