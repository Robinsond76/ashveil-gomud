package company

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/require"
)

// Phase 81: the classes no earlier phase tuned, measured one swapped member
// at a time. Opt-in (ASHVEIL_BALANCE=1); it reports and asserts nothing.
// ASHVEIL_BALANCE_FIGHTS sets the fights per cell and ASHVEIL_BALANCE_ONLY
// limits the run to cells whose label contains it. The numbers recorded in
// docs/plans/2026-10-07-phase-81-class-tuning.md came from this run.
//
// One company member (Oswin the cleric, Garrick the wizard) becomes the
// lineage's base class, an advanced class or an elite class, and the company
// fights a five-foe group of its own level in the even mirror. A single
// swapped caster, not two, so the class moves the result instead of the
// caster floor hiding it.
func TestPhase81ClassTuning(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to measure the phase 81 classes")
	}
	fights := 60
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && v > 0 {
		fights = v
	}
	only := os.Getenv("ASHVEIL_BALANCE_ONLY")
	// ASHVEIL_BALANCE_LINE=1 stands the company in the shipped line (the
	// harness default leaves it unplaced, so foes reach casters at will).
	line := os.Getenv("ASHVEIL_BALANCE_LINE") == "1"
	type cell struct {
		lineage, class string
		member, level  int
	}
	var cells []cell
	for _, level := range []int{15, 25, 35, 45} {
		for _, class := range []string{"", "priest", "druid", "blood-priest"} {
			cells = append(cells, cell{"cleric", class, 2, level})
		}
		if level >= 30 {
			for _, class := range []string{"hierarch", "elder-druid", "demonologist"} {
				cells = append(cells, cell{"cleric", class, 2, level})
			}
		}
	}
	for _, level := range []int{15, 25, 40, 50} {
		for _, class := range []string{"", "theurgist", "arcanist", "warlock", "sorcerer"} {
			cells = append(cells, cell{"wizard", class, 3, level})
		}
		if level >= 30 {
			for _, class := range []string{"archon", "archmage", "necromancer", "high-sorcerer"} {
				cells = append(cells, cell{"wizard", class, 3, level})
			}
		}
	}
	for _, c := range cells {
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
					EnemyCount: 5, Coordination: 1, LegacyMirror: true,
					Classes: map[int]string{c.member: c.lineage},
				})
				if c.class != "" {
					f.brawl.companion(c.member).Character.SetClassState(c.class, nil)
				}
				if line {
					for key, at := range map[domain.MemberKey][2]int{
						domain.CompanionMemberKey(1): {0, 0}, domain.LeaderMemberKey: {0, 1}, domain.CompanionMemberKey(4): {1, 2},
						domain.CompanionMemberKey(2): {2, 0}, domain.CompanionMemberKey(3): {2, 1},
					} {
						require.NoError(t, module.registry.PlaceMember(7, key, at[0], at[1]))
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
		fallen := 0
		for _, r := range results {
			fallen += r.Fallen[sideCompany]
		}
		n := float64(len(results))
		t.Logf("PHASE81 | %-30s | wins %3d%% | median rounds %2d | HP lost %5.1f%% | fallen %.2f", label, wins, median, lostPct/n, float64(fallen)/n)
	}
}
