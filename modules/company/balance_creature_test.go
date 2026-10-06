package company

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// Phase 38e: a Hound and a Stone Golem in the third slot against the
// Warrior it replaces (the design: creatures sit near their siblings, and a
// creature company wins within about 10 points of the warrior company).
// Opt-in (ASHVEIL_BALANCE=1); it reports and asserts nothing.
// ASHVEIL_BALANCE_FIGHTS sets the fights per cell and ASHVEIL_BALANCE_ONLY
// limits the run to cells whose label contains it. The numbers recorded in
// docs/PROJECT_STATUS.md came from this run.
//
// Garrick (the third slot) becomes the creature: its archetype, race (canine
// or golem, so its own bites or fist), and its own gear (a hound's collar and
// harness) replace the human's; the company fights a five-foe group of its
// own level through the real round with real dice.
func TestPhase38eCreatures(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to measure the creatures")
	}
	fights := 60
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && v > 0 {
		fights = v
	}
	only := os.Getenv("ASHVEIL_BALANCE_ONLY")
	for _, level := range []int{5, 10, 20} {
		for _, a := range []string{"warrior", "hound", "stone-golem"} {
			label := fmt.Sprintf("L%d/%s", level, a)
			if only != "" && !strings.Contains(label, only) {
				continue
			}
			var results []balanceResult
			var lostPct float64
			for i := 0; i < fights; i++ {
				t.Run(fmt.Sprintf("%s/%d", label, i), func(t *testing.T) {
					f := newBalanceFightWithOptions(t, level, companyDefault, enemyDefault, balanceFightOptions{
						EnemyCount: 5, Coordination: 1, LegacyMirror: true,
						Classes: map[int]string{3: a},
					})
					creatureInSlot(t, f, 3, a)
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
			t.Logf("CREATURE | %-18s | wins %3d%% | median rounds %2d | HP lost %5.1f%% | fallen %.2f", label, wins, median, lostPct/n, float64(fallen)/n)
		}
	}
}

// creatureInSlot turns a companion into a creature of species: its race, its
// own gear, and its health at its level.
func creatureInSlot(t *testing.T, f *balanceFight, id int, species string) {
	t.Helper()
	raceID := map[string]int{"hound": 11, "stone-golem": 16}[species]
	if raceID == 0 {
		return
	}
	c := &f.brawl.companion(id).Character
	c.RaceId = raceID
	c.Equipment = characters.Worn{}
	if species == "hound" {
		c.Equipment.Neck = items.New(20026)
		c.Equipment.Body = items.New(20500)
	}
	levelTo(c, c.Level)
	c.RecalculateStats()
	c.Health = c.HealthMax.Value
}
