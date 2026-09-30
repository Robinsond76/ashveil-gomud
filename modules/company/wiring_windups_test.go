package company

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 30d2 wiring: physical wind-ups (the ogre's Crushing Blow) through
// the real combat round (shipped config, DoCombat), in the brawl world.

// windUpDice fixes the wind-up start roll (0..99) for the rest of the test:
// 0 starts any wind-up a foe may start.
func windUpDice(t *testing.T, roll int) {
	t.Helper()
	t.Cleanup(hooks.UseWindUpRollForTest(func(int) int { return roll }))
}

// ogreBrawl is guardBrawl with the fixture hill ogre (mob 9109, Crushing
// Blow) in the bandits' band from the start, so it is in the battle.
func ogreBrawl(t *testing.T, strategies ...string) (*brawl, *mobs.Mob) {
	t.Helper()
	b := newBrawl(t)
	loadStatusBuffs(t)
	// Statuses land (Crushing Blow's knockdown, a crit's stagger).
	buffId := events.RegisterListener(events.Buff{}, hooks.ApplyBuffs)
	t.Cleanup(func() { events.UnregisterListener(events.Buff{}, buffId) })
	// A group is two to five (29b2): the ogre stands in for a cutthroat.
	cut := b.bandits["bandit cutthroat"][1]
	b.road.RemoveMob(cut)
	mobs.DestroyInstance(cut)
	b.bandits["bandit cutthroat"] = b.bandits["bandit cutthroat"][:1]
	ogre := mobs.NewMobById(9109, b.road.RoomId)
	require.NotNil(t, ogre)
	require.Equal(t, 50, ogre.WindUps["crushing-blow"])
	ogre.SpawnGroup = brawlBandits
	b.road.AddMob(ogre.InstanceId)
	b.bandits["hill ogre"] = []int{ogre.InstanceId}

	b.withArchetypes("")
	b.unplaced()
	for _, s := range strategies {
		b.cmd("strategy", s)
	}
	b.cmd("attack", fmt.Sprintf("#%d", b.bandits["bandit captain"][0]))
	b.toughen()
	b.hold(nil)
	for _, m := range b.livingBandits() {
		m.Character.SetAggro(0, b.companion(4).InstanceId, characters.DefaultAttack)
		m.Character.Aggro.RoundsWaiting = 1
	}
	b.fight()
	fight, inBattle := battle.Current(7)
	require.True(t, inBattle)
	require.True(t, fight.Has(ogre.InstanceId), "the ogre is in the battle")
	b.toughen()
	b.hold(nil)
	return b, ogre
}

// ogreOn sets every bandit but the ogre on Ysolde, holding its blow, and
// the ogre on its target (0 for Aria, else a companion id), with no wait.
func (b *brawl) ogreOn(ogre *mobs.Mob, target int) {
	b.t.Helper()
	b.toughen()
	b.hold(nil)
	for _, m := range b.livingBandits() {
		if m.InstanceId == ogre.InstanceId {
			continue
		}
		m.Character.SetAggro(0, b.companion(4).InstanceId, characters.DefaultAttack)
		m.Character.Aggro.RoundsWaiting = 1
	}
	if target == 0 {
		ogre.Character.SetAggro(7, 0, characters.DefaultAttack, 0)
	} else {
		ogre.Character.SetAggro(0, b.companion(target).InstanceId, characters.DefaultAttack, 0)
	}
}

// attacksBy lists the weapon attacks a ref key made.
func attacksBy(stream []combatstream.Event, by string) []combatstream.Event {
	var out []combatstream.Event
	for _, e := range ofKind(stream, combatstream.Attack) {
		if e.Source.Key() == by {
			out = append(out, e)
		}
	}
	return out
}

// windUpsOf lists a foe's wind-up events of a kind.
func windUpsOf(stream []combatstream.Event, kind combatstream.Kind, by string) []combatstream.Event {
	return castsBy(stream, kind, by)
}

// The ogre winds up in plain view (Aria told "you"), swings at nothing
// that turn, and lands at its next: the release line, a Crushing Blow that
// knocks Aria down, and the land event.
func TestOgreWindsUpThenLands(t *testing.T) {
	b, ogre := ogreBrawl(t)
	forceBlows(t, true)
	noCounters(t)
	windUpDice(t, 0)
	stream := b.listen()

	b.ogreOn(ogre, 0)
	b.aria.Character.SetAggro(0, b.captain().InstanceId, characters.DefaultAttack)
	out := b.fight()
	require.Contains(t, out, "The hill ogre plants his feet and drags his club up over his shoulder, eyes on you. (winding up: Crushing Blow, 1 round)")
	starts := windUpsOf(*stream, combatstream.WindUpStart, key(ogre))
	require.Len(t, starts, 1)
	assert.Equal(t, 7, starts[0].Target.UserId)
	assert.Equal(t, "crushing-blow", starts[0].SpellId)
	assert.Equal(t, "Crushing Blow", starts[0].Status)
	assert.Empty(t, attacksBy(*stream, key(ogre)), "no swing while it winds up")
	assert.True(t, hooks.WindingUp(ogre.InstanceId))

	b.ogreOn(ogre, 4) // re-aimed meanwhile: the blow still falls on Aria, as named
	b.aria.Character.SetAggro(0, b.captain().InstanceId, characters.DefaultAttack)
	out = b.fight()
	assert.Contains(t, out, "The hill ogre brings his club down with all his weight.")
	assert.Regexp(t, `\(Crushing Blow, \d+ damage, knocked down\)`, out)
	lands := windUpsOf(*stream, combatstream.WindUpLand, key(ogre))
	require.Len(t, lands, 1)
	assert.Equal(t, combatstream.OutcomeHit, lands[0].Outcome)
	assert.Equal(t, 7, lands[0].Target.UserId, "on the target it named")
	assert.Greater(t, lands[0].Damage, 0)
	swings := attacksBy(*stream, key(ogre))
	require.Len(t, swings, 1, "one strike")
	assert.True(t, b.aria.Character.HasBuff(status.KnockedDown), "Aria is knocked down: %v", b.aria.Character.Buffs)
	assert.False(t, hooks.WindingUp(ogre.InstanceId))
	assert.Empty(t, interruptsOf(*stream, key(ogre)))
}

// A crit on the winding ogre breaks the wind-up: the line, an interrupt
// credited to the company, no blow; then it swings normally for two turns
// before it winds up again, and the summary lists the break.
func TestCritBreaksWindUp(t *testing.T) {
	b, ogre := ogreBrawl(t)
	forceBlows(t, true)
	noCounters(t)
	windUpDice(t, 0)
	stream := b.listen()

	b.ogreOn(ogre, 0)
	b.aria.Character.SetAggro(0, b.captain().InstanceId, characters.DefaultAttack)
	b.fight()
	require.True(t, hooks.WindingUp(ogre.InstanceId))

	forceCrits(t)
	b.ogreOn(ogre, 0)
	b.aria.Character.SetAggro(0, ogre.InstanceId, characters.DefaultAttack)
	swingsBefore := len(attacksBy(*stream, key(ogre)))
	out := b.fight()
	assert.Contains(t, out, "The hill ogre staggers, and the blow dies before it can fall. (Crushing Blow interrupted)")
	broken := interruptsOf(*stream, key(ogre))
	require.NotEmpty(t, broken)
	assert.Equal(t, combatstream.OutcomeSucceeded, broken[0].Outcome)
	assert.Equal(t, "Crushing Blow", broken[0].Status)
	assert.Equal(t, 7, broken[0].Source.UserId, "Aria's crit")
	assert.Empty(t, windUpsOf(*stream, combatstream.WindUpLand, key(ogre)), "no blow")
	assert.NotContains(t, out, "with all his weight")

	// The cooldown: two ordinary swings (the first may come in the round
	// of the break) before the next wind-up. A turn a crit's status costs
	// it doesn't count.
	forceBlows(t, true) // no more crits
	for i := 0; i < 10 && len(windUpsOf(*stream, combatstream.WindUpStart, key(ogre))) < 2; i++ {
		b.ogreOn(ogre, 0)
		b.aria.Character.SetAggro(0, b.captain().InstanceId, characters.DefaultAttack)
		b.fight()
	}
	require.Len(t, windUpsOf(*stream, combatstream.WindUpStart, key(ogre)), 2, "it winds up again")
	assert.Equal(t, 2, len(attacksBy(*stream, key(ogre)))-swingsBefore, "after two ordinary swings")

	windUpDice(t, 99) // no more wind-ups
	for _, m := range b.livingBandits() {
		m.Character.Health = 1
	}
	out = b.fightItOut(10)
	found := false
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Interrupts") {
			found = true
			assert.Contains(t, line, "Crushing Blow")
		}
	}
	assert.True(t, found, "an Interrupts line: %s", out)
}

// Ordinary blows on a winding ogre neither break it nor are told: no line,
// no interrupt, and the blow lands.
func TestOrdinaryBlowsLeaveWindUpWhole(t *testing.T) {
	b, ogre := ogreBrawl(t)
	forceBlows(t, true)
	noCounters(t)
	windUpDice(t, 0)
	stream := b.listen()

	var out string
	for i := 0; i < 2; i++ {
		b.ogreOn(ogre, 0)
		b.aria.Character.SetAggro(0, ogre.InstanceId, characters.DefaultAttack)
		out += b.fight()
	}
	hitsOnOgre := 0
	for _, e := range ofKind(*stream, combatstream.Attack) {
		if e.Target.Key() == key(ogre) && e.Damage > 0 {
			hitsOnOgre++
		}
	}
	require.Greater(t, hitsOnOgre, 0, "Aria's blows landed on the ogre")
	assert.Empty(t, interruptsOf(*stream, key(ogre)))
	assert.NotContains(t, out, "interrupted")
	assert.NotContains(t, out, "holds")
	require.Len(t, windUpsOf(*stream, combatstream.WindUpLand, key(ogre)), 1, "it landed")
}

// A guardian steps in front of its ward and takes the Crushing Blow.
func TestGuardianTakesCrushingBlow(t *testing.T) {
	b, ogre := ogreBrawl(t, "tamsin guard me")
	forceBlows(t, true)
	noCounters(t)
	windUpDice(t, 0)
	stream := b.listen()
	tamsin := b.companion(1)

	b.ogreOn(ogre, 0)
	b.aria.Character.SetAggro(0, b.captain().InstanceId, characters.DefaultAttack)
	b.fight()
	require.True(t, hooks.WindingUp(ogre.InstanceId))
	require.Empty(t, guardEvents(*stream, combatstream.GuardUsed), "no guard spent on the wind-up")

	b.ogreOn(ogre, 0)
	b.aria.Character.SetAggro(0, b.captain().InstanceId, characters.DefaultAttack)
	out := b.fight()
	assert.Contains(t, out, "Tamsin Reed steps in front of you.")
	lands := windUpsOf(*stream, combatstream.WindUpLand, key(ogre))
	require.Len(t, lands, 1)
	assert.Equal(t, key(tamsin), lands[0].Target.Key(), "Tamsin took it")
	assert.Equal(t, combatstream.OutcomeHit, lands[0].Outcome)
	assert.False(t, b.aria.Character.HasBuff(status.KnockedDown), "Aria stands")
}

// With the target it named gone, the blow falls on its aim.
func TestWindUpFallsOnAimWhenNamedTargetGone(t *testing.T) {
	b, ogre := ogreBrawl(t)
	forceBlows(t, true)
	noCounters(t)
	windUpDice(t, 0)
	stream := b.listen()
	tamsin := b.companion(1)

	b.ogreOn(ogre, 1)
	b.fight()
	require.True(t, hooks.WindingUp(ogre.InstanceId))

	b.ogreOn(ogre, 0) // its aim: Aria
	tamsin.Character.RoomId = 920102
	t.Cleanup(func() { tamsin.Character.RoomId = b.road.RoomId })
	b.aria.Character.SetAggro(0, b.captain().InstanceId, characters.DefaultAttack)
	b.fight()
	tamsin.Character.RoomId = b.road.RoomId
	lands := windUpsOf(*stream, combatstream.WindUpLand, key(ogre))
	require.Len(t, lands, 1)
	assert.Equal(t, 7, lands[0].Target.UserId, "on its aim")
}

// A shield bash is a counter strike only: the Crushing Blow that misses
// Tamsin is countered, and nothing is broken or interrupted.
func TestShieldBashBreaksNoWindUp(t *testing.T) {
	b, ogre := ogreBrawl(t)
	forceBlows(t, false)
	counterDice(t, 0, 0, 99) // every fumble countered, no stun
	windUpDice(t, 0)
	stream := b.listen()
	tamsin := b.companion(1)
	require.True(t, tamsin.Character.HasShield(), "Tamsin bears a shield")

	b.ogreOn(ogre, 1)
	b.fight()
	require.True(t, hooks.WindingUp(ogre.InstanceId))
	b.ogreOn(ogre, 1)
	b.fight()

	lands := windUpsOf(*stream, combatstream.WindUpLand, key(ogre))
	require.Len(t, lands, 1)
	assert.Equal(t, combatstream.OutcomeMiss, lands[0].Outcome)
	assert.NotEmpty(t, bashesBy(*stream, key(tamsin)), "Tamsin countered the blow")
	assert.Empty(t, interruptsOf(*stream, key(ogre)), "a bash interrupts nothing")
}

// No wind-up starts for a foe without `windups`, or at the pinned roll.
func TestNoWindUpWithoutTheRoll(t *testing.T) {
	b, ogre := ogreBrawl(t)
	forceBlows(t, true)
	noCounters(t)
	stream := b.listen()
	for i := 0; i < 4; i++ {
		b.ogreOn(ogre, 0)
		b.fight()
	}
	assert.Empty(t, ofKind(*stream, combatstream.WindUpStart), "newBrawl's roll starts none")
	assert.NotEmpty(t, attacksBy(*stream, key(ogre)), "it swings")
}

// The shipped forest ogre, in a band with the bandits, winds up Crushing
// Blow at Aria with its great club and lands it through the real round.
func TestShippedForestOgreCrushes(t *testing.T) {
	b := newBrawl(t)
	copyShipped(t, configs.GetFilePathsConfig().DataFiles.String(), "mobs/dark_forest/85-forest_ogre.yaml")
	mobs.LoadDataFiles()
	cut := b.bandits["bandit cutthroat"][1]
	b.road.RemoveMob(cut)
	mobs.DestroyInstance(cut)
	b.bandits["bandit cutthroat"] = b.bandits["bandit cutthroat"][:1]
	ogre := mobs.NewMobById(85, b.road.RoomId)
	require.NotNil(t, ogre)
	ogre.SpawnGroup = brawlBandits
	b.road.AddMob(ogre.InstanceId)
	b.bandits["forest ogre"] = []int{ogre.InstanceId}
	b.unplaced()
	forceBlows(t, true)
	noCounters(t)
	windUpDice(t, 0)
	stream := b.listen()
	b.cmd("attack", fmt.Sprintf("#%d", b.bandits["bandit captain"][0]))

	var out string
	for i := 0; i < 6 && len(windUpsOf(*stream, combatstream.WindUpLand, key(ogre))) == 0; i++ {
		b.ogreOn(ogre, 0)
		out += b.fight()
	}
	assert.Contains(t, out, "The forest ogre plants his feet and drags his great club up over his shoulder, eyes on you. (winding up: Crushing Blow, 1 round)")
	assert.Contains(t, out, "The forest ogre brings his great club down with all his weight.")
	lands := windUpsOf(*stream, combatstream.WindUpLand, key(ogre))
	require.Len(t, lands, 1, out)
	assert.Equal(t, 7, lands[0].Target.UserId)
	assert.Equal(t, combatstream.OutcomeHit, lands[0].Outcome)
}

// Review fix: a status not from a blow (here set directly, as a spell's
// would be) costs the winding ogre its turn, and the wind-up with it, told
// with the neutral line and credited to no one.
func TestWindUpLostToAStatusThroughTheRound(t *testing.T) {
	b, ogre := ogreBrawl(t)
	forceBlows(t, true)
	noCounters(t)
	windUpDice(t, 0)
	stream := b.listen()

	b.ogreOn(ogre, 0)
	b.aria.Character.SetAggro(0, b.captain().InstanceId, characters.DefaultAttack)
	b.fight()
	require.True(t, hooks.WindingUp(ogre.InstanceId))

	windUpDice(t, 99)
	b.ogreOn(ogre, 0)
	b.aria.Character.SetAggro(0, b.captain().InstanceId, characters.DefaultAttack)
	require.NoError(t, ogre.Character.AddBuff(status.Stunned, false))
	out := b.fight()
	assert.Contains(t, out, "The blow the hill ogre was winding up is lost. (Crushing Blow interrupted)")
	assert.False(t, hooks.WindingUp(ogre.InstanceId))
	assert.Empty(t, windUpsOf(*stream, combatstream.WindUpLand, key(ogre)), "no blow")
	assert.Empty(t, interruptsOf(*stream, key(ogre)), "no one to credit")
	assert.Empty(t, attacksBy(*stream, key(ogre)), "no swing")
}
