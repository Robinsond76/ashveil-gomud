package company

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hexes"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Phase 38a wiring: the Witch's hexes through the real strategy pass, the
// real spell scripts, the buff event and the combat round (shipped config
// and spells, DoCombat), in the brawl world.

// loadShippedBuff registers one shipped buff (poison) for the test.
func loadShippedBuff(t *testing.T, file string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(shippedWorld(t), "buffs", file))
	require.NoError(t, err)
	var spec buffs.BuffSpec
	require.NoError(t, yaml.Unmarshal(data, &spec))
	require.NoError(t, spec.Validate())
	buffs.SetTestBuffSpec(&spec)
	id := spec.BuffId
	t.Cleanup(func() { buffs.RemoveTestBuffSpec(id) })
}

// witchBrawl is a brawl whose player is a level-level Witch with every hex
// she owns at that level, mana to spare, hexes that always land, and a
// fight begun on the bandits. The bandits hold their blows and nobody is
// struck (so no chant breaks).
func witchBrawl(t *testing.T, level int) *brawl {
	t.Helper()
	b := newBrawl(t)
	loadStatusBuffs(t)
	hexes.Default.Reset()
	t.Cleanup(hexes.Default.Reset)
	t.Cleanup(scripting.UseHexRollForTest(func(int) int { return 0 }))
	// Chants are timed to the round in these tests (Dread Whisper is given
	// exactly its chant's rounds), so no blow may break one: a broken chant
	// restarts with no morale check, as the Phase 83 shuffled run saw.
	t.Cleanup(hooks.UseBreakRollForTest(func(n int) int { return n - 1 }))
	// The game registers these (hooks.RegisterListeners).
	buffListener := events.RegisterListener(events.Buff{}, hooks.ApplyBuffs)
	t.Cleanup(func() { events.UnregisterListener(events.Buff{}, buffListener) })
	freshEvents(t)
	dreadListener := events.RegisterListener(events.MoraleCheck{}, hooks.DreadCheck)
	t.Cleanup(func() { events.UnregisterListener(events.MoraleCheck{}, dreadListener) })
	b.withArchetypes("witch")
	b.unplaced()
	forceBlows(t, false)
	noCounters(t)
	c := b.aria.Character
	c.Level = level
	c.SetSkill("cast", 1)
	for _, h := range hexes.All {
		if level >= h.Level {
			c.SpellBook[h.Spell] = 1
		}
	}
	c.SpellBook["hex"] = 1
	c.ManaMax.Value, c.Mana = 200, 200
	return b
}

// start begins the fight on the captain, and runs the round that opens the
// battle.
func (b *brawl) startWitchFight() {
	b.t.Helper()
	b.cmd("attack", "#"+itoa(b.bandits["bandit captain"][0]))
	b.toughen()
	b.hold(nil)
	for _, m := range b.livingBandits() {
		m.Character.SetAggro(0, b.companion(4).InstanceId, characters.DefaultAttack)
		m.Character.Aggro.RoundsWaiting = 1
	}
	b.fight()
	_, inBattle := battle.Current(7)
	require.True(b.t, inBattle)
}

func TestWitchDefaultsToControllerAndHexes(t *testing.T) {
	b := witchBrawl(t, 3)
	assert.Regexp(t, `You\s+witch\s+controller`, b.cmd("strategy", ""))
	stream := b.listen()
	b.startWitchFight() // the opening round: she starts her first chant
	agg := b.aria.Character.Aggro
	require.NotNil(t, agg)
	require.Equal(t, characters.SpellCast, agg.Type, "a controller casts with no command")
	assert.Equal(t, "slumber", agg.SpellInfo.SpellId, "the first hex she owns that has a foe worth it")
	assert.Len(t, agg.SpellInfo.TargetMobInstanceIds, 1, "one foe at level 3")
	assert.Equal(t, 1, castEvents(*stream, combatstream.CastStart, "Aria"))
	assert.Equal(t, 200-6, b.aria.Character.Mana)
}

// sleeper is the bandit asleep, if any.
func (b *brawl) sleeper() *mobs.Mob {
	for _, m := range b.livingBandits() {
		if status.Live(&m.Character, status.Asleep) {
			return m
		}
	}
	return nil
}

func TestSlumberPutsAFoeToSleepAndTheFirstDamageWakesIt(t *testing.T) {
	b := witchBrawl(t, 1)
	b.startWitchFight()
	var out string
	for i := 0; i < 3 && b.sleeper() == nil; i++ {
		b.toughen()
		b.hold(nil)
		out += b.fight()
	}
	sleeper := b.sleeper()
	require.NotNil(t, sleeper, "a foe sleeps")
	assert.Contains(t, out, "(asleep, 2 rounds)")
	assert.True(t, hexes.Default.Immune("m"+itoa(sleeper.InstanceId), status.Asleep), "no lock loops")

	b.toughen()
	b.hold(nil)
	assert.Contains(t, b.fight(), "sleeps on, and loses the action. (asleep)")

	// The first damage wakes it, and says so.
	forceBlows(t, true)
	b.aria.Character.Aggro = nil
	b.aria.Character.SetAggro(0, sleeper.InstanceId, characters.DefaultAttack)
	b.toughen()
	b.hold(nil)
	out = b.fight()
	assert.False(t, status.Live(&sleeper.Character, status.Asleep), "damage woke it")
	assert.Contains(t, out, "wakes with a start")
}

// witchHexes limits the Witch to the given hexes (and Withering Hex).
func (b *brawl) witchHexes(spells ...string) {
	b.t.Helper()
	for id := range b.aria.Character.SpellBook {
		delete(b.aria.Character.SpellBook, id)
	}
	b.aria.Character.SpellBook["hex"] = 1
	for _, id := range spells {
		b.aria.Character.SpellBook[id] = 1
	}
}

// chantTargets is the foes of the chant she has started.
func (b *brawl) chantTargets() []int {
	b.t.Helper()
	agg := b.aria.Character.Aggro
	require.NotNil(b.t, agg)
	require.Equal(b.t, characters.SpellCast, agg.Type)
	return agg.SpellInfo.TargetMobInstanceIds
}

func TestHexReachGrowsWithLevel(t *testing.T) {
	for _, tc := range []struct{ level, reach int }{{1, 1}, {7, 1}, {8, 2}, {16, 3}, {24, 4}} {
		t.Run(fmt.Sprintf("level %d", tc.level), func(t *testing.T) {
			b := witchBrawl(t, tc.level)
			b.witchHexes("slumber")
			b.startWitchFight()
			assert.Len(t, b.chantTargets(), tc.reach)
		})
	}
	assert.Equal(t, hexes.ReachGroup, hexes.Reach(30), "the whole group at 30")
}

func TestMiasmaPoisonsOneRowNotTheWholeGroup(t *testing.T) {
	b := witchBrawl(t, 24) // reach 4
	loadShippedBuff(t, "13-poisoned.yaml")
	b.witchHexes("miasma")
	b.startWitchFight()
	targets := b.chantTargets()
	require.NotEmpty(t, targets)
	assert.Less(t, len(targets), 4, "the five stand in rows of at most three: a row, not her whole reach")
	for i := 0; i < 2; i++ {
		b.toughen()
		b.hold(nil)
		b.fight()
	}
	poisoned := 0
	for _, m := range b.livingBandits() {
		if m.Character.HasBuff(13) {
			poisoned++
		}
	}
	assert.Positive(t, poisoned, "the cloud poisons the row")
	assert.Less(t, poisoned, 5)
}

func TestHexResistedLeavesNoStatusAndNoImmunity(t *testing.T) {
	b := witchBrawl(t, 1)
	t.Cleanup(scripting.UseHexRollForTest(func(n int) int { return n - 1 })) // always resisted
	b.startWitchFight()
	var out string
	for i := 0; i < 2; i++ {
		b.toughen()
		b.hold(nil)
		out += b.fight()
	}
	assert.Contains(t, out, "shrugs the hex off. (resisted)")
	assert.Nil(t, b.sleeper())
	for _, m := range b.livingBandits() {
		assert.False(t, hexes.Default.Immune("m"+itoa(m.InstanceId), status.Asleep))
	}
}

func TestBindingHexParalyzesAndDamageDoesNotBreakIt(t *testing.T) {
	b := witchBrawl(t, 10)
	b.witchHexes("binding")
	b.startWitchFight()
	var held *mobs.Mob
	for i := 0; i < 3 && held == nil; i++ {
		b.toughen()
		b.hold(nil)
		b.fight()
		for _, m := range b.livingBandits() {
			if status.Live(&m.Character, status.Paralyzed) {
				held = m
			}
		}
	}
	require.NotNil(t, held, "a foe is paralyzed")
	assert.Equal(t, 2, held.Character.GetBuffs(status.Paralyzed)[0].TriggersInitial, "1 round at level 10 (one more than it lasts)")
	assert.True(t, held.Character.HasBuffFlag(status.FlagNoDodge))

	// Damage does not break paralysis, and it holds a lost action.
	forceBlows(t, true)
	for id := 1; id <= 4; id++ {
		b.companion(id).Character.SetAggro(0, held.InstanceId, characters.DefaultAttack)
	}
	b.toughen()
	b.hold(map[int]int{held.InstanceId: 700}) // the weakest: the company's blows go to it
	out := b.fight()
	assert.Contains(t, out, "cannot move, and loses the action. (paralyzed)")
	assert.Less(t, held.Character.Health, 700, "the company struck it")
	assert.True(t, status.Live(&held.Character, status.Paralyzed), "damage left the paralysis alone")
	assert.NotContains(t, out, "wakes with a start")
}

func TestBindingHexLastsTwoRoundsAtLevelTwentyAndABossHalvesIt(t *testing.T) {
	for _, tc := range []struct {
		level   int
		boss    bool
		initial int
	}{{20, false, 3}, {20, true, 2}, {10, true, 2}} {
		t.Run(fmt.Sprintf("level %d boss %v", tc.level, tc.boss), func(t *testing.T) {
			b := witchBrawl(t, tc.level)
			b.witchHexes("binding")
			for _, m := range b.livingBandits() {
				m.Boss = tc.boss
			}
			b.startWitchFight()
			var held *mobs.Mob
			for i := 0; i < 3 && held == nil; i++ {
				b.toughen()
				b.hold(nil)
				b.fight()
				for _, m := range b.livingBandits() {
					if status.Live(&m.Character, status.Paralyzed) {
						held = m
					}
				}
			}
			require.NotNil(t, held)
			assert.Equal(t, tc.initial, held.Character.GetBuffs(status.Paralyzed)[0].TriggersInitial)
		})
	}
}

func TestEarthbindLeadenFrailtyLandTheirStatuses(t *testing.T) {
	for _, tc := range []struct {
		spell string
		buff  int
		level int
	}{{"earthbind", status.KnockedDown, 3}, {"leaden", status.Hobbled, 5}, {"frailty", status.Exposed, 12}} {
		t.Run(tc.spell, func(t *testing.T) {
			b := witchBrawl(t, tc.level)
			b.witchHexes(tc.spell)
			b.startWitchFight()
			hit := 0
			for i := 0; i < 3 && hit == 0; i++ {
				b.toughen()
				b.hold(nil)
				b.fight()
				for _, m := range b.livingBandits() {
					if status.Live(&m.Character, tc.buff) {
						hit++
					}
				}
			}
			assert.Positive(t, hit, "%s lands buff %d", tc.spell, tc.buff)
		})
	}
}

func TestControllerFallsBackToWitheringHexWhenEveryFoeIsHexed(t *testing.T) {
	b := witchBrawl(t, 1)
	b.witchHexes("slumber")
	for _, m := range b.livingBandits() {
		require.NoError(t, m.Character.AddBuff(status.Asleep, false, 6))
	}
	b.startWitchFight()
	assert.Equal(t, "hex", b.aria.Character.Aggro.SpellInfo.SpellId, "nobody left to put to sleep: the weak curse")
}

func TestDreadWhisperMakesAFoeTakeAMoraleCheck(t *testing.T) {
	b := witchBrawl(t, 15)
	b.witchHexes("dread")
	for _, m := range b.livingBandits() {
		m.Temperament = "craven" // roll 99: flees
	}
	b.startWitchFight()
	var out string
	for i := 0; i < 3 && len(b.livingBandits()) == 5; i++ {
		b.toughen()
		b.hold(nil)
		out += b.fight()
	}
	assert.Contains(t, out, "loses nerve and flees.")
	assert.Contains(t, out, "(dread)")
}

func TestDreadWhisperMovesNoUnbreakableFoe(t *testing.T) {
	b := witchBrawl(t, 15)
	b.witchHexes("dread")
	for _, m := range b.livingBandits() {
		m.NeverBreak = true
	}
	b.startWitchFight()
	require.NotNil(t, b.aria.Character.Aggro)
	assert.Equal(t, "hex", b.aria.Character.Aggro.SpellInfo.SpellId, "no mana spent on a foe that can't break: the weak curse")
	var out string
	for i := 0; i < 3; i++ {
		b.toughen()
		b.hold(nil)
		out += b.fight()
	}
	assert.NotContains(t, out, "loses nerve and flees.")
	assert.Len(t, b.livingBandits(), 5)
}

// Review: poison, a script's AddHealth, wakes a sleeper like any damage.
func TestScriptDamageWakesASleeper(t *testing.T) {
	b := witchBrawl(t, 1)
	b.startWitchFight()
	foe := b.livingBandits()[0]
	foe.Character.AddBuff(status.Asleep, false, 3)
	require.True(t, status.Live(&foe.Character, status.Asleep))
	scripting.GetMob(foe.InstanceId).AddHealth(-1)
	assert.False(t, status.Live(&foe.Character, status.Asleep))
}

// Review: sleep, paralysis and knockdown share one immunity, so chaining
// them can't hold a foe past the half-of-a-fight bound.
func TestActionLosingHexesShareOneImmunity(t *testing.T) {
	hexes.Default.Reset()
	t.Cleanup(hexes.Default.Reset)
	hexes.Default.Land("m1", status.Asleep, 2)
	for _, buff := range []int{status.Asleep, status.Paralyzed, status.KnockedDown} {
		assert.True(t, hexes.Default.Immune("m1", buff), "buff %d", buff)
	}
	assert.False(t, hexes.Default.Immune("m1", status.Blighted))
	assert.False(t, hexes.Default.Immune("m2", status.Paralyzed))
}

func TestBlightIsNotCastWithoutAHealer(t *testing.T) {
	b := witchBrawl(t, 18)
	b.witchHexes("blight")
	b.startWitchFight()
	assert.Equal(t, "hex", b.aria.Character.Aggro.SpellInfo.SpellId, "no healer among them: the weak curse")
}

func TestBlightIsCastAtAnEnemyHealer(t *testing.T) {
	b := witchBrawl(t, 18)
	b.witchHexes("blight")
	healer := b.livingBandits()[2]
	healer.Role = "healer"
	b.startWitchFight()
	assert.Equal(t, "blight", b.aria.Character.Aggro.SpellInfo.SpellId)
	assert.Equal(t, []int{healer.InstanceId}, b.chantTargets(), "at the healer alone")
}

func TestBlightHalvesHealingItsHolderReceives(t *testing.T) {
	b := witchBrawl(t, 1)
	c := b.aria.Character
	require.NoError(t, c.AddBuff(status.Blighted, false)) // (it recalculates, so before the health is set)
	hardTo(c, 1000)
	c.Health = 100
	assert.Equal(t, 5, c.ApplyHealthChange(10), "halved while blighted")
	c.RemoveBuff(status.Blighted)
	hardMaxTo(c, 1000)
	assert.Equal(t, 10, c.ApplyHealthChange(10), "whole once it lifts")
	assert.Equal(t, -10, c.ApplyHealthChange(-10), "damage is not")
}

func TestTwoWitchesHexDifferentFoes(t *testing.T) {
	b := witchBrawl(t, 1)
	b.witchHexes("slumber")
	b.withArchetypesFor("witch", map[int]string{1: "witch", 2: "cleric", 3: "warrior", 4: "ranger"})
	tamsin := b.companion(1)
	tamsin.Character.SpellBook["slumber"] = 1
	tamsin.Character.SetSkill("cast", 1)
	tamsin.Character.ManaMax.Value, tamsin.Character.Mana = 100, 100
	b.startWitchFight()
	aria := b.aria.Character.Aggro
	require.NotNil(t, aria)
	if tamsin.Character.Aggro != nil && tamsin.Character.Aggro.Type == characters.SpellCast && aria.Type == characters.SpellCast {
		assert.NotEqual(t, aria.SpellInfo.TargetMobInstanceIds, tamsin.Character.Aggro.SpellInfo.TargetMobInstanceIds, "never the same foe")
	}
}

func TestStrategyCommandOffersControllerAndItsHexes(t *testing.T) {
	b := witchBrawl(t, 3)
	out := b.cmd("strategy", "me controller")
	assert.Contains(t, out, "controller")
	assert.NotContains(t, out, "no hex yet")
	assert.Contains(t, b.cmd("strategy", "tamsin controller"), "controller")
}

// Review fix: a status's tick damage (a bleed) wakes a sleeper too.
func TestBleedingWakesASleeper(t *testing.T) {
	b := witchBrawl(t, 1)
	b.startWitchFight()
	foe := b.livingBandits()[0]
	b.toughen()
	foe.Character.AddBuff(status.Asleep, false, 4)
	foe.Character.AddBuff(status.Bleeding, false)
	require.True(t, status.Live(&foe.Character, status.Asleep))
	b.hold(nil)
	out := b.fight()
	assert.False(t, status.Live(&foe.Character, status.Asleep), "the bleed woke it")
	assert.Contains(t, out, "wakes with a start")
}

// Review fix: a foe whose temperament is unset takes no morale check, so
// Dread Whisper isn't spent on it either.
func TestDreadWhisperSkipsAFoeWithNoTemperament(t *testing.T) {
	b := witchBrawl(t, 15)
	b.witchHexes("dread")
	foe := b.livingBandits()[0]
	r := races.GetRace(foe.Character.GetRaceId())
	require.NotNil(t, r)
	prev := r.Temperament
	r.Temperament = ""
	t.Cleanup(func() { r.Temperament = prev })
	for _, m := range b.livingBandits() {
		m.Temperament = ""
	}
	b.startWitchFight()
	require.NotNil(t, b.aria.Character.Aggro)
	assert.Equal(t, "hex", b.aria.Character.Aggro.SpellInfo.SpellId, "the weak curse, not a dread that can't land")
}

// Review fix: a foe that fell while a hex was chanted takes nothing from it.
func TestCastHexAtAFallenFoeIsInvalid(t *testing.T) {
	b := witchBrawl(t, 1)
	b.startWitchFight()
	foe := b.livingBandits()[0]
	foe.Character.Health = 0
	out := scripting.GetActor(b.aria.UserId, 0).CastHex("slumber", *scripting.GetMob(foe.InstanceId))
	assert.Equal(t, "invalid", out["reason"])
	assert.False(t, hexes.Default.Immune("m"+itoa(foe.InstanceId), status.Asleep))
}
