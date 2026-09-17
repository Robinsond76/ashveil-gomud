package survival

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBandForCoversEveryBoundary(t *testing.T) {
	cases := []struct {
		value int
		want  Band
	}{
		{-5, BandDepleted},
		{0, BandDepleted},
		{1, BandCritical},
		{25, BandCritical},
		{26, BandLow},
		{50, BandLow},
		{51, BandSteady},
		{75, BandSteady},
		{76, BandFull},
		{100, BandFull},
		{150, BandFull},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, BandFor(tc.value), "value %d", tc.value)
	}
}

func TestNormalizeClampsOutOfRangeValues(t *testing.T) {
	got := Normalize(Needs{Hunger: -10, Thirst: 120, Fatigue: 50})
	assert.Equal(t, Needs{Hunger: 0, Thirst: 100, Fatigue: 50}, got)
}

func TestEnsureInitializesFullySuppliedMember(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, LeaderMemberKey))

	needs, ok := r.NeedsFor(7, LeaderMemberKey)
	require.True(t, ok)
	assert.Equal(t, FullNeeds(), needs)
}

func TestEnsureIsIdempotent(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, CompanionMemberKey(2)))
	require.NoError(t, r.PutNeeds(7, CompanionMemberKey(2), Needs{Hunger: 40, Thirst: 40, Fatigue: 40}))
	require.NoError(t, r.Ensure(7, CompanionMemberKey(2)))

	needs, _ := r.NeedsFor(7, CompanionMemberKey(2))
	assert.Equal(t, 40, needs.Hunger)
}

func TestConsumeFoodCapsNeedAndReportsCrossing(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, LeaderMemberKey))
	require.NoError(t, r.PutNeeds(7, LeaderMemberKey, Needs{Hunger: 25, Thirst: 100, Fatigue: 100}))

	change, _, err := r.ConsumeFood(7, LeaderMemberKey, 90, 0)
	require.NoError(t, err)
	assert.Equal(t, BandCritical, change.Before)
	assert.Equal(t, BandFull, change.After)
	assert.Equal(t, 100, r.MustNeedsFor(7, LeaderMemberKey).Hunger)
}

func TestConsumeFoodAppliesBothNutritionAndHydration(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, LeaderMemberKey))
	require.NoError(t, r.PutNeeds(7, LeaderMemberKey, Needs{Hunger: 10, Thirst: 10, Fatigue: 100}))

	hunger, thirst, err := r.ConsumeFood(7, LeaderMemberKey, 30, 20)
	require.NoError(t, err)
	assert.Equal(t, BandCritical, hunger.Before)
	assert.Equal(t, BandLow, hunger.After)
	assert.Equal(t, BandCritical, thirst.Before)
	assert.Equal(t, BandLow, thirst.After)
}

func TestConsumeFoodRejectsNonPositiveBenefits(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, LeaderMemberKey))
	before := r.MustNeedsFor(7, LeaderMemberKey)

	for _, tc := range []struct{ nutrition, hydration int }{
		{0, 0},
		{-1, 0},
		{0, -1},
		{-5, -5},
	} {
		_, _, err := r.ConsumeFood(7, LeaderMemberKey, tc.nutrition, tc.hydration)
		assert.ErrorIs(t, err, ErrInvalidAmount, "%+v", tc)
	}
	assert.Equal(t, before, r.MustNeedsFor(7, LeaderMemberKey))
}

func TestConsumeWaterIncreasesOnlyThirst(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, LeaderMemberKey))
	require.NoError(t, r.PutNeeds(7, LeaderMemberKey, Needs{Hunger: 30, Thirst: 20, Fatigue: 30}))

	change, err := r.ConsumeWater(7, LeaderMemberKey, 40)
	require.NoError(t, err)
	assert.Equal(t, BandCritical, change.Before)
	assert.Equal(t, BandSteady, change.After)
	assert.Equal(t, Needs{Hunger: 30, Thirst: 60, Fatigue: 30}, r.MustNeedsFor(7, LeaderMemberKey))
}

func TestConsumeWaterRejectsNonPositive(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, LeaderMemberKey))
	_, err := r.ConsumeWater(7, LeaderMemberKey, 0)
	assert.ErrorIs(t, err, ErrInvalidAmount)
}

func TestApplyRestRecoveryTouchesOnlyFatigue(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, LeaderMemberKey))
	require.NoError(t, r.PutNeeds(7, LeaderMemberKey, Needs{Hunger: 30, Thirst: 40, Fatigue: 10}))

	change, err := r.ApplyRestRecovery(7, LeaderMemberKey, 20)
	require.NoError(t, err)
	assert.Equal(t, BandCritical, change.Before)
	assert.Equal(t, BandLow, change.After)
	assert.Equal(t, Needs{Hunger: 30, Thirst: 40, Fatigue: 30}, r.MustNeedsFor(7, LeaderMemberKey))
}

func TestApplyRestRecoveryRejectsNonPositive(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, LeaderMemberKey))
	_, err := r.ApplyRestRecovery(7, LeaderMemberKey, -3)
	assert.ErrorIs(t, err, ErrInvalidAmount)
}

func TestApplyExertionReducesRequestedNeedsOnly(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, LeaderMemberKey))
	require.NoError(t, r.PutNeeds(7, LeaderMemberKey, Needs{Hunger: 80, Thirst: 80, Fatigue: 80}))

	hunger, thirst, fatigue, err := r.ApplyExertion(7, LeaderMemberKey, Exertion{Hunger: 10, Thirst: 5, Fatigue: 2})
	require.NoError(t, err)
	assert.Equal(t, BandFull, hunger.Before)
	assert.Equal(t, BandSteady, hunger.After)
	assert.Equal(t, BandSteady, thirst.After)
	assert.Equal(t, BandFull, fatigue.After)
	assert.Equal(t, Needs{Hunger: 70, Thirst: 75, Fatigue: 78}, r.MustNeedsFor(7, LeaderMemberKey))
}

func TestApplyExertionReportsExactBoundaryCrossing(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, LeaderMemberKey))
	require.NoError(t, r.PutNeeds(7, LeaderMemberKey, Needs{Hunger: 26, Thirst: 100, Fatigue: 100}))

	hunger, _, _, err := r.ApplyExertion(7, LeaderMemberKey, Exertion{Hunger: 1})
	require.NoError(t, err)
	assert.Equal(t, BandLow, hunger.Before)
	assert.Equal(t, BandCritical, hunger.After)
	assert.Equal(t, 25, r.MustNeedsFor(7, LeaderMemberKey).Hunger)
}

func TestApplyExertionClampsAtZero(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, LeaderMemberKey))
	require.NoError(t, r.PutNeeds(7, LeaderMemberKey, Needs{Hunger: 3, Thirst: 3, Fatigue: 3}))

	_, _, _, err := r.ApplyExertion(7, LeaderMemberKey, Exertion{Hunger: 50, Thirst: 50, Fatigue: 50})
	require.NoError(t, err)
	assert.Equal(t, Needs{Hunger: 0, Thirst: 0, Fatigue: 0}, r.MustNeedsFor(7, LeaderMemberKey))
}

func TestApplyExertionRejectsNonPositiveCosts(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, LeaderMemberKey))
	for _, cost := range []Exertion{
		{},
		{Hunger: -1},
		{Thirst: -1},
		{Fatigue: -1},
	} {
		_, _, _, err := r.ApplyExertion(7, LeaderMemberKey, cost)
		assert.ErrorIs(t, err, ErrInvalidAmount, "%+v", cost)
	}
}

func TestMutationsRejectInvalidMemberIdentity(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, LeaderMemberKey))

	assert.ErrorIs(t, r.Ensure(0, LeaderMemberKey), ErrInvalidMember)
	assert.ErrorIs(t, r.Ensure(7, MemberKey("")), ErrInvalidMember)
	assert.ErrorIs(t, r.Ensure(7, MemberKey("companion:0")), ErrInvalidMember)
	assert.ErrorIs(t, r.Ensure(7, MemberKey("bogus")), ErrInvalidMember)

	_, _, err := r.ConsumeFood(0, LeaderMemberKey, 10, 0)
	assert.ErrorIs(t, err, ErrInvalidMember)
	_, _, err = r.ConsumeFood(7, MemberKey("bogus"), 10, 0)
	assert.ErrorIs(t, err, ErrInvalidMember)
}

func TestMutationsRejectUnknownMember(t *testing.T) {
	r := NewRegistry()
	_, _, err := r.ConsumeFood(7, LeaderMemberKey, 10, 0)
	assert.ErrorIs(t, err, ErrUnknownMember)

	_, err = r.ApplyRestRecovery(7, CompanionMemberKey(1), 10)
	assert.ErrorIs(t, err, ErrUnknownMember)
}

func TestNeedsForReturnsNormalizedValues(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.PutNeeds(7, LeaderMemberKey, Needs{Hunger: -20, Thirst: 999, Fatigue: 42}))

	needs, ok := r.NeedsFor(7, LeaderMemberKey)
	require.True(t, ok)
	assert.Equal(t, Needs{Hunger: 0, Thirst: 100, Fatigue: 42}, needs)
}

func TestRemoveDeletesMemberAndEmptyLeader(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, CompanionMemberKey(1)))
	r.Remove(7, CompanionMemberKey(1))

	_, ok := r.NeedsFor(7, CompanionMemberKey(1))
	assert.False(t, ok)
	assert.NotContains(t, r.Leaders, 7)
}

func TestRemoveLeaderDropsEveryMember(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, LeaderMemberKey))
	require.NoError(t, r.Ensure(7, CompanionMemberKey(1)))
	r.RemoveLeader(7)

	_, ok := r.NeedsFor(7, LeaderMemberKey)
	assert.False(t, ok)
	_, ok = r.NeedsFor(7, CompanionMemberKey(1))
	assert.False(t, ok)
}

func TestMembersListsLeaderFirstThenCompanionsInOrder(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, CompanionMemberKey(2)))
	require.NoError(t, r.Ensure(7, CompanionMemberKey(1)))
	require.NoError(t, r.Ensure(7, LeaderMemberKey))

	assert.Equal(t, []MemberKey{LeaderMemberKey, CompanionMemberKey(1), CompanionMemberKey(2)}, r.Members(7))
}

func TestCloneIsDeepCopy(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Ensure(7, LeaderMemberKey))
	clone := r.Clone()

	require.NoError(t, r.PutNeeds(7, LeaderMemberKey, Needs{Hunger: 1, Thirst: 1, Fatigue: 1}))
	assert.Equal(t, FullNeeds(), clone.MustNeedsFor(7, LeaderMemberKey))
}

func TestNeedLabelsFollowBands(t *testing.T) {
	assert.Equal(t, "Well fed", HungerLabel(100))
	assert.Equal(t, "Sated", HungerLabel(60))
	assert.Equal(t, "Hungry", HungerLabel(30))
	assert.Equal(t, "Starving", HungerLabel(10))
	assert.Equal(t, "Starving", HungerLabel(0))

	assert.Equal(t, "Hydrated", ThirstLabel(100))
	assert.Equal(t, "Comfortable", ThirstLabel(60))
	assert.Equal(t, "Thirsty", ThirstLabel(30))
	assert.Equal(t, "Parched", ThirstLabel(10))
	assert.Equal(t, "Dehydrated", ThirstLabel(0))

	assert.Equal(t, "Rested", FatigueLabel(100))
	assert.Equal(t, "Ready", FatigueLabel(60))
	assert.Equal(t, "Tired", FatigueLabel(30))
	assert.Equal(t, "Exhausted", FatigueLabel(10))
	assert.Equal(t, "Collapsed", FatigueLabel(0))
}

type recordingProvisioner struct {
	calls           int
	gotLeader       int
	gotSelector     string
	gotBenefit      Benefit
	result          ProvisionResult
	err             error
	memberSelectors map[string]bool
}

func (r *recordingProvisioner) Provision(leaderUserID int, selector string, benefit Benefit) (ProvisionResult, error) {
	r.calls++
	r.gotLeader = leaderUserID
	r.gotSelector = selector
	r.gotBenefit = benefit
	return r.result, r.err
}

func (r *recordingProvisioner) IsMemberSelector(_ int, selector string) bool {
	return r.memberSelectors[selector]
}

func TestProvisionIsUnavailableWithoutRegisteredModule(t *testing.T) {
	SetProvisioner(nil)
	t.Cleanup(func() { SetProvisioner(nil) })

	_, err := Provision(7, "leader", Benefit{Nutrition: 10})
	assert.ErrorIs(t, err, ErrProvisionUnavailable)
}

func TestProvisionForwardsToRegisteredProvisioner(t *testing.T) {
	fake := &recordingProvisioner{result: ProvisionResult{Member: LeaderMemberKey, Name: "Hero"}}
	SetProvisioner(fake)
	t.Cleanup(func() { SetProvisioner(nil) })

	result, err := Provision(7, "#2", Benefit{Nutrition: 30, Hydration: 10})
	require.NoError(t, err)
	assert.Equal(t, "Hero", result.Name)
	assert.Equal(t, 1, fake.calls)
	assert.Equal(t, 7, fake.gotLeader)
	assert.Equal(t, "#2", fake.gotSelector)
	assert.Equal(t, Benefit{Nutrition: 30, Hydration: 10}, fake.gotBenefit)
}

func TestIsMemberSelectorIsFalseWithoutModuleAndForwardsOtherwise(t *testing.T) {
	SetProvisioner(nil)
	t.Cleanup(func() { SetProvisioner(nil) })
	assert.False(t, IsMemberSelector(7, "#2"))

	fake := &recordingProvisioner{memberSelectors: map[string]bool{"#2": true}}
	SetProvisioner(fake)
	t.Cleanup(func() { SetProvisioner(nil) })
	assert.True(t, IsMemberSelector(7, "#2"))
	assert.False(t, IsMemberSelector(7, "#3"))
}

func TestProvisionResultReportsCrossing(t *testing.T) {
	assert.True(t, ProvisionResult{Hunger: Change{Before: BandLow, After: BandSteady}}.Crossed())
	assert.False(t, ProvisionResult{Hunger: Change{Before: BandLow, After: BandLow}}.Crossed())
}

type staticRoster struct{ members []MemberRef }

func (s staticRoster) Roster(int) []MemberRef { return s.members }

func TestCurrentRosterIsEmptyWithoutProvider(t *testing.T) {
	SetRosterProvider(nil)
	t.Cleanup(func() { SetRosterProvider(nil) })
	assert.Nil(t, CurrentRoster(7))
}

func TestCurrentRosterForwardsToProvider(t *testing.T) {
	SetRosterProvider(staticRoster{members: []MemberRef{{Key: LeaderMemberKey, Name: "Hero"}}})
	t.Cleanup(func() { SetRosterProvider(nil) })

	roster := CurrentRoster(7)
	require.Len(t, roster, 1)
	assert.Equal(t, "Hero", roster[0].Name)
}

type restoreCall struct {
	leader    int
	companion int
	snapshot  MemberSnapshot
}

type recordingLifecycle struct {
	ensured    [][2]int
	removed    [][2]int
	removedAll []int
	err        error

	snapshot     MemberSnapshot
	snapshotErr  error
	snapshots    [][2]int
	restored     []restoreCall
	restoreErr   error
	reconciled   []map[int][]MemberRef
	reconcileErr error
	nextReserved int
}

func (r *recordingLifecycle) EnsureCompanyMember(leader, companion int) error {
	r.ensured = append(r.ensured, [2]int{leader, companion})
	return r.err
}

func (r *recordingLifecycle) NextReservedCompanionID(_ int) (int, error) {
	if r.nextReserved < 1 {
		return 1, nil
	}
	return r.nextReserved, r.err
}

func (r *recordingLifecycle) RemoveCompanyMember(leader, companion int) error {
	r.removed = append(r.removed, [2]int{leader, companion})
	return r.err
}

func (r *recordingLifecycle) RemoveAllCompanyMembers(leader int) error {
	r.removedAll = append(r.removedAll, leader)
	return r.err
}

func (r *recordingLifecycle) SnapshotCompanyMember(leader, companion int) (MemberSnapshot, error) {
	r.snapshots = append(r.snapshots, [2]int{leader, companion})
	return r.snapshot, r.snapshotErr
}

func (r *recordingLifecycle) RestoreCompanyMember(leader, companion int, snapshot MemberSnapshot) error {
	r.restored = append(r.restored, restoreCall{leader: leader, companion: companion, snapshot: snapshot})
	return r.restoreErr
}

func (r *recordingLifecycle) ReconcileCompanyRosters(rosters map[int][]MemberRef) error {
	r.reconciled = append(r.reconciled, rosters)
	return r.reconcileErr
}

func TestSnapshotRestoreAndReconcileSeamForwarding(t *testing.T) {
	SetLifecycle(nil)
	t.Cleanup(func() { SetLifecycle(nil) })

	snap, err := SnapshotCompanyMember(7, 1)
	require.NoError(t, err)
	assert.Equal(t, MemberSnapshot{}, snap)
	require.NoError(t, RestoreCompanyMember(7, 1, MemberSnapshot{Exists: true, Needs: FullNeeds()}))
	require.NoError(t, ReconcileCompanyRosters(map[int][]MemberRef{7: {{Key: LeaderMemberKey}}}))

	exact := MemberSnapshot{Exists: true, Needs: Needs{Hunger: 12, Thirst: 34, Fatigue: 56}}
	rec := &recordingLifecycle{snapshot: exact}
	SetLifecycle(rec)

	got, err := SnapshotCompanyMember(7, 2)
	require.NoError(t, err)
	assert.Equal(t, exact, got)
	require.NoError(t, RestoreCompanyMember(7, 3, got))
	require.NoError(t, ReconcileCompanyRosters(map[int][]MemberRef{7: {{Key: LeaderMemberKey, Name: "Hero"}}}))

	assert.Equal(t, [][2]int{{7, 2}}, rec.snapshots)
	assert.Equal(t, []restoreCall{{leader: 7, companion: 3, snapshot: exact}}, rec.restored)
	require.Len(t, rec.reconciled, 1)
	assert.Equal(t, "Hero", rec.reconciled[0][7][0].Name)

	rec.snapshotErr = errors.New("boom")
	_, err = SnapshotCompanyMember(7, 4)
	assert.ErrorIs(t, err, rec.snapshotErr)
	rec.restoreErr = errors.New("boom")
	assert.ErrorIs(t, RestoreCompanyMember(7, 4, MemberSnapshot{}), rec.restoreErr)
	rec.reconcileErr = errors.New("boom")
	assert.ErrorIs(t, ReconcileCompanyRosters(nil), rec.reconcileErr)
}

func TestLifecycleSeamNoOpsWithoutModuleAndForwardsOtherwise(t *testing.T) {
	SetLifecycle(nil)
	t.Cleanup(func() { SetLifecycle(nil) })
	require.NoError(t, EnsureCompanyMember(7, 1))
	require.NoError(t, RemoveCompanyMember(7, 1))
	require.NoError(t, RemoveAllCompanyMembers(7))

	rec := &recordingLifecycle{}
	SetLifecycle(rec)
	require.NoError(t, EnsureCompanyMember(7, 2))
	require.NoError(t, RemoveCompanyMember(7, 3))
	require.NoError(t, RemoveAllCompanyMembers(7))
	assert.Equal(t, [][2]int{{7, 2}}, rec.ensured)
	assert.Equal(t, [][2]int{{7, 3}}, rec.removed)
	assert.Equal(t, []int{7}, rec.removedAll)

	next, err := NextReservedCompanionID(7)
	require.NoError(t, err)
	assert.Equal(t, 1, next)
	rec.nextReserved = 4
	next, err = NextReservedCompanionID(7)
	require.NoError(t, err)
	assert.Equal(t, 4, next)

	rec.err = errors.New("boom")
	assert.ErrorIs(t, EnsureCompanyMember(7, 4), rec.err)
	_, err = NextReservedCompanionID(7)
	assert.ErrorIs(t, err, rec.err)
}
