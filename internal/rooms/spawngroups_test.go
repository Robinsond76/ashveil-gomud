package rooms

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func ids() func() string {
	n := 0
	return func() string {
		n++
		return fmt.Sprintf("g%d", n)
	}
}

func TestPlanSpawnGroupsFormsAndTopsUp(t *testing.T) {
	// One hostile mob alone: a group of one, topped up.
	plan := planSpawnGroups([]groupable{{InstanceId: 1, MobId: 10}}, ids())
	assert.Equal(t, map[int]string{1: "g1"}, plan.Assign)
	assert.Equal(t, []string{"g1"}, plan.TopUp)

	// A rat and a ruffian: one mixed group, nothing to top up.
	plan = planSpawnGroups([]groupable{{InstanceId: 1, MobId: 10}, {InstanceId: 2, MobId: 20}}, ids())
	assert.Equal(t, map[int]string{1: "g1", 2: "g1"}, plan.Assign)
	assert.Empty(t, plan.TopUp)

	// Six: two groups of three, never five and one.
	var six []groupable
	for i := 1; i <= 6; i++ {
		six = append(six, groupable{InstanceId: i, MobId: 10})
	}
	plan = planSpawnGroups(six, ids())
	count := map[string]int{}
	for _, g := range plan.Assign {
		count[g]++
	}
	assert.Equal(t, map[string]int{"g1": 3, "g2": 3}, count)
	assert.Empty(t, plan.TopUp)
}

func TestPlanSpawnGroupsJoinsAnIdleGroupWithRoom(t *testing.T) {
	ms := []groupable{
		{InstanceId: 1, MobId: 10, Group: "a"},
		{InstanceId: 2, MobId: 10, Group: "a"},
		{InstanceId: 3, MobId: 10, Group: "b"},
		{InstanceId: 4, MobId: 10, Group: "b"},
		{InstanceId: 5, MobId: 10, Group: "b"},
		{InstanceId: 6, MobId: 10}, // respawned
	}
	plan := planSpawnGroups(ms, ids())
	assert.Equal(t, map[int]string{6: "a"}, plan.Assign, "the smallest group with room")
	assert.Empty(t, plan.TopUp)

	// A group in a fight takes no one; the newcomer forms its own, topped up.
	ms[0].Fighting = true
	ms[2].Fighting = true
	plan = planSpawnGroups(ms, ids())
	assert.Equal(t, map[int]string{6: "g1"}, plan.Assign)
	assert.Equal(t, []string{"g1"}, plan.TopUp)

	// A full group takes no one.
	full := []groupable{}
	for i := 1; i <= 5; i++ {
		full = append(full, groupable{InstanceId: i, MobId: 10, Group: "a"})
	}
	full = append(full, groupable{InstanceId: 9, MobId: 10})
	plan = planSpawnGroups(full, ids())
	assert.Equal(t, map[int]string{9: "g1"}, plan.Assign)
}

func TestPlanSpawnGroupsLeavesAFightingGroupOfOne(t *testing.T) {
	plan := planSpawnGroups([]groupable{{InstanceId: 1, MobId: 10, Group: "a", Fighting: true}}, ids())
	assert.Empty(t, plan.Assign)
	assert.Empty(t, plan.TopUp, "a group down to one in a fight isn't reinforced")
}

func TestTopUpEntry(t *testing.T) {
	_, ok := topUpEntry(nil, 10)
	assert.False(t, ok)
	pool := []SpawnInfo{{MobId: 10}, {MobId: 20}}
	e, ok := topUpEntry(pool, 10)
	assert.True(t, ok)
	assert.Equal(t, 20, e.MobId, "another kind from the list first")
	e, _ = topUpEntry(pool, 30)
	assert.Equal(t, 10, e.MobId)
	e, _ = topUpEntry([]SpawnInfo{{MobId: 10}}, 10)
	assert.Equal(t, 10, e.MobId, "the only kind: another of the same")
}

func TestPlanSpawnGroupsNeverReinforcesASurvivor(t *testing.T) {
	// A group down to one after a fight, idle now: left as it is.
	plan := planSpawnGroups([]groupable{{InstanceId: 1, MobId: 10, Group: "a"}}, ids())
	assert.Empty(t, plan.Assign)
	assert.Empty(t, plan.TopUp, "a survivor isn't topped up")

	// A fresh spawn joins the survivor: a pair again, nothing new.
	plan = planSpawnGroups([]groupable{{InstanceId: 1, MobId: 10, Group: "a"}, {InstanceId: 2, MobId: 10}}, ids())
	assert.Equal(t, map[int]string{2: "a"}, plan.Assign)
	assert.Empty(t, plan.TopUp)
}

func TestPlanSpawnGroupsLeavesAnUngroupedFighterAlone(t *testing.T) {
	plan := planSpawnGroups([]groupable{{InstanceId: 1, MobId: 10, Fighting: true}}, ids())
	assert.Empty(t, plan.Assign, "grouped once its fight is over")
	assert.Empty(t, plan.TopUp)
}
