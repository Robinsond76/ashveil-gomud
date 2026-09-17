package company

import (
	"errors"
	"maps"
	"strings"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLifecycle struct {
	ensured      [][2]int
	removed      [][2]int
	removedAll   []int
	ensureErr    error
	removeErr    error
	removeAllErr error
}

func (f *fakeLifecycle) EnsureCompanyMember(leader, companion int) error {
	f.ensured = append(f.ensured, [2]int{leader, companion})
	return f.ensureErr
}

func (f *fakeLifecycle) RemoveCompanyMember(leader, companion int) error {
	f.removed = append(f.removed, [2]int{leader, companion})
	return f.removeErr
}

func (f *fakeLifecycle) RemoveAllCompanyMembers(leader int) error {
	f.removedAll = append(f.removedAll, leader)
	return f.removeAllErr
}

func useFakeLifecycle(t *testing.T, fake *fakeLifecycle) {
	t.Helper()
	survival.SetLifecycle(fake)
	t.Cleanup(func() { survival.SetLifecycle(nil) })
}

type fakeRuntime struct {
	nextInstanceID    int
	spawnedTemplateID int
	spawnedRoomID     int
	spawnErr          error
	resolved          map[string]int
	live              map[int]bool
	detachCalls       int
	spawnCalls        int
	failSpawnOnCall   int // 1-based; when >0, the Nth Spawn call returns an error
}

func (f *fakeRuntime) ResolveTemplate(name string) (int, bool) {
	id, ok := f.resolved[name]
	return id, ok
}
func (f *fakeRuntime) Spawn(_ int, roomID int, templateID int) (int, error) {
	f.spawnCalls++
	if f.failSpawnOnCall > 0 && f.spawnCalls == f.failSpawnOnCall {
		return 0, errors.New("spawn failed")
	}
	f.spawnedTemplateID = templateID
	f.spawnedRoomID = roomID
	if f.spawnErr != nil {
		return 0, f.spawnErr
	}
	if f.live == nil {
		f.live = map[int]bool{}
	}
	f.live[f.nextInstanceID] = true
	return f.nextInstanceID, nil
}
func (f *fakeRuntime) IsLive(instanceID int) bool            { return f.live[instanceID] }
func (f *fakeRuntime) IsAttached(_ int, instanceID int) bool { return f.live[instanceID] }
func (f *fakeRuntime) Detach(_ int, instanceID int) {
	f.detachCalls++
	delete(f.live, instanceID)
}

type fakeStore struct {
	saved                domain.Registry
	loadErr, saveErr     error
	loadCalls, saveCalls int
}

func cloneRegistry(in domain.Registry) domain.Registry {
	out := domain.Registry{Companies: map[int]domain.Record{}}
	for leader, record := range in.Companies {
		clone := record
		clone.Companions = append([]domain.Companion(nil), record.Companions...)
		out.Companies[leader] = clone
	}
	return out
}

func (f *fakeStore) Load(reg *domain.Registry) error {
	f.loadCalls++
	*reg = cloneRegistry(f.saved)
	return f.loadErr
}
func (f *fakeStore) Save(reg domain.Registry) error {
	f.saveCalls++
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = cloneRegistry(reg)
	return nil
}

func newTestModule(registry domain.Registry, runtime Runtime) *CompanyModule {
	return &CompanyModule{registry: registry, instances: map[int]map[int]int{}, runtime: runtime, store: &fakeStore{}}
}

func TestRestoreForLeaderSpawnsFreshInstanceFromSavedTemplate(t *testing.T) {
	runtime := &fakeRuntime{nextInstanceID: 101}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}}},
	}}, runtime)

	require.NoError(t, module.restoreForLeader(7, 12))
	assert.Equal(t, 58, runtime.spawnedTemplateID)
	instanceID, ok := module.instance(7, 1)
	assert.True(t, ok)
	assert.Equal(t, 101, instanceID)
}

func TestRestoreForLeaderKeepsRecordWhenTemplateCannotSpawn(t *testing.T) {
	runtime := &fakeRuntime{spawnErr: errors.New("unknown template")}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}}},
	}}, runtime)
	assert.Error(t, module.restoreForLeader(7, 12))
	_, exists := module.registry.Get(7)
	assert.True(t, exists)
}

func TestRestoreForLeaderDoesNothingWithoutSavedRecord(t *testing.T) {
	runtime := &fakeRuntime{nextInstanceID: 101}
	module := newTestModule(*domain.NewRegistry(), runtime)
	require.NoError(t, module.restoreForLeader(7, 12))
	assert.Equal(t, 0, runtime.spawnedTemplateID)
}

func TestRestoreForLeaderDoesNotDuplicateLiveInstance(t *testing.T) {
	runtime := &fakeRuntime{nextInstanceID: 101, live: map[int]bool{99: true}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}}},
	}}, runtime)
	module.setInstance(7, 1, 99)
	require.NoError(t, module.restoreForLeader(7, 12))
	assert.Equal(t, 0, runtime.spawnedTemplateID)
	instanceID, ok := module.instance(7, 1)
	assert.True(t, ok)
	assert.Equal(t, 99, instanceID)
}

func TestRestoreForLeaderReplacesStaleInstance(t *testing.T) {
	runtime := &fakeRuntime{nextInstanceID: 101, live: map[int]bool{99: false}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}}},
	}}, runtime)
	module.setInstance(7, 1, 99)
	require.NoError(t, module.restoreForLeader(7, 12))
	instanceID, ok := module.instance(7, 1)
	assert.True(t, ok)
	assert.Equal(t, 101, instanceID)
}

func TestMobDeathClearsMatchingInstanceButKeepsRecord(t *testing.T) {
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}}},
	}}, &fakeRuntime{})
	module.setInstance(7, 1, 99)
	require.Equal(t, events.Continue, module.onMobDeath(events.MobDeath{InstanceId: 99}))
	_, exists := module.registry.Get(7)
	assert.True(t, exists)
	assert.Empty(t, module.instances)
}

func TestPlayerSpawnRestoresUsingCurrentUserRoom(t *testing.T) {
	users.ResetActiveUsers()
	defer users.ResetActiveUsers()
	user := users.NewUserRecord(7, 1)
	user.Character.RoomId = 44
	users.SetTestUser(user)
	runtime := &fakeRuntime{nextInstanceID: 101}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}}},
	}}, runtime)

	require.Equal(t, events.Continue, module.onPlayerSpawn(events.PlayerSpawn{UserId: 7, RoomId: 12}))
	assert.Equal(t, 44, runtime.spawnedRoomID)
}

func TestCompanySummonPersistsAndAttachesAllowedTemplate(t *testing.T) {
	module := newTestModule(*domain.NewRegistry(), &fakeRuntime{resolved: map[string]int{"training dummy": 58}, nextInstanceID: 101})
	text, err := module.summon(7, 12, "training dummy")
	require.NoError(t, err)
	assert.Contains(t, text, "training dummy")
	record, saved := module.registry.Get(7)
	require.True(t, saved)
	assert.Equal(t, 58, record.Companions[0].MobTemplateID)
	store := module.store.(*fakeStore)
	assert.Equal(t, 1, store.saveCalls)
	assert.Equal(t, record, store.saved.Companies[7])
}

func TestCompanyDismissClearsSavedAndLiveState(t *testing.T) {
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}}},
	}}, &fakeRuntime{})
	module.setInstance(7, 1, 101)
	_, err := module.dismiss(7, "1")
	require.NoError(t, err)
	_, saved := module.registry.Get(7)
	assert.False(t, saved)
	assert.Empty(t, module.instances)
}

func TestCompanyStatusReportsSavedCompanionAwaitingRestoration(t *testing.T) {
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}}},
	}}, &fakeRuntime{})
	assert.Contains(t, module.status(7), "awaiting restoration")
}

func TestCompanyDismissSkipsStaleTrackedInstance(t *testing.T) {
	runtime := &fakeRuntime{live: map[int]bool{101: false}}
	store := &fakeStore{}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}}},
	}}, runtime)
	module.store = store
	module.setInstance(7, 1, 101)

	_, err := module.dismiss(7, "1")
	require.NoError(t, err)
	_, saved := module.registry.Get(7)
	assert.False(t, saved)
	assert.Empty(t, module.instances)
	assert.Equal(t, 0, runtime.detachCalls)
	_, saved = store.saved.Get(7)
	assert.False(t, saved)
	assert.Equal(t, 1, store.saveCalls)
}

func captureCompanyMessages(t *testing.T) *[]string {
	t.Helper()
	events.ProcessEvents()
	var messages []string
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		messages = append(messages, e.(events.Message).Text)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	return &messages
}

func TestCompanyCommandSaveFailureRollsBackAndCanRetry(t *testing.T) {
	for _, command := range []string{"summon 58", "dismiss 1"} {
		t.Run(command, func(t *testing.T) {
			messages := captureCompanyMessages(t)
			runtime := &fakeRuntime{nextInstanceID: 101}
			module := newTestModule(*domain.NewRegistry(), runtime)
			store := module.store.(*fakeStore)
			if command == "dismiss 1" {
				_, err := module.summon(7, 12, "58")
				require.NoError(t, err)
			}
			before := domain.Registry{Companies: maps.Clone(module.registry.Companies)}
			beforeSaves := store.saveCalls
			store.saveErr = errors.New("disk full")
			user := users.NewUserRecord(7, 1)
			user.Character.RoomId = 12
			handled, err := module.userCommand(command, user, nil, 0)
			assert.True(t, handled)
			assert.ErrorIs(t, err, store.saveErr)
			events.ProcessEvents()
			assert.NotContains(t, strings.Join(*messages, ""), "Companion summoned:")
			assert.NotContains(t, strings.Join(*messages, ""), "Companion dismissed:")
			assert.Equal(t, before, module.registry)
			assert.Equal(t, beforeSaves+1, store.saveCalls)
			if command == "summon 58" {
				assert.Empty(t, module.instances)
				assert.False(t, runtime.IsLive(101))
			} else {
				instanceID, tracked := module.instance(7, 1)
				assert.True(t, tracked)
				assert.Equal(t, 101, instanceID)
				assert.True(t, runtime.IsLive(101))
				assert.Equal(t, before, store.saved, "failed dismissal must not mutate the saved snapshot")
			}
			store.saveErr = nil
			handled, err = module.userCommand(command, user, nil, 0)
			assert.True(t, handled)
			require.NoError(t, err)
			assert.Equal(t, beforeSaves+2, store.saveCalls)
			assert.Equal(t, module.registry, store.saved)
			events.ProcessEvents()
			if command == "summon 58" {
				assert.Contains(t, strings.Join(*messages, ""), "Companion summoned:")
				instanceID, tracked := module.instance(7, 1)
				assert.True(t, tracked)
				assert.True(t, runtime.IsLive(instanceID))
			} else {
				assert.Contains(t, strings.Join(*messages, ""), "Companion dismissed:")
				assert.Empty(t, module.instances)
				assert.False(t, runtime.IsLive(101))
			}
		})
	}
}

func TestCompanyFailedLoadBlocksWritesUntilRecovery(t *testing.T) {
	runtime := &fakeRuntime{nextInstanceID: 101}
	module := newTestModule(*domain.NewRegistry(), runtime)
	store := &fakeStore{loadErr: errors.New("cannot read companies"), saved: domain.Registry{Companies: map[int]domain.Record{
		8: {LeaderUserID: 8, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}}},
	}}}
	module.store = store
	module.load()
	assert.Empty(t, module.registry.Companies, "a failed/partial decode must not replace active state")
	for _, command := range []string{"summon 58", "dismiss 1"} {
		handled, err := module.userCommand(command, users.NewUserRecord(7, 1), nil, 0)
		assert.True(t, handled)
		assert.ErrorIs(t, err, store.loadErr)
	}
	module.save()
	assert.Zero(t, store.saveCalls, "neither commands nor save callbacks may overwrite unread data")
	assert.Zero(t, runtime.spawnCalls)
	assert.ErrorIs(t, module.restoreForLeader(8, 12), store.loadErr)
	assert.Contains(t, module.status(8), "unavailable")
	assert.Contains(t, store.saved.Companies, 8)

	store.loadErr = nil
	module.load()
	require.Equal(t, 2, store.loadCalls)
	require.Contains(t, module.registry.Companies, 8)
	_, err := module.summon(7, 12, "58")
	require.NoError(t, err)
	assert.Equal(t, 1, store.saveCalls)
	assert.Contains(t, store.saved.Companies, 8, "recovery must preserve preexisting ownership")
	assert.Contains(t, store.saved.Companies, 7)
}

func TestCompanyRejectedSummonDoesNotWriteOrReplaceCompanion(t *testing.T) {
	for _, selector := range []string{"missing", "59", "0", "-1", "58"} {
		t.Run(selector, func(t *testing.T) {
			messages := captureCompanyMessages(t)
			runtime := &fakeRuntime{nextInstanceID: 101}
			module := newTestModule(*domain.NewRegistry(), runtime)
			if selector == "58" {
				for i := 0; i < 4; i++ {
					runtime.nextInstanceID++
					_, err := module.summon(7, 12, "58")
					require.NoError(t, err)
				}
			}
			before := domain.Registry{Companies: maps.Clone(module.registry.Companies)}
			store := module.store.(*fakeStore)
			beforeSaves, beforeSpawns := store.saveCalls, runtime.spawnCalls
			handled, err := module.userCommand("summon "+selector, users.NewUserRecord(7, 1), nil, 0)
			assert.True(t, handled)
			assert.Error(t, err)
			events.ProcessEvents()
			assert.NotContains(t, strings.Join(*messages, ""), "Companion summoned:")
			assert.Equal(t, beforeSaves, store.saveCalls)
			assert.Equal(t, beforeSpawns, runtime.spawnCalls)
			assert.Equal(t, before, module.registry)
		})
	}
}

func TestSummonRespectsMaxCompanions(t *testing.T) {
	runtime := &fakeRuntime{nextInstanceID: 100}
	module := newTestModule(*domain.NewRegistry(), runtime)
	for i := 0; i < 4; i++ {
		runtime.nextInstanceID++
		_, err := module.summon(7, 12, "58")
		require.NoError(t, err)
	}
	_, err := module.summon(7, 12, "58")
	assert.ErrorIs(t, err, domain.ErrCompanyFull)
}

func TestRestoreForLeaderSpawnsEveryCompanion(t *testing.T) {
	runtime := &fakeRuntime{nextInstanceID: 200}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}, {ID: 2, MobTemplateID: 58}}},
	}}, runtime)
	require.NoError(t, module.restoreForLeader(7, 12))
	assert.Equal(t, 2, runtime.spawnCalls)
	_, ok := module.instance(7, 1)
	assert.True(t, ok)
	_, ok = module.instance(7, 2)
	assert.True(t, ok)
}

func TestRestoreForLeaderContinuesAfterOneSpawnFails(t *testing.T) {
	runtime := &fakeRuntime{nextInstanceID: 200, failSpawnOnCall: 2}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}, {ID: 2, MobTemplateID: 58}, {ID: 3, MobTemplateID: 58}}},
	}}, runtime)

	err := module.restoreForLeader(7, 12)
	require.Error(t, err)
	assert.Equal(t, 3, runtime.spawnCalls, "restoration must continue past a failed spawn")

	_, ok := module.instance(7, 1)
	assert.True(t, ok, "the first companion must remain tracked")
	_, ok = module.instance(7, 2)
	assert.False(t, ok, "the failed companion must not be tracked")
	_, ok = module.instance(7, 3)
	assert.True(t, ok, "a later companion must still be restored")

	record, exists := module.registry.Get(7)
	require.True(t, exists)
	assert.Len(t, record.Companions, 3, "a spawn failure must not remove durable ownership")
}

func TestMobDeathClearsOnlyMatchingCompanionInstance(t *testing.T) {
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}, {ID: 2, MobTemplateID: 58}}},
	}}, &fakeRuntime{})
	module.setInstance(7, 1, 91)
	module.setInstance(7, 2, 92)
	require.Equal(t, events.Continue, module.onMobDeath(events.MobDeath{InstanceId: 91}))
	_, ok := module.instance(7, 1)
	assert.False(t, ok)
	_, ok = module.instance(7, 2)
	assert.True(t, ok)
	record, exists := module.registry.Get(7)
	assert.True(t, exists)
	assert.Len(t, record.Companions, 2)
}

func TestDismissOneCompanionDetachesOnlyItsInstance(t *testing.T) {
	runtime := &fakeRuntime{live: map[int]bool{91: true, 92: true}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}, {ID: 2, MobTemplateID: 58}}},
	}}, runtime)
	module.setInstance(7, 1, 91)
	module.setInstance(7, 2, 92)
	_, err := module.dismiss(7, "1")
	require.NoError(t, err)
	assert.Equal(t, 1, runtime.detachCalls)
	_, ok := module.instance(7, 1)
	assert.False(t, ok)
	_, ok = module.instance(7, 2)
	assert.True(t, ok)
	record, _ := module.registry.Get(7)
	assert.Len(t, record.Companions, 1)
	assert.Equal(t, 2, record.Companions[0].ID)
}

func TestDismissAllCompanionsDetachesEveryInstance(t *testing.T) {
	runtime := &fakeRuntime{live: map[int]bool{91: true, 92: true}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}, {ID: 2, MobTemplateID: 58}}},
	}}, runtime)
	module.setInstance(7, 1, 91)
	module.setInstance(7, 2, 92)
	_, err := module.dismiss(7, "all")
	require.NoError(t, err)
	assert.Equal(t, 2, runtime.detachCalls)
	assert.Empty(t, module.instances)
}

func TestDismissAllSaveFailureRollsBack(t *testing.T) {
	runtime := &fakeRuntime{live: map[int]bool{91: true, 92: true}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}, {ID: 2, MobTemplateID: 58}}},
	}}, runtime)
	module.setInstance(7, 1, 91)
	module.setInstance(7, 2, 92)
	store := module.store.(*fakeStore)
	before := cloneRegistry(module.registry)
	beforeSaves := store.saveCalls
	store.saveErr = errors.New("disk full")

	_, err := module.dismiss(7, "all")
	assert.ErrorIs(t, err, store.saveErr)
	assert.Equal(t, before, module.registry, "failed bulk dismissal must restore the roster")
	assert.Equal(t, 0, runtime.detachCalls, "failed bulk dismissal must not detach live companions")
	_, ok := module.instance(7, 1)
	assert.True(t, ok)
	_, ok = module.instance(7, 2)
	assert.True(t, ok)
	assert.Equal(t, beforeSaves+1, store.saveCalls)

	store.saveErr = nil
	_, err = module.dismiss(7, "all")
	require.NoError(t, err)
	assert.Equal(t, 2, runtime.detachCalls)
	assert.Empty(t, module.instances)
}

func TestConfigIntAcceptsYAMLAndStringValues(t *testing.T) {
	for _, value := range []any{4, int64(4), float64(4), "4"} {
		got, ok := configInt(value)
		require.True(t, ok)
		assert.Equal(t, 4, got)
	}
	_, ok := configInt("not-a-number")
	assert.False(t, ok)
}

func TestMaxCompanionsFromConfigClampsToDomainLimit(t *testing.T) {
	assert.Equal(t, domain.MaxCompanions, maxCompanionsFromConfig(domain.MaxCompanions+1))
	assert.Equal(t, domain.MaxCompanions, maxCompanionsFromConfig("999"))
	assert.Equal(t, 2, maxCompanionsFromConfig(2))
	assert.Equal(t, domain.MaxCompanions, maxCompanionsFromConfig(0))
	assert.Equal(t, domain.MaxCompanions, maxCompanionsFromConfig("invalid"))
}

func TestCompanyRosterIncludesLeaderAndCompanions(t *testing.T) {
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 1)
	user.Character.Name = "Hero"
	users.SetTestUser(user)

	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}}},
	}}, &fakeRuntime{})

	roster := module.Roster(7)
	require.Len(t, roster, 2)
	assert.Equal(t, survival.LeaderMemberKey, roster[0].Key)
	assert.Equal(t, "Hero", roster[0].Name)
	assert.Equal(t, survival.CompanionMemberKey(1), roster[1].Key)
	assert.NotEmpty(t, roster[1].Name)
}

func TestCompanySummonSynchronizesSurvivalState(t *testing.T) {
	fake := &fakeLifecycle{}
	useFakeLifecycle(t, fake)
	module := newTestModule(*domain.NewRegistry(), &fakeRuntime{nextInstanceID: 101})

	_, err := module.summon(7, 12, "58")
	require.NoError(t, err)
	assert.Equal(t, [][2]int{{7, 1}}, fake.ensured)
	assert.Empty(t, fake.removed)
}

func TestCompanySummonRollsBackWhenSurvivalSyncFails(t *testing.T) {
	fake := &fakeLifecycle{ensureErr: errors.New("survival disk full")}
	useFakeLifecycle(t, fake)
	runtime := &fakeRuntime{nextInstanceID: 101}
	module := newTestModule(*domain.NewRegistry(), runtime)
	before := cloneRegistry(module.registry)

	_, err := module.summon(7, 12, "58")
	require.ErrorIs(t, err, fake.ensureErr)
	assert.Equal(t, before, module.registry)
	assert.Zero(t, runtime.spawnCalls, "no native companion may spawn when survival sync fails")
	assert.Empty(t, module.instances)
}

func TestCompanyDismissRemovesSurvivalState(t *testing.T) {
	fake := &fakeLifecycle{}
	useFakeLifecycle(t, fake)
	runtime := &fakeRuntime{live: map[int]bool{101: true}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}}},
	}}, runtime)
	module.setInstance(7, 1, 101)

	_, err := module.dismiss(7, "1")
	require.NoError(t, err)
	assert.Equal(t, [][2]int{{7, 1}}, fake.removed)
	assert.Equal(t, 1, runtime.detachCalls)
}

func TestCompanyDismissRollsBackWhenSurvivalSyncFails(t *testing.T) {
	fake := &fakeLifecycle{removeErr: errors.New("survival disk full")}
	useFakeLifecycle(t, fake)
	runtime := &fakeRuntime{live: map[int]bool{101: true}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}}},
	}}, runtime)
	module.setInstance(7, 1, 101)
	before := cloneRegistry(module.registry)

	_, err := module.dismiss(7, "1")
	require.ErrorIs(t, err, fake.removeErr)
	assert.Equal(t, before, module.registry)
	assert.Zero(t, runtime.detachCalls, "no native companion may detach when survival sync fails")
	instanceID, tracked := module.instance(7, 1)
	assert.True(t, tracked)
	assert.Equal(t, 101, instanceID)
}

func TestCompanyDismissAllRemovesSurvivalState(t *testing.T) {
	fake := &fakeLifecycle{}
	useFakeLifecycle(t, fake)
	runtime := &fakeRuntime{live: map[int]bool{91: true, 92: true}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}, {ID: 2, MobTemplateID: 58}}},
	}}, runtime)
	module.setInstance(7, 1, 91)
	module.setInstance(7, 2, 92)

	_, err := module.dismiss(7, "all")
	require.NoError(t, err)
	assert.Equal(t, []int{7}, fake.removedAll)
	assert.Equal(t, 2, runtime.detachCalls)
}

func TestCompanyDismissAllRollsBackWhenSurvivalSyncFails(t *testing.T) {
	fake := &fakeLifecycle{removeAllErr: errors.New("survival disk full")}
	useFakeLifecycle(t, fake)
	runtime := &fakeRuntime{live: map[int]bool{91: true, 92: true}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}, {ID: 2, MobTemplateID: 58}}},
	}}, runtime)
	module.setInstance(7, 1, 91)
	module.setInstance(7, 2, 92)
	before := cloneRegistry(module.registry)

	_, err := module.dismiss(7, "all")
	require.ErrorIs(t, err, fake.removeAllErr)
	assert.Equal(t, before, module.registry)
	assert.Zero(t, runtime.detachCalls)
}

func TestCompanySaveFailureUndoesSurvivalEnsure(t *testing.T) {
	fake := &fakeLifecycle{}
	useFakeLifecycle(t, fake)
	module := newTestModule(*domain.NewRegistry(), &fakeRuntime{nextInstanceID: 101})
	store := module.store.(*fakeStore)
	store.saveErr = errors.New("disk full")

	_, err := module.summon(7, 12, "58")
	require.ErrorIs(t, err, store.saveErr)
	assert.Equal(t, [][2]int{{7, 1}}, fake.ensured)
	assert.Equal(t, [][2]int{{7, 1}}, fake.removed, "a failed company save must undo the survival record")
}
