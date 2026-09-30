package company

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 30c2 wiring: guardians through the real strategy command, attack,
// and combat round (shipped config, DoCombat), in the brawl world.

// guardBrawl is a brawl with archetypes set and everyone unplaced (so
// reach fails open and only the strategies decide), the fight begun on
// the bandits, and the company and bandits made hard to kill.
func guardBrawl(t *testing.T, strategies ...string) *brawl {
	t.Helper()
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	for _, s := range strategies {
		b.cmd("strategy", s)
	}
	b.cmd("attack", fmt.Sprintf("#%d", b.bandits["bandit captain"][0]))
	// The battle begins at the next round's battle pass. The bandits hold
	// their blows that round, so no guard is spent before a test looks.
	b.toughen()
	b.hold(nil)
	for _, m := range b.livingBandits() {
		m.Character.SetAggro(0, b.companion(4).InstanceId, characters.DefaultAttack)
		m.Character.Aggro.RoundsWaiting = 1
	}
	b.fight()
	_, inBattle := battle.Current(7)
	require.True(t, inBattle)
	require.Equal(t, battle.MaxGuards, battle.GuardsLeft(7, "companion:1"))
	require.Equal(t, battle.MaxGuards, battle.GuardsLeft(7, "leader"))
	b.toughen()
	b.hold(nil)
	return b
}

// strike sets the captain on the ward (a companion id, or 0 for Aria).
// Unless all, every other bandit is set on Ysolde (companion 4, never a
// ward here) and holds its blow this round, so the captain's is the only
// blow of the round.
func (b *brawl) strike(ward int, all bool) {
	b.t.Helper()
	captain := b.bandits["bandit captain"][0]
	for _, m := range b.livingBandits() {
		target := 4
		if all || m.InstanceId == captain {
			target = ward
		}
		if target == 0 {
			m.Character.SetAggro(7, 0, characters.DefaultAttack)
		} else {
			m.Character.SetAggro(0, b.companion(target).InstanceId, characters.DefaultAttack)
		}
		if target == 4 {
			m.Character.Aggro.RoundsWaiting = 1
		}
	}
}

// guardEvents counts the guard events since listen.
func guardEvents(events []combatstream.Event, kind combatstream.Kind) []combatstream.Event {
	var out []combatstream.Event
	for _, e := range events {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	return out
}

// blowsOn counts the bandits' weapon attacks landing on (aimed at and
// resolved against) a ref key.
func blowsOn(events []combatstream.Event, key string) int {
	n := 0
	for _, e := range events {
		if e.Kind == combatstream.Attack && e.Target.Key() == key {
			n++
		}
	}
	return n
}

func TestGuardianGuardsWard(t *testing.T) {
	b := guardBrawl(t, "tamsin guard me")
	got := b.listen()
	b.strike(0, false)
	tamsin := b.companion(1)
	out := b.fight()

	assert.Contains(t, out, "Tamsin Reed steps in front of you. (guard, 1 left)")
	used := guardEvents(*got, combatstream.GuardUsed)
	require.Len(t, used, 1)
	assert.Equal(t, tamsin.InstanceId, used[0].Source.MobInstanceId)
	assert.Equal(t, 7, used[0].Target.UserId)
	fight, _ := battle.Current(7)
	assert.Equal(t, fight.FightID, used[0].FightID, "in the battle's fight")
	assert.Equal(t, 1, battle.GuardsLeft(7, "companion:1"))
	assert.Zero(t, blowsOn(*got, "u:7"), "the blow went to Tamsin, not Aria")
	assert.GreaterOrEqual(t, blowsOn(*got, fmt.Sprintf("m:%d", tamsin.InstanceId)), 1, "Tamsin took it")
	assert.Equal(t, b.bandits["bandit captain"][0], b.captain().InstanceId)
	assert.Equal(t, 7, b.captain().Character.Aggro.UserId, "the captain's aim is untouched")
}

func TestGuardianPlayerGuardsCompanion(t *testing.T) {
	b := guardBrawl(t, "me guard garrick")
	got := b.listen()
	b.strike(3, false)
	garrick := b.companion(3)
	out := b.fight()

	assert.Contains(t, out, "You step in front of Garrick Vane. (guard, 1 left)")
	require.Len(t, guardEvents(*got, combatstream.GuardUsed), 1)
	assert.Zero(t, blowsOn(*got, fmt.Sprintf("m:%d", garrick.InstanceId)), "Garrick untouched")
	assert.Equal(t, 1, blowsOn(*got, "u:7"), "Aria took the blow")
	assert.Equal(t, 1, battle.GuardsLeft(7, "leader"))
}

func TestGuardianCompanionGuardsCompanion(t *testing.T) {
	b := guardBrawl(t, "tamsin guard oswin")
	got := b.listen()
	b.strike(2, false)
	oswin := b.companion(2)
	out := b.fight()

	assert.Contains(t, out, "Tamsin Reed steps in front of Brother Oswin. (guard, 1 left)")
	require.Len(t, guardEvents(*got, combatstream.GuardUsed), 1)
	assert.Zero(t, blowsOn(*got, fmt.Sprintf("m:%d", oswin.InstanceId)), "Oswin untouched")
}

func TestGuardsExhaustAndRefill(t *testing.T) {
	b := guardBrawl(t, "tamsin guard me")
	got := b.listen()
	round := func() string {
		b.toughen()
		b.hold(nil)
		b.strike(0, true)
		*got = nil
		return b.fight()
	}

	out := round()
	assert.Len(t, guardEvents(*got, combatstream.GuardUsed), 2, "both guards spent on five blows")
	assert.Len(t, guardEvents(*got, combatstream.GuardExhausted), 1)
	assert.Contains(t, out, "(guard, none left)")
	assert.Equal(t, 3, blowsOn(*got, "u:7"), "the rest land on Aria")

	round()
	assert.Empty(t, guardEvents(*got, combatstream.GuardUsed), "none back after one round")
	assert.Equal(t, 5, blowsOn(*got, "u:7"))

	out = round()
	assert.Len(t, guardEvents(*got, combatstream.GuardUsed), 1, "one back after two rounds")
	assert.Contains(t, out, "Tamsin Reed steps in front of you. (guard, none left)")

	// A new battle starts full.
	battle.End(7)
	assert.Equal(t, 0, battle.GuardsLeft(7, "companion:1"))
}

func TestGuardianUnguardedWardIsStruck(t *testing.T) {
	b := guardBrawl(t) // no guardian
	got := b.listen()
	b.strike(0, false)
	b.fight()
	assert.Empty(t, guardEvents(*got, combatstream.GuardUsed))
	assert.Equal(t, 1, blowsOn(*got, "u:7"), "the captain's blow lands on Aria")
}

func TestGuardianKnockedDown(t *testing.T) {
	b := guardBrawl(t, "tamsin guard me")
	loadStatusBuffs(t)
	got := b.listen()
	tamsin := b.companion(1)
	require.NoError(t, tamsin.Character.AddBuff(status.KnockedDown, false))
	b.strike(0, false)
	b.fight()
	assert.Empty(t, guardEvents(*got, combatstream.GuardUsed), "no guard while down")
	assert.Equal(t, 1, blowsOn(*got, "u:7"))
	assert.Equal(t, 2, battle.GuardsLeft(7, "companion:1"), "guards kept")

	status.Clear(&tamsin.Character)
	require.NoError(t, tamsin.Character.AddBuff(status.Stunned, false))
	*got = nil
	b.toughen()
	b.strike(0, false)
	b.fight()
	assert.Empty(t, guardEvents(*got, combatstream.GuardUsed), "no guard while stunned")

	status.Clear(&tamsin.Character)
	*got = nil
	b.toughen()
	b.strike(0, false)
	b.fight()
	assert.Len(t, guardEvents(*got, combatstream.GuardUsed), 1, "back up, guarding")
}

func TestGuardianMostHurt(t *testing.T) {
	b := guardBrawl(t, "tamsin guard")
	got := b.listen()
	// Nobody hurt: nobody guarded.
	b.strike(0, false)
	b.fight()
	assert.Empty(t, guardEvents(*got, combatstream.GuardUsed), "nobody hurt, nobody guarded")

	// Aria the most hurt: guarded. Oswin, less hurt, is not.
	*got = nil
	b.toughen()
	b.aria.Character.Health = 400
	b.companion(2).Character.Health = 900
	b.strike(0, false)
	out := b.fight()
	assert.Contains(t, out, "Tamsin Reed steps in front of you.")
	require.Len(t, guardEvents(*got, combatstream.GuardUsed), 1)

	*got = nil
	b.toughen()
	b.aria.Character.Health = 400
	b.companion(2).Character.Health = 900
	b.strike(2, false)
	b.fight()
	assert.Empty(t, guardEvents(*got, combatstream.GuardUsed), "Oswin isn't the most hurt")
}

func TestGuardianRefusedInBattle(t *testing.T) {
	b := guardBrawl(t)
	assert.Equal(t, usercommands.BattleUnderWay, strings.TrimSpace(b.cmd("strategy", "tamsin guard me")))
	assert.Contains(t, b.cmd("strategy", ""), "Tamsin Reed")
	assert.NotContains(t, b.cmd("strategy", ""), "guardian")
}

// placeBrawl is a brawl with the formation set by moves, before any fight.
func placeBrawl(t *testing.T, moves ...string) *brawl {
	t.Helper()
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	for _, mv := range moves {
		require.Contains(t, b.cmd("formation", "move "+mv), "Placed")
	}
	return b
}

func TestGuardianOutOfReach(t *testing.T) {
	// Aria front left, Tamsin front right: two columns apart.
	b := placeBrawl(t, "me 1 1", "tamsin 1 3", "oswin 2 2")
	out := b.cmd("strategy", "tamsin guard me")
	assert.Contains(t, out, "Out of reach: Tamsin Reed can't step in for you from there.")
	assert.Contains(t, b.cmd("formation", ""), "Out of reach: Tamsin Reed can't step in for you from there.")

	// Moving her to the next column clears the warning; back again, it
	// warns on the move.
	out = b.cmd("formation", "move tamsin 1 2")
	assert.NotContains(t, out, "Out of reach")
	assert.NotContains(t, b.cmd("strategy", ""), "Out of reach")
	out = b.cmd("formation", "move tamsin 1 3")
	assert.Contains(t, out, "Out of reach: Tamsin Reed can't step in for you from there.")

	// In the fight, no guard for Aria from there.
	got := b.listen()
	b.cmd("attack", fmt.Sprintf("#%d", b.bandits["bandit captain"][0]))
	for i := 0; i < 2; i++ {
		b.toughen()
		b.hold(nil)
		for _, m := range b.livingBandits() {
			m.Character.SetAggro(7, 0, characters.DefaultAttack)
		}
		b.fight()
	}
	assert.Empty(t, guardEvents(*got, combatstream.GuardUsed))
	assert.NotZero(t, blowsOn(*got, "u:7"), "Aria is struck")
}

func TestGuardianInReachPlaced(t *testing.T) {
	// Aria front left, Tamsin front middle: the next column.
	b := placeBrawl(t, "me 1 1", "tamsin 1 2", "oswin 2 3")
	assert.NotContains(t, b.cmd("strategy", "tamsin guard me"), "Out of reach")
	got := b.listen()
	b.cmd("attack", fmt.Sprintf("#%d", b.bandits["bandit captain"][0]))
	for i := 0; i < 2 && len(guardEvents(*got, combatstream.GuardUsed)) == 0; i++ {
		b.toughen()
		b.hold(nil)
		for _, m := range b.livingBandits() {
			m.Character.SetAggro(7, 0, characters.DefaultAttack)
		}
		b.fight()
	}
	assert.NotEmpty(t, guardEvents(*got, combatstream.GuardUsed), "Tamsin guards from the next column")
}

// The battle summary counts the guards (29b's Guards line).
func TestGuardianInTheSummary(t *testing.T) {
	b := guardBrawl(t, "tamsin guard me")
	b.strike(0, false)
	b.fight()
	for _, m := range b.livingBandits() {
		m.Character.Health = 1
	}
	out := b.fightItOut(10)
	found := false
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Guards") {
			found = true
			assert.Contains(t, line, "Tamsin Reed")
		}
	}
	assert.True(t, found, "a Guards line: %s", out)
}
