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

func (f fakeFormationProvider) LeaderAndKeyForInstance(instanceId int) (int, company.MemberKey, bool) {
	return 0, "", false
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

type fakeLeaderLookup struct {
	fakeFormationProvider
	leaderUserID int
	key          company.MemberKey
	found        bool
}

func (f fakeLeaderLookup) LeaderAndKeyForInstance(instanceId int) (int, company.MemberKey, bool) {
	return f.leaderUserID, f.key, f.found
}

func TestLeaderAndKeyForInstanceReturnsFalseWithNoProviderRegistered(t *testing.T) {
	company.SetFormationProvider(nil)

	_, _, ok := company.LeaderAndKeyForInstance(9)
	assert.False(t, ok)
}

func TestLeaderAndKeyForInstanceCallsThroughToRegisteredProvider(t *testing.T) {
	company.SetFormationProvider(fakeLeaderLookup{leaderUserID: 7, key: company.CompanionMemberKey(3), found: true})
	defer company.SetFormationProvider(nil)

	leaderUserID, key, ok := company.LeaderAndKeyForInstance(9)
	require.True(t, ok)
	assert.Equal(t, 7, leaderUserID)
	assert.Equal(t, company.CompanionMemberKey(3), key)
}

type alignmentStub struct {
	fakeFormationProviderForAlignment
}

type fakeFormationProviderForAlignment struct{}

func (fakeFormationProviderForAlignment) FormationFor(int) (company.Formation, bool) {
	return company.Formation{}, false
}
func (fakeFormationProviderForAlignment) InstanceFor(int, int) (int, bool) { return 0, false }
func (fakeFormationProviderForAlignment) LeaderAndKeyForInstance(int) (int, company.MemberKey, bool) {
	return 0, "", false
}
func (alignmentStub) CompanyAlignment(leader int) (int, bool) { return leader * 10, leader > 0 }

func TestCompanyAlignmentSeam(t *testing.T) {
	company.SetFormationProvider(nil)
	_, ok := company.CompanyAlignment(3)
	assert.False(t, ok, "no provider")
	company.SetFormationProvider(fakeFormationProviderForAlignment{})
	_, ok = company.CompanyAlignment(3)
	assert.False(t, ok, "provider without alignment")
	company.SetFormationProvider(alignmentStub{})
	t.Cleanup(func() { company.SetFormationProvider(nil) })
	got, ok := company.CompanyAlignment(3)
	assert.True(t, ok)
	assert.Equal(t, 30, got)
}
