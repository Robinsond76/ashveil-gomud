package company_test

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func twoCompanions(t *testing.T) *company.Registry {
	t.Helper()
	registry := company.NewRegistry()
	for i := 0; i < 2; i++ {
		_, err := registry.Summon(7, 58, allowed58(), 4)
		require.NoError(t, err)
	}
	return registry
}

func TestMarkDeadClearsCell(t *testing.T) {
	registry := twoCompanions(t)
	require.NoError(t, registry.PlaceMember(7, company.CompanionMemberKey(1), 0, 2))
	require.NoError(t, registry.MarkDead(7, 1, company.CompanionDeath{OpID: "op", Allowance: 100, Remaining: 100}))

	record, _ := registry.Get(7)
	assert.True(t, record.Companions[0].Dead())
	assert.False(t, record.Companions[1].Dead())
	_, _, placed := record.Formation.Find(company.CompanionMemberKey(1))
	assert.False(t, placed, "out of the formation")
	d := record.Companions[0].Death
	assert.True(t, d.Placed)
	assert.Equal(t, [2]int{0, 2}, [2]int{d.Row, d.Col}, "the cell is remembered")

	assert.ErrorIs(t, registry.MarkDead(7, 1, company.CompanionDeath{}), company.ErrMemberDead, "dies once")
	assert.ErrorIs(t, registry.MarkDead(7, 9, company.CompanionDeath{}), company.ErrUnknownMember)
	assert.ErrorIs(t, registry.PlaceMember(7, company.CompanionMemberKey(1), 1, 1), company.ErrMemberDead, "the dead aren't placed")
	require.NoError(t, registry.PlaceMember(7, company.CompanionMemberKey(2), 1, 1))
	assert.ErrorIs(t, registry.SwapMembers(7, company.CompanionMemberKey(1), company.CompanionMemberKey(2)), company.ErrMemberDead)
}

func TestReviveRestoresFreeCell(t *testing.T) {
	registry := twoCompanions(t)
	require.NoError(t, registry.PlaceMember(7, company.CompanionMemberKey(1), 0, 2))
	require.NoError(t, registry.MarkDead(7, 1, company.CompanionDeath{OpID: "op", Remaining: 5}))
	require.NoError(t, registry.Revive(7, 1))
	record, _ := registry.Get(7)
	assert.False(t, record.Companions[0].Dead())
	r, c, placed := record.Formation.Find(company.CompanionMemberKey(1))
	assert.True(t, placed)
	assert.Equal(t, [2]int{0, 2}, [2]int{r, c})
	assert.ErrorIs(t, registry.Revive(7, 1), company.ErrNotDead)
	assert.ErrorIs(t, registry.Revive(7, 9), company.ErrUnknownMember)
}

func TestReviveLeavesTakenCell(t *testing.T) {
	registry := twoCompanions(t)
	require.NoError(t, registry.PlaceMember(7, company.CompanionMemberKey(1), 0, 2))
	require.NoError(t, registry.MarkDead(7, 1, company.CompanionDeath{OpID: "op", Remaining: 5}))
	require.NoError(t, registry.PlaceMember(7, company.CompanionMemberKey(2), 0, 2))
	require.NoError(t, registry.Revive(7, 1))
	record, _ := registry.Get(7)
	_, _, placed := record.Formation.Find(company.CompanionMemberKey(1))
	assert.False(t, placed, "the cell was taken")
	assert.Equal(t, company.CompanionMemberKey(2), record.Formation.At(0, 2))
}

func TestSetRemaining(t *testing.T) {
	registry := twoCompanions(t)
	assert.ErrorIs(t, registry.SetRemaining(7, 1, 5, false), company.ErrNotDead)
	require.NoError(t, registry.MarkDead(7, 1, company.CompanionDeath{OpID: "op", Allowance: 10, Remaining: 10}))
	require.NoError(t, registry.SetRemaining(7, 1, -3, true))
	record, _ := registry.Get(7)
	assert.Equal(t, 0, record.Companions[0].Death.Remaining, "never below zero")
	assert.True(t, record.Companions[0].Death.Warned)
}

func TestLoseArchivesAndCaps(t *testing.T) {
	registry := company.NewRegistry()
	for i := 1; i <= company.MaxLost+2; i++ {
		c, err := registry.Summon(7, 58, allowed58(), 4)
		require.NoError(t, err)
		require.NoError(t, registry.MarkDead(7, c.ID, company.CompanionDeath{OpID: fmt.Sprintf("op-%d", i)}))
		require.True(t, registry.Lose(7, c.ID, company.LostCompanion{ID: c.ID, MobTemplateID: 58, Name: "dummy", Level: 3, OpID: fmt.Sprintf("op-%d", i)}))
	}
	record, ok := registry.Get(7)
	require.True(t, ok, "a record holding only the fallen is kept")
	assert.Empty(t, record.Companions)
	require.Len(t, record.Lost, company.MaxLost)
	assert.Equal(t, 3, record.Lost[0].ID, "the oldest are dropped")
	assert.Equal(t, company.MaxLost+2, record.Lost[company.MaxLost-1].ID)
	assert.Equal(t, company.MaxLost+3, record.NextCompanionID, "IDs are never reused")
	assert.False(t, registry.Lose(7, 99, company.LostCompanion{}))
}

func TestPutKeepsLostOnlyRecord(t *testing.T) {
	registry := company.NewRegistry()
	registry.Put(company.Record{LeaderUserID: 7, Lost: []company.LostCompanion{{ID: 1, Name: "x"}}})
	_, ok := registry.Get(7)
	assert.True(t, ok)
}

func TestGetClonesDeathAndLost(t *testing.T) {
	registry := twoCompanions(t)
	require.NoError(t, registry.MarkDead(7, 1, company.CompanionDeath{OpID: "op", Remaining: 5}))
	require.True(t, registry.Lose(7, 2, company.LostCompanion{ID: 2, Name: "x"}))
	record, _ := registry.Get(7)
	record.Companions[0].Death.Remaining = 99
	record.Lost[0].Name = "changed"
	again, _ := registry.Get(7)
	assert.Equal(t, 5, again.Companions[0].Death.Remaining)
	assert.Equal(t, "x", again.Lost[0].Name)
	clone := registry.Clone()
	clone.Companies[7].Companions[0].Death.Remaining = 42
	again, _ = registry.Get(7)
	assert.Equal(t, 5, again.Companions[0].Death.Remaining)
}

func TestResurrectionProviderNone(t *testing.T) {
	company.SetFormationProvider(nil)
	assert.Nil(t, company.DeadCompanions(7))
	_, err := company.ResurrectCompanion(7, "x", 1)
	assert.ErrorIs(t, err, company.ErrNoResurrection)
}

type fakeResurrector struct {
	fakeFormationProvider
	calls int
}

func (f *fakeResurrector) DeadCompanions(int) []company.DeadCompanionView {
	return []company.DeadCompanionView{{ID: 1, Name: "x", Level: 2, Remaining: 60}}
}

func (f *fakeResurrector) ResurrectCompanion(leader int, selector string, room int) (company.ResurrectionResult, error) {
	f.calls++
	return company.ResurrectionResult{ID: 1, Name: selector, Level: 1, Spawned: room > 0}, nil
}

func TestResurrectionProviderDelegates(t *testing.T) {
	fake := &fakeResurrector{}
	company.SetFormationProvider(fake)
	t.Cleanup(func() { company.SetFormationProvider(nil) })
	assert.Len(t, company.DeadCompanions(7), 1)
	result, err := company.ResurrectCompanion(7, "x", 5)
	require.NoError(t, err)
	assert.True(t, result.Spawned)
	assert.Equal(t, 1, fake.calls)
}

type fakeMembers struct{ fakeFormationProvider }

func (fakeMembers) CompanyMembers(int) ([]company.MemberView, bool) {
	return []company.MemberView{{ID: 1, Status: company.MemberDead, RescueSeconds: 60}}, true
}

func TestMemberViewProviderNone(t *testing.T) {
	company.SetFormationProvider(nil)
	_, ok := company.CompanyMembers(7)
	assert.False(t, ok)
	company.SetFormationProvider(fakeFormationProvider{})
	t.Cleanup(func() { company.SetFormationProvider(nil) })
	_, ok = company.CompanyMembers(7)
	assert.False(t, ok, "a provider without member views")
	company.SetFormationProvider(fakeMembers{})
	members, ok := company.CompanyMembers(7)
	assert.True(t, ok)
	assert.Equal(t, company.MemberDead, members[0].Status)
}
