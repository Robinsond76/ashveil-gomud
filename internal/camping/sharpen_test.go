package camping

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBladedSubtypes(t *testing.T) {
	for _, s := range []string{"slashing", "cleaving", "stabbing"} {
		assert.True(t, Bladed(s), s)
	}
	for _, s := range []string{"bludgeoning", "shooting", "claws", "whipping", "generic", ""} {
		assert.False(t, Bladed(s), s)
	}
}

func outcomes(plan SharpenPlan) map[int]SharpenOutcome {
	out := map[int]SharpenOutcome{}
	for _, e := range plan.Entries {
		out[e.Member.ID] = e.Outcome
	}
	return out
}

func TestPlanOneUsePerMemberEvenWithTwoBlades(t *testing.T) {
	plan := PlanSharpen([]SharpenMember{
		{ID: LeaderID, Name: "Hero", Present: true, Dull: 2},
		{ID: 1, Name: "Bran", Present: true, Dull: 1},
	}, 10)
	assert.Equal(t, 2, plan.UsesSpent)
	assert.Equal(t, map[int]SharpenOutcome{LeaderID: Sharpened, 1: Sharpened}, outcomes(plan))
}

func TestPlanSkipsSharpAndBladeless(t *testing.T) {
	plan := PlanSharpen([]SharpenMember{
		{ID: LeaderID, Name: "Hero", Present: true, Sharp: 1},
		{ID: 1, Name: "Bran", Present: true},
		{ID: 2, Name: "Cara", Present: true, Dull: 1, Sharp: 1},
	}, 10)
	assert.Equal(t, 1, plan.UsesSpent, "only Cara's dull blade costs a use")
	assert.Equal(t, map[int]SharpenOutcome{LeaderID: AlreadySharp, 1: NoBlade, 2: Sharpened}, outcomes(plan))
}

func TestPlanLeaderFirstThenByNumberAndLeftOut(t *testing.T) {
	plan := PlanSharpen([]SharpenMember{
		{ID: 3, Name: "Dag", Present: true, Dull: 1},
		{ID: 1, Name: "Bran", Present: true, Dull: 1},
		{ID: LeaderID, Name: "Hero", Present: true, Dull: 1},
		{ID: 2, Name: "Cara", Present: true, Sharp: 1},
	}, 2)
	assert.Equal(t, 2, plan.UsesSpent)
	ids := []int{}
	for _, e := range plan.Entries {
		ids = append(ids, e.Member.ID)
	}
	assert.Equal(t, []int{LeaderID, 1, 2, 3}, ids)
	assert.Equal(t, map[int]SharpenOutcome{LeaderID: Sharpened, 1: Sharpened, 2: AlreadySharp, 3: LeftOut}, outcomes(plan))
	assert.Equal(t, []string{"Dag"}, plan.Names(LeftOut))

	none := PlanSharpen([]SharpenMember{{ID: LeaderID, Name: "Hero", Present: true, Dull: 1}}, 0)
	assert.Zero(t, none.UsesSpent)
	assert.Equal(t, LeftOut, none.Entries[0].Outcome)
}

func TestPlanAbsentMembersCostNothing(t *testing.T) {
	plan := PlanSharpen([]SharpenMember{
		{ID: LeaderID, Name: "Hero", Present: true, Dull: 1},
		{ID: 1, Name: "Bran", Present: false, Dull: 1},
	}, 1)
	assert.Equal(t, 1, plan.UsesSpent)
	assert.Equal(t, map[int]SharpenOutcome{LeaderID: Sharpened, 1: NotHere}, outcomes(plan))
	assert.Equal(t, 1, plan.Count(Sharpened))
}
