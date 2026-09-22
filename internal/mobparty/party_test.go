package mobparty_test

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssembleSoloMobsEachBecomeOwnParty(t *testing.T) {
	parties := mobparty.Assemble([]mobparty.MobSummary{
		{InstanceId: 1},
		{InstanceId: 2},
	})

	require.Len(t, parties, 2)
	for _, p := range parties {
		require.Len(t, p.Members, 1)
		row, col, ok := p.Formation.Find(company.MemberKey(memberKeyFor(p.Members[0])))
		require.True(t, ok)
		assert.Equal(t, 0, row, "solo party occupies the front row")
		assert.Equal(t, 1, col, "solo party occupies the center column")
	}
	assert.NotEqual(t, parties[0].ID, parties[1].ID)
}

func TestAssembleGroupsSharedTagIntoOneParty(t *testing.T) {
	parties := mobparty.Assemble([]mobparty.MobSummary{
		{InstanceId: 1, Groups: []string{"goblin-raiders"}},
		{InstanceId: 2, Groups: []string{"goblin-raiders"}},
		{InstanceId: 3, Groups: []string{"goblin-raiders"}},
		{InstanceId: 4}, // untagged, solo
	})

	require.Len(t, parties, 2)

	var grouped, solo mobparty.Party
	for _, p := range parties {
		if len(p.Members) == 3 {
			grouped = p
		} else {
			solo = p
		}
	}

	assert.ElementsMatch(t, []int{1, 2, 3}, grouped.Members)
	assert.Equal(t, []int{4}, solo.Members)
}

func TestAssembleOrdersFormationByEHPDescending(t *testing.T) {
	parties := mobparty.Assemble([]mobparty.MobSummary{
		{InstanceId: 1, Groups: []string{"pack"}, EHP: 10},
		{InstanceId: 2, Groups: []string{"pack"}, EHP: 50},
		{InstanceId: 3, Groups: []string{"pack"}, EHP: 30},
	})

	require.Len(t, parties, 1)
	f := parties[0].Formation

	// Highest EHP (instance 2) fills the front row first, left to right,
	// in descending EHP order: 2 (50), 3 (30), 1 (10).
	assert.Equal(t, company.MemberKey(memberKeyFor(2)), f.At(0, 0))
	assert.Equal(t, company.MemberKey(memberKeyFor(3)), f.At(0, 1))
	assert.Equal(t, company.MemberKey(memberKeyFor(1)), f.At(0, 2))
}

func TestAssembleCapsAtFiveAndSplits(t *testing.T) {
	members := make([]mobparty.MobSummary, 0, 6)
	for i := 1; i <= 6; i++ {
		members = append(members, mobparty.MobSummary{InstanceId: i, Groups: []string{"horde"}})
	}

	parties := mobparty.Assemble(members)

	require.Len(t, parties, 2)
	assert.Len(t, parties[0].Members, mobparty.MaxPartySize)
	assert.Len(t, parties[1].Members, 1)
	assert.ElementsMatch(t, []int{1, 2, 3, 4, 5}, parties[0].Members)
	assert.Equal(t, []int{6}, parties[1].Members)
	assert.NotEqual(t, parties[0].ID, parties[1].ID)
}

func memberKeyFor(instanceId int) string {
	return fmt.Sprintf("mob:%d", instanceId)
}
