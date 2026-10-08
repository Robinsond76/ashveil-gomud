package company

import (
	"strconv"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/coordination"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 33i2 wiring: coordinated enemies through the real combat round
// (shipped config, DoCombat), in the brawl world. The bandits are level 1
// to 4 (a rabble by level); a test sets its tier on the battle as a
// template's `coordination` would.

// coordBrawl is a guardBrawl whose battle fights at tier.
func coordBrawl(t *testing.T, tier coordination.Tier) *brawl {
	t.Helper()
	b := guardBrawl(t)
	bt, ok := battle.Current(7)
	require.True(t, ok)
	require.Equal(t, int(coordination.Rabble), bt.Coordination, "level 1-4 bandits are a rabble")
	battle.SetCoordination(7, int(tier))
	return b
}

// bandit is the first live instance of a bandit by name.
func (b *brawl) bandit(name string) *mobs.Mob {
	b.t.Helper()
	ids := b.bandits[name]
	require.NotEmpty(b.t, ids, name)
	m := mobs.GetInstance(ids[0])
	require.NotNil(b.t, m)
	return m
}

// healerRole makes a bandit an enemy healer who knows Minor Heal.
func healerRole(m *mobs.Mob) {
	m.Role = "healer"
	m.Character.SpellBook["heal"] = 250
	m.Character.ManaMax.Value, m.Character.Mana = 100, 100
}

// quiet keeps every bandit's blow held this round, aimed at Ysolde.
func (b *brawl) quiet() {
	for _, m := range b.livingBandits() {
		if m.Character.Aggro != nil && m.Character.Aggro.Type == characters.SpellCast {
			continue
		}
		m.Character.SetAggro(0, b.companion(4).InstanceId, characters.DefaultAttack)
		m.Character.Aggro.RoundsWaiting = 1
	}
}

func castStarts(events []combatstream.Event, by *mobs.Mob, spell string) int {
	n := 0
	for _, e := range events {
		if e.Kind == combatstream.CastStart && e.Source.MobInstanceId == by.InstanceId && e.SpellId == spell {
			n++
		}
	}
	return n
}

// An enemy healer heals a member of its group below its tier's threshold
// (a band's: half), paying the mana, and not one above it.
func TestEnemyHealerHealsBelowItsTiersThreshold(t *testing.T) {
	b := coordBrawl(t, coordination.Band)
	stream := b.listen()
	slinger, bruiser := b.bandit("bandit slinger"), b.bandit("bandit bruiser")
	healerRole(slinger)

	b.toughen()
	b.hold(map[int]int{bruiser.InstanceId: 600})
	b.quiet()
	b.fight()
	require.Equal(t, 0, castStarts(*stream, slinger, "heal"), "60%% is above a band's 50%%")

	b.toughen()
	b.hold(map[int]int{bruiser.InstanceId: 400})
	b.quiet()
	b.fight()
	require.Equal(t, 1, castStarts(*stream, slinger, "heal"), "40%% is below it")
	require.Equal(t, characters.SpellCast, slinger.Character.Aggro.Type)
	assert.Equal(t, []int{bruiser.InstanceId}, slinger.Character.Aggro.SpellInfo.TargetMobInstanceIds)
	assert.Equal(t, 97, slinger.Character.Mana, "Minor Heal costs 3")

	// The chant runs its course and the heal lands on the bruiser.
	for i := 0; i < 4 && slinger.Character.Aggro != nil && slinger.Character.Aggro.Type == characters.SpellCast; i++ {
		b.toughen()
		b.quiet()
		b.fight()
	}
	var healed bool
	for _, e := range *stream {
		healed = healed || (e.Kind == combatstream.Heal && e.Source.MobInstanceId == slinger.InstanceId && e.Target.MobInstanceId == bruiser.InstanceId)
	}
	assert.True(t, healed, "the heal landed on the bruiser")
}

// A rabble starts one heal a round however many healers it has; a band
// starts one per healer, never two on one patient.
func TestARabbleStartsOneHealARound(t *testing.T) {
	for _, c := range []struct {
		tier coordination.Tier
		want int
	}{{coordination.Rabble, 1}, {coordination.Band, 2}} {
		t.Run(coordination.SpecOf(c.tier).Word, func(t *testing.T) {
			b := coordBrawl(t, c.tier)
			stream := b.listen()
			slinger, bruiser := b.bandit("bandit slinger"), b.bandit("bandit bruiser")
			cutthroats := b.bandits["bandit cutthroat"]
			require.Len(t, cutthroats, 2)
			second := mobs.GetInstance(cutthroats[1])
			healerRole(slinger)
			healerRole(second)
			b.toughen()
			b.hold(map[int]int{bruiser.InstanceId: 200, cutthroats[0]: 200})
			b.quiet()
			b.fight()
			started := castStarts(*stream, slinger, "heal") + castStarts(*stream, second, "heal")
			assert.Equal(t, c.want, started, "tier %d", c.tier)
			if started == 2 {
				assert.NotEqual(t, slinger.Character.Aggro.SpellInfo.TargetMobInstanceIds, second.Character.Aggro.SpellInfo.TargetMobInstanceIds, "two patients")
			}
		})
	}
}

// A healer that has given up the fight (withdrawn) casts nothing.
func TestAWithdrawnEnemyHealerDoesNotCast(t *testing.T) {
	b := coordBrawl(t, coordination.Band)
	stream := b.listen()
	slinger, bruiser := b.bandit("bandit slinger"), b.bandit("bandit bruiser")
	healerRole(slinger)
	b.toughen()
	b.hold(map[int]int{bruiser.InstanceId: 200})
	b.quiet()
	slinger.Character.CombatWithdrawn = true
	t.Cleanup(func() { slinger.Character.CombatWithdrawn = false })
	b.fight()
	assert.Equal(t, 0, castStarts(*stream, slinger, "heal"))
	assert.Equal(t, 100, slinger.Character.Mana)
}

// A band's fighters take its focus: the captain (its leader, the highest
// level) turns on Tamsin, half the fighters (rounded up, the captain
// among them) follow, and the band says so once.
func TestABandTakesItsLeadersFocus(t *testing.T) {
	b := coordBrawl(t, coordination.Band)
	captain := b.bandit("bandit captain")
	tamsin, ysolde := b.companion(1), b.companion(4)
	b.toughen()
	b.hold(nil)
	b.quiet()
	captain.Character.SetAggro(0, tamsin.InstanceId, characters.DefaultAttack)
	captain.Character.Aggro.RoundsWaiting = 1
	out := b.fight()

	onTamsin := 0
	for _, m := range b.livingBandits() {
		if m.Character.Aggro != nil && m.Character.Aggro.MobInstanceId == tamsin.InstanceId {
			onTamsin++
		} else {
			assert.Equal(t, ysolde.InstanceId, m.Character.Aggro.MobInstanceId, "%s keeps its own aim", m.Character.Name)
		}
	}
	assert.Equal(t, coordination.FocusCount(coordination.Band, 5), onTamsin, "three of five fighters")
	assert.Contains(t, out, "closes in on Tamsin Reed.")

	b.toughen()
	b.hold(nil)
	assert.NotContains(t, b.fight(), "closes in on", "said once per change")
}

// A rabble keeps no focus and says nothing.
func TestARabbleKeepsNoFocus(t *testing.T) {
	b := coordBrawl(t, coordination.Rabble)
	captain := b.bandit("bandit captain")
	tamsin := b.companion(1)
	b.toughen()
	b.hold(nil)
	b.quiet()
	captain.Character.SetAggro(0, tamsin.InstanceId, characters.DefaultAttack)
	captain.Character.Aggro.RoundsWaiting = 1
	out := b.fight()
	for _, m := range b.livingBandits() {
		if m.InstanceId != captain.InstanceId {
			assert.NotEqual(t, tamsin.InstanceId, m.Character.Aggro.MobInstanceId, "%s keeps its own aim", m.Character.Name)
		}
	}
	assert.NotContains(t, out, "closes in on")
}

// guardianNextTo makes the bandit standing within a column of the
// captain an enemy guardian.
func (b *brawl) guardianNextTo(captain *mobs.Mob) *mobs.Mob {
	b.t.Helper()
	party, ok := enemyparty.PartyOf(b.road, captain.InstanceId)
	require.True(b.t, ok)
	for _, id := range party.Members {
		if id != captain.InstanceId && formationcombat.GuardReach(party.Formation, mobparty.MemberKeyFor(id), mobparty.MemberKeyFor(captain.InstanceId)) {
			g := mobs.GetInstance(id)
			g.Role = "guardian"
			return g
		}
	}
	b.t.Fatal("no bandit stands beside the captain")
	return nil
}

// A band's guardian steps in front of its hurt leader once a battle; a
// rabble's never does.
func TestEnemyGuardianStepsInByTier(t *testing.T) {
	for _, c := range []struct {
		tier coordination.Tier
		want int
	}{{coordination.Rabble, 0}, {coordination.Band, 1}, {coordination.Drilled, 2}} {
		t.Run(coordination.SpecOf(c.tier).Word, func(t *testing.T) {
			b := coordBrawl(t, c.tier)
			stream := b.listen()
			captain := b.bandit("bandit captain")
			guard := b.guardianNextTo(captain)
			var out string
			for i := 0; i < 3; i++ {
				b.toughen()
				b.hold(map[int]int{captain.InstanceId: 300})
				b.quiet()
				b.aria.Character.SetAggro(0, captain.InstanceId, characters.DefaultAttack)
				out += b.fight()
			}
			used := 0
			for _, e := range *stream {
				if e.Kind == combatstream.GuardUsed && e.Source.MobInstanceId == guard.InstanceId {
					used++
					assert.Equal(t, guard.InstanceId, e.Source.MobInstanceId)
					assert.Equal(t, captain.InstanceId, e.Target.MobInstanceId)
				}
			}
			assert.Equal(t, c.want, used, "tier %d", c.tier)
			if c.want > 0 {
				assert.Contains(t, out, "steps in front of the bandit captain. (guard, none left)")
			}
		})
	}
}

// A drilled company's leader picks its focus by the casters rule while the
// company has one: the captain, re-aiming, turns on Oswin (a cleric, so a
// healer).
func TestADrilledLeaderGoesForTheCasters(t *testing.T) {
	b := coordBrawl(t, coordination.Drilled)
	captain := b.bandit("bandit captain")
	oswin := b.companion(2)
	b.toughen()
	b.hold(nil)
	b.quiet()
	captain.Character.Aggro = nil
	b.fight()
	require.NotNil(t, captain.Character.Aggro)
	assert.Equal(t, oswin.InstanceId, captain.Character.Aggro.MobInstanceId)
}

// A drilled company's caster casts at its focus.
func TestADrilledCasterCastsAtTheFocus(t *testing.T) {
	b := coordBrawl(t, coordination.Drilled)
	stream := b.listen()
	slinger := b.bandit("bandit slinger")
	slinger.Role = "caster"
	slinger.Character.SpellBook["mm"] = 250
	slinger.Character.ManaMax.Value, slinger.Character.Mana = 100, 100
	tamsin := b.companion(1)
	b.toughen()
	b.hold(nil)
	b.quiet()
	captain := b.bandit("bandit captain")
	captain.Character.SetAggro(0, tamsin.InstanceId, characters.DefaultAttack) // the focus
	captain.Character.Aggro.RoundsWaiting = 1
	b.fight()
	require.Equal(t, 1, castStarts(*stream, slinger, "mm"))
	assert.Contains(t, slinger.Character.Aggro.SpellInfo.TargetMobInstanceIds, tamsin.InstanceId)
}

// A veteran company's leader turns on a company member chanting a heal;
// a drilled one keeps its aim.
func TestAVeteranLeaderBreaksAHeal(t *testing.T) {
	for _, c := range []struct {
		tier   coordination.Tier
		breaks bool
	}{{coordination.Drilled, false}, {coordination.Veteran, true}} {
		t.Run(coordination.SpecOf(c.tier).Word, func(t *testing.T) {
			b := coordBrawl(t, c.tier)
			captain := b.bandit("bandit captain")
			oswin, ysolde := b.companion(2), b.companion(4)
			b.toughen()
			b.hold(nil)
			b.quiet()
			captain.Character.SetAggro(0, ysolde.InstanceId, characters.DefaultAttack)
			captain.Character.Aggro.RoundsWaiting = 1
			b.mobCasts(oswin, "heal #"+itoa(b.companion(1).InstanceId))
			oswin.Character.Aggro.RoundsWaiting = 5
			b.fight()
			want := ysolde.InstanceId
			if c.breaks {
				want = oswin.InstanceId
			}
			assert.Equal(t, want, captain.Character.Aggro.MobInstanceId)
		})
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

// 33i2 review finding 3: a hidden guardian never steps in.
func TestAHiddenEnemyGuardianDoesNotStepIn(t *testing.T) {
	b := coordBrawl(t, coordination.Drilled)
	stream := b.listen()
	captain := b.bandit("bandit captain")
	guard := b.guardianNextTo(captain)
	buffs.SetTestFlag("hidden")
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 93304, Name: "hidden", TriggerCount: 1000, RoundInterval: 1, Flags: []string{"hidden"}})
	t.Cleanup(func() { buffs.RemoveTestBuffSpec(93304) })
	require.NoError(t, guard.Character.AddBuff(93304, true))
	guard.Character.Validate()
	b.toughen()
	b.hold(map[int]int{captain.InstanceId: 300})
	b.quiet()
	b.aria.Character.SetAggro(0, captain.InstanceId, characters.DefaultAttack)
	b.fight()
	for _, e := range *stream {
		assert.False(t, e.Kind == combatstream.GuardUsed && e.Source.MobInstanceId == guard.InstanceId, "the hidden guardian stepped in")
	}
}

// 33i2 review finding 4: while the leader is busy chanting, the band
// keeps the focus it had.
func TestABandKeepsItsFocusWhileTheLeaderChants(t *testing.T) {
	b := coordBrawl(t, coordination.Band)
	captain := b.bandit("bandit captain")
	tamsin := b.companion(1)
	require.True(t, battle.SetEnemyFocus(7, "companion:1"))
	b.toughen()
	b.hold(nil)
	b.quiet()
	b.mobCasts(captain, "mm #"+itoa(b.companion(4).InstanceId))
	captain.Character.Aggro.RoundsWaiting = 5
	b.fight()
	onTamsin := 0
	for _, m := range b.livingBandits() {
		if m.Character.Aggro != nil && m.Character.Aggro.MobInstanceId == tamsin.InstanceId {
			onTamsin++
		}
	}
	assert.Equal(t, coordination.FocusCount(coordination.Band, 5)-1, onTamsin, "the followers, without their chanting leader")
}

// An enemy caster aimed at the player turns back to them when its spell
// ends.
func TestAnEnemyCasterTurnsBackToThePlayer(t *testing.T) {
	b := coordBrawl(t, coordination.Band)
	slinger := b.bandit("bandit slinger")
	slinger.Role = "caster"
	slinger.Character.SpellBook["mm"] = 250
	slinger.Character.ManaMax.Value, slinger.Character.Mana = 100, 100
	b.toughen()
	b.hold(nil)
	b.quiet()
	slinger.Character.SetAggro(7, 0, characters.DefaultAttack)
	slinger.Character.Aggro.RoundsWaiting = 1
	b.fight()
	require.Equal(t, characters.SpellCast, slinger.Character.Aggro.Type, "the slinger chants")
	for i := 0; i < 4 && slinger.Character.Aggro != nil && slinger.Character.Aggro.Type == characters.SpellCast; i++ {
		b.toughen()
		b.hold(nil)
		slinger.Character.Mana = 0 // no second spell
		b.fight()
	}
	require.NotNil(t, slinger.Character.Aggro)
	assert.Equal(t, characters.DefaultAttack, slinger.Character.Aggro.Type)
	assert.Equal(t, 7, slinger.Character.Aggro.UserId, "back on Aria")
}

// Enemy healers heal against a player fighting alone, too.
func TestAnEnemyHealerHealsAgainstASoloPlayer(t *testing.T) {
	b := newBrawl(t)
	b.cmd("company", "dismiss all")
	stream := b.listen()
	slinger, bruiser := b.bandit("bandit slinger"), b.bandit("bandit bruiser")
	healerRole(slinger)
	b.aimAt("bandit captain")
	for i := 0; i < 3 && castStarts(*stream, slinger, "heal") == 0; i++ {
		hardTo(b.aria.Character, 1000)
		b.hold(map[int]int{bruiser.InstanceId: 100})
		b.fight()
	}
	assert.Equal(t, 1, castStarts(*stream, slinger, "heal"))
}
