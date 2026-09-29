package company

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 32d review: regression tests for the reviewer's findings, and the
// integration points the first pass left untested.

// hardenBandits makes every bandit outlast the test: aims are sticky, so
// they stay where they were set.
func (b *brawl) hardenBandits() {
	for _, m := range b.livingBandits() {
		m.Character.HealthMax.Value, m.Character.Health = 1000, 1000
	}
}

func TestAWizardCompanionCastsSingleThenArea(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypesFor("", map[int]string{1: "warrior", 2: "cleric", 3: "wizard", 4: "ranger"})
	b.unplaced()
	captain, _, _, _, _ := b.shapeBandits()
	garrick := b.companion(3)
	garrick.Character.ManaMax.Value, garrick.Character.Mana = 40, 40

	b.cmd("attack", fmt.Sprintf("#%d", captain))
	b.hardenBandits()
	aim := aimOf(&garrick.Character)
	require.NotZero(t, aim)
	b.toughen()
	out := b.fight()
	require.NotNil(t, garrick.Character.Aggro)
	assert.Equal(t, characters.SpellCast, garrick.Character.Aggro.Type, "a wizard companion casts")
	assert.Equal(t, "mm", garrick.Character.Aggro.SpellInfo.SpellId, "level 1: only Magic Missile")
	assert.Len(t, garrick.Character.Aggro.SpellInfo.TargetMobInstanceIds, 1)
	assert.Equal(t, 34, garrick.Character.Mana)
	assert.Contains(t, out, "Garrick Vane begins to chant", "mm.js narrates a mob caster")

	b.toughen()
	b.fight() // the spell ends
	require.NotNil(t, garrick.Character.Aggro)
	assert.Equal(t, characters.DefaultAttack, garrick.Character.Aggro.Type)
	assert.Equal(t, aim, aimOf(&garrick.Character), "back to his foe")

	// At level 5 he knows Shower of Sparks: five foes stand, so the area spell.
	garrick.Character.Level = 5
	b.toughen()
	b.fight()
	require.NotNil(t, garrick.Character.Aggro)
	assert.Equal(t, "sparks", garrick.Character.Aggro.SpellInfo.SpellId)
	assert.Len(t, garrick.Character.Aggro.SpellInfo.TargetMobInstanceIds, 5, "every foe of the battle")
	assert.Equal(t, 24, garrick.Character.Mana)
}

func TestAGroupHealWhenTwoAreHurt(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, _, _, _, _ := b.shapeBandits()
	oswin := b.companion(2)
	oswin.Character.Level = 5
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 20, 20
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	b.hardenBandits()
	b.toughen()
	b.aria.Character.Health = 300
	b.companion(1).Character.Health = 300
	b.fight()
	require.NotNil(t, oswin.Character.Aggro)
	assert.Equal(t, "healall", oswin.Character.Aggro.SpellInfo.SpellId)
	assert.Contains(t, oswin.Character.Aggro.SpellInfo.TargetUserIds, 7)
	assert.Contains(t, oswin.Character.Aggro.SpellInfo.TargetMobInstanceIds, b.companion(1).InstanceId)
	assert.Equal(t, 14, oswin.Character.Mana)
}

// Finding 2: a player who is down, bleeding out, is healed.
func TestADownedPlayerIsHealed(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, _, _, _, _ := b.shapeBandits()
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 20, 20
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	b.hardenBandits()
	b.toughen()
	b.fight()
	b.aria.Character.Health = -3
	b.fight()
	require.NotNil(t, oswin.Character.Aggro)
	assert.Equal(t, characters.SpellCast, oswin.Character.Aggro.Type)
	assert.Equal(t, []int{7}, oswin.Character.Aggro.SpellInfo.TargetUserIds, "Oswin heals the downed Aria")
}

// Finding 1: a leader on defend who turns by the rule isn't told she
// "can't reach" a foe she can.
func TestADefendingLeaderTurnsWithoutCantReach(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, _, slinger, _, _ := b.shapeBandits()
	b.cmd("strategy", "me defend")
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	b.hardenBandits()
	b.toughen()
	b.fight()
	b.aria.Character.SetAggro(0, captain, characters.DefaultAttack)

	tamsin := b.companion(1)
	for _, m := range b.livingBandits() {
		m.Character.SetAggro(0, b.companion(3).InstanceId, characters.DefaultAttack) // on Garrick, unhurt
	}
	tamsin.Character.Health = 200
	mobs.GetInstance(slinger).Character.SetAggro(0, tamsin.InstanceId, characters.DefaultAttack)
	out := b.fight()
	assert.Equal(t, slinger, aimOf(b.aria.Character), "she goes for the foe on Tamsin, the most hurt")
	assert.Contains(t, out, "You turn toward the bandit slinger.")
	assert.NotContains(t, out, "can't reach")
}

// Finding 3: no mana comes back between blows in a battle.
func TestNoManaBackBetweenBlowsInABattle(t *testing.T) {
	b := newBrawl(t)
	captain, _, _, _, _ := b.shapeBandits()
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	b.toughen()
	b.fight()
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 20, 5
	oswin.Character.Aggro = nil // a spell just ended with its foe gone
	b.aria.Character.ManaMax.Value, b.aria.Character.Mana = 20, 5
	b.aria.Character.Aggro = nil
	hooks.AutoHeal(events.NewRound{RoundNumber: 3})
	assert.Equal(t, 5, oswin.Character.Mana, "a companion in its leader's battle")
	assert.Equal(t, 5, b.aria.Character.Mana, "a player in a battle")
}

// Reach through the real attack: a rule's choice out of reach falls back to
// the nearest foe in reach (the design's "leader with a reach limit").
func TestARuleOutOfReachTakesTheNearestInReach(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, bruiser, slinger, _, _ := b.shapeBandits()
	require.Contains(t, b.cmd("formation", "move tamsin 1 3"), "Placed Tamsin Reed")
	b.cmd("strategy", "tamsin leader")

	b.cmd("attack", fmt.Sprintf("#%d", captain))
	aim := aimOf(&b.companion(1).Character)
	assert.NotEqual(t, captain, aim, "the captain (front left) is out of her reach from the right")
	assert.Contains(t, []int{bruiser, slinger}, aim, "the nearest foe she can reach")
	assert.Contains(t, b.cmd("formation", "reach tamsin"), mobs.GetInstance(aim).Character.Name)
}
