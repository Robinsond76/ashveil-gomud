package company_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFormationProvider struct {
	formation company.Formation
	ok        bool
}

func (f fakeFormationProvider) FormationFor(leaderUserID int) (company.Formation, bool) {
	return f.formation, f.ok
}

func (f fakeFormationProvider) InstanceFor(leaderUserID, companionID int) (int, bool) {
	return 0, false
}

type fakeInstanceLookup struct {
	fakeFormationProvider
	instanceId int
	ok         bool
}

func (f fakeInstanceLookup) InstanceFor(leaderUserID, companionID int) (int, bool) {
	return f.instanceId, f.ok
}

func TestFormationForReturnsFalseWithNoProviderRegistered(t *testing.T) {
	company.SetFormationProvider(nil)

	_, ok := company.FormationFor(1)
	assert.False(t, ok)
}

func TestFormationForCallsThroughToRegisteredProvider(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(company.LeaderMemberKey, 0, 1))

	company.SetFormationProvider(fakeFormationProvider{formation: f, ok: true})
	defer company.SetFormationProvider(nil)

	got, ok := company.FormationFor(42)
	require.True(t, ok)
	assert.Equal(t, company.LeaderMemberKey, got.At(0, 1))
}

func TestInstanceForReturnsFalseWithNoProviderRegistered(t *testing.T) {
	company.SetFormationProvider(nil)

	_, ok := company.InstanceFor(1, 2)
	assert.False(t, ok)
}

func TestInstanceForCallsThroughToRegisteredProvider(t *testing.T) {
	company.SetFormationProvider(fakeInstanceLookup{instanceId: 55, ok: true})
	defer company.SetFormationProvider(nil)

	got, ok := company.InstanceFor(1, 2)
	require.True(t, ok)
	assert.Equal(t, 55, got)
}
