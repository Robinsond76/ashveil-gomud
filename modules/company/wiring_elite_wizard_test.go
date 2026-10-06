package company

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/summons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 38c3 wiring: the wizard elites (Archon, Archmage, Necromancer)
// through the real strategy pass, the shipped spell scripts and the combat
// round, in the brawl world. Aria is the elite; her blows never land and
// neither do the bandits' unless a test says so.

// eliteCaster is a classCaster whose archetype is the wizard's.
func eliteCaster(t *testing.T, class string, level int, spells ...string) *brawl {
	t.Helper()
	b := newBrawl(t)
	loadStatusBuffs(t)
	loadShippedBuff(t, "13-poisoned.yaml")
	buffListener := events.RegisterListener(events.Buff{}, hooks.ApplyBuffs)
	t.Cleanup(func() { events.UnregisterListener(events.Buff{}, buffListener) })
	classes.SetProvider(&fakeClassStore{state: classes.State{Class: class}})
	t.Cleanup(func() { classes.SetProvider(nil) })
	hooks.ResetEliteForTest()
	t.Cleanup(hooks.ResetEliteForTest)
	b.withArchetypes("wizard")
	b.unplaced()
	forceBlows(t, false)
	noCounters(t)
	c := b.aria.Character
	c.Level = level
	c.SetSkill("cast", 1)
	c.SpellBook = map[string]int{}
	for _, id := range spells {
		c.SpellBook[id] = 1
	}
	c.ManaMax.Value, c.Mana = 300, 300
	return b
}

func TestArchonCounterspellBreaksAChantAndCostsAHeldTurn(t *testing.T) {
	b := eliteCaster(t, "archon", 30)
	t.Cleanup(hooks.UseCounterspellRollForTest(func(int) int { return 0 }))
	stream := b.listen()
	b.startWitchFight()
	captain := b.captain()
	b.toughen()
	b.hold(nil)
	b.mobCasts(captain, "mm #"+itoa(b.companion(1).InstanceId))
	captain.Character.Aggro.RoundsWaiting = 5
	mana := b.aria.Character.Mana
	before := len(*stream)
	out := b.fight()
	assert.Contains(t, out, "chant comes apart")
	assert.Equal(t, mana-12, b.aria.Character.Mana, "Counterspell costs 12")
	broken := interruptsOf(*stream, key(captain))
	require.Len(t, broken, 1)
	assert.Equal(t, combatstream.OutcomeSucceeded, broken[0].Outcome)
	assert.Empty(t, swingsBy(since(*stream, before), "Aria"), "a held turn is no swing")
}

func TestArchonCounterspellCanFail(t *testing.T) {
	b := eliteCaster(t, "archon", 30)
	t.Cleanup(hooks.UseCounterspellRollForTest(func(int) int { return 99 }))
	stream := b.listen()
	b.startWitchFight()
	captain := b.captain()
	b.toughen()
	b.hold(nil)
	b.mobCasts(captain, "mm #"+itoa(b.companion(1).InstanceId))
	captain.Character.Aggro.RoundsWaiting = 5
	mana := b.aria.Character.Mana
	out := b.fight()
	assert.Contains(t, out, "shrugs it off")
	assert.Equal(t, mana-12, b.aria.Character.Mana, "the turn and the mana are spent either way")
	broken := interruptsOf(*stream, key(captain))
	require.Len(t, broken, 1)
	assert.Equal(t, combatstream.OutcomeFailed, broken[0].Outcome)
	assert.Equal(t, characters.SpellCast, captain.Character.Aggro.Type, "the chant holds")
}

func TestArchonCounterspellNeedsAChantAndTheMana(t *testing.T) {
	b := eliteCaster(t, "archon", 30)
	t.Cleanup(hooks.UseCounterspellRollForTest(func(int) int { return 0 }))
	b.startWitchFight()
	mana := b.aria.Character.Mana
	b.classRounds(2, nil)
	assert.Equal(t, mana, b.aria.Character.Mana, "nothing chants: no counter, no cost")

	captain := b.captain()
	b.mobCasts(captain, "mm #"+itoa(b.companion(1).InstanceId))
	captain.Character.Aggro.RoundsWaiting = 5
	b.aria.Character.Mana = 11
	out := b.fight()
	assert.NotContains(t, out, "counter-word", "11 mana is not enough")
	assert.Equal(t, 11, b.aria.Character.Mana)
}

func TestArchonCounterspellDrainsTheCasterAtRank45(t *testing.T) {
	b := eliteCaster(t, "archon", 45)
	t.Cleanup(hooks.UseCounterspellRollForTest(func(int) int { return 0 }))
	b.startWitchFight()
	captain := b.captain()
	b.toughen()
	b.hold(nil)
	b.mobCasts(captain, "mm #"+itoa(b.companion(1).InstanceId))
	captain.Character.Aggro.RoundsWaiting = 5
	out := b.fight()
	assert.Contains(t, out, "10 mana lost", "a tenth of its 100")
	assert.Equal(t, 100-6-10, captain.Character.Mana, "its 6 for the chant, then a tenth of its 100")
}

func TestArchonAegisWardsEveryAllyOnceABattle(t *testing.T) {
	b := eliteCaster(t, "archon", 60)
	stream := b.listen()
	b.startWitchFight() // the opening round
	assert.Equal(t, 5, warded(b))
	require.Len(t, abilityEvents(*stream, "Aria"), 1)
	assert.Equal(t, "Archon's Aegis", abilityEvents(*stream, "Aria")[0].Status)
	// The wards go; she does not spend it again.
	for _, c := range []*characters.Character{&b.companion(1).Character, &b.companion(2).Character} {
		c.RT.Ward = 0
	}
	b.aria.Character.RT.Ward = 0
	out := b.fight()
	assert.NotContains(t, out, "archon's aegis")
}

func TestArchonTwinWardCoversASecondAlly(t *testing.T) {
	b := eliteCaster(t, "archon", 35, "arcaneward")
	b.startWitchFight()
	b.classRounds(4, nil)
	assert.GreaterOrEqual(t, warded(b), 2, "one Arcane Ward covers two")
}

func TestArchonManaShieldGivesItsRowTheAura(t *testing.T) {
	b := eliteCaster(t, "archon", 40)
	require.Contains(t, b.cmd("formation", "move me 1 1"), "Placed")
	require.Contains(t, b.cmd("formation", "move tamsin 1 2"), "Placed")
	require.Contains(t, b.cmd("formation", "move oswin 2 1"), "Placed")
	b.startWitchFight()
	b.classRounds(1, nil)
	assert.Equal(t, 15, b.aria.Character.Aura.SpellResolve, "her own row")
	assert.Equal(t, 15, b.companion(1).Character.Aura.SpellResolve, "beside her")
	assert.Zero(t, b.companion(2).Character.Aura.SpellResolve, "the row behind is not covered")
}

func TestArchmageOverchannelsEveryThirdRound(t *testing.T) {
	b := eliteCaster(t, "archmage", 30, "mm")
	stream := b.listen()
	b.startWitchFight()
	var out string
	for i := 0; i < 8; i++ {
		b.toughen()
		b.hold(nil)
		out += b.fight()
	}
	assert.Regexp(t, `\(overchannel, \+50% damage, \d+ mana\)`, out)
	assert.NotContains(t, out, "MISSING")
	casts := castEvents(*stream, combatstream.CastStart, "Aria")
	overs := strings.Count(out, "(overchannel")
	assert.Greater(t, casts, overs, "it is not on every cast")
	assert.Positive(t, overs)
}

func TestArchmageQuickCastingChantsOneRoundLess(t *testing.T) {
	count := func(level int) int {
		b := eliteCaster(t, "archmage", level, "mm")
		stream := b.listen()
		b.startWitchFight()
		for i := 0; i < 8; i++ {
			b.toughen()
			b.hold(nil)
			b.fight()
		}
		return castEvents(*stream, combatstream.CastStart, "Aria")
	}
	assert.Greater(t, count(35), count(30), "more casts in the same rounds")
}

func TestArchmageStormDoublesSparksOnce(t *testing.T) {
	b := eliteCaster(t, "archmage", 60, "mm", "sparks")
	b.startWitchFight()
	assert.Contains(t, strings.Join(*b.messages, "\n"), "(archmage's storm, double damage)")
	var out string
	for i := 0; i < 10; i++ {
		b.toughen()
		b.hold(nil)
		out += b.fight()
	}
	assert.True(t, b.aria.Character.RT.StormUsed, "the opening Shower of Sparks was the storm")
	assert.NotContains(t, out, "archmage's storm", "once a battle")
	assert.Contains(t, out, "sparks bursts", "later Showers are ordinary")
}

func TestNecromancerRaisesAFallenFoeAsAThrall(t *testing.T) {
	t.Cleanup(company.ResetSummonsForTest)
	b := eliteCaster(t, "necromancer", 30, "raisefallen")
	b.startWitchFight()
	fall := b.livingBandits()[0]
	fall.Character.Health = 0
	room := rooms.LoadRoom(fall.Character.RoomId)
	_, err := mobcommands.Suicide("", fall, room)
	require.NoError(t, err)
	assert.True(t, summons.HasFallen(7), "the foe is remembered")

	thrall := b.summonRounds(6)
	require.NotNil(t, thrall, "a thrall rises")
	assert.Equal(t, "thrall", thrall.Character.RT.Summon.Kind)
	assert.Contains(t, thrall.Character.Name, "risen")
	whole := mobs.NewMobByIdNoElite(fall.MobId, fall.Character.RoomId, fall.Character.Level)
	require.NotNil(t, whole)
	assert.InDelta(t, whole.Character.HealthMax.Value*60/100, thrall.Character.HealthMax.Value, 2, "60% of its health")
	assert.Equal(t, thrall.Character.HealthMax.Value, thrall.Character.Health)
	assert.Equal(t, 300-30, b.aria.Character.Mana, "a tenth of her mana")
	assert.Empty(t, thrall.Character.SpellBook, "weapon attacks only")
	assert.False(t, summons.HasFallen(7), "once a battle")

	battle.End(7)
	for _, m := range b.livingBandits() {
		_, err := mobcommands.Suicide("vanish", m, rooms.LoadRoom(m.Character.RoomId))
		require.NoError(t, err)
	}
	b.fight()
	assert.Empty(t, company.SummonInstances(), "it crumbles when the battle ends")
	assert.Nil(t, mobs.GetInstance(thrall.InstanceId))
	assert.False(t, summons.HasFallen(7))
}

func TestNecromancerCannotRaiseABossOrTwiceBelowRank50(t *testing.T) {
	t.Cleanup(company.ResetSummonsForTest)
	b := eliteCaster(t, "necromancer", 30, "raisefallen")
	b.startWitchFight()
	boss := b.captain()
	boss.Boss = true
	boss.Character.Health = 0
	_, err := mobcommands.Suicide("", boss, rooms.LoadRoom(boss.Character.RoomId))
	require.NoError(t, err)
	assert.False(t, summons.HasFallen(7), "a boss is never raised")
	assert.Nil(t, b.summonRounds(3))
}

func TestNecromancerDeathsHarvestFeedsMana(t *testing.T) {
	b := eliteCaster(t, "necromancer", 55)
	b.startWitchFight()
	b.aria.Character.Mana = 100
	fall := b.livingBandits()[0]
	fall.Character.Health = 0
	_, err := mobcommands.Suicide("", fall, rooms.LoadRoom(fall.Character.RoomId))
	require.NoError(t, err)
	assert.Equal(t, 109, b.aria.Character.Mana, "3% of 300")
}

func TestNecromancerBargainLeavesOneHealthOnce(t *testing.T) {
	b := eliteCaster(t, "necromancer", 60)
	forceBlows(t, true)
	b.startWitchFight()
	b.toughen()
	b.hold(nil)
	b.aria.Character.Health = 3
	b.strike(0, false)
	out := b.fight()
	assert.Contains(t, out, "lich's bargain, 1 health left")
	assert.Equal(t, 1, b.aria.Character.Health)
	assert.True(t, b.aria.Character.RT.BargainUsed)
}

func TestNecromancerLifeDrainHealsMoreAndChillsAtRanks(t *testing.T) {
	b := eliteCaster(t, "necromancer", 40, "siphon")
	b.startWitchFight()
	tamsin := &b.companion(1).Character
	b.toughen()
	tamsin.Health = 300
	var out string
	for i := 0; i < 4; i++ {
		b.hold(nil)
		out += b.fight()
	}
	assert.Greater(t, tamsin.Health, 300, "drained life heals the most hurt ally")
	chilled := 0
	for _, m := range b.livingBandits() {
		if status.Live(&m.Character, status.Hobbled) {
			chilled++
		}
	}
	assert.Positive(t, chilled, "Grave Chill hobbles the first target")
}

func TestArchmageBarrageStrikesASecondFoe(t *testing.T) {
	b := eliteCaster(t, "archmage", 45, "mm")
	b.startWitchFight()
	var out string
	for i := 0; i < 4; i++ {
		b.toughen()
		b.hold(nil)
		out += b.fight()
	}
	assert.Contains(t, out, "You release the light, and it streaks into")
	assert.Regexp(t, `A second streak of cold light arcs into the .*\(barrage, \d+ damage\)`, out)
}

func TestArchonReflectionReturnsHalfAWardedBlowOnce(t *testing.T) {
	b := eliteCaster(t, "archon", 55)
	b.startWitchFight()
	forceBlows(t, true)
	b.toughen()
	b.hold(nil)
	captain := b.captain()
	rt := b.aria.Character.RTState()
	rt.Ward, rt.WardCap, rt.WardReflectBy = 2, 6, rt
	before := captain.Character.Health
	b.strike(0, false)
	out := b.fight()
	assert.Contains(t, out, "(reflection,")
	assert.Less(t, captain.Character.Health, before, "the blow came back")
	assert.True(t, rt.ReflectUsed, "once a battle")
}
