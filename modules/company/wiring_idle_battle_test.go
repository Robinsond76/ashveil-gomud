package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestABattleGoesOnWhenEveryAimFellInOneRound (Phase 83): when the last
// target of every fighter on both sides fell in the same round (the leader's
// foe, a companion's winding shot, a bandit's mark among the company), the
// next round's upkeep still sees a fight, turns the company onto the foes
// that stand, and the battle is not broken off and opened again as a second
// fight. It was, about one run in three hundred of the battle-event test:
// engagedWith only counted an aim at someone still alive, so no aim was a
// fight over.
func TestABattleGoesOnWhenEveryAimFellInOneRound(t *testing.T) {
	b := newBrawl(t)
	got := b.listen()
	b.toughen()
	b.aimAt("bandit captain")
	b.fight()
	opened, ok := battle.Current(7)
	require.True(t, ok, "the fight is a battle")
	require.NotEmpty(t, b.livingBandits(), "bandits stand")

	fallen := mobs.NewMobById(mobs.MobId(9101), b.road.RoomId)
	require.NotNil(t, fallen)
	fallen.Character.Health = 0
	aimAtTheFallen := func(c *characters.Character) {
		c.SetAggro(0, fallen.InstanceId, characters.DefaultAttack)
	}
	aimAtTheFallen(b.aria.Character)
	for id := 1; id <= 4; id++ {
		aimAtTheFallen(&b.companion(id).Character)
	}
	for _, m := range b.livingBandits() {
		aimAtTheFallen(&m.Character)
	}

	b.toughen()
	b.fight()
	now, ok := battle.Current(7)
	require.True(t, ok, "the battle goes on")
	assert.Equal(t, opened.FightID, now.FightID, "the same fight")
	for _, e := range *got {
		assert.NotEqual(t, combatstream.FightEnd, e.Kind, "the fight did not end: %+v", e)
	}
}
