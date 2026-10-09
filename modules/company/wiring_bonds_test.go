package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/bonds"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 65 wiring: bonds in the real battle round (shipped config,
// DoCombat), in the guardian brawl: a friend steps in for a hurt friend, a
// guardian set to guard a rival won't, and each moves the pair's bond.

// setBond saves a bond between two of leader 7's companions.
func setBond(t *testing.T, a, b, value int) {
	t.Helper()
	record, ok := module.registry.Get(7)
	require.True(t, ok)
	record.SetBond(domain.Bond{A: a, B: b, Value: value})
	module.registry.Put(record)
}

func bondOf(t *testing.T, a, b int) domain.Bond {
	t.Helper()
	record, _ := module.registry.Get(7)
	bond, _ := record.BondOf(a, b)
	return bond
}

func TestAFriendStepsInForAHurtFriend(t *testing.T) {
	b := guardBrawl(t)
	setBond(t, 1, 3, 40) // Tamsin trusts Garrick
	b.companion(3).Character.Health = 300
	got := b.listen()
	b.strike(3, false)
	out := b.fight()

	assert.Contains(t, out, "Tamsin Reed steps in front of Garrick Vane for a friend. (bond guard, none left)")
	used := guardEvents(*got, combatstream.GuardUsed)
	require.Len(t, used, 1)
	assert.Equal(t, "bond", used[0].Status)
	assert.Equal(t, b.companion(1).InstanceId, used[0].Source.MobInstanceId)
	assert.Equal(t, 43, bondOf(t, 1, 3).Value, "a rescue draws them closer")

	// One step a battle: Tamsin has no more for a second blow, and the bond
	// is inside its rescue cooldown.
	b.toughen()
	b.companion(3).Character.Health = 300
	b.strike(3, false)
	out = b.fight()
	assert.NotContains(t, out, "bond guard")
	assert.Equal(t, 43, bondOf(t, 1, 3).Value)
}

func TestAFriendDoesNotStepInForAnUnhurtOrUnboundMember(t *testing.T) {
	b := guardBrawl(t)
	setBond(t, 1, 3, 40)
	b.strike(3, false)
	assert.NotContains(t, b.fight(), "bond guard", "Garrick is unhurt")

	b = guardBrawl(t)
	setBond(t, 1, 3, 10) // acquaintances
	b.companion(3).Character.Health = 300
	b.strike(3, false)
	assert.NotContains(t, b.fight(), "bond guard", "not friends yet")
}

func TestKinStepInTwiceABattle(t *testing.T) {
	b := guardBrawl(t)
	setBond(t, 1, 3, 90)
	got := b.listen()
	b.companion(3).Character.Health = 300
	b.strike(3, false)
	out := b.fight()
	assert.Contains(t, out, "(bond guard, 1 left)")
	b.toughen()
	b.companion(3).Character.Health = 300
	b.strike(3, false)
	out = b.fight()
	assert.Contains(t, out, "(bond guard, none left)")
	assert.Len(t, guardEvents(*got, combatstream.GuardUsed), 2)
}

func TestAGuardianSetToGuardARivalWontAndTheBondSours(t *testing.T) {
	b := guardBrawl(t, "tamsin guard garrick")
	setBond(t, 1, 3, -60)
	got := b.listen()
	b.strike(3, false)
	out := b.fight()

	assert.Contains(t, out, "Tamsin Reed lets the blow fall on Garrick Vane. (no guard: they can't stand each other)")
	assert.Empty(t, guardEvents(*got, combatstream.GuardUsed), "no guard was made")
	refused := guardEvents(*got, combatstream.GuardRefused)
	require.Len(t, refused, 1)
	assert.Equal(t, "bond", refused[0].Status)
	assert.Equal(t, -62, bondOf(t, 1, 3).Value)
	assert.GreaterOrEqual(t, blowsOn(*got, "m:"+itoa(b.companion(3).InstanceId)), 1, "Garrick took the blow himself")

	// Said once a battle.
	b.toughen()
	b.strike(3, false)
	assert.NotContains(t, b.fight(), "lets the blow fall")
}

func TestAGuardianWithNoWardPassesOverARival(t *testing.T) {
	b := guardBrawl(t, "tamsin guardian")
	setBond(t, 1, 3, -60)
	b.companion(3).Character.Health = 300 // the most hurt, but a rival
	got := b.listen()
	b.strike(3, false)
	out := b.fight()
	assert.NotContains(t, out, "steps in front of Garrick")
	assert.Empty(t, guardEvents(*got, combatstream.GuardUsed))
	assert.Equal(t, -60, bondOf(t, 1, 3).Value, "passing over a rival is not a clash")
}

func TestAnOrdinaryGuardForACompanionDrawsThePairCloser(t *testing.T) {
	b := guardBrawl(t, "tamsin guard garrick")
	got := b.listen()
	b.strike(3, false)
	out := b.fight()
	assert.Contains(t, out, "Tamsin Reed steps in front of Garrick Vane. (guard, 1 left)")
	assert.Len(t, guardEvents(*got, combatstream.GuardUsed), 1)
	assert.Equal(t, bonds.RescueGain, bondOf(t, 1, 3).Value, "stepping in for a companion is a rescue")
}

func TestAnOrdinaryGuardForTheLeaderMovesNoBond(t *testing.T) {
	b := guardBrawl(t, "tamsin guard me")
	b.strike(0, false)
	b.fight()
	record, _ := module.registry.Get(7)
	assert.Empty(t, record.Bonds, "bonds are between companions")
}

func TestACompanyMakesTwoBondStepsABattleAtMost(t *testing.T) {
	b := guardBrawl(t)
	// Tamsin and Oswin are kin of Garrick; each could step in twice, but the
	// company only makes two bond steps in a battle.
	setBond(t, 1, 3, 90)
	setBond(t, 2, 3, 90)
	got := b.listen()
	steps := 0
	for round := 0; round < 3; round++ {
		b.toughen()
		b.companion(3).Character.Health = 300
		b.strike(3, false)
		b.fight()
		steps = len(guardEvents(*got, combatstream.GuardUsed))
	}
	assert.Equal(t, 2, steps, "three rounds of blows on a hurt friend, two steps in all")
}

func TestAFriendSteppingInNeedsTheWardAtFortyPercentOrLess(t *testing.T) {
	b := guardBrawl(t)
	setBond(t, 1, 3, 40)
	b.companion(3).Character.Health = 450 // 45%
	b.strike(3, false)
	assert.NotContains(t, b.fight(), "bond guard")
}
