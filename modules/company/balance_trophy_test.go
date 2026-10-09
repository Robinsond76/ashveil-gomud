package company

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

// Phase 71 review: what a fully enchanted company does to a fight. Every
// member (the leader and four companions) wears enchants up to the per
// wearer limit on every combat effect the shipped trophies give (+1 damage,
// +2 Attack, +2 Evasion, +1% damage reduction since the review; it was
// +2/+6/+6/+5), against the same company with none. ASHVEIL_TROPHY_FX
// ("attack:2,evasion:2") measures other values, one trophy each, and
// ASHVEIL_TROPHY_FOE_LEVELS puts the foes that many levels lower, to price
// an enchant in levels. Empty armor slots get a plain piece with no stats, so only
// the enchants differ. Opt-in (ASHVEIL_BALANCE=1); ASHVEIL_BALANCE_FIGHTS sets
// the fights per variant, ASHVEIL_BALANCE_ONLY limits the cells by label.

const (
	trophyBalancePlain = 99760 // + slot index: a plain piece for each slot
	trophyBalanceFirst = 99780 // four test trophies, one per effect
)

var trophyBalanceEffects = []map[string]int{
	{classes.Damage: 1}, {classes.Attack: 4}, {classes.Evasion: 4}, {classes.Armor: 2},
}

// trophyBalancePlan is which test trophy goes in each worn slot: two
// hearts, two hollow hearts, two chitin plates and three hides reach the
// per-wearer limit on all four effects.
var trophyBalancePlan = []int{0, 0, 1, 1, 2, 2, 3, 3, 3}

func setTrophyBalanceSpecs(t *testing.T) {
	effects := trophyBalanceEffects
	if v := os.Getenv("ASHVEIL_TROPHY_FX"); v != "" {
		// e.g. "damage:1,attack:3": one test trophy per effect, at that value
		effects = nil
		for _, kv := range strings.Split(v, ",") {
			k, n, _ := strings.Cut(kv, ":")
			val, _ := strconv.Atoi(n)
			effects = append(effects, map[string]int{k: val})
		}
		trophyBalancePlan = trophyBalancePlan[:0]
		for i := range effects {
			trophyBalancePlan = append(trophyBalancePlan, i)
		}
	}
	for i, fx := range effects {
		id := trophyBalanceFirst + i
		items.SetTestItemSpec(&items.ItemSpec{ItemId: id, Name: fmt.Sprintf("test trophy %d", i), Type: items.Commodity,
			Trophy: &items.TrophySpec{Part: items.TrophyHeart, Races: []string{"nobody"}, Chance: 1, Effects: fx}})
		t.Cleanup(func() { items.RemoveTestItemSpec(id) })
	}
	for i, slot := range characters.AllSlots() {
		id := trophyBalancePlain + i
		items.SetTestItemSpec(&items.ItemSpec{ItemId: id, Name: "plain " + string(slot), Type: slot, Subtype: items.Wearable})
		t.Cleanup(func() { items.RemoveTestItemSpec(id) })
	}
}

// enchantFully fills the character's empty slots with plain pieces and
// enchants them to the plan.
func enchantFully(c *characters.Character) {
	n := 0
	for i, slot := range characters.AllSlots() {
		if slot == items.Pack || n >= len(trophyBalancePlan) {
			continue
		}
		it := c.Equipment.Get(slot)
		if it == nil {
			continue
		}
		if it.ItemId < 1 {
			if slot == items.Weapon || slot == items.Offhand {
				continue // a plain "weapon" with no damage would replace fists
			}
			*it = items.New(trophyBalancePlain + i)
		}
		it.Trophy = trophyBalanceFirst + trophyBalancePlan[n]
		n++
	}
	c.RecalculateStats()
}

func TestBalanceTrophyEnchants(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to measure trophy enchants")
	}
	fights := balanceFights()
	only := os.Getenv("ASHVEIL_BALANCE_ONLY")
	for _, level := range []int{5, 15} {
		for _, foes := range []int{3, 5} {
			label := fmt.Sprintf("trophy/L%d/%dfoes", level, foes)
			if only != "" && !strings.Contains(label, only) {
				continue
			}
			var plain, with []balanceResult
			for i := 0; i < fights; i++ {
				for _, enchanted := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%v/%d", label, enchanted, i), func(t *testing.T) {
						f := newBalanceFightWithOptions(t, level, companyDefault, enemyDefault, balanceFightOptions{
							EnemyCount: foes, EnemyLevels: foeLevels(level), Coordination: 1, LegacyMirror: true,
							Setup: func(b *brawl) {
								setTrophyBalanceSpecs(t) // after the brawl world loads its items
								if !enchanted {
									return
								}
								enchantFully(b.aria.Character)
								for id := 1; id <= 4; id++ {
									enchantFully(&b.companion(id).Character)
								}
								if os.Getenv("ASHVEIL_TROPHY_FX") == "" {
									fx, _ := b.companion(1).Character.WornGear()
									assert.Equal(t, map[string]int{classes.Damage: 1, classes.Attack: 2, classes.Evasion: 2, classes.Armor: 1}, fx, "each effect at a wearer's limit")
								}
							}})
						r := f.run()
						if enchanted {
							with = append(with, r)
						} else {
							plain = append(plain, r)
						}
					})
				}
			}
			if len(plain) == 0 {
				continue
			}
			_, plainWins := balanceMedianAndWins(plain)
			_, withWins := balanceMedianAndWins(with)
			plainLost, withLost := balanceHPLost(plain, sideCompany), balanceHPLost(with, sideCompany)
			t.Logf("TROPHY | %-18s | wins %3d%% -> %3d%% (z %.1f) | company HP lost %.0f -> %.0f (z %.1f) | rounds %.1f -> %.1f",
				label, plainWins, withWins, balanceWelchZ(balanceWins(with), balanceWins(plain)),
				balanceMean(plainLost), balanceMean(withLost), balanceWelchZ(withLost, plainLost),
				balanceMean(roundsOf(plain)), balanceMean(roundsOf(with)))
		}
	}
}

// foeLevels is ASHVEIL_TROPHY_FOE_LEVELS levels below the company (to price
// an enchant in levels), or the company's own.
func foeLevels(level int) []int {
	if n, err := strconv.Atoi(os.Getenv("ASHVEIL_TROPHY_FOE_LEVELS")); err == nil {
		return []int{level - n}
	}
	return nil
}
