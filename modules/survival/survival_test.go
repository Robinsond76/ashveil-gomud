package survival

import (
	"errors"
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	domain "github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "low", "", false)
	os.Exit(m.Run())
}

type fakeStore struct {
	saved                domain.Registry
	loadErr, saveErr     error
	loadCalls, saveCalls int
	absent               bool
}

func (f *fakeStore) Load(registry *domain.Registry) error {
	f.loadCalls++
	if f.loadErr != nil {
		return f.loadErr
	}
	if f.absent {
		*registry = *domain.NewRegistry()
		return nil
	}
	*registry = f.saved.Clone()
	return nil
}

func (f *fakeStore) Save(registry domain.Registry) error {
	f.saveCalls++
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = registry.Clone()
	return nil
}

type fakeRoster struct {
	members map[int][]domain.MemberRef
}

func (f fakeRoster) Roster(leaderUserID int) []domain.MemberRef {
	return f.members[leaderUserID]
}

func newTestModule(registry domain.Registry) *SurvivalModule {
	return &SurvivalModule{registry: registry, store: &fakeStore{}}
}

func useRoster(t *testing.T, roster fakeRoster) {
	t.Helper()
	domain.SetRosterProvider(roster)
	t.Cleanup(func() { domain.SetRosterProvider(nil) })
}

func TestLoadAbsentFileCreatesEmptyRegistry(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	m.store = &fakeStore{absent: true}
	m.load()

	require.NoError(t, m.loadErr)
	assert.Empty(t, m.registry.Leaders)
	assert.Equal(t, 1, m.store.(*fakeStore).loadCalls)
}

func TestPluginStoreAbsentFileLoadsEmptyRegistry(t *testing.T) {
	t.Chdir(t.TempDir())
	plug := plugins.New("survival_absent_regression", "1.0")
	m := newTestModule(*domain.NewRegistry())
	m.store = pluginStore{plug: plug}
	m.load()

	require.NoError(t, m.loadErr)
	assert.Empty(t, m.registry.Leaders)
}

func TestPluginStoreRejectsMalformedDataWithoutOverwritingIt(t *testing.T) {
	t.Chdir(t.TempDir())
	plug := plugins.New("survival_load_regression", "1.0")
	malformed := []byte("leaders: [invalid")
	require.NoError(t, plug.WriteBytes("survival", malformed))
	m := newTestModule(*domain.NewRegistry())
	m.store = pluginStore{plug: plug}
	m.load()

	assert.Error(t, m.loadErr)
	require.Error(t, m.EnsureCompanyMember(7, 1))
	m.save()
	data, err := plug.ReadBytes("survival")
	require.NoError(t, err)
	assert.Equal(t, malformed, data)
}

func TestDecodeNormalizesValuesAndDropsMalformedEntries(t *testing.T) {
	data := []byte("leaders:\n" +
		"  7:\n" +
		"    leader:\n" +
		"      hunger: -5\n" +
		"      thirst: 200\n" +
		"      fatigue: 42\n" +
		"    companion:1:\n" +
		"      hunger: 50\n" +
		"      thirst: 50\n" +
		"      fatigue: 50\n" +
		"    bogus:\n" +
		"      hunger: 1\n" +
		"  0:\n" +
		"    leader:\n" +
		"      hunger: 1\n")

	var registry domain.Registry
	require.NoError(t, decodeRegistry(data, &registry))
	assert.Equal(t, domain.Needs{Hunger: 0, Thirst: 100, Fatigue: 42}, registry.MustNeedsFor(7, domain.LeaderMemberKey))
	assert.Equal(t, domain.Needs{Hunger: 50, Thirst: 50, Fatigue: 50}, registry.MustNeedsFor(7, domain.CompanionMemberKey(1)))
	_, ok := registry.NeedsFor(7, domain.MemberKey("bogus"))
	assert.False(t, ok)
	assert.NotContains(t, registry.Leaders, 0)
}

func TestMalformedBytesBlockMutationsAndWrites(t *testing.T) {
	loadErr := errors.New("cannot read survival")
	store := &fakeStore{loadErr: loadErr, saved: domain.Registry{Leaders: map[int]map[domain.MemberKey]domain.Needs{
		8: {domain.LeaderMemberKey: domain.FullNeeds()},
	}}}
	m := newTestModule(*domain.NewRegistry())
	m.store = store
	m.load()

	require.ErrorIs(t, m.loadErr, loadErr)
	assert.Empty(t, m.registry.Leaders, "a failed decode must not replace active state")

	require.ErrorIs(t, m.EnsureCompanyMember(7, 1), loadErr)
	require.ErrorIs(t, m.RemoveCompanyMember(7, 1), loadErr)
	require.ErrorIs(t, m.RemoveAllCompanyMembers(7), loadErr)
	_, err := m.Provision(7, "leader", domain.Benefit{Nutrition: 10})
	require.ErrorIs(t, err, loadErr)
	m.save()
	assert.Zero(t, store.saveCalls, "unread data must not be overwritten")
	assert.Contains(t, m.status(7), "unavailable")
	assert.Contains(t, store.saved.Leaders, 8)
}

func TestProvisionResolvesLeaderAliases(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.registry.Ensure(7, domain.LeaderMemberKey))

	for _, selector := range []string{"", "leader", "me", "self", "LEADER"} {
		result, err := m.Provision(7, selector, domain.Benefit{Nutrition: 10})
		require.NoError(t, err, selector)
		assert.Equal(t, domain.LeaderMemberKey, result.Member, selector)
	}
}

func TestProvisionResolvesCompanionBySelectorAndName(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.registry.Ensure(7, domain.CompanionMemberKey(2)))
	useRoster(t, fakeRoster{members: map[int][]domain.MemberRef{
		7: {
			{Key: domain.LeaderMemberKey, Name: "Hero"},
			{Key: domain.CompanionMemberKey(2), Name: "Bear"},
		},
	}})

	for _, selector := range []string{"#2", "2", "Bear", "bear", "bea"} {
		result, err := m.Provision(7, selector, domain.Benefit{Nutrition: 10})
		require.NoError(t, err, selector)
		assert.Equal(t, domain.CompanionMemberKey(2), result.Member, selector)
		assert.Equal(t, "Bear", result.Name, selector)
	}
}

func TestProvisionRejectsDismissedCompanionWithoutChangingState(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.registry.Ensure(7, domain.LeaderMemberKey))
	before := m.registry.Clone()

	_, err := m.Provision(7, "#2", domain.Benefit{Nutrition: 30})
	require.ErrorIs(t, err, domain.ErrUnknownMember)
	assert.Equal(t, before, m.registry)
}

func TestProvisionRejectsUnknownAndAmbiguousCompanions(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.registry.Ensure(7, domain.CompanionMemberKey(1)))
	require.NoError(t, m.registry.Ensure(7, domain.CompanionMemberKey(2)))
	useRoster(t, fakeRoster{members: map[int][]domain.MemberRef{
		7: {
			{Key: domain.LeaderMemberKey, Name: "Hero"},
			{Key: domain.CompanionMemberKey(1), Name: "Bear"},
			{Key: domain.CompanionMemberKey(2), Name: "Bear Cub"},
		},
	}})

	_, err := m.Provision(7, "#9", domain.Benefit{Nutrition: 10})
	assert.ErrorIs(t, err, domain.ErrUnknownMember)

	_, err = m.Provision(7, "wolf", domain.Benefit{Nutrition: 10})
	assert.ErrorIs(t, err, domain.ErrUnknownMember)

	_, err = m.Provision(7, "be", domain.Benefit{Nutrition: 10})
	assert.ErrorIs(t, err, domain.ErrAmbiguousMember)

	result, err := m.Provision(7, "Bear", domain.Benefit{Nutrition: 10})
	require.NoError(t, err)
	assert.Equal(t, domain.CompanionMemberKey(1), result.Member, "an exact name wins over a substring match")
}

func TestProvisionRejectsNonPositiveBenefit(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.registry.Ensure(7, domain.LeaderMemberKey))
	before := m.registry.Clone()

	_, err := m.Provision(7, "leader", domain.Benefit{})
	assert.ErrorIs(t, err, domain.ErrInvalidAmount)
	_, err = m.Provision(7, "leader", domain.Benefit{Nutrition: -5})
	assert.ErrorIs(t, err, domain.ErrInvalidAmount)
	assert.Equal(t, before, m.registry)
}

func TestProvisionPersistsBeforeReportingSuccess(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.registry.Ensure(7, domain.LeaderMemberKey))
	require.NoError(t, m.registry.PutNeeds(7, domain.LeaderMemberKey, domain.Needs{Hunger: 50, Thirst: 100, Fatigue: 100}))

	result, err := m.Provision(7, "leader", domain.Benefit{Nutrition: 10})
	require.NoError(t, err)
	assert.Equal(t, 60, result.Needs.Hunger)
	assert.Equal(t, domain.BandLow, result.Hunger.Before)
	assert.Equal(t, domain.BandSteady, result.Hunger.After)
	assert.True(t, result.Crossed())

	store := m.store.(*fakeStore)
	assert.Equal(t, 1, store.saveCalls)
	assert.Equal(t, m.registry, store.saved)
}

func TestProvisionRollsBackOnSaveFailure(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.registry.Ensure(7, domain.LeaderMemberKey))
	require.NoError(t, m.registry.PutNeeds(7, domain.LeaderMemberKey, domain.Needs{Hunger: 50, Thirst: 50, Fatigue: 50}))
	before := m.registry.Clone()

	store := m.store.(*fakeStore)
	store.saveErr = errors.New("disk full")
	_, err := m.Provision(7, "leader", domain.Benefit{Nutrition: 10})
	assert.ErrorIs(t, err, store.saveErr)
	assert.Equal(t, before, m.registry, "a failed write must not leave the benefit applied")

	store.saveErr = nil
	_, err = m.Provision(7, "leader", domain.Benefit{Nutrition: 10})
	require.NoError(t, err)
	assert.Equal(t, 60, m.registry.MustNeedsFor(7, domain.LeaderMemberKey).Hunger)
}

func TestEnsureCompanyMemberCreatesAndPersistsDefaultState(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.EnsureCompanyMember(7, 1))

	needs, ok := m.registry.NeedsFor(7, domain.CompanionMemberKey(1))
	require.True(t, ok)
	assert.Equal(t, domain.FullNeeds(), needs)
	assert.Equal(t, 1, m.store.(*fakeStore).saveCalls)
}

func TestEnsureCompanyMemberRejectsInvalidIDs(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	assert.ErrorIs(t, m.EnsureCompanyMember(0, 1), domain.ErrInvalidMember)
	assert.ErrorIs(t, m.EnsureCompanyMember(7, 0), domain.ErrInvalidMember)
}

func TestRemoveCompanyMemberPrunesAndPersists(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.EnsureCompanyMember(7, 1))
	require.NoError(t, m.EnsureCompanyMember(7, 2))

	require.NoError(t, m.RemoveCompanyMember(7, 1))
	_, ok := m.registry.NeedsFor(7, domain.CompanionMemberKey(1))
	assert.False(t, ok)
	_, ok = m.registry.NeedsFor(7, domain.CompanionMemberKey(2))
	assert.True(t, ok)
	assert.Equal(t, m.registry, m.store.(*fakeStore).saved)
}

func TestRemoveAllCompanyMembersKeepsLeaderState(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.registry.Ensure(7, domain.LeaderMemberKey))
	require.NoError(t, m.registry.PutNeeds(7, domain.LeaderMemberKey, domain.Needs{Hunger: 40, Thirst: 40, Fatigue: 40}))
	require.NoError(t, m.EnsureCompanyMember(7, 1))
	require.NoError(t, m.EnsureCompanyMember(7, 2))

	require.NoError(t, m.RemoveAllCompanyMembers(7))
	assert.Equal(t, 40, m.registry.MustNeedsFor(7, domain.LeaderMemberKey).Hunger)
	_, ok := m.registry.NeedsFor(7, domain.CompanionMemberKey(1))
	assert.False(t, ok)
	_, ok = m.registry.NeedsFor(7, domain.CompanionMemberKey(2))
	assert.False(t, ok)
}

func TestLifecycleSaveFailureRestoresSnapshot(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.EnsureCompanyMember(7, 1))
	before := m.registry.Clone()

	store := m.store.(*fakeStore)
	store.saveErr = errors.New("disk full")

	require.ErrorIs(t, m.RemoveCompanyMember(7, 1), store.saveErr)
	assert.Equal(t, before, m.registry, "failed removal must restore the companion state")

	require.ErrorIs(t, m.EnsureCompanyMember(7, 2), store.saveErr)
	assert.Equal(t, before, m.registry, "failed ensure must not add a partial record")

	store.saveErr = nil
	require.NoError(t, m.RemoveCompanyMember(7, 1))
	_, ok := m.registry.NeedsFor(7, domain.CompanionMemberKey(1))
	assert.False(t, ok)
}

func TestStatusRendersLeaderAndCompanionsWithLabels(t *testing.T) {
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 1)
	user.Character.Name = "Hero"
	users.SetTestUser(user)

	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.registry.Ensure(7, domain.LeaderMemberKey))
	require.NoError(t, m.registry.PutNeeds(7, domain.LeaderMemberKey, domain.Needs{Hunger: 10, Thirst: 60, Fatigue: 100}))
	require.NoError(t, m.registry.Ensure(7, domain.CompanionMemberKey(2)))
	require.NoError(t, m.registry.PutNeeds(7, domain.CompanionMemberKey(2), domain.Needs{Hunger: 100, Thirst: 0, Fatigue: 30}))
	useRoster(t, fakeRoster{members: map[int][]domain.MemberRef{
		7: {
			{Key: domain.LeaderMemberKey, Name: "Hero"},
			{Key: domain.CompanionMemberKey(2), Name: "Bear"},
		},
	}})

	text := m.status(7)
	assert.Contains(t, text, "Hero")
	assert.Contains(t, text, "Hunger 10 (Starving)")
	assert.Contains(t, text, "Thirst 60 (Comfortable)")
	assert.Contains(t, text, "Fatigue 100 (Rested)")
	assert.Contains(t, text, "Bear")
	assert.Contains(t, text, "Thirst 0 (Dehydrated)")
	assert.Contains(t, text, "Fatigue 30 (Tired)")
}

func TestStatusPrunesStaleCompanionRecords(t *testing.T) {
	m := newTestModule(*domain.NewRegistry())
	require.NoError(t, m.registry.Ensure(7, domain.LeaderMemberKey))
	require.NoError(t, m.registry.Ensure(7, domain.CompanionMemberKey(1)))
	require.NoError(t, m.registry.Ensure(7, domain.CompanionMemberKey(2)))
	useRoster(t, fakeRoster{members: map[int][]domain.MemberRef{
		7: {
			{Key: domain.LeaderMemberKey, Name: "Hero"},
			{Key: domain.CompanionMemberKey(1), Name: "Bear"},
		},
	}})

	text := m.status(7)
	assert.Contains(t, text, "Bear")
	_, ok := m.registry.NeedsFor(7, domain.CompanionMemberKey(2))
	assert.False(t, ok, "stale companion state must be pruned")
	assert.Zero(t, m.store.(*fakeStore).saveCalls, "status must not write to the store")
}

func TestStatusCreatesLeaderRecordInMemoryOnly(t *testing.T) {
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 1)
	user.Character.Name = "Hero"
	users.SetTestUser(user)

	m := newTestModule(*domain.NewRegistry())
	text := m.status(7)
	assert.Contains(t, text, "Hero")
	assert.Contains(t, text, "Hunger 100 (Well fed)")
	_, ok := m.registry.NeedsFor(7, domain.LeaderMemberKey)
	assert.True(t, ok)
	assert.Zero(t, m.store.(*fakeStore).saveCalls)
}

func TestUserCommandRendersStatus(t *testing.T) {
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 1)
	user.Character.Name = "Hero"
	users.SetTestUser(user)

	m := newTestModule(*domain.NewRegistry())
	handled, err := m.userCommand("", user, nil, 0)
	require.NoError(t, err)
	assert.True(t, handled)

	handled, err = m.userCommand("garbage", user, nil, 0)
	require.NoError(t, err)
	assert.True(t, handled)
}
