package combat

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/stance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 69: weapon stances reach the strike, the tempo, the block and the
// expected-damage figure the company's assessment reads.

const (
	stanceGlaive  = 10151 // militia glaive: two-handed, slashing
	stanceBow     = 10172 // longbow
	stanceDagger  = 10004 // dagger
	stanceSword   = 10001 // a one-handed sword (not a great weapon)
	stanceTower   = 20048 // tower shield
	stanceBuckler = 20046
)

func stanceSetup(t *testing.T) {
	t.Helper()
	loadTestData(t)
	races.LoadDataFiles()
}

// stanceFighter is an even-stat level-10 fighter holding gear, in a stance.
func stanceFighter(weapon, offhand int, st stance.Stance) *characters.Character {
	c := baselineCharacter()
	c.RaceId = 1
	c.Health, c.HealthMax.Value = 1000000, 1000000
	c.Equipment.Weapon, c.Equipment.Offhand = items.Item{}, items.Item{}
	if weapon > 0 {
		c.Equipment.Weapon = items.New(weapon)
	}
	if offhand > 0 {
		c.Equipment.Offhand = items.New(offhand)
	}
	c.RecalculateStats()
	if st != stance.None {
		c.RTState().Stance = st
	}
	return &c
}

// stanceRun is the mean damage per turn, hit rate and crit rate of n blows.
func stanceRun(src, target *characters.Character, n int) (dmg, hit, crit float64) {
	var total, hits, crits int
	for i := 0; i < n; i++ {
		s, tg := *src, *target
		s.SetAggro(0, 1, characters.DefaultAttack)
		res := calculateCombat(s, tg, User, Mob, 0, 0)
		total += res.DamageToTarget
		if res.Hit {
			hits++
		}
		if res.Crit {
			crits++
		}
	}
	f := float64(n)
	return float64(total) / f, float64(hits) / f, float64(crits) / f
}

func stanceTarget() *characters.Character {
	c := stanceFighter(0, 0, stance.None)
	c.SetAggro(0, 1, characters.DefaultAttack)
	return c
}

func TestStanceNeedsItsWeapon(t *testing.T) {
	stanceSetup(t)
	cases := []struct {
		name            string
		weapon, offhand int
		st              stance.Stance
		active          bool
	}{
		{"heavy with a glaive", stanceGlaive, 0, stance.Heavy, true},
		{"heavy with a one-handed sword", stanceSword, 0, stance.Heavy, false},
		{"heavy with a bow", stanceBow, 0, stance.Heavy, false},
		{"keen with a dagger", stanceDagger, 0, stance.Keen, true},
		{"keen with a glaive", stanceGlaive, 0, stance.Keen, false},
		{"quick with a bow", stanceBow, 0, stance.Quick, true},
		{"quick with a dagger", stanceDagger, 0, stance.Quick, false},
		{"wall with a shield", stanceSword, stanceTower, stance.Wall, true},
		{"wall with nothing in the offhand", stanceSword, 0, stance.Wall, false},
		{"none", stanceSword, stanceTower, stance.None, false},
	}
	for _, tc := range cases {
		c := stanceFighter(tc.weapon, tc.offhand, tc.st)
		assert.Equal(t, tc.active, !c.StanceEffect().IsZero(), tc.name)
		if tc.active {
			assert.Equal(t, tc.st, c.Stance(), tc.name)
		} else {
			assert.Equal(t, stance.None, c.Stance(), tc.name)
		}
	}
	// Put the weapon away mid-battle and the stance goes quiet.
	c := stanceFighter(stanceGlaive, 0, stance.Heavy)
	require.False(t, c.StanceEffect().IsZero())
	c.Equipment.Weapon = items.Item{}
	assert.True(t, c.StanceEffect().IsZero())
}

func TestHeavyBlowsTradeAccuracyForDamage(t *testing.T) {
	stanceSetup(t)
	target := stanceTarget()
	const n = 12000
	baseDmg, baseHit, _ := stanceRun(stanceFighter(stanceGlaive, 0, stance.None), target, n)
	heavyDmg, heavyHit, _ := stanceRun(stanceFighter(stanceGlaive, 0, stance.Heavy), target, n)
	assert.InDelta(t, baseHit-0.15, heavyHit, 0.03, "15 points less likely to hit")
	// A landed blow is a quarter harder, so the mean per hit rises by about that.
	assert.InDelta(t, 1.30, (heavyDmg/heavyHit)/(baseDmg/baseHit), 0.08, "blows land 30% harder")
	assert.Less(t, heavyDmg/baseDmg, 1.12, "a sidegrade, not a flat gain")
}

func TestKeenEdgeTradesDepthForCrits(t *testing.T) {
	stanceSetup(t)
	target := stanceTarget()
	const n = 20000
	baseDmg, baseHit, baseCrit := stanceRun(stanceFighter(stanceDagger, 0, stance.None), target, n)
	keenDmg, keenHit, keenCrit := stanceRun(stanceFighter(stanceDagger, 0, stance.Keen), target, n)
	assert.InDelta(t, baseHit, keenHit, 0.03, "accuracy is untouched")
	assert.InDelta(t, 0.10, (keenCrit/keenHit)-(baseCrit/baseHit), 0.04, "10 points more critical hits")
	assert.Less(t, keenDmg/baseDmg, 1.1)
}

func TestQuickDrawAndShieldWallMoveTempo(t *testing.T) {
	stanceSetup(t)
	bow, bowQuick := stanceFighter(stanceBow, 0, stance.None), stanceFighter(stanceBow, 0, stance.Quick)
	assert.InDelta(t, 1.25, Tempo(bowQuick)/Tempo(bow), 0.01, "25% more turns")
	sword, wall := stanceFighter(stanceSword, stanceTower, stance.None), stanceFighter(stanceSword, stanceTower, stance.Wall)
	assert.InDelta(t, 0.70, Tempo(wall)/Tempo(sword), 0.01, "30% fewer turns")
	// Without the gear the stance moves nothing.
	idle := stanceFighter(stanceSword, 0, stance.Quick)
	assert.Equal(t, Tempo(stanceFighter(stanceSword, 0, stance.None)), Tempo(idle))
}

func TestShieldWallRaisesTheBlock(t *testing.T) {
	stanceSetup(t)
	atk := stanceTarget()
	base, wall := stanceFighter(stanceSword, stanceTower, stance.None), stanceFighter(stanceSword, stanceTower, stance.Wall)
	assert.Equal(t, blockChance(base, atk)+12, blockChance(wall, atk))
	assert.Equal(t, blockChance(stanceFighter(stanceSword, 0, stance.None), atk), blockChance(stanceFighter(stanceSword, 0, stance.Wall), atk), "no shield, no change")
}

// The strike's breakdown names the stance, so `why` says what it did.
func TestStanceIsNamedInTheStrikeBreakdown(t *testing.T) {
	stanceSetup(t)
	src := stanceFighter(stanceGlaive, 0, stance.Heavy)
	target := stanceTarget()
	var noted, modified bool
	for i := 0; i < 400 && !(noted && modified); i++ {
		s := *src
		s.SetAggro(0, 1, characters.DefaultAttack)
		res := calculateCombat(s, *target, User, Mob, 0, 0)
		for _, st := range res.Strikes {
			if st.Modifier <= -15 {
				modified = true
			}
			for _, n := range st.Notes {
				if strings.Contains(n, "Heavy blows stance changed the blow by +") {
					noted = true
				}
			}
		}
	}
	assert.True(t, noted, "a landed blow names the stance and its change")
	assert.True(t, modified, "the strike's hit modifier carries the penalty")
}

// The company's expected-damage figure reads the stance too.
func TestExpectedDamageReadsTheStance(t *testing.T) {
	stanceSetup(t)
	target := stanceTarget()
	base := ExpectedDamage(stanceFighter(stanceBow, 0, stance.None), target)
	quick := ExpectedDamage(stanceFighter(stanceBow, 0, stance.Quick), target)
	assert.InDelta(t, 1.25*0.85, quick/base, 0.01)
	idle := ExpectedDamage(stanceFighter(stanceSword, 0, stance.Quick), target)
	assert.Equal(t, ExpectedDamage(stanceFighter(stanceSword, 0, stance.None), target), idle)
}

// TestStancesAreSidegrades is the balance cell (Phase 69 acceptance): at
// even stats and across armored and unarmored foes, no stance beats going
// without by more than a few percent on what it trades for, and none is
// useless. Expected damage per round is the engine's own figure
// (ExpectedDamage) so the check is exact, not sampled.
func TestStancesAreSidegrades(t *testing.T) {
	stanceSetup(t)
	cfg := configs.GetCombatConfig()
	require.NotZero(t, cfg.ToHitEven)
	armored := stanceFighter(0, 0, stance.None)
	armored.Equipment.Body = items.New(20038)
	armored.RecalculateStats()
	armored.SetAggro(0, 1, characters.DefaultAttack)
	foes := map[string]*characters.Character{"bare": stanceTarget(), "armored": armored}

	for foeName, foe := range foes {
		for _, tc := range []struct {
			name            string
			weapon, offhand int
			st              stance.Stance
		}{
			{"heavy glaive", stanceGlaive, 0, stance.Heavy},
			{"keen dagger", stanceDagger, 0, stance.Keen},
			{"quick bow", stanceBow, 0, stance.Quick},
		} {
			base := ExpectedDamage(stanceFighter(tc.weapon, tc.offhand, stance.None), foe)
			with := ExpectedDamage(stanceFighter(tc.weapon, tc.offhand, tc.st), foe)
			ratio := with / base
			t.Logf("%s vs %s foe: %.2f -> %.2f damage a round (%.3f)", tc.name, foeName, base, with, ratio)
			assert.Greater(t, ratio, 0.88, "%s vs %s is not a trap", tc.name, foeName)
			assert.Less(t, ratio, 1.09, "%s vs %s is not a free win", tc.name, foeName)
		}
	}

	// The shield wall trades its wearer's offence for the blows it turns:
	// its damage dealt falls by the tempo, and the damage it takes falls by
	// the extra blocks. The exchange stays close to even.
	attacker := stanceTarget()
	base := stanceFighter(stanceSword, stanceTower, stance.None)
	wall := stanceFighter(stanceSword, stanceTower, stance.Wall)
	dealt := ExpectedDamage(wall, attacker) / ExpectedDamage(base, attacker)
	taken := ExpectedDamage(attacker, wall) / ExpectedDamage(attacker, base)
	t.Logf("shield wall: deals %.3f, takes %.3f of the usual", dealt, taken)
	assert.InDelta(t, 0.70, dealt, 0.02)
	assert.Less(t, taken, 0.9, "it turns blows")
	assert.Greater(t, taken, 0.55, "but not into a wall nothing passes")
	assert.Less(t, dealt/taken, 1.1, "the exchange is not a free win")
	assert.Greater(t, dealt/taken, 0.85)
}
