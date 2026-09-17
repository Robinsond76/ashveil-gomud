package company_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormationPlaceAndFind(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(company.LeaderMemberKey, 2, 1))
	assert.Equal(t, company.LeaderMemberKey, f.At(2, 1))
	row, col, ok := f.Find(company.LeaderMemberKey)
	require.True(t, ok)
	assert.Equal(t, 2, row)
	assert.Equal(t, 1, col)
}

func TestFormationPlaceRejectsInvalidSlot(t *testing.T) {
	var f company.Formation
	assert.ErrorIs(t, f.Place(company.LeaderMemberKey, 3, 0), company.ErrInvalidSlot)
	assert.ErrorIs(t, f.Place(company.LeaderMemberKey, -1, 0), company.ErrInvalidSlot)
	assert.ErrorIs(t, f.Place(company.LeaderMemberKey, 0, 3), company.ErrInvalidSlot)
}

func TestFormationPlaceRejectsOccupiedSlot(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(company.LeaderMemberKey, 0, 0))
	err := f.Place(company.CompanionMemberKey(1), 0, 0)
	assert.ErrorIs(t, err, company.ErrSlotOccupied)
	// The rejected move must not have altered the grid.
	assert.Equal(t, company.LeaderMemberKey, f.At(0, 0))
	_, _, placed := f.Find(company.CompanionMemberKey(1))
	assert.False(t, placed)
}

func TestFormationPlaceMovesMemberClearingOldCell(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(company.LeaderMemberKey, 0, 0))
	require.NoError(t, f.Place(company.LeaderMemberKey, 2, 2))
	assert.Equal(t, company.MemberKey(""), f.At(0, 0))
	assert.Equal(t, company.LeaderMemberKey, f.At(2, 2))
}

func TestFormationSwapTwoMembers(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(company.LeaderMemberKey, 0, 0))
	require.NoError(t, f.Place(company.CompanionMemberKey(1), 2, 2))
	require.NoError(t, f.Swap(company.LeaderMemberKey, company.CompanionMemberKey(1)))
	assert.Equal(t, company.CompanionMemberKey(1), f.At(0, 0))
	assert.Equal(t, company.LeaderMemberKey, f.At(2, 2))
}

func TestFormationSwapRejectsUnknownMember(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(company.LeaderMemberKey, 0, 0))
	assert.ErrorIs(t, f.Swap(company.LeaderMemberKey, company.CompanionMemberKey(9)), company.ErrUnknownMember)
}

func TestFormationClearAndPrune(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(company.LeaderMemberKey, 0, 0))
	require.NoError(t, f.Place(company.CompanionMemberKey(1), 1, 1))

	f.Clear(company.LeaderMemberKey)
	assert.Equal(t, company.MemberKey(""), f.At(0, 0))
	assert.Equal(t, company.CompanionMemberKey(1), f.At(1, 1))

	f.Prune(map[company.MemberKey]bool{company.LeaderMemberKey: true})
	assert.Equal(t, company.MemberKey(""), f.At(1, 1))
}

func TestFormationPruneKeepsFirstDuplicateValidMember(t *testing.T) {
	var f company.Formation
	f[0][2] = company.LeaderMemberKey
	f[2][0] = company.LeaderMemberKey

	f.Prune(map[company.MemberKey]bool{company.LeaderMemberKey: true})

	assert.Equal(t, company.LeaderMemberKey, f.At(0, 2))
	assert.Equal(t, company.MemberKey(""), f.At(2, 0))
}
