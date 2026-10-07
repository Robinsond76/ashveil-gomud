package company

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encounters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// At-level fight tuning (owner, 2026-10-07): a company at the top of a
// zone's band against the zone's own encounters. The mirror cells elsewhere
// stay as class-balance checks; this one asks whether a player has a good
// time: short fights that cost little, and a rhythm of fights and rests.
// Encounters are fixed by the zone, so a smaller company simply has less to
// fight with: the rest rhythm follows company size (see atLevelTargets).
//
// Each fight is measured alone from full health (the harness builds one
// fight at a time), as every member's share of health and mana lost. A
// campaign is then drawn from those fights: seven in eight are ordinary
// groups of two or three, one in eight is four at the band's low, and the
// company rests when a member has gone down, the company has lost
// restCompanyLost of its health, or a caster is under restManaFloor of
// mana. The count of fights before that is the rhythm. A variant also rests
// when the weakest member is under restHealthFloor.
const (
	atLevelFourShare = 8   // one random encounter in this many is four foes
	restHealthFloor  = 20  // percent: the weakest-member variant's floor
	restCompanyLost  = 50  // percent of the company's health
	restManaFloor    = 20  // percent mana left
	atLevelCampaigns = 400 // campaigns drawn per cell
	atLevelMaxFights = 60
)

// atLevelTarget is a company size's designed range of fights before a rest
// (owner, 2026-10-07: five members 15-20, four 8-14, three 5-7, a solo
// player suffers in a zone for their own level; two and one are ours), for a
// martial company and one leaning on magic. Measured medians must land in
// the range widened by atLevelSlack fights each way: the curve is steeper
// at the full company than the harness company's makeup reaches (recorded in
// docs/plans/2026-10-07-at-level-fight-tuning.md).
const atLevelSlack = 5

type atLevelTarget struct{ martialLo, martialHi, magicLo, magicHi int }

var atLevelTargets = map[int]atLevelTarget{
	5: {15, 20, 7, 12},
	4: {8, 14, 5, 8},
	3: {5, 7, 3, 5},
	2: {3, 4, 2, 3},
	1: {1, 2, 1, 2},
}

// fightCost is one fight's cost to each member, as shares (0-1) of what
// they started with. Absent members cost nothing.
type fightCost struct {
	Health [5]float64
	Mana   [5]float64
	Size   int
	Rounds int
	Won    bool
}

// dressLeader gives the leader what a companion fighter wears (the harness
// leader goes bare-handed and bare-backed, which no player at level does).
func dressLeader(c *characters.Character) {
	for slot, id := range map[items.ItemType]int{items.Weapon: 10002, items.Head: 20020, items.Body: 20008, items.Legs: 20029, items.Feet: 20003} {
		c.Equipment.Set(slot, items.New(id))
	}
	c.RecalculateStats()
}

// companionNames are the harness companions by slot, for dismissing.
var companionNames = map[int]string{1: "tamsin", 2: "oswin", 3: "garrick", 4: "ysolde"}

// joinOrder is the order companions join a growing company: a warrior
// first, then the healer, the third fighter (a wizard in the magic
// company), the ranger. A magic company's wizard comes before the rest.
func joinOrder(magic bool) []int {
	if magic {
		return []int{3, 1, 2, 4}
	}
	return []int{1, 2, 3, 4}
}

func measureAtLevelFight(t *testing.T, band encounters.Band, level, size int, magic bool, count int, enemies []int) fightCost {
	classes := map[int]string(nil)
	if magic {
		classes = map[int]string{3: "wizard"}
	}
	keep := map[int]bool{}
	for _, id := range joinOrder(magic)[:size-1] {
		keep[id] = true
	}
	f := newBalanceFightWithOptions(t, level, companyDefault, enemyDefault, balanceFightOptions{
		EnemyCount: count, EnemyLevels: enemies, Classes: classes,
		OrdinaryHPPercent: encounters.HPPercent(level, band),
		PlayerClass:       "warrior",
		Setup: func(b *brawl) {
			dressLeader(b.aria.Character)
			for id := 1; id <= 4; id++ {
				if !keep[id] {
					_, err := module.dismiss(7, companionNames[id])
					require.NoError(t, err, companionNames[id])
				}
			}
			placeAtLevelFormation(t, keep, magic)
		},
	})
	members := []*characters.Character{f.aria.Character}
	slots := []int{0}
	for id := 1; id <= 4; id++ {
		if keep[id] {
			members = append(members, &f.companion(id).Character)
			slots = append(slots, id)
		}
	}
	require.Len(t, members, size)
	startHP, startMana := make([]int, size), make([]int, size)
	for i, c := range members {
		startHP[i], startMana[i] = c.Health, c.Mana
	}
	res := f.run()
	cost := fightCost{Size: size, Rounds: res.Rounds, Won: res.Won}
	for i, c := range members {
		if startHP[i] > 0 {
			cost.Health[slots[i]] = float64(startHP[i]-max(c.Health, 0)) / float64(c.HealthMax.Value)
		}
		if startMana[i] > 0 {
			cost.Mana[slots[i]] = float64(startMana[i]-c.Mana) / float64(c.ManaMax.Value)
		}
	}
	return cost
}

// campaignLength counts the fights drawn from the pools before the company
// would rest (the fight that tips it counts: the company fought it).
func campaignLength(rng *rand.Rand, ordinary, four []fightCost, memberFloor bool) int {
	var hp, mana [5]float64
	for n := 1; n <= atLevelMaxFights; n++ {
		pool := ordinary
		if rng.Intn(atLevelFourShare) == 0 && len(four) > 0 {
			pool = four
		}
		c := pool[rng.Intn(len(pool))]
		total := 0.0
		for i := range hp {
			hp[i] += c.Health[i]
			mana[i] += c.Mana[i]
			total += hp[i]
		}
		if !c.Won {
			return n
		}
		for i := range hp {
			// A member who went down always ends the stretch; with
			// memberFloor, so does one gone below the health floor.
			if hp[i] >= 1 || memberFloor && hp[i]*100 >= 100-restHealthFloor {
				return n
			}
			if mana[i]*100 >= 100-restManaFloor && c.Mana[i] > 0 {
				return n
			}
		}
		if total/float64(c.Size)*100 >= restCompanyLost {
			return n
		}
	}
	return atLevelMaxFights
}

func median(v []int) int {
	s := append([]int(nil), v...)
	sort.Ints(s)
	return s[len(s)/2]
}

func quartiles(v []int) string {
	s := append([]int(nil), v...)
	sort.Ints(s)
	return fmt.Sprintf("%d/%d/%d", s[len(s)/4], s[len(s)/2], s[3*len(s)/4])
}

// TestBalanceAtLevel measures a company of each size at the top of a band
// against the band's ordinary encounters. ASHVEIL_BALANCE=1 runs it;
// ASHVEIL_BALANCE_FIGHTS sets fights per cell (default 40) and
// ASHVEIL_BALANCE_ONLY (a comma list such as "5,4") limits the sizes.
func TestBalanceAtLevel(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to measure at-level fights")
	}
	fights := 40
	if n, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && n > 0 {
		fights = n
	}
	onlySizes := os.Getenv("ASHVEIL_BALANCE_ONLY")
	rng := rand.New(rand.NewSource(1))
	for _, bandRange := range [][2]int{{3, 5}, {10, 12}, {20, 22}} {
		low, high := bandRange[0], bandRange[1]
		band := encounters.Band{Low: low, High: high}
		for _, magic := range []bool{false, true} {
			for size := 5; size >= 1; size-- {
				if magic && size == 1 {
					continue // a lone leader is the same company either way
				}
				if onlySizes != "" && !strings.Contains(","+onlySizes+",", fmt.Sprintf(",%d,", size)) {
					continue
				}
				name := map[bool]string{false: "martial", true: "magic"}[magic]
				var ordinary, four []fightCost
				for i := 0; i < fights; i++ {
					t.Run(fmt.Sprintf("%d-%d/%s/%d/%d", low, high, name, size, i), func(t *testing.T) {
						// Levels as encounters.Plan draws them: low to one under the top.
						level := low + rng.Intn(max(low, high-1)-low+1)
						ordinary = append(ordinary, measureAtLevelFight(t, band, high, size, magic, 2+i%2, []int{level}))
						if i%2 == 0 {
							four = append(four, measureAtLevelFight(t, band, high, size, magic, 4, []int{low}))
						}
					})
				}
				if len(ordinary) == 0 {
					continue
				}
				var rounds, lengths, strict []int
				var lost, won float64
				var slotHP, slotMana [5]float64
				for _, c := range ordinary {
					rounds = append(rounds, c.Rounds)
					for i := range slotHP {
						slotHP[i] += c.Health[i] / float64(len(ordinary))
						slotMana[i] += c.Mana[i] / float64(len(ordinary))
					}
					for _, h := range c.Health {
						lost += h / float64(size)
					}
					if c.Won {
						won++
					}
				}
				for i := 0; i < atLevelCampaigns; i++ {
					lengths = append(lengths, campaignLength(rng, ordinary, four, false))
					strict = append(strict, campaignLength(rng, ordinary, four, true))
				}
				n := float64(len(ordinary))
				t.Logf("ATLEVEL | %d-%d | %s | %d members | wins %.0f%% | rounds %d | HP lost %.1f%% | fights before rest p25/median/p75 %s | weakest member kept above %d%%: %s |",
					low, high, name, size, 100*won/n, median(rounds), 100*lost/n, quartiles(lengths), restHealthFloor, quartiles(strict))
				t.Logf("ATLEVEL | %d-%d | %s | %d members | health lost a fight by slot (leader, tamsin, oswin, garrick, ysolde) %.0f/%.0f/%.0f/%.0f/%.0f%% | mana %.0f/%.0f/%.0f/%.0f/%.0f%% |",
					low, high, name, size, 100*slotHP[0], 100*slotHP[1], 100*slotHP[2], 100*slotHP[3], 100*slotHP[4], 100*slotMana[0], 100*slotMana[1], 100*slotMana[2], 100*slotMana[3], 100*slotMana[4])
				target := atLevelTargets[size]
				lo, hi := target.martialLo, target.martialHi
				if magic {
					lo, hi = target.magicLo, target.magicHi
				}
				got := median(lengths)
				assert.GreaterOrEqual(t, got, lo-atLevelSlack, fmt.Sprintf("band %d-%d %s %d: fights before a rest, target %d-%d", low, high, name, size, lo, hi))
				assert.LessOrEqual(t, got, hi+atLevelSlack-1, fmt.Sprintf("band %d-%d %s %d: fights before a rest, target %d-%d", low, high, name, size, lo, hi))
				if size >= 3 {
					assert.GreaterOrEqual(t, 100*won/n, 97.0, fmt.Sprintf("band %d-%d %s %d: won", low, high, name, size))
					assert.LessOrEqual(t, median(rounds), 7, "fights at the band's top are short")
				}
			}
		}
	}
}

// placeAtLevelFormation stands the company as a player would: fighters in
// the front row, the healer, the ranger and a wizard behind them.
func placeAtLevelFormation(t *testing.T, keep map[int]bool, magic bool) {
	rec, ok := module.registry.Get(7)
	require.True(t, ok)
	front := []domain.MemberKey{domain.LeaderMemberKey}
	var back []domain.MemberKey
	for _, id := range []int{1, 3, 2, 4} {
		if !keep[id] {
			continue
		}
		if id == 1 || id == 3 && !magic {
			front = append(front, domain.CompanionMemberKey(id))
		} else {
			back = append(back, domain.CompanionMemberKey(id))
		}
	}
	var f domain.Formation
	cols := []int{1, 0, 2}
	for i, k := range front {
		f[0][cols[i]] = k
	}
	for i, k := range back {
		f[1][cols[i]] = k
	}
	rec.Formation = f
	module.registry.Put(rec)
}
