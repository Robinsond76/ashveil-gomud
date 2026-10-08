package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// turnOrderFixture pins tempo to Speed / 10 and rolls to zero, so the
// order is decided by the slot rule alone.
func turnOrderFixture(t *testing.T) {
	t.Helper()
	users.ResetActiveUsers()
	ResetTempoForTest()
	t.Cleanup(users.ResetActiveUsers)
	t.Cleanup(ResetTempoForTest)
	t.Cleanup(UseTempoForTest(func(c *characters.Character) float64 { return float64(c.Stats.Speed.ValueAdj) / 10 }))
	t.Cleanup(UseTurnOrderRollForTest(func() float64 { return 0 }))
}

func orderNames(slots []turnSlot) []string {
	var out []string
	for _, s := range slots {
		name := ""
		if s.who.userId > 0 {
			name = users.GetByUserId(s.who.userId).Character.Name
		} else {
			name = engagementMobName(s.who.mobId)
		}
		if s.k > 1 {
			name += "#2"
		}
		out = append(out, name)
	}
	return out
}

func engagementMobName(id int) string {
	switch id {
	case 8801:
		return "fast-foe"
	case 8802:
		return "slow-foe"
	case 8803:
		return "twin-foe"
	}
	return "?"
}

func TestTurnOrderFastestFirstAndSecondTurnsInterleave(t *testing.T) {
	turnOrderFixture(t)
	u := users.NewUserRecord(8800, 1)
	users.SetTestUser(u)
	u.Character.Name = "leader"
	u.Character.Health = 50
	u.Character.Stats.Speed.ValueAdj = 10 // tempo 1.0: slot at 1.0
	u.Character.SetAggro(0, 8801, characters.DefaultAttack, 0)

	fast := engagementMob(t, 8801, 50, 1)
	fast.Character.Stats.Speed.ValueAdj = 15 // tempo 1.5: slots at 0.67 and 1.33
	fast.Character.SetAggro(8800, 0, characters.DefaultAttack, 0)
	slow := engagementMob(t, 8802, 50, 1)
	slow.Character.Stats.Speed.ValueAdj = 6 // tempo 0.6: slot at 1.67
	slow.Character.SetAggro(8800, 0, characters.DefaultAttack, 0)

	tempoTurns[caster{mobId: 8801}] = 2
	tempoTurns[caster{mobId: 8802}] = 1
	tempoTurns[caster{userId: 8800}] = 1

	got := orderNames(buildTurnOrder(5))
	assert.Equal(t, []string{"fast-foe", "leader", "fast-foe#2", "slow-foe"}, got,
		"a fast foe strikes before the player, its second turn comes before the slowest fighter's first")
}

func TestTurnOrderOpeningBonusMovesTheFirstSlotInTheFirstRoundOnly(t *testing.T) {
	turnOrderFixture(t)
	u := users.NewUserRecord(8800, 1)
	users.SetTestUser(u)
	u.Character.Name = "leader"
	u.Character.Health = 50
	u.Character.Stats.Speed.ValueAdj = 10
	u.Character.SetAggro(0, 8801, characters.DefaultAttack, 0)
	foe := engagementMob(t, 8801, 50, 1)
	foe.Character.Stats.Speed.ValueAdj = 12 // tempo 1.2: slot at 0.83, ahead of the leader's 1.0
	foe.Character.SetAggro(8800, 0, characters.DefaultAttack, 0)
	tempoTurns[caster{userId: 8800}] = 1
	tempoTurns[caster{mobId: 8801}] = 1

	// The leader opens the fight with half a turn on the meter (Iaijutsu):
	// in the round the meter opened, the slot is (1 - 50/100) / 1.0 = 0.5.
	st := &tempoState{char: u.Character, opened: 7}
	st.meter.Bonus = 50
	tempoMeters[caster{userId: 8800}] = st
	assert.Equal(t, []string{"leader", "fast-foe"}, orderNames(buildTurnOrder(7)), "the opening bonus strikes first")
	assert.Equal(t, []string{"fast-foe", "leader"}, orderNames(buildTurnOrder(8)), "the next round is by tempo alone")
}

func TestTurnOrderTiesBreakBySpeedThenPerceptionThenRoll(t *testing.T) {
	turnOrderFixture(t)
	a := engagementMob(t, 8801, 50, 1)
	b := engagementMob(t, 8802, 50, 1)
	c := engagementMob(t, 8803, 50, 1)
	for _, m := range []*mobs.Mob{a, b, c} {
		m.Character.Stats.Speed.ValueAdj = 10
		m.Character.SetAggro(8800, 0, characters.DefaultAttack, 0)
		tempoTurns[caster{mobId: m.InstanceId}] = 1
	}
	// The same tempo (pinned to Speed/10 the fixture can't tell apart), so
	// Perception decides between the first two...
	a.Character.Stats.Perception.ValueAdj = 1
	b.Character.Stats.Perception.ValueAdj = 5
	c.Character.Stats.Perception.ValueAdj = 5
	// ...and the roll between the second and third.
	rolls := []float64{0.9, 0.2, 0.1}
	i := 0
	restore := UseTurnOrderRollForTest(func() float64 { r := rolls[i%len(rolls)]; i++; return r })
	t.Cleanup(restore)
	got := orderNames(buildTurnOrder(1))
	require.Len(t, got, 3)
	assert.Equal(t, []string{"twin-foe", "slow-foe", "fast-foe"}, got, "Perception first, then the lowest roll")
}

func TestTurnOrderLeavesRetreatingPlayersToTheUpkeep(t *testing.T) {
	turnOrderFixture(t)
	u := users.NewUserRecord(8800, 1)
	users.SetTestUser(u)
	u.Character.Name = "leader"
	u.Character.Health = 50
	u.Character.Stats.Speed.ValueAdj = 10
	u.Character.SetAggro(0, 8801, characters.Retreat, 0)
	foe := engagementMob(t, 8801, 50, 1)
	foe.Character.Stats.Speed.ValueAdj = 10
	foe.Character.SetAggro(8800, 0, characters.DefaultAttack, 0)
	assert.Equal(t, []string{"fast-foe"}, orderNames(buildTurnOrder(1)), "a retreat is resolved before the order, not in it")
}
