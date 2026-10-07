package company

import (
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompanyMembersStates (Phase 26a): present with vitals, awaiting, and
// dead with rescue time, in ID order, with archetype, level, and cell.
func TestCompanyMembersStates(t *testing.T) {
	module, _, runtime, _ := newDeathModule(t)
	runtime.vitals = map[int][2]int{102: {17, 30}}
	killOne(module)
	record, _ := module.registry.Get(7)
	record.Companions = append(record.Companions, domain.Companion{ID: 3, MobTemplateID: deathTemplate, Archetype: "ranger", State: &domain.MemberState{Level: 4}})
	module.registry.Put(record)
	require.NoError(t, module.registry.PlaceMember(7, domain.CompanionMemberKey(2), 1, 2))

	members, ok := module.CompanyMembers(7)
	require.True(t, ok)
	require.Len(t, members, 3)
	assert.Equal(t, domain.MemberView{ID: 1, Name: "#1", Status: domain.MemberDead, Level: 5, Alignment: 10, Personality: "cheerful", RescueSeconds: 3600, TrainingPoints: earnedTrainingPoints(5)}, members[0])
	assert.Equal(t, domain.MemberView{ID: 2, Name: "#2", Status: domain.MemberPresent, Level: 2, Alignment: 10, Personality: "grim", HP: 17, HPMax: 30, HPLimit: 30, Placed: true, Row: 1, Col: 2, TrainingPoints: earnedTrainingPoints(2)}, members[1])
	assert.Equal(t, domain.MemberView{ID: 3, Name: "#3", Status: domain.MemberAwaiting, Level: 4, Archetype: "ranger", Personality: "boastful", TrainingPoints: earnedTrainingPoints(4)}, members[2])

	none, ok := module.CompanyMembers(8)
	assert.True(t, ok)
	assert.Empty(t, none)
	module.loadErr = assert.AnError
	_, ok = module.CompanyMembers(7)
	assert.False(t, ok, "unreadable")
}

// TestCompanyMembersLiveLevel (review finding 5): a present companion with
// no saved state yet shows its live mob's level.
func TestCompanyMembersLiveLevel(t *testing.T) {
	module, _, runtime, _ := newDeathModule(t)
	record, _ := module.registry.Get(7)
	record.Companions[1].State = nil
	module.registry.Put(record)
	runtime.liveState = map[int]domain.MemberState{102: {Level: 6}}
	members, ok := module.CompanyMembers(7)
	require.True(t, ok)
	assert.Equal(t, 6, members[1].Level)
}

func TestHasClaimed(t *testing.T) {
	module, _, _, _ := newDeathModule(t)
	assert.False(t, module.HasClaimed(7, 61))
	require.NoError(t, module.registry.Claim(7, 61))
	assert.True(t, module.HasClaimed(7, 61))
	assert.False(t, module.HasClaimed(8, 61))
}

// TestCompanyMembersLiveProgress (Phase 32e): a present companion shows its
// live level and progress, ahead of its record between snapshots.
func TestCompanyMembersLiveProgress(t *testing.T) {
	module, _, runtime, _ := newDeathModule(t)
	runtime.progress = map[int][3]int{102: {5, 40, 160}}
	members, ok := module.CompanyMembers(7)
	require.True(t, ok)
	assert.Equal(t, 5, members[1].Level, "the live level, not the record's")
	assert.True(t, members[1].ExpKnown)
	assert.Equal(t, 40, members[1].ExpInto)
	assert.Equal(t, 160, members[1].ExpTNL)
	assert.False(t, members[0].ExpKnown, "the fallen show no progress")
}

// TestCompanyMembersMana (Phase 32g): a present companion carries its live
// mana; the fallen and the awaiting carry none.
func TestCompanyMembersMana(t *testing.T) {
	module, _, runtime, _ := newDeathModule(t)
	runtime.mana = map[int][2]int{102: {8, 20}}
	killOne(module)
	members, ok := module.CompanyMembers(7)
	require.True(t, ok)
	assert.Equal(t, 8, members[1].MP)
	assert.Equal(t, 20, members[1].MPMax)
	assert.Zero(t, members[0].MPMax, "the fallen carry no mana")
}
