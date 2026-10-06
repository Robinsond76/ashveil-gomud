package company

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hexes"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 38c3 wiring: the witch elites (Wise One, Coven Mother, Crone of
// Ash) through the real strategy pass, the shipped hex scripts and the
// combat round. Hexes always land unless a test says otherwise.

// witchElite is a witchBrawl whose class is the given elite.
func witchElite(t *testing.T, class string, level int) *brawl {
	t.Helper()
	b := witchBrawl(t, level)
	classes.SetProvider(&fakeClassStore{state: classes.State{Class: class}})
	t.Cleanup(func() { classes.SetProvider(nil) })
	hooks.ResetEliteForTest()
	t.Cleanup(hooks.ResetEliteForTest)
	return b
}

// firstHeld runs rounds until a bandit carries the status, and returns it.
func (b *brawl) firstHeld(id int, rounds int) *mobs.Mob {
	b.t.Helper()
	for i := 0; i <= rounds; i++ {
		for _, m := range b.livingBandits() {
			if status.Live(&m.Character, id) || id == status.Poisoned && m.Character.HasBuff(id) {
				return m
			}
		}
		b.toughen()
		b.hold(nil)
		b.fight()
	}
	return nil
}

func TestWiseOneHearthwardWardsTheMostHurtAlliesWithTheirGifts(t *testing.T) {
	b := witchElite(t, "wise-one", 60)
	b.witchHexes("slumber")
	b.startWitchFight()
	tamsin, oswin, garrick := &b.companion(1).Character, &b.companion(2).Character, &b.companion(3).Character
	b.toughen()
	tamsin.Health, oswin.Health, garrick.Health = 100, 200, 300
	for i := 0; i < 3 && warded(b) < 3; i++ {
		b.hold(nil)
		b.fight()
	}
	assert.GreaterOrEqual(t, warded(b), 3, "a landed hex wards the three most hurt allies")
	for _, c := range []*characters.Character{tamsin, oswin, garrick} {
		require.NotNil(t, c.RT)
		assert.Positive(t, c.RT.Ward)
		assert.Positive(t, c.RT.WardMend, "Mend Charm")
		assert.True(t, c.RT.WardCleanse, "Cleansing ward")
		assert.Equal(t, 5, c.RT.WardPeace, "Hearth's Peace")
		assert.NotNil(t, c.RT.WardLifeBy, "Ward of Life")
	}
	// A ward holder has the Evasion for the round.
	b.hold(nil)
	b.fight()
	assert.GreaterOrEqual(t, tamsin.Aura.Evasion, 5)
}

func TestWiseOneDeepSlumberLastsLonger(t *testing.T) {
	initial := map[int]int{}
	for _, level := range []int{30, 35} {
		t.Run(fmt.Sprintf("level %d", level), func(t *testing.T) {
			b := witchElite(t, "wise-one", level)
			b.witchHexes("slumber")
			b.startWitchFight()
			m := b.firstHeld(status.Asleep, 3)
			require.NotNil(t, m)
			initial[level] = m.Character.GetBuffs(status.Asleep)[0].TriggersInitial
		})
	}
	assert.Greater(t, initial[35], initial[30], "Deep Slumber at 35")
}

func TestAWardBreakingHealsCleansesAndTheLastWardHoldsOneHealth(t *testing.T) {
	b := witchElite(t, "wise-one", 60)
	b.witchHexes()
	b.aria.Character.Mana = 0
	b.cmd("strategy", "oswin fighter") // no healer to mend her before the blow
	b.startWitchFight()
	forceBlows(t, true)
	tamsin := &b.companion(1).Character
	b.toughen()
	b.hold(nil)
	tamsin.Health = 400
	tamsin.RT = &characters.ClassRT{Ward: 1, WardCap: 1, WardMend: 20, WardCleanse: true, WardPeace: 5}
	tamsin.RT.WardLifeBy = b.aria.Character.RTState()
	b.strike(1, false)
	out := b.fight()
	assert.Contains(t, out, "mend charm")
	assert.Equal(t, characters.ClassRT{}.WardMend, tamsin.RT.WardMend, "the gifts end with the ward")
	assert.Zero(t, tamsin.RT.Ward)

	// A fatal blow on a warded ally leaves it at 1 health, once.
	b.toughen()
	b.hold(nil)
	tamsin.Health = 1
	tamsin.RT.Ward, tamsin.RT.WardCap = 1, 0 // a ward that soaks nothing: any blow would fell her
	tamsin.RT.WardLifeBy = b.aria.Character.RTState()
	b.strike(1, false)
	out = b.fight()
	assert.Contains(t, out, "ward of life")
	assert.Equal(t, 1, tamsin.Health)
	assert.True(t, b.aria.Character.RT.LifeUsed)

	// The guard is spent: the next such blow falls.
	b.toughen()
	b.hold(nil)
	tamsin.Health = 1
	tamsin.RT.Ward, tamsin.RT.WardCap = 1, 0 // a ward that soaks nothing: any blow would fell her
	tamsin.RT.WardLifeBy = b.aria.Character.RTState()
	b.strike(1, false)
	b.fight()
	assert.Less(t, tamsin.Health, 1, "Ward of Life is once a battle")
}

func TestCovenMotherHexesLastLongerAndCostLess(t *testing.T) {
	initial, cost := map[int]int{}, map[int]int{}
	for _, level := range []int{30, 35, 40} {
		t.Run(fmt.Sprintf("level %d", level), func(t *testing.T) {
			b := witchElite(t, "coven-mother", level)
			b.witchHexes("binding")
			b.startWitchFight()
			cost[level] = 200 - b.aria.Character.Mana
			m := b.firstHeld(status.Paralyzed, 4)
			require.NotNil(t, m)
			initial[level] = m.Character.GetBuffs(status.Paralyzed)[0].TriggersInitial
		})
	}
	assert.GreaterOrEqual(t, initial[35], initial[30], "Lasting hexes at 35 never shorten a hex")
	assert.Less(t, cost[40], cost[35], "Cheaper hexes at 40")
}

func TestCovenMothersChantsAreShorter(t *testing.T) {
	// A Coven Sage's Binding chants a round and lands next round; the
	// Mother's chants a round less again and lands as the battle opens.
	for _, tc := range []struct {
		class string
		held  bool
	}{{"coven-sage", false}, {"coven-mother", true}} {
		t.Run(tc.class, func(t *testing.T) {
			b := witchElite(t, tc.class, 30)
			b.witchHexes("binding")
			b.startWitchFight()
			var held bool
			for _, m := range b.livingBandits() {
				held = held || status.Live(&m.Character, status.Paralyzed)
			}
			assert.Equal(t, tc.held, held)
		})
	}
}

func TestCovenMothersBossResistIsHalved(t *testing.T) {
	assert.Less(t, hexes.BossResist/2, hexes.BossResist)
	chance := func(class string) int {
		return hexes.LandChanceWith(0, hexes.BossResist/2, 0)
	}
	assert.Greater(t, chance("coven-mother"), hexes.LandChanceWith(0, hexes.BossResist, 0))
}

func TestCovenMothersTwinHexLeavesTheThirdTargetExposed(t *testing.T) {
	b := witchElite(t, "coven-mother", 50)
	b.witchHexes("slumber")
	b.startWitchFight()
	assert.NotNil(t, b.firstHeld(status.Exposed, 8), "every third hex also exposes its target")
}

func TestCovenCircleLandsTheFirstResistedHex(t *testing.T) {
	b := witchElite(t, "coven-mother", 60)
	t.Cleanup(scripting.UseHexRollForTest(func(n int) int { return n - 1 })) // always resisted
	b.witchHexes("slumber")
	b.startWitchFight()
	out := strings.Join(*b.messages, "\n")
	for i := 0; i < 3 && b.sleeper() == nil; i++ {
		b.toughen()
		b.hold(nil)
		out += b.fight()
	}
	assert.NotNil(t, b.sleeper(), "the circle lands the hex anyway:\n%s", out)
	assert.True(t, b.aria.Character.RT.CircleUsed)
	assert.Contains(t, out, "coven circle")

	// Only the first: the next hex is resisted as usual.
	hexes.Default.Reset()
	for _, m := range b.livingBandits() {
		m.Character.RemoveBuff(status.Asleep)
	}
	b.aria.Character.Aggro = nil
	for i := 0; i < 3; i++ {
		b.toughen()
		b.hold(nil)
		b.fight()
	}
	assert.Nil(t, b.sleeper(), "once a battle")
}

func TestCroneRottingMiasmaDoublesThePoison(t *testing.T) {
	for _, tc := range []struct {
		class   string
		doubled bool
	}{{"crone-of-ash", true}, {"coven-mother", false}} {
		t.Run(tc.class, func(t *testing.T) {
			b := witchElite(t, tc.class, 35)
			loadShippedBuff(t, "13-poisoned.yaml")
			b.witchHexes("miasma")
			b.startWitchFight()
			m := b.firstHeld(status.Poisoned, 4)
			require.NotNil(t, m)
			assert.Equal(t, tc.doubled, m.Character.RT != nil && m.Character.RT.PoisonX2)
		})
	}
}

func TestCroneHexedFoesAreEasierToHit(t *testing.T) {
	b := witchElite(t, "crone-of-ash", 30)
	b.witchHexes("slumber")
	b.startWitchFight()
	m := b.firstHeld(status.Asleep, 3)
	require.NotNil(t, m)
	assert.Equal(t, 8, b.aria.Character.ClassEffects().Int(classes.CurseAtk))
}

func TestCroneLingeringCurseLeavesTheFoeExposed(t *testing.T) {
	b := witchElite(t, "crone-of-ash", 45)
	b.witchHexes("slumber")
	b.startWitchFight()
	m := b.firstHeld(status.Asleep, 3)
	require.NotNil(t, m)
	b.aria.Character.Mana = 0 // no new hexes
	var exposed bool
	for i := 0; i < 8 && !exposed; i++ {
		b.toughen()
		b.hold(nil)
		b.fight()
		exposed = status.Live(&m.Character, status.Exposed)
	}
	assert.True(t, exposed, "exposed after the hex ends")
}

func TestCroneSoulRotFrightensTheGroupWhenAHexedFoeFalls(t *testing.T) {
	b := witchElite(t, "crone-of-ash", 55)
	b.witchHexes("slumber")
	b.startWitchFight()
	m := b.firstHeld(status.Asleep, 3)
	require.NotNil(t, m)
	require.NotNil(t, m.Character.RT)
	require.True(t, m.Character.RT.SoulRot, "marked when hexed")
	m.Character.Health = 0
	_, err := mobcommands.Suicide("", m, rooms.LoadRoom(m.Character.RoomId))
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*b.messages, "\n"), "soul rot")
}

func TestCroneDoomFellsAFoeHexedThreeRounds(t *testing.T) {
	loadShippedBuff(t, "13-poisoned.yaml")
	for _, tc := range []struct {
		name  string
		level int
	}{{"too early", 55}, {"at 60", 60}} {
		t.Run(tc.name, func(t *testing.T) {
			b := witchElite(t, "crone-of-ash", tc.level)
			b.witchHexes("miasma")
			b.startWitchFight()
			var out string
			for i := 0; i < 8; i++ {
				b.toughen()
				b.hold(nil)
				out += b.fight()
			}
			if tc.level == 60 {
				assert.Contains(t, out, "crone's doom", fmt.Sprint("rounds run:\n", out))
				assert.True(t, b.aria.Character.RT.DoomUsed)
			} else {
				assert.NotContains(t, out, "crone's doom")
			}
		})
	}
}
