package company

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Phase 30a wiring: statuses through the real combat round (DoCombat), on
// the shipped buffs, with the events they emit.

// loadStatusBuffs registers the nine shipped status buffs and their flags
// for one test, and only those: loading every shipped buff into the brawl
// world changes what the other brawl tests see.
func loadStatusBuffs(t *testing.T) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(shippedWorld(t), "buffs", "11??-*.yaml"))
	require.NoError(t, err)
	require.Len(t, files, len(status.Ids()))
	flags, err := filepath.Glob(filepath.Join(shippedWorld(t), "buffs-flags", "*.yaml"))
	require.NoError(t, err)
	for _, f := range flags {
		buffs.SetTestFlag(strings.TrimSuffix(filepath.Base(f), ".yaml"))
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		require.NoError(t, err)
		var spec buffs.BuffSpec
		require.NoError(t, yaml.Unmarshal(data, &spec))
		require.NoError(t, spec.Validate())
		buffs.SetTestBuffSpec(&spec)
		id := spec.BuffId
		t.Cleanup(func() { buffs.RemoveTestBuffSpec(id) })
	}
}

// statusEvents collects every event of kind the stream emits.
func statusEvents(t *testing.T, kinds ...combatstream.Kind) *[]combatstream.Event {
	t.Helper()
	var got []combatstream.Event
	off := combatstream.Default().Subscribe(func(e combatstream.Event) {
		for _, k := range kinds {
			if e.Kind == k {
				got = append(got, e)
			}
		}
	})
	t.Cleanup(off)
	return &got
}

func (b *brawl) captain() *mobs.Mob {
	b.t.Helper()
	ids := b.bandits["bandit captain"]
	require.Len(b.t, ids, 1)
	m := mobs.GetInstance(ids[0])
	require.NotNil(b.t, m)
	return m
}

func TestBleedingTicksStacksAndEndsThroughTheRealRound(t *testing.T) {
	b := newBrawl(t)
	loadStatusBuffs(t)
	ticks := statusEvents(t, combatstream.StatusTick, combatstream.StatusExpired)
	b.aimAt("bandit captain")
	b.toughen() // the company must outlast the round, whatever the dice
	captain := b.captain()
	hardTo(&captain.Character, 1000)
	require.NoError(t, captain.Character.AddBuff(status.Bleeding, false))
	require.NoError(t, captain.Character.AddBuff(status.Bleeding, false))
	captain.Character.Health = 1000

	out := b.fight()
	assert.Regexp(t, `The bandit captain bleeds\. \(2 damage, bleeding\)`, out, "two stacks, one tick per combat round")
	require.NotEmpty(t, *ticks)
	assert.Equal(t, combatstream.StatusTick, (*ticks)[0].Kind)
	assert.Equal(t, 2, (*ticks)[0].Damage)
	assert.Equal(t, status.Bleeding, (*ticks)[0].BuffId)
	assert.Equal(t, captain.InstanceId, (*ticks)[0].Target.MobInstanceId)

	b.toughen()
	assert.Regexp(t, `bandit captain bleeds\. \(2 damage, bleeding\)`, b.fight())
	b.toughen()
	final := b.fight()
	assert.Regexp(t, `bandit captain bleeds\. \(2 damage, bleeding\)`, final)
	assert.Contains(t, final, "The bandit captain's bleeding stops.")
	last := (*ticks)[len(*ticks)-1]
	assert.Equal(t, combatstream.StatusExpired, last.Kind)
	assert.False(t, status.Has(&captain.Character), "three combat rounds later it is over")
}

func TestBleedingCanFellAFoeInTheRoundItHappens(t *testing.T) {
	b := newBrawl(t)
	loadStatusBuffs(t)
	b.aimAt("bandit captain")
	b.toughen() // the company must outlast the round, whatever the dice
	captain := b.captain()
	captain.Character.AddBuff(status.Bleeding, false)
	captain.Character.Health = 1

	attacks := statusEvents(t, combatstream.Attack)
	out := b.fight()
	assert.Regexp(t, `bandit captain bleeds\. \(1 damage, bleeding\)`, out)
	for _, e := range *attacks {
		assert.NotEqual(t, captain.InstanceId, e.Source.MobInstanceId, "a foe the bleed felled strikes no blow that round")
	}
	assert.Less(t, captain.Character.Health, 1, "the bleed took it down")
	assert.Nil(t, mobs.GetInstance(captain.InstanceId), "and its fall was resolved that round")
}

func TestStaggeredFoeLosesExactlyOneAction(t *testing.T) {
	b := newBrawl(t)
	loadStatusBuffs(t)
	b.aimAt("bandit captain")
	b.toughen() // the company must outlast the round, whatever the dice
	captain := b.captain()
	require.NoError(t, captain.Character.AddBuff(status.Staggered, false))

	assert.Contains(t, b.fight(), "The bandit captain reels, and loses the action. (staggered)")
	b.toughen()
	assert.NotContains(t, b.fight(), "loses the action", "the second round it acts")
}

func TestKnockedDownLeaderLosesTheNextActionAndStaysDown(t *testing.T) {
	b := newBrawl(t)
	loadStatusBuffs(t)
	b.aimAt("bandit captain")
	b.toughen() // the company must outlast the round, whatever the dice
	require.NoError(t, b.aria.Character.AddBuff(status.KnockedDown, false))
	b.toughen()

	out := b.fight()
	assert.Contains(t, out, "You scramble up off the ground, and lose your action. (knocked down)")
	assert.True(t, b.aria.Character.HasBuff(status.KnockedDown), "still down for its later rounds")
	b.toughen()
	assert.NotContains(t, b.fight(), "lose your action")
}

// Owner, 2026-09-30: stunned lasts 2 rounds (two lost actions), and a
// stunned fighter can't dodge: with every dodge certain, the captain twists
// aside only once the stun is over.
func TestStunnedLosesTwoActionsAndCantDodge(t *testing.T) {
	b := newBrawl(t)
	loadStatusBuffs(t)
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.DodgeChanceMin, gameplay.Combat.DodgeChanceMax = 100, 100
	gameplay.Combat.ToHitMin, gameplay.Combat.ToHitMax = 100, 100
	gameplay.Combat.CritChanceMin, gameplay.Combat.CritChanceMax = 0, 0
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	b.aimAt("bandit captain")
	b.toughen() // the company must outlast the round, whatever the dice
	captain := b.captain()
	require.NoError(t, captain.Character.AddBuff(status.Stunned, false))
	lost := 0
	for i := 0; i < 4; i++ {
		b.toughen()
		out := b.fight()
		if strings.Contains(out, "stands stunned, and loses the action") {
			lost++
		}
		if i < 2 { // its two stunned rounds; the third tick ends it
			assert.NotContains(t, out, "captain twists aside", "round %d: stunned, no dodge", i+1)
		} else {
			assert.Contains(t, out, "captain twists aside", "round %d: the stun over, every dodge certain", i+1)
		}
	}
	assert.Equal(t, 2, lost, "stunned lasts 2 rounds")
}

func TestStatusesEndWithTheFight(t *testing.T) {
	b := newBrawl(t)
	loadStatusBuffs(t)
	b.aimAt("bandit captain")
	b.toughen() // the company must outlast the round, whatever the dice
	require.NoError(t, b.aria.Character.AddBuff(status.ArmorBroken, false))
	b.toughen()
	for i := 0; i < 200 && len(b.livingBandits()) > 0; i++ {
		b.toughen()
		b.fight()
	}
	require.Empty(t, b.livingBandits())
	b.fight() // the fight's last round ends it
	assert.False(t, status.Has(b.aria.Character), "a status ends with the fight it was struck in")
}

func TestStrayStatusWithNoFightIsClearedQuietly(t *testing.T) {
	b := newBrawl(t)
	loadStatusBuffs(t)
	b.aria.Character.Aggro = nil
	require.NoError(t, b.aria.Character.AddBuff(status.Bleeding, false))
	require.True(t, status.Has(b.aria.Character))

	out := b.fight()
	assert.False(t, status.Has(b.aria.Character), "a leftover of a restart is cleared")
	assert.NotRegexp(t, regexp.MustCompile(`bleed`), out, "without a word")
}

// A crit struck in the real round leaves its weapon's status on the foe,
// named in the hit line and reported on the stream.
func TestCritThroughTheRealRoundLeavesItsStatus(t *testing.T) {
	b := newBrawl(t)
	loadStatusBuffs(t)
	// The world loop's buff listener: a blow's buffs are queued as events.
	buffId := events.RegisterListener(events.Buff{}, hooks.ApplyBuffs)
	t.Cleanup(func() { events.UnregisterListener(events.Buff{}, buffId) })
	applied := statusEvents(t, combatstream.StatusApplied)
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.ToHitMin, gameplay.Combat.ToHitMax = 100, 100
	gameplay.Combat.CritChanceMin, gameplay.Combat.CritChanceMax = 100, 100
	gameplay.Combat.DodgeChanceMin, gameplay.Combat.DodgeChanceMax = 0, 0
	gameplay.Combat.ParryChanceMin, gameplay.Combat.ParryChanceMax = 0, 0
	gameplay.Combat.BlockChanceMin, gameplay.Combat.BlockChanceMax = 0, 0
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	const maceID = 99401
	items.SetTestItemSpec(&items.ItemSpec{ItemId: maceID, Name: "test mace", Type: items.Weapon, Subtype: items.Bludgeoning, Hands: 1,
		Damage: items.Damage{DiceRoll: "1d4", Attacks: 1, DiceCount: 1, SideCount: 4}})
	t.Cleanup(func() { items.RemoveTestItemSpec(maceID) })
	b.aria.Character.Equipment.Weapon = items.New(maceID)

	b.aimAt("bandit captain")
	b.toughen() // the company must outlast the round, whatever the dice
	captain := b.captain()
	hardTo(&captain.Character, 1000)

	out := b.fight()
	// Phase 33i2: the crit's light wound is named after its status.
	assert.Regexp(t, `critical hit, \d+ damage, staggered, wounded\)`, out)
	assert.True(t, captain.Character.HasBuff(status.Staggered), "the mace's crit staggered it")
	var seen bool
	for _, e := range *applied {
		seen = seen || (e.BuffId == status.Staggered && e.Status == "Staggered" && e.Target.MobInstanceId == captain.InstanceId)
	}
	assert.True(t, seen, "and the stream reported it: %+v", *applied)
}

// Shower of Sparks overloads each target it sears, through a real cast.
func TestSparksOverloadsItsTargetsThroughARealCast(t *testing.T) {
	b := newBrawl(t)
	loadStatusBuffs(t)
	freshEvents(t)
	buffId := events.RegisterListener(events.Buff{}, hooks.ApplyBuffs)
	t.Cleanup(func() { events.UnregisterListener(events.Buff{}, buffId) })
	cutthroat := b.bandits["bandit cutthroat"][0]
	b.aimAt("bandit captain")
	b.toughen() // the company must outlast the round, whatever the dice
	c := b.aria.Character
	c.SpellBook["sparks"] = 5000
	c.Stats.Mysticism.ValueAdj = 1000

	// Every cast can fizzle (a roll of 100 fails even a 100% chance), and a
	// companion's crit can leave another status on the cutthroat, so recast
	// until the sparks themselves have overloaded it.
	target := mobs.GetInstance(cutthroat)
	var transcript string
	for attempt := 0; attempt < 30 && !target.Character.HasBuff(status.Overloaded); attempt++ {
		b.toughen()
		hardTo(&target.Character, 1000) // it must outlast the company
		c.SetCast(0, characters.SpellAggroInfo{SpellId: "sparks", TargetMobInstanceIds: []int{cutthroat}})
		transcript += b.fight() + "\n"
	}
	require.Contains(t, transcript, "Sparks sear")
	assert.Regexp(t, `Sparks sear .*\(\d+ damage, overloaded\)`, transcript)
	assert.True(t, target.Character.HasBuff(status.Overloaded))
}

// Phase 30a review: a flight begun before a hobbling blow is held by it
// (since Phase 33c, flee is the retreat order).
func TestPendingFlightIsHeldByHobbled(t *testing.T) {
	b := newBrawl(t)
	loadStatusBuffs(t)
	b.aimAt("bandit captain")
	b.toughen()
	b.fight()
	t.Cleanup(hooks.UseRetreatRollForTest(func(int) int { return 0 }))
	require.Contains(t, b.cmd("flee", ""), "begins an ordered retreat")
	require.NoError(t, b.aria.Character.AddBuff(status.Hobbled, false))
	b.toughen()
	out := b.fight()
	b.toughen()
	out += b.fight()
	assert.Contains(t, out, "Your legs will not carry you out of this.")
	assert.NotContains(t, out, "withdraws")
	assert.Equal(t, b.road.RoomId, b.aria.Character.RoomId)
}

// Phase 30a review: a status lands only on someone still in a fight. One
// struck in a fight's last round arrives after the fight ended and cleared
// its statuses, and lands on nobody.
func TestStatusLandsOnlyInAFight(t *testing.T) {
	b := newBrawl(t)
	loadStatusBuffs(t)
	freshEvents(t)
	buffId := events.RegisterListener(events.Buff{}, hooks.ApplyBuffs)
	t.Cleanup(func() { events.UnregisterListener(events.Buff{}, buffId) })

	b.aria.AddBuff(status.ArmorBroken, `combat`)
	events.ProcessEvents()
	assert.False(t, b.aria.Character.HasBuff(status.ArmorBroken), "no fight: it lands on nobody")

	b.aimAt("bandit captain")
	b.toughen()
	b.fight()
	b.aria.AddBuff(status.ArmorBroken, `combat`)
	events.ProcessEvents()
	assert.True(t, b.aria.Character.HasBuff(status.ArmorBroken), "in the fight it lands")
}
