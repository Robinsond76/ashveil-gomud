package company

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 30c1 wiring: company tactics through the real strategy module's
// store, attack, and combat round (shipped config, DoCombat).

// saveTactics stores Aria's tactics in the real strategy module.
func (b *brawl) saveTactics(t strategy.Tactics) {
	b.t.Helper()
	require.NoError(b.t, strategy.SaveTactics(7, t))
}

// hold keeps every bandit standing at the given health (of 1000), so the
// rules have fixed answers through a round.
func (b *brawl) hold(hp map[int]int) {
	for _, m := range b.livingBandits() {
		m.Character.HealthMax.Value, m.Character.Health = 1000, 800
		if v, ok := hp[m.InstanceId]; ok {
			m.Character.Health = v
		}
	}
}

func TestTacticsFocusOverridesStrategies(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, bruiser, slinger, _, _ := b.shapeBandits()
	b.cmd("strategy", "tamsin strongest")
	b.cmd("strategy", "ysolde wounded")
	// Garrick stands at the right of the front row: the captain (front
	// left) is out of his reach.
	require.Contains(t, b.cmd("formation", "move garrick 1 3"), "Placed Garrick Vane")
	b.saveTactics(strategy.Tactics{Focus: strategy.Leader})

	b.cmd("attack", fmt.Sprintf("#%d", slinger))
	assert.Equal(t, captain, aimOf(b.aria.Character), "Aria: the focus, their leader")
	assert.Equal(t, captain, aimOf(&b.companion(1).Character), "Tamsin: the focus over her strongest")
	assert.Equal(t, captain, aimOf(&b.companion(2).Character), "Oswin: the focus")
	assert.Equal(t, captain, aimOf(&b.companion(4).Character), "Ysolde: the focus over her wounded")
	garrick := aimOf(&b.companion(3).Character)
	assert.NotEqual(t, captain, garrick, "reach still binds")
	assert.Contains(t, []int{bruiser, slinger}, garrick, "the nearest he can reach")

	// Roles stay: Oswin, a healer, still heals under a focus.
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 20, 20
	b.hold(nil)
	b.toughen()
	b.aria.Character.Health = 300
	b.fight()
	require.NotNil(t, oswin.Character.Aggro)
	assert.Equal(t, characters.SpellCast, oswin.Character.Aggro.Type, "the healer heals")
	assert.Equal(t, []int{7}, oswin.Character.Aggro.SpellInfo.TargetUserIds)
}

func TestTacticsHealingThreshold(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, _, _, _, _ := b.shapeBandits()
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 20, 20
	b.cmd("attack", fmt.Sprintf("#%d", captain))

	// Aria at 60%: above the default half, Oswin swings.
	b.hold(nil)
	b.toughen()
	b.aria.Character.Health = 600
	b.fight()
	require.NotNil(t, oswin.Character.Aggro)
	assert.NotEqual(t, characters.SpellCast, oswin.Character.Aggro.Type, "60%: no heal at the default threshold")

	// Raised to 70: he heals her.
	b.saveTactics(strategy.Tactics{Healing: 70})
	b.hold(nil)
	b.toughen()
	b.aria.Character.Health = 600
	b.fight()
	require.NotNil(t, oswin.Character.Aggro)
	assert.Equal(t, characters.SpellCast, oswin.Character.Aggro.Type, "60%: healed under a 70 threshold")
	assert.Equal(t, []int{7}, oswin.Character.Aggro.SpellInfo.TargetUserIds)
}

func TestTacticsFocusMidBattleTurnsEveryoneOnce(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, bruiser, _, cutA, _ := b.shapeBandits()
	stream := b.listen()
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	require.Equal(t, cutA, aimOf(&b.companion(1).Character), "by her default, the weakest")
	b.hold(map[int]int{cutA: 700})
	b.toughen()
	b.fight() // the battle begins
	require.Equal(t, cutA, aimOf(&b.companion(1).Character))

	// The order: this battle's focus is the strongest.
	require.NoError(t, battle.SetFocus(7, string(strategy.Strongest)))
	b.hold(map[int]int{bruiser: 990, captain: 900})
	b.toughen()
	out := b.fight()
	assert.Equal(t, bruiser, aimOf(b.aria.Character), "Aria turns at once")
	for id := 1; id <= 4; id++ {
		c := &b.companion(id).Character
		if c.Aggro != nil && c.Aggro.Type == characters.SpellCast {
			continue
		}
		assert.Equal(t, bruiser, aimOf(c), "%s turns at once, from a foe still standing", c.Name)
	}
	assert.Contains(t, out, "You turn toward the bandit bruiser.")
	focus := 0
	for _, e := range *stream {
		if e.Kind == combatstream.FocusChange {
			focus++
			assert.Equal(t, "strongest", e.Rule)
			assert.Equal(t, 7, e.Source.UserId)
		}
	}
	assert.Equal(t, 1, focus, "one focus-change event")
	assert.True(t, battle.FocusReady(7), "the order is applied: another may be given")

	// The round after, aims stick: the captain is now the strongest, but
	// nobody turns.
	b.hold(map[int]int{bruiser: 900, captain: 990})
	b.toughen()
	b.fight()
	assert.Equal(t, bruiser, aimOf(b.aria.Character), "a kept aim is sticky after the turn")
	assert.Equal(t, bruiser, aimOf(&b.companion(1).Character))
	// The saved tactics never changed.
	assert.Equal(t, strategy.NoFocus, strategy.TacticsFor(7).Focus)
}

func TestTacticsFocusTurnsAPlayerAlone(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	pair := spawnedHostiles(t, 920103)
	require.Len(t, pair, 2)
	b.into(920103)
	pair[0].Character.HealthMax.Value, pair[0].Character.Health = 1000, 900
	pair[1].Character.HealthMax.Value, pair[1].Character.Health = 1000, 500

	b.cmd("attack", "ruffians")
	require.Equal(t, pair[1].InstanceId, aimOf(b.aria.Character), "the weakest, by default")
	b.aria.Character.HealthMax.Value, b.aria.Character.Health = 1000, 1000
	b.fight() // the battle begins
	require.Equal(t, pair[1].InstanceId, aimOf(b.aria.Character))
	pair[0].Character.Health, pair[1].Character.Health = 900, 500
	require.NoError(t, battle.SetFocus(7, string(strategy.Strongest)))
	b.aria.Character.HealthMax.Value, b.aria.Character.Health = 1000, 1000
	b.fight()
	assert.Equal(t, pair[0].InstanceId, aimOf(b.aria.Character), "alone, she turns by the focus at once")
	assert.NotNil(t, mobs.GetInstance(pair[1].InstanceId), "from a foe still standing")
}
