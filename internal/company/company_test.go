package company_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func allowed58() map[int]struct{} { return map[int]struct{}{58: {}} }

func TestRegistrySummonAssignsIncrementingIDs(t *testing.T) {
	registry := company.NewRegistry()
	first, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	second, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	assert.Equal(t, 1, first.ID)
	assert.Equal(t, 2, second.ID)

	record, ok := registry.Get(7)
	require.True(t, ok)
	assert.Equal(t, []company.Companion{first, second}, record.Companions)
}

func TestRegistrySummonEnforcesCap(t *testing.T) {
	registry := company.NewRegistry()
	for i := 0; i < 2; i++ {
		_, err := registry.Summon(7, 58, allowed58(), 2)
		require.NoError(t, err)
	}
	_, err := registry.Summon(7, 58, allowed58(), 2)
	assert.ErrorIs(t, err, company.ErrCompanyFull)
}

func TestRegistrySummonValidatesIDsAndAllowlist(t *testing.T) {
	tests := []struct {
		name     string
		leaderID int
		template int
		allowed  map[int]struct{}
		wantErr  error
	}{
		{name: "zero leader", leaderID: 0, template: 58, allowed: allowed58(), wantErr: company.ErrInvalidLeader},
		{name: "zero template", leaderID: 7, template: 0, allowed: map[int]struct{}{0: {}}, wantErr: company.ErrInvalidTemplate},
		{name: "disallowed template", leaderID: 7, template: 58, allowed: map[int]struct{}{59: {}}, wantErr: company.ErrTemplateNotAllowed},
		{name: "nil allowlist", leaderID: 7, template: 58, allowed: nil, wantErr: company.ErrTemplateNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := company.NewRegistry().Summon(tt.leaderID, tt.template, tt.allowed, 4)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestRegistryDismissRemovesOneAndPrunesItsFormationCells(t *testing.T) {
	registry := company.NewRegistry()
	first, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	second, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	require.NoError(t, registry.PlaceMember(7, company.CompanionMemberKey(first.ID), 0, 0))
	require.NoError(t, registry.PlaceMember(7, company.CompanionMemberKey(second.ID), 1, 1))

	assert.True(t, registry.Dismiss(7, first.ID))
	record, ok := registry.Get(7)
	require.True(t, ok)
	require.Len(t, record.Companions, 1)
	assert.Equal(t, second.ID, record.Companions[0].ID)
	assert.Equal(t, company.MemberKey(""), record.Formation.At(0, 0))
	assert.Equal(t, company.CompanionMemberKey(second.ID), record.Formation.At(1, 1))
	assert.False(t, registry.Dismiss(7, first.ID), "dismiss is idempotent")
}

func TestRegistryDismissLastCompanionKeepsLeaderPlacement(t *testing.T) {
	registry := company.NewRegistry()
	first, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	require.NoError(t, registry.PlaceMember(7, company.LeaderMemberKey, 2, 1))

	require.True(t, registry.Dismiss(7, first.ID))
	record, ok := registry.Get(7)
	require.True(t, ok, "a leader placement keeps the record alive")
	assert.Empty(t, record.Companions)
	assert.Equal(t, company.LeaderMemberKey, record.Formation.At(2, 1))
}

func TestRegistryDismissLastCompanionKeepsHighWaterMark(t *testing.T) {
	registry := company.NewRegistry()
	first, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	require.True(t, registry.Dismiss(7, first.ID))
	record, ok := registry.Get(7)
	require.True(t, ok, "an empty record must persist the companion-ID high-water mark")
	assert.Empty(t, record.Companions)
	assert.Equal(t, 2, record.NextCompanionID)
}

func TestRegistryNeverReusesDismissedHighestCompanionID(t *testing.T) {
	r := company.NewRegistry()
	_, err := r.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	second, err := r.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	require.True(t, r.Dismiss(7, second.ID))

	replacement, err := r.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	assert.Equal(t, 3, replacement.ID)
}

func TestRegistryNeverReusesIDsAfterDismissAll(t *testing.T) {
	r := company.NewRegistry()
	_, err := r.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	_, err = r.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	assert.Equal(t, 2, r.DismissAll(7))

	replacement, err := r.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	assert.Equal(t, 3, replacement.ID)
}

func TestRegistryHighWaterMarkSurvivesYAMLRoundTrip(t *testing.T) {
	r := company.NewRegistry()
	_, err := r.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	_, err = r.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	require.Equal(t, 2, r.DismissAll(7))

	encoded, err := yaml.Marshal(r)
	require.NoError(t, err)

	var reloaded company.Registry
	require.NoError(t, yaml.Unmarshal(encoded, &reloaded))
	record, ok := reloaded.Get(7)
	require.True(t, ok)
	assert.Empty(t, record.Companions)
	assert.Equal(t, 3, record.NextCompanionID)

	replacement, err := reloaded.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	assert.Equal(t, 3, replacement.ID)
}

func TestRegistryPutPreservesExplicitHighWaterMark(t *testing.T) {
	registry := company.NewRegistry()
	registry.Put(company.Record{LeaderUserID: 7, NextCompanionID: 9})
	record, ok := registry.Get(7)
	require.True(t, ok)
	assert.Equal(t, 9, record.NextCompanionID)
}

func TestRegistryDismissAllKeepsLeaderPlacement(t *testing.T) {
	registry := company.NewRegistry()
	first, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	second, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	require.NoError(t, registry.PlaceMember(7, company.LeaderMemberKey, 0, 0))
	require.NoError(t, registry.PlaceMember(7, company.CompanionMemberKey(first.ID), 0, 1))
	require.NoError(t, registry.PlaceMember(7, company.CompanionMemberKey(second.ID), 0, 2))

	assert.Equal(t, 2, registry.DismissAll(7))
	record, ok := registry.Get(7)
	require.True(t, ok)
	assert.Empty(t, record.Companions)
	assert.Equal(t, company.LeaderMemberKey, record.Formation.At(0, 0))
	assert.Equal(t, company.MemberKey(""), record.Formation.At(0, 1))
	assert.Equal(t, company.MemberKey(""), record.Formation.At(0, 2))
}

func TestRegistryPlaceMemberValidatesMembershipAndSlots(t *testing.T) {
	registry := company.NewRegistry()
	first, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)

	assert.ErrorIs(t, registry.PlaceMember(7, company.CompanionMemberKey(99), 0, 0), company.ErrUnknownMember)
	assert.ErrorIs(t, registry.PlaceMember(7, company.LeaderMemberKey, 3, 0), company.ErrInvalidSlot)

	require.NoError(t, registry.PlaceMember(7, company.CompanionMemberKey(first.ID), 0, 0))
	assert.ErrorIs(t, registry.PlaceMember(7, company.LeaderMemberKey, 0, 0), company.ErrSlotOccupied)
}

func TestRegistryLeaderFormationCreatesRecordWithoutCompanions(t *testing.T) {
	registry := company.NewRegistry()
	require.NoError(t, registry.PlaceMember(7, company.LeaderMemberKey, 1, 1))
	record, ok := registry.Get(7)
	require.True(t, ok)
	assert.Equal(t, company.LeaderMemberKey, record.Formation.At(1, 1))
}

func TestRegistryGetReturnsIndependentSnapshot(t *testing.T) {
	registry := company.NewRegistry()
	first, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	second, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)

	before, ok := registry.Get(7)
	require.True(t, ok)
	require.Len(t, before.Companions, 2)

	require.True(t, registry.Dismiss(7, first.ID))

	assert.Equal(t, []company.Companion{first, second}, before.Companions, "a retained snapshot must not change")
	after, _ := registry.Get(7)
	require.Len(t, after.Companions, 1)
	assert.Equal(t, second.ID, after.Companions[0].ID)
}

func TestRegistrySummonClampsCapToOne(t *testing.T) {
	registry := company.NewRegistry()
	_, err := registry.Summon(7, 58, allowed58(), 0)
	require.NoError(t, err)
	_, err = registry.Summon(7, 58, allowed58(), 0)
	assert.ErrorIs(t, err, company.ErrCompanyFull)
}

func TestRegistryPutPrunesStaleFormationKeys(t *testing.T) {
	registry := company.NewRegistry()
	require.NoError(t, registry.PlaceMember(7, company.LeaderMemberKey, 0, 0))
	record, _ := registry.Get(7)
	record.Formation[1][1] = company.CompanionMemberKey(99)
	registry.Put(record)
	got, _ := registry.Get(7)
	assert.Equal(t, company.MemberKey(""), got.Formation.At(1, 1), "Put prunes keys with no matching member")
	assert.Equal(t, company.LeaderMemberKey, got.Formation.At(0, 0))
}

func TestRegistrySwapAndClearMembers(t *testing.T) {
	registry := company.NewRegistry()
	first, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	require.NoError(t, registry.PlaceMember(7, company.LeaderMemberKey, 0, 0))
	require.NoError(t, registry.PlaceMember(7, company.CompanionMemberKey(first.ID), 2, 2))

	require.NoError(t, registry.SwapMembers(7, company.LeaderMemberKey, company.CompanionMemberKey(first.ID)))
	record, _ := registry.Get(7)
	assert.Equal(t, company.CompanionMemberKey(first.ID), record.Formation.At(0, 0))
	assert.Equal(t, company.LeaderMemberKey, record.Formation.At(2, 2))

	require.NoError(t, registry.ClearMember(7, company.CompanionMemberKey(first.ID)))
	record, _ = registry.Get(7)
	assert.Equal(t, company.MemberKey(""), record.Formation.At(0, 0))
	assert.ErrorIs(t, registry.ClearMember(7, company.CompanionMemberKey(first.ID)), company.ErrUnknownMember)
}

func TestRegistrySummonClampsCapToPartyMaximum(t *testing.T) {
	registry := company.NewRegistry()
	for i := 0; i < company.MaxCompanions; i++ {
		_, err := registry.Summon(7, 58, allowed58(), company.MaxCompanions+10)
		require.NoError(t, err)
	}
	_, err := registry.Summon(7, 58, allowed58(), company.MaxCompanions+10)
	assert.ErrorIs(t, err, company.ErrCompanyFull)
}

func TestRegistryPutKeepsFirstDuplicateFormationOccupant(t *testing.T) {
	registry := company.NewRegistry()
	record := company.Record{LeaderUserID: 7}
	record.Formation[0][0] = company.LeaderMemberKey
	record.Formation[1][1] = company.LeaderMemberKey
	registry.Put(record)

	got, ok := registry.Get(7)
	require.True(t, ok)
	assert.Equal(t, company.LeaderMemberKey, got.Formation.At(0, 0))
	assert.Equal(t, company.MemberKey(""), got.Formation.At(1, 1))
}

func TestRegistrySetCompanionArchetypeOnce(t *testing.T) {
	registry := company.NewRegistry()
	c, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)

	require.NoError(t, registry.SetCompanionArchetype(7, c.ID, "Warrior"))
	record, _ := registry.Get(7)
	assert.Equal(t, "warrior", record.Companions[0].Archetype)

	assert.ErrorIs(t, registry.SetCompanionArchetype(7, c.ID, "rogue"), company.ErrArchetypeAlreadySet)
	assert.ErrorIs(t, registry.SetCompanionArchetype(7, 99, "rogue"), company.ErrUnknownMember)
	assert.ErrorIs(t, registry.SetCompanionArchetype(8, c.ID, "rogue"), company.ErrUnknownMember)
	assert.ErrorIs(t, registry.SetCompanionArchetype(7, c.ID, " "), company.ErrInvalidArchetype)
}

func TestRegistryCompanionArchetypeRoundTripsAndLegacyIsEmpty(t *testing.T) {
	registry := company.NewRegistry()
	c, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	require.NoError(t, registry.SetCompanionArchetype(7, c.ID, "rogue"))

	data, err := yaml.Marshal(registry)
	require.NoError(t, err)
	var loaded company.Registry
	require.NoError(t, yaml.Unmarshal(data, &loaded))
	record, _ := loaded.Get(7)
	assert.Equal(t, "rogue", record.Companions[0].Archetype)

	var legacy company.Registry
	require.NoError(t, yaml.Unmarshal([]byte("companies:\n  7:\n    leader_user_id: 7\n    companions:\n      - id: 1\n        mob_template_id: 58\n"), &legacy))
	record, _ = legacy.Get(7)
	assert.Empty(t, record.Companions[0].Archetype, "legacy companions have no archetype")
}

// Phase 22c: tutorial recruit claims.

func TestClaimRecordsTemplateOnce(t *testing.T) {
	registry := company.NewRegistry()
	assert.ErrorIs(t, registry.Claim(0, 61), company.ErrInvalidLeader)
	assert.ErrorIs(t, registry.Claim(7, 0), company.ErrInvalidTemplate)
	require.NoError(t, registry.Claim(7, 61))
	require.NoError(t, registry.Claim(7, 61))
	require.NoError(t, registry.Claim(7, 62))
	record, ok := registry.Get(7)
	require.True(t, ok)
	assert.Equal(t, []int{61, 62}, record.Claimed)
	assert.True(t, record.HasClaimed(61))
	assert.False(t, record.HasClaimed(63))
}

func TestGetDeepCopiesClaims(t *testing.T) {
	registry := company.NewRegistry()
	require.NoError(t, registry.Claim(7, 61))
	record, _ := registry.Get(7)
	record.Claimed[0] = 99
	again, _ := registry.Get(7)
	assert.Equal(t, []int{61}, again.Claimed)
	clone := registry.Clone()
	cloned := clone.Companies[7]
	cloned.Claimed[0] = 98
	again, _ = registry.Get(7)
	assert.Equal(t, []int{61}, again.Claimed)
}

func TestPutKeepsClaimsOnlyRecord(t *testing.T) {
	registry := company.NewRegistry()
	c, err := registry.Summon(7, 61, map[int]struct{}{61: {}}, 4)
	require.NoError(t, err)
	require.NoError(t, registry.Claim(7, 61))
	assert.Equal(t, 1, registry.DismissAll(7))
	record, ok := registry.Get(7)
	require.True(t, ok, "a claim outlives the last companion")
	assert.True(t, record.HasClaimed(61))

	// Even with no ID high-water mark, a claims-only record is kept.
	registry.Put(company.Record{LeaderUserID: 8, Claimed: []int{62}})
	_, ok = registry.Get(8)
	assert.True(t, ok)
	assert.False(t, registry.Dismiss(7, c.ID), "already dismissed")
}

func TestClaimsYAMLRoundTrip(t *testing.T) {
	registry := company.NewRegistry()
	require.NoError(t, registry.Claim(7, 61))
	data, err := yaml.Marshal(registry)
	require.NoError(t, err)
	assert.Contains(t, string(data), "claimed:")
	loaded := company.NewRegistry()
	require.NoError(t, yaml.Unmarshal(data, loaded))
	record, ok := loaded.Get(7)
	require.True(t, ok)
	assert.Equal(t, []int{61}, record.Claimed)
}
