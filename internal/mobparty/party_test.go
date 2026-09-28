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
		{InstanceId: 1, Groups: []string{"goblin-raiders"}, Hostile: true},
		{InstanceId: 2, Groups: []string{"goblin-raiders"}, Hostile: true},
		{InstanceId: 3, Groups: []string{"goblin-raiders"}, Hostile: true},
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
		{InstanceId: 1, Groups: []string{"pack"}, Hostile: true, EHP: 10},
		{InstanceId: 2, Groups: []string{"pack"}, Hostile: true, EHP: 50},
		{InstanceId: 3, Groups: []string{"pack"}, Hostile: true, EHP: 30},
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
		members = append(members, mobparty.MobSummary{InstanceId: i, Groups: []string{"horde"}, Hostile: true})
	}

	parties := mobparty.Assemble(members)

	// Phase 29b2: split evenly, never leaving one alone.
	require.Len(t, parties, 2)
	assert.ElementsMatch(t, []int{1, 2, 3}, parties[0].Members)
	assert.ElementsMatch(t, []int{4, 5, 6}, parties[1].Members)
	assert.NotEqual(t, parties[0].ID, parties[1].ID)
}

func TestEvenSizes(t *testing.T) {
	assert.Nil(t, mobparty.EvenSizes(0, 5))
	assert.Equal(t, []int{1}, mobparty.EvenSizes(1, 5))
	assert.Equal(t, []int{5}, mobparty.EvenSizes(5, 5))
	assert.Equal(t, []int{3, 3}, mobparty.EvenSizes(6, 5))
	assert.Equal(t, []int{4, 3}, mobparty.EvenSizes(7, 5))
	assert.Equal(t, []int{4, 4, 3}, mobparty.EvenSizes(11, 5))
}

// TestAssembleGroupsBySpawnGroup (29b2): a spawn group decides a mob's
// party over its zone-wide group tag, so a rat and a ruffian spawned
// together are one party, and two spawn groups sharing a tag are two.
func TestAssembleGroupsBySpawnGroup(t *testing.T) {
	parties := mobparty.Assemble([]mobparty.MobSummary{
		{InstanceId: 1, SpawnGroup: "spawn:441:1", Groups: []string{"rats"}},
		{InstanceId: 2, SpawnGroup: "spawn:441:1", Groups: []string{"slum-ruffians"}},
		{InstanceId: 3, SpawnGroup: "spawn:441:2", Groups: []string{"rats"}},
		{InstanceId: 4, SpawnGroup: "spawn:441:2", Groups: []string{"rats"}},
		{InstanceId: 5, Groups: []string{"rats"}, Hostile: true},
	})
	require.Len(t, parties, 3)
	assert.ElementsMatch(t, []int{1, 2}, parties[0].Members)
	assert.Equal(t, "spawngroup:spawn:441:1", parties[0].ID)
	assert.ElementsMatch(t, []int{3, 4}, parties[1].Members)
	assert.Equal(t, []int{5}, parties[2].Members, "no spawn group: its tag, as before")
}

func memberKeyFor(instanceId int) string {
	return fmt.Sprintf("mob:%d", instanceId)
}

func TestMemberKeyForAndInstanceIdFromMemberKeyRoundTrip(t *testing.T) {
	key := mobparty.MemberKeyFor(42)
	id, ok := mobparty.InstanceIdFromMemberKey(key)
	require.True(t, ok)
	assert.Equal(t, 42, id)
}

func TestInstanceIdFromMemberKeyRejectsNonMobKeys(t *testing.T) {
	_, ok := mobparty.InstanceIdFromMemberKey(company.LeaderMemberKey)
	assert.False(t, ok)

	_, ok = mobparty.InstanceIdFromMemberKey(company.CompanionMemberKey(3))
	assert.False(t, ok)
}

// TestAPeacefulTagGroupsNobody (Phase 32d, the owner): non-hostile mobs
// sharing a tag (townsfolk) are each their own party; a spawn group still
// groups them, and hostile ones group by the tag as before.
func TestAPeacefulTagGroupsNobody(t *testing.T) {
	parties := mobparty.Assemble([]mobparty.MobSummary{
		{InstanceId: 1, Groups: []string{"townsfolk"}},
		{InstanceId: 2, Groups: []string{"townsfolk"}},
		{InstanceId: 3, SpawnGroup: "practice:7", Groups: []string{"straw"}},
		{InstanceId: 4, SpawnGroup: "practice:7", Groups: []string{"straw"}},
		{InstanceId: 5, Groups: []string{"wolves"}, Hostile: true},
		{InstanceId: 6, Groups: []string{"wolves"}, Hostile: true},
	})
	require.Len(t, parties, 4)
	assert.Equal(t, []int{1}, parties[0].Members)
	assert.Equal(t, []int{2}, parties[1].Members)
	assert.ElementsMatch(t, []int{3, 4}, parties[2].Members)
	assert.ElementsMatch(t, []int{5, 6}, parties[3].Members)
}
