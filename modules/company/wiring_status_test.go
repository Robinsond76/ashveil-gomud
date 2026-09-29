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
	captain.Character.HealthMax.Value, captain.Character.Health = 1000, 1000
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

	out := b.fight()
	assert.Regexp(t, `bandit captain bleeds\. \(1 damage, bleeding\)`, out)
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

func TestStunnedLosesTwoActions(t *testing.T) {
	b := newBrawl(t)
	loadStatusBuffs(t)
	b.aimAt("bandit captain")
	b.toughen() // the company must outlast the round, whatever the dice
	captain := b.captain()
	require.NoError(t, captain.Character.AddBuff(status.Stunned, false))
	lost := 0
	for i := 0; i < 4; i++ {
		b.toughen()
		if strings.Contains(b.fight(), "stands stunned, and loses the action") {
			lost++
		}
	}
	assert.Equal(t, 2, lost)
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
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	const maceID = 99401
	items.SetTestItemSpec(&items.ItemSpec{ItemId: maceID, Name: "test mace", Type: items.Weapon, Subtype: items.Bludgeoning, Hands: 1,
		Damage: items.Damage{DiceRoll: "1d4", Attacks: 1, DiceCount: 1, SideCount: 4}})
	t.Cleanup(func() { items.RemoveTestItemSpec(maceID) })
	b.aria.Character.Equipment.Weapon = items.New(maceID)

	b.aimAt("bandit captain")
	b.toughen() // the company must outlast the round, whatever the dice
	captain := b.captain()
	captain.Character.HealthMax.Value, captain.Character.Health = 1000, 1000

	out := b.fight()
	assert.Regexp(t, `critical hit, \d+ damage, staggered\)`, out)
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
	buffId := events.RegisterListener(events.Buff{}, hooks.ApplyBuffs)
	t.Cleanup(func() { events.UnregisterListener(events.Buff{}, buffId) })
	cutthroat := b.bandits["bandit cutthroat"][0]
	b.aimAt("bandit captain")
	b.toughen() // the company must outlast the round, whatever the dice
	c := b.aria.Character
	c.SpellBook["sparks"] = 5000
	c.Stats.Mysticism.ValueAdj = 1000

	target := mobs.GetInstance(cutthroat)
	var transcript string
	for attempt := 0; attempt < 30 && !status.Has(&target.Character); attempt++ {
		b.toughen()
		target.Character.HealthMax.Value, target.Character.Health = 1000, 1000 // it must outlast the company
		c.SetCast(0, characters.SpellAggroInfo{SpellId: "sparks", TargetMobInstanceIds: []int{cutthroat}})
		transcript += b.fight() + "\n"
	}
	require.Contains(t, transcript, "Sparks sear")
	assert.Regexp(t, `Sparks sear .*\(\d+ damage, overloaded\)`, transcript)
	assert.True(t, target.Character.HasBuff(status.Overloaded))
}
