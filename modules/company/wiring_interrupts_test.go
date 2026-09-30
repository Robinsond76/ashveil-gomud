package company

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 30d1 wiring: broken chants and shield counters through the real
// combat round (shipped config, DoCombat), in the brawl world.

// forceBlows makes every blow land (hit) or every blow miss, undodged and
// with no crits, for the rest of the test.
func forceBlows(t *testing.T, hit bool) {
	t.Helper()
	gameplay := configs.GetGamePlayConfig()
	chance := 0
	if hit {
		chance = 100
	}
	gameplay.Combat.ToHitMin, gameplay.Combat.ToHitMax = configs.ConfigInt(chance), configs.ConfigInt(chance)
	gameplay.Combat.DodgeChanceMin, gameplay.Combat.DodgeChanceMax = 0, 0
	gameplay.Combat.CritChanceMin, gameplay.Combat.CritChanceMax = 0, 0
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
}

// counterDice scripts the counter's rolls: the bash roll, the damage face,
// the stun roll, again and again.
func counterDice(t *testing.T, bash, face, stun int) {
	t.Helper()
	i := 0
	t.Cleanup(hooks.UseCounterRollForTest(func(int) int {
		v := []int{bash, face, stun}[i%3]
		i++
		return v
	}))
}

// noCounters makes every counter roll fail, so a test about chants is not
// disturbed by a shield.
func noCounters(t *testing.T) {
	t.Helper()
	t.Cleanup(hooks.UseCounterRollForTest(func(n int) int { return n - 1 }))
}

// ofKind lists the events of a kind.
func ofKind(stream []combatstream.Event, kind combatstream.Kind) []combatstream.Event {
	var out []combatstream.Event
	for _, e := range stream {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	return out
}

// castsBy lists a caster's cast events of a kind.
func castsBy(stream []combatstream.Event, kind combatstream.Kind, key string) []combatstream.Event {
	var out []combatstream.Event
	for _, e := range ofKind(stream, kind) {
		if e.Source.Key() == key {
			out = append(out, e)
		}
	}
	return out
}

// interruptsOf lists the interrupts of a chanter.
func interruptsOf(stream []combatstream.Event, key string) []combatstream.Event {
	var out []combatstream.Event
	for _, e := range ofKind(stream, combatstream.Interrupt) {
		if e.Target.Key() == key {
			out = append(out, e)
		}
	}
	return out
}

// mobCasts has a mob start a spell with the real cast command, with the
// mana it needs.
func (b *brawl) mobCasts(m *mobs.Mob, spell string) {
	b.t.Helper()
	m.Character.SpellBook[strings.Fields(spell)[0]] = 1
	m.Character.ManaMax.Value, m.Character.Mana = 100, 100
	_, err := mobcommands.TryCommand("cast", spell, m.InstanceId)
	require.NoError(b.t, err)
	require.NotNil(b.t, m.Character.Aggro)
	require.Equal(b.t, characters.SpellCast, m.Character.Aggro.Type, "%s chants %s", m.Character.Name, spell)
}

func key(m *mobs.Mob) string { return fmt.Sprintf("m:%d", m.InstanceId) }

// A companion's heal breaks when a blow draws blood: the spell is lost,
// half its mana comes back, the events say so, and no heal lands.
func TestCompanionHealBrokenByBlow(t *testing.T) {
	b := guardBrawl(t)
	forceBlows(t, true)
	noCounters(t)
	stream := b.listen()
	oswin := b.companion(2)

	var out string
	for try := 0; try < 8 && len(interruptsOf(*stream, key(oswin))) == 0; try++ {
		b.toughen()
		b.hold(nil)
		if oswin.Character.Aggro == nil || oswin.Character.Aggro.Type != characters.SpellCast {
			b.mobCasts(oswin, "heal aria")
			require.Equal(t, 97, oswin.Character.Mana, "Minor Heal costs 3")
		}
		b.strike(2, false) // the captain on Oswin
		out = b.fight()
	}
	broken := interruptsOf(*stream, key(oswin))
	require.Len(t, broken, 1, "a blow broke the chant:\n%s", out)
	assert.Equal(t, b.captain().InstanceId, broken[0].Source.MobInstanceId, "the captain's blow")
	assert.Equal(t, "heal", broken[0].SpellId)
	assert.Equal(t, "Minor Heal", broken[0].Status)
	assert.Equal(t, combatstream.OutcomeSucceeded, broken[0].Outcome)
	assert.Contains(t, out, "Brother Oswin's chant breaks off under the blow. (Minor Heal interrupted)")
	done := castsBy(*stream, combatstream.CastComplete, key(oswin))
	require.NotEmpty(t, done)
	assert.Equal(t, combatstream.OutcomeInterrupted, done[len(done)-1].Outcome)
	assert.Equal(t, 98, oswin.Character.Mana, "3 spent, 1 back (half, rounded down)")
	require.NotNil(t, oswin.Character.Aggro, "back to his foe")
	assert.NotEqual(t, characters.SpellCast, oswin.Character.Aggro.Type)
	assert.Empty(t, ofKind(*stream, combatstream.Heal), "no heal landed")
}

// The player's own chant (a wizard's automatic Magic Missile) breaks the
// same way, and she is told what she lost and got back.
func TestPlayerChantBrokenByBlow(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("wizard")
	b.unplaced()
	captain, _, _, _, _ := b.shapeBandits()
	forceBlows(t, true)
	noCounters(t)
	stream := b.listen()
	b.aria.Character.SetSkill("cast", 1)
	b.aria.Character.LearnSpell("mm")
	b.cmd("attack", fmt.Sprintf("#%d", captain))

	var out string
	for try := 0; try < 8 && len(interruptsOf(*stream, "u:7")) == 0; try++ {
		b.toughen()
		b.hold(nil)
		b.aria.Character.ManaMax.Value, b.aria.Character.Mana = 20, 20
		b.strike(0, false) // the captain on Aria
		out = b.fight()
	}
	broken := interruptsOf(*stream, "u:7")
	require.Len(t, broken, 1, "a blow broke her chant:\n%s", out)
	assert.Equal(t, "Magic Missile", broken[0].Status)
	assert.Contains(t, out, "The blow breaks your chant, and Magic Missile is lost. (Magic Missile interrupted, 3 mana back)")
	assert.Equal(t, 17, b.aria.Character.Mana, "6 spent, 3 back")
	require.NotNil(t, b.aria.Character.Aggro)
	assert.NotEqual(t, characters.SpellCast, b.aria.Character.Aggro.Type)
}

// A blow that misses draws no blood: the chant holds and the heal lands.
func TestMissLeavesChantWhole(t *testing.T) {
	b := guardBrawl(t)
	forceBlows(t, false)
	noCounters(t)
	stream := b.listen()
	oswin := b.companion(2)

	healed := false
	for try := 0; try < 4 && !healed; try++ {
		b.toughen()
		b.aria.Character.Health = 500
		b.hold(nil)
		b.mobCasts(oswin, "heal aria")
		for i := 0; i < 3 && oswin.Character.Aggro != nil && oswin.Character.Aggro.Type == characters.SpellCast; i++ {
			b.strike(2, true) // every bandit on Oswin
			b.fight()
		}
		healed = len(ofKind(*stream, combatstream.Heal)) > 0
	}
	assert.True(t, healed, "the heal landed")
	assert.Empty(t, ofKind(*stream, combatstream.Interrupt), "no miss breaks a chant")
	assert.Greater(t, len(ofKind(*stream, combatstream.Attack)), 0, "blows were struck")
}

// An enemy's chant breaks on a company blow and starts again from the
// first word at its turn, then lands after its full chant time.
func TestEnemyChantBreaksAndRestarts(t *testing.T) {
	b := guardBrawl(t)
	forceBlows(t, true)
	noCounters(t)
	stream := b.listen()
	captain := b.captain()
	tamsin := b.companion(1)

	var out string
	for try := 0; try < 8 && len(interruptsOf(*stream, key(captain))) == 0; try++ {
		b.toughen()
		b.hold(nil)
		for _, m := range b.livingBandits() {
			m.Character.SetAggro(0, b.companion(4).InstanceId, characters.DefaultAttack)
			m.Character.Aggro.RoundsWaiting = 1
		}
		if captain.Character.Aggro.Type != characters.SpellCast {
			b.mobCasts(captain, fmt.Sprintf("mm #%d", tamsin.InstanceId))
		}
		// Only Aria strikes: the player's blows come before any mob's
		// turn, so a break always precedes the captain's.
		b.aria.Character.SetAggro(0, captain.InstanceId, characters.DefaultAttack)
		for id := 1; id <= 4; id++ {
			b.companion(id).Character.SetAggro(0, captain.InstanceId, characters.DefaultAttack)
			b.companion(id).Character.Aggro.RoundsWaiting = 1
		}
		out = b.fight()
	}
	broken := interruptsOf(*stream, key(captain))
	require.NotEmpty(t, broken, "a company blow broke the captain's chant:\n%s", out)
	assert.Equal(t, "Magic Missile", broken[0].Status)
	assert.Contains(t, out, "The bandit captain's chant breaks off under the blow. (Magic Missile interrupted)")
	// The company struck before his turn, so he starts again this round.
	assert.Contains(t, out, "The bandit captain starts the chant again from the first word. (chanting: Magic Missile, 2 rounds)")
	assert.Len(t, castsBy(*stream, combatstream.CastStart, key(captain)), 1, "the restart is a cast start")
	require.Equal(t, characters.SpellCast, captain.Character.Aggro.Type, "still chanting")
	assert.Equal(t, 100-6, captain.Character.Mana, "the restart costs nothing")

	// No more blows land: a round of chanting, then the spell ends.
	forceBlows(t, false)
	ended := func() int {
		n := 0
		for _, e := range castsBy(*stream, combatstream.CastComplete, key(captain)) {
			if e.Outcome != combatstream.OutcomeInterrupted {
				n++
			}
		}
		return n
	}
	b.toughen()
	b.fight()
	assert.Zero(t, ended(), "a round of chanting first")
	b.toughen()
	b.fight()
	assert.Equal(t, 1, ended(), "then the missile is cast (or fizzles), its full chant told")
}

// A blow a guard redirects breaks the guardian's own chant, not the
// ward's.
func TestGuardedBlowBreaksGuardianChant(t *testing.T) {
	b := guardBrawl(t, "tamsin guard me")
	forceBlows(t, true)
	noCounters(t)
	stream := b.listen()
	tamsin := b.companion(1)

	for try := 0; try < 8 && len(interruptsOf(*stream, key(tamsin))) == 0; try++ {
		b.toughen()
		b.hold(nil)
		if tamsin.Character.Aggro == nil || tamsin.Character.Aggro.Type != characters.SpellCast {
			b.mobCasts(tamsin, "heal aria")
		}
		b.aria.Character.SetCast(3, characters.SpellAggroInfo{SpellId: "heal", TargetUserIds: []int{7}})
		b.strike(0, false) // the captain on Aria; Tamsin steps in
		b.fight()
	}
	require.NotEmpty(t, interruptsOf(*stream, key(tamsin)), "the guarded blow broke Tamsin's chant")
	assert.NotEmpty(t, ofKind(*stream, combatstream.GuardUsed))
	assert.Empty(t, interruptsOf(*stream, "u:7"), "the ward's chant is whole")
	require.NotNil(t, b.aria.Character.Aggro)
	assert.Equal(t, characters.SpellCast, b.aria.Character.Aggro.Type)
}

// A missed blow on Tamsin (a wooden shield) is countered: the line, the
// damage, an attack event for the bash.
func TestShieldCounterOnMiss(t *testing.T) {
	b := guardBrawl(t)
	forceBlows(t, false)
	counterDice(t, 0, 3, 99) // a bash, 4 damage, no stun
	stream := b.listen()
	captain := b.captain()
	tamsin := b.companion(1)
	require.True(t, tamsin.Character.HasShield(), "Tamsin carries a shield")

	b.strike(1, false) // the captain on Tamsin
	before := captain.Character.Health
	out := b.fight()

	assert.Regexp(t, `Tamsin Reed turns the blow and drives \w+ shield into the bandit captain\. \(shield bash, 4 damage\)`, out)
	var bashes []combatstream.Event
	for _, e := range ofKind(*stream, combatstream.Attack) {
		if e.WeaponType == "shield-bash" {
			bashes = append(bashes, e)
		}
	}
	require.Len(t, bashes, 1)
	assert.Equal(t, key(tamsin), bashes[0].Source.Key())
	assert.Equal(t, key(captain), bashes[0].Target.Key())
	assert.Equal(t, 4, bashes[0].Damage)
	assert.Equal(t, before-4, captain.Character.Health, "only the bash drew blood")
	assert.False(t, captain.Character.HasBuff(status.Stunned))
}

// A bash that stuns leaves the attacker stunned, with its event.
func TestShieldCounterStuns(t *testing.T) {
	b := guardBrawl(t)
	loadStatusBuffs(t)
	buffId := events.RegisterListener(events.Buff{}, hooks.ApplyBuffs)
	t.Cleanup(func() { events.UnregisterListener(events.Buff{}, buffId) })
	forceBlows(t, false)
	counterDice(t, 0, 0, 0) // a bash, 1 damage, a stun
	stream := b.listen()
	b.strike(1, false)
	out := b.fight()

	assert.Regexp(t, `into the bandit captain\. \(shield bash, 1 damage, stunned\)`, out)
	assert.True(t, b.captain().Character.HasBuff(status.Stunned), "the captain is stunned")
	var stuns int
	for _, e := range ofKind(*stream, combatstream.StatusApplied) {
		if e.BuffId == status.Stunned && e.Target.Key() == key(b.captain()) {
			stuns++
		}
	}
	assert.Equal(t, 1, stuns)
}

// Counters are once a round: every bandit misses Tamsin, one is bashed.
func TestShieldCounterOncePerRound(t *testing.T) {
	b := guardBrawl(t)
	forceBlows(t, false)
	counterDice(t, 0, 0, 99)
	stream := b.listen()
	b.strike(1, true) // every bandit on Tamsin
	b.fight()
	var bashes, missed int
	for _, e := range ofKind(*stream, combatstream.Attack) {
		switch {
		case e.WeaponType == "shield-bash":
			bashes++
		case e.Target.Key() == key(b.companion(1)):
			missed++
		}
	}
	require.Greater(t, missed, 1, "more than one blow missed her")
	assert.Equal(t, 1, bashes, "one counter a round")

	b.strike(1, true)
	b.fight()
	bashes = 0
	for _, e := range ofKind(*stream, combatstream.Attack) {
		if e.WeaponType == "shield-bash" {
			bashes++
		}
	}
	assert.Equal(t, 2, bashes, "and one again the next round")
}

// A sling shot is not countered, nor is anything while the bearer is
// stunned.
func TestNoShieldCounterAgainstBowOrWhileStunned(t *testing.T) {
	b := guardBrawl(t)
	loadStatusBuffs(t)
	forceBlows(t, false)
	counterDice(t, 0, 0, 99)
	stream := b.listen()
	bashes := func() int {
		n := 0
		for _, e := range ofKind(*stream, combatstream.Attack) {
			if e.WeaponType == "shield-bash" {
				n++
			}
		}
		return n
	}

	b.captain().Character.Equipment.Weapon = items.New(10014) // a sling
	b.strike(1, false)
	b.fight()
	assert.Zero(t, bashes(), "no counter to a sling shot")

	b.captain().Character.Equipment.Weapon = items.Item{}
	tamsin := b.companion(1)
	require.NoError(t, tamsin.Character.AddBuff(status.Stunned, false))
	b.strike(1, false)
	b.fight()
	assert.Zero(t, bashes(), "no counter while stunned")
}

// An enemy with a shield counters the player.
func TestEnemyShieldCountersPlayer(t *testing.T) {
	b := guardBrawl(t)
	forceBlows(t, false)
	counterDice(t, 0, 3, 99)
	stream := b.listen()
	captain := b.captain()
	captain.Character.Equipment.Offhand = items.New(20019) // an iron shield
	require.True(t, captain.Character.HasShield())
	b.aria.Character.SetAggro(0, captain.InstanceId, characters.DefaultAttack)
	b.strike(4, false)
	out := b.fight()
	assert.Regexp(t, `The bandit captain turns your blow and drives \w+ shield into you\. \(shield bash, 4 damage\)`, out)
	var onAria int
	for _, e := range ofKind(*stream, combatstream.Attack) {
		if e.WeaponType == "shield-bash" && e.Target.Key() == "u:7" {
			onAria++
		}
	}
	assert.Equal(t, 1, onAria)
}

// A counter that kills its attacker ends it this round.
func TestShieldCounterKills(t *testing.T) {
	b := guardBrawl(t)
	forceBlows(t, false)
	counterDice(t, 0, 3, 99)
	stream := b.listen()
	captain := b.captain()
	b.strike(1, false)
	captain.Character.Health = 2
	b.fight()
	var slain bool
	for _, e := range ofKind(*stream, combatstream.Death) {
		if e.Target.Key() == key(captain) {
			slain = true
		}
	}
	assert.True(t, slain, "the bash killed the captain this round")
}

// Review finding 3: a companion's bash counts for its leader, as its blows
// do.
func TestCompanionCounterCreditsLeader(t *testing.T) {
	b := guardBrawl(t)
	forceBlows(t, false)
	counterDice(t, 0, 3, 99)
	captain := b.captain()
	captain.Character.PlayerDamage = nil
	b.strike(1, false) // the captain on Tamsin, who bashes
	b.fight()
	assert.Equal(t, 4, captain.Character.PlayerDamage[7], "Tamsin's bash is Aria's damage")
}

// The leader-intercept site (11c, and a player guardian): Aria steps in
// for Oswin; the blow she takes breaks her chant, and a miss she takes
// with a shield she counters.
func TestPlayerGuardianChantAndCounter(t *testing.T) {
	b := guardBrawl(t, "me guard oswin")
	forceBlows(t, true)
	noCounters(t)
	stream := b.listen()
	for try := 0; try < 8 && len(interruptsOf(*stream, "u:7")) == 0; try++ {
		b.toughen()
		b.hold(nil)
		b.aria.Character.SetCast(3, characters.SpellAggroInfo{SpellId: "heal", TargetUserIds: []int{7}})
		b.strike(2, false) // the captain on Oswin; Aria steps in
		b.fight()
	}
	require.NotEmpty(t, interruptsOf(*stream, "u:7"), "the blow Aria took for Oswin broke her chant")
	assert.NotEmpty(t, ofKind(*stream, combatstream.GuardUsed))

	forceBlows(t, false)
	counterDice(t, 0, 3, 99)
	b.aria.Character.Aggro = nil
	b.aria.Character.Equipment.Offhand = items.New(20019)
	require.True(t, b.aria.Character.HasShield())
	for try := 0; try < 4 && len(bashesBy(*stream, "u:7")) == 0; try++ {
		b.toughen()
		b.hold(nil)
		b.strike(2, false)
		b.fight()
	}
	assert.NotEmpty(t, bashesBy(*stream, "u:7"), "Aria countered the blow she took for Oswin")
}

// bashesBy lists the shield bashes a ref key struck.
func bashesBy(stream []combatstream.Event, by string) []combatstream.Event {
	var out []combatstream.Event
	for _, e := range ofKind(stream, combatstream.Attack) {
		if e.WeaponType == "shield-bash" && e.Source.Key() == by {
			out = append(out, e)
		}
	}
	return out
}

// The player-vs-player site: Aria's blow breaks Brom's chant, and Brom,
// with a shield, counters her miss.
func TestPlayerVsPlayerChantAndCounter(t *testing.T) {
	b := newBrawl(t)
	brom := users.NewUserRecord(8, 2)
	brom.Username = "brom"
	brom.Password = "$2a$test"
	brom.Character.Name = "Brom"
	brom.Character.RaceId = 1
	brom.Character.Level = 3
	brom.Character.RoomId = b.road.RoomId
	brom.Character.Validate()
	users.SetTestUser(brom)
	b.road.AddPlayer(brom.UserId)
	t.Cleanup(func() { b.road.RemovePlayer(8) })
	stream := b.listen()

	forceBlows(t, true)
	for try := 0; try < 8 && len(interruptsOf(*stream, "u:8")) == 0; try++ {
		brom.Character.HealthMax.Value, brom.Character.Health = 1000, 1000
		b.toughen()
		brom.Character.SetCast(3, characters.SpellAggroInfo{SpellId: "heal", TargetUserIds: []int{8}})
		b.aria.Character.SetAggro(8, 0, characters.DefaultAttack)
		b.fight()
	}
	require.NotEmpty(t, interruptsOf(*stream, "u:8"), "Aria's blow broke Brom's chant")
	assert.Nil(t, brom.Character.Aggro, "a hand cast ends with no aim, as a finished one does")

	forceBlows(t, false)
	counterDice(t, 0, 3, 99)
	brom.Character.Equipment.Offhand = items.New(20019)
	brom.Character.HealthMax.Value, brom.Character.Health = 1000, 1000
	b.toughen()
	b.aria.Character.SetAggro(8, 0, characters.DefaultAttack)
	b.fight()
	assert.Len(t, bashesBy(*stream, "u:8"), 1, "Brom countered Aria's miss")
	assert.Equal(t, 4, b.aria.Character.PlayerDamage[8], "credited to Brom")
}

// The shipped goblin hexer, in a band with the bandits, chants Withering
// Hex at the company's healer (its goblin race aims by casters) and it
// lands when no blow breaks it.
func TestGoblinHexerHexesTheHealer(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("") // Brother Oswin is a cleric: a healer
	b.unplaced()
	copyShipped(t, configs.GetFilePathsConfig().DataFiles.String(), "mobs/dark_forest/70-goblin_hexer.yaml")
	mobs.LoadDataFiles()
	hexer := mobs.NewMobById(70, b.road.RoomId)
	require.NotNil(t, hexer)
	require.GreaterOrEqual(t, hexer.Character.ManaMax.Value, 8, "enough mana for a hex")
	hexer.SpawnGroup = brawlBandits
	hexer.ActivityLevel = 100 // it chants whenever it can
	b.road.AddMob(hexer.InstanceId)
	b.bandits["goblin hexer"] = []int{hexer.InstanceId}
	forceBlows(t, false) // no blow breaks the hex
	stream := b.listen()
	oswin := b.companion(2)
	b.cmd("attack", fmt.Sprintf("#%d", b.bandits["bandit captain"][0]))

	var hit *combatstream.Event
	for i := 0; i < 15 && hit == nil; i++ { // a hex is three rounds; it may fizzle
		b.toughen()
		b.hold(nil)
		b.fight()
		for j := range *stream {
			if e := (*stream)[j]; e.Kind == combatstream.SpellHit && e.SpellId == "hex" {
				hit = &e
			}
		}
	}
	require.NotNil(t, hit, "the hexer's hex landed")
	assert.Equal(t, key(hexer), hit.Source.Key())
	assert.Equal(t, key(oswin), hit.Target.Key(), "on the healer")
	assert.Greater(t, hit.Damage, 0)
}
