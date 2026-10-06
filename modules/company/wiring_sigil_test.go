package company

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/sigils"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 54 wiring: a sigil laid with `cast sigil of`, picked up by the battle
// that begins in its room, and its effects through the real combat round,
// the shipped spell scripts and the buff event.

func sigilBrawl(t *testing.T) *brawl {
	t.Helper()
	b := newBrawl(t)
	loadStatusBuffs(t)
	freshEvents(t)
	buffListener := events.RegisterListener(events.Buff{}, hooks.ApplyBuffs)
	t.Cleanup(func() { events.UnregisterListener(events.Buff{}, buffListener) })
	forceBlows(t, false)
	noCounters(t)
	b.aria.Character.SetSkill("cast", 1)
	b.aria.Character.ManaMax.Value, b.aria.Character.Mana = 300, 300
	return b
}

// lay puts a sigil of kind in the road room, lit for a while.
func (b *brawl) lay(kind sigils.Kind) {
	b.aria.Character.Sigil = sigils.Lay(kind, b.road.RoomId, time.Now())
}

// beginFight starts the fight on the captain and runs the opening round,
// returning what Aria saw.
func (b *brawl) beginFight() string {
	b.t.Helper()
	b.cmd("attack", "#"+itoa(b.bandits["bandit captain"][0]))
	b.toughen()
	b.hold(nil)
	for _, m := range b.livingBandits() {
		m.Character.SetAggro(0, b.companion(4).InstanceId, characters.DefaultAttack)
		m.Character.Aggro.RoundsWaiting = 1
	}
	out := b.fight()
	_, inBattle := battle.Current(7)
	require.True(b.t, inBattle)
	return out
}

func TestStillnessSigilChillsEveryFoeAtTheStartOfTheBattle(t *testing.T) {
	b := sigilBrawl(t)
	b.lay(sigils.Stillness)
	out := b.beginFight()
	assert.Equal(t, sigils.Stillness, battle.SigilOf(7))
	assert.Contains(t, out, "The stillness sigil turns the air still around the foe.")
	assert.Equal(t, len(b.livingBandits()), b.marked(status.Windchilled), "every foe starts windchilled")
	assert.Equal(t, 4, b.livingBandits()[0].Character.GetBuffs(status.Windchilled)[0].TriggersInitial, "3 rounds, one more than it lasts")
}

func TestNoSigilChangesNothing(t *testing.T) {
	b := sigilBrawl(t)
	b.beginFight()
	assert.Equal(t, sigils.None, battle.SigilOf(7))
	assert.Zero(t, b.marked(status.Windchilled))
}

func TestAFadedOrFarAwaySigilIsNotUsed(t *testing.T) {
	b := sigilBrawl(t)
	b.aria.Character.Sigil = sigils.Lay(sigils.Stillness, b.road.RoomId, time.Now().Add(-time.Hour))
	b.beginFight()
	assert.Equal(t, sigils.None, battle.SigilOf(7), "it faded while the company was away")
	assert.Zero(t, b.marked(status.Windchilled))

	b2 := sigilBrawl(t)
	b2.aria.Character.Sigil = sigils.Lay(sigils.Stillness, b2.road.RoomId+1, time.Now())
	b2.beginFight()
	assert.Equal(t, sigils.None, battle.SigilOf(7), "it was laid in another room")
	assert.Zero(t, b2.marked(status.Windchilled))
}

func TestWardSigilWardsTheFrontRowOnly(t *testing.T) {
	b := sigilBrawl(t)
	b.lay(sigils.Ward)
	for who, cell := range map[string]string{"me": "1 1", "tamsin": "1 2", "garrick": "2 1", "oswin": "3 1", "ysolde": "3 2"} {
		b.cmd("formation", "move "+who+" "+cell)
	}
	f, _ := domainFormationFor(7)
	chars := map[string]*characters.Character{string(domainLeaderKey()): b.aria.Character}
	for id := 1; id <= 4; id++ {
		chars[string(domainCompanionKey(id))] = &b.companion(id).Character
	}
	front, back := 0, 0
	for key := range chars {
		if r, _, placed := f.Find(domainMemberKey(key)); placed && r == 0 {
			front++
		} else if placed {
			back++
		}
	}
	require.Positive(t, front, "the shipped formation has a front row")
	require.Positive(t, back, "and members behind it")

	out := b.beginFight()
	assert.Contains(t, out, "a ward settles over the front row")
	for key, c := range chars {
		r, _, placed := f.Find(domainMemberKey(key))
		if placed && r == 0 {
			require.NotNil(t, c.RT, key)
			assert.Equal(t, sigils.WardBlows, c.RT.Ward, "%s starts warded", key)
			assert.Equal(t, sigils.WardCap(c.Level), c.RT.WardCap, key)
		} else {
			assert.True(t, c.RT == nil || c.RT.Ward == 0, "%s is not in the front row", key)
		}
	}
}

func TestFireSigilStrengthensFireSpellsAndLeavesTheFoeBurning(t *testing.T) {
	cast := func(b *brawl, until func(*mobs.Mob) bool) (*mobs.Mob, string) {
		cutthroat := b.bandits["bandit cutthroat"][0]
		b.aimAt("bandit captain")
		b.toughen()
		c := b.aria.Character
		c.SpellBook["sparks"] = 5000
		c.Stats.Mysticism.ValueAdj = 1000
		target := mobs.GetInstance(cutthroat)
		var transcript string
		for attempt := 0; attempt < 30 && !until(target); attempt++ {
			b.toughen()
			target.Character.HealthMax.Value, target.Character.Health = 1000, 1000
			c.SetCast(0, characters.SpellAggroInfo{SpellId: "sparks", TargetMobInstanceIds: []int{cutthroat}})
			transcript += b.fight() + "\n"
		}
		return target, transcript
	}

	plain := sigilBrawl(t)
	target, _ := cast(plain, func(m *mobs.Mob) bool { return m.Character.HasBuff(status.Overloaded) })
	require.True(t, target.Character.HasBuff(status.Overloaded), "the sparks landed")
	assert.False(t, target.Character.HasBuff(status.Burning), "with no sigil the sparks do not burn")

	b := sigilBrawl(t)
	b.lay(sigils.Fire)
	target, transcript := cast(b, func(m *mobs.Mob) bool { return m.Character.HasBuff(status.Burning) })
	assert.Equal(t, sigils.Fire, battle.SigilOf(7))
	assert.Contains(t, transcript, "Sparks sear")
	assert.True(t, target.Character.HasBuff(status.Burning), "a fire spell under a fire sigil leaves the foe burning")
}

func TestFireAndMendingSigilsScaleSpellsBySharedFactors(t *testing.T) {
	b := sigilBrawl(t)
	b.beginFight()
	aria := scripting.GetActor(7, 0)
	foe := scripting.GetActor(0, b.livingBandits()[0].InstanceId)
	b.aria.Character.SetCast(0, characters.SpellAggroInfo{SpellId: "sparks"})
	plain := aria.SpellFactor(*foe)
	healPlain := aria.HealFactor()

	battle.SetSigil(7, sigils.Fire, time.Now().Add(time.Hour).Unix())
	assert.InDelta(t, plain*1.25, aria.SpellFactor(*foe), 0.0001, "fire spells hit 25% harder under a fire sigil")
	assert.Equal(t, healPlain, aria.HealFactor(), "and heals are untouched")
	b.aria.Character.SetCast(0, characters.SpellAggroInfo{SpellId: "gust"})
	assert.InDelta(t, plain, aria.SpellFactor(*foe), 0.0001, "a spell that is not fire is untouched")

	battle.SetSigil(7, sigils.Mending, time.Now().Add(time.Hour).Unix())
	assert.InDelta(t, healPlain+0.25, aria.HealFactor(), 0.0001, "heals land 25% stronger under a mending sigil")
	b.aria.Character.SetCast(0, characters.SpellAggroInfo{SpellId: "sparks"})
	assert.InDelta(t, plain, aria.SpellFactor(*foe), 0.0001, "and sparks are untouched")
}

func TestCastSigilLaysOneAndSpendsManaAndChalk(t *testing.T) {
	b := sigilBrawl(t)
	c := b.aria.Character
	assert.Contains(t, b.cmd("cast", "sigil"), "Lay a sigil", "no kind lists them")
	assert.Contains(t, b.cmd("cast", "sigil"), "no sigil laid")
	assert.Contains(t, b.cmd("cast", "sigil of frost"), "stillness")

	assert.Contains(t, b.cmd("cast", "sigil of ward"), "no sigil chalk")
	assert.Equal(t, 300, c.Mana)
	assert.False(t, c.Sigil.Live(time.Now()))

	c.StoreItem(items.New(sigils.ChalkItemID))
	c.StoreItem(items.New(sigils.ChalkItemID))
	c.Mana = 5
	assert.Contains(t, b.cmd("cast", "sigil of ward"), "enough mana")
	c.Mana = 300

	out := b.cmd("cast", "sigil of ward")
	assert.Contains(t, out, "draw a ward sigil in chalk")
	assert.Equal(t, 300-sigils.Ward.ManaCost(), c.Mana)
	assert.True(t, c.Sigil.In(b.road.RoomId, time.Now()))
	assert.Equal(t, sigils.Ward, c.Sigil.Kind)
	chalk := 0
	for _, it := range c.GetAllBackpackItems() {
		if it.ItemId == sigils.ChalkItemID {
			chalk++
		}
	}
	assert.Equal(t, 1, chalk, "one chalk used up")
	assert.Contains(t, b.cmd("look", ""), "ward sigil")
	assert.Contains(t, b.cmd("cast", "sigil"), "Your company's ward sigil lasts")

	assert.Contains(t, b.cmd("cast", "sigil of fire"), "already lies here")
	assert.Equal(t, 300-sigils.Ward.ManaCost(), c.Mana, "a refused sigil costs nothing")

	// A company keeps one: a sigil laid elsewhere lets the old one fade.
	c.Sigil = sigils.Lay(sigils.Fire, b.road.RoomId+1, time.Now())
	assert.Contains(t, b.cmd("cast", "sigil of mending"), "The sigil you laid before fades.")
	assert.Equal(t, sigils.Mending, c.Sigil.Kind)
	assert.Equal(t, b.road.RoomId, c.Sigil.RoomId)
}

func TestNothingIsLaidInABattleOrWithoutTheCastSkill(t *testing.T) {
	b := sigilBrawl(t)
	b.aria.Character.StoreItem(items.New(sigils.ChalkItemID))
	b.beginFight()
	assert.Contains(t, b.cmd("cast", "sigil of fire"), "The battle is under way")
	assert.False(t, b.aria.Character.Sigil.Live(time.Now()))

	// A leader without the Cast skill and no caster companion lays nothing.
	b2 := sigilBrawl(t)
	b2.withArchetypesFor("", map[int]string{1: "warrior", 2: "warrior", 3: "warrior", 4: "ranger"})
	b2.aria.Character.SetSkill("cast", 0)
	b2.aria.Character.StoreItem(items.New(sigils.ChalkItemID))
	assert.Contains(t, b2.cmd("cast", "sigil of fire"), "needs a caster")
	assert.False(t, b2.aria.Character.Sigil.Live(time.Now()))
	assert.Equal(t, 300, b2.aria.Character.Mana)
}

// Review: a leader without the Cast skill has a caster companion standing
// with it draw the sigil, from the companion's mana.
func TestACasterCompanionDrawsTheSigilForALeaderWhoCannotCast(t *testing.T) {
	b := sigilBrawl(t)
	b.withArchetypesFor("", map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"})
	b.aria.Character.SetSkill("cast", 0)
	b.aria.Character.StoreItem(items.New(sigils.ChalkItemID))
	oswin := &b.companion(2).Character
	oswin.ManaMax.Value, oswin.Mana = 50, 5
	assert.Contains(t, b.cmd("cast", "sigil of ward"), "doesn't have enough mana")
	assert.False(t, b.aria.Character.Sigil.Live(time.Now()))

	oswin.Mana = 50
	out := b.cmd("cast", "sigil of ward")
	assert.Contains(t, out, "who kneels and draws a ward sigil")
	assert.Equal(t, 50-sigils.Ward.ManaCost(), oswin.Mana, "the companion pays the mana")
	assert.Equal(t, 300, b.aria.Character.Mana, "the leader pays none")
	assert.True(t, b.aria.Character.Sigil.In(b.road.RoomId, time.Now()))
}

// Small aliases for the company domain, so this file reads shortly.
var (
	domainFormationFor = domain.FormationFor
	domainLeaderKey    = func() domain.MemberKey { return domain.LeaderMemberKey }
	domainCompanionKey = domain.CompanionMemberKey
	domainMemberKey    = func(s string) domain.MemberKey { return domain.MemberKey(s) }
)

// A companion's fire spell burns too: the sigil is the company's, not the
// player's.
func TestACompanionsFireSpellBurnsUnderTheFireSigil(t *testing.T) {
	b := sigilBrawl(t)
	b.lay(sigils.Fire)
	b.beginFight()
	require.Equal(t, sigils.Fire, battle.SigilOf(7))
	foe := b.bandit("bandit cutthroat")
	ysolde := b.companion(4)
	for attempt := 0; attempt < 30 && !foe.Character.HasBuff(status.Burning); attempt++ {
		b.toughen()
		foe.Character.HealthMax.Value, foe.Character.Health = 1000, 1000
		ysolde.Character.SetCast(0, characters.SpellAggroInfo{SpellId: "sparks", TargetMobInstanceIds: []int{foe.InstanceId}})
		b.fight()
	}
	assert.True(t, foe.Character.HasBuff(status.Burning))

}
