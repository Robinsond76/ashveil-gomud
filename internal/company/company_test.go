package company_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestRegistryDismissLastCompanionRemovesEmptyRecord(t *testing.T) {
	registry := company.NewRegistry()
	first, err := registry.Summon(7, 58, allowed58(), 4)
	require.NoError(t, err)
	require.True(t, registry.Dismiss(7, first.ID))
	_, ok := registry.Get(7)
	assert.False(t, ok)
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
