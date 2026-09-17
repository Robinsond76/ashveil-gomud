package company

import (
	"errors"
	"maps"
	"strings"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRuntime struct {
	nextInstanceID    int
	spawnedTemplateID int
	spawnedRoomID     int
	spawnErr          error
	resolved          map[string]int
	live              map[int]bool
	detachCalls       int
	spawnCalls        int
}

func (f *fakeRuntime) ResolveTemplate(name string) (int, bool) {
	id, ok := f.resolved[name]
	return id, ok
}
func (f *fakeRuntime) Spawn(_ int, roomID int, templateID int) (int, error) {
	f.spawnCalls++
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

func (f *fakeStore) Load(reg *domain.Registry) error {
	f.loadCalls++
	*reg = domain.Registry{Companies: maps.Clone(f.saved.Companies)}
	return f.loadErr
}
func (f *fakeStore) Save(reg domain.Registry) error {
	f.saveCalls++
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = domain.Registry{Companies: maps.Clone(reg.Companies)}
	return nil
}

func newTestModule(registry domain.Registry, runtime Runtime) *CompanyModule {
	return &CompanyModule{registry: registry, liveByLeader: map[int]int{}, runtime: runtime, store: &fakeStore{}}
}

func TestRestoreForLeaderSpawnsFreshInstanceFromSavedTemplate(t *testing.T) {
	runtime := &fakeRuntime{nextInstanceID: 101}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companion: domain.Companion{MobTemplateID: 58}},
	}}, runtime)

	require.NoError(t, module.restoreForLeader(7, 12))
	assert.Equal(t, 58, runtime.spawnedTemplateID)
	assert.Equal(t, 101, module.liveByLeader[7])
}

func TestRestoreForLeaderKeepsRecordWhenTemplateCannotSpawn(t *testing.T) {
	runtime := &fakeRuntime{spawnErr: errors.New("unknown template")}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companion: domain.Companion{MobTemplateID: 58}},
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
		7: {LeaderUserID: 7, Companion: domain.Companion{MobTemplateID: 58}},
	}}, runtime)
	module.liveByLeader[7] = 99
	require.NoError(t, module.restoreForLeader(7, 12))
	assert.Equal(t, 0, runtime.spawnedTemplateID)
	assert.Equal(t, 99, module.liveByLeader[7])
}

func TestRestoreForLeaderReplacesStaleInstance(t *testing.T) {
	runtime := &fakeRuntime{nextInstanceID: 101, live: map[int]bool{99: false}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companion: domain.Companion{MobTemplateID: 58}},
	}}, runtime)
	module.liveByLeader[7] = 99
	require.NoError(t, module.restoreForLeader(7, 12))
	assert.Equal(t, 101, module.liveByLeader[7])
}

func TestMobDeathClearsMatchingInstanceButKeepsRecord(t *testing.T) {
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companion: domain.Companion{MobTemplateID: 58}},
	}}, &fakeRuntime{})
	module.liveByLeader[7] = 99
	require.Equal(t, events.Continue, module.onMobDeath(events.MobDeath{InstanceId: 99}))
	_, exists := module.registry.Get(7)
	assert.True(t, exists)
	assert.NotContains(t, module.liveByLeader, 7)
}

func TestPlayerSpawnRestoresUsingCurrentUserRoom(t *testing.T) {
	users.ResetActiveUsers()
	defer users.ResetActiveUsers()
	user := users.NewUserRecord(7, 1)
	user.Character.RoomId = 44
	users.SetTestUser(user)
	runtime := &fakeRuntime{nextInstanceID: 101}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companion: domain.Companion{MobTemplateID: 58}},
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
	assert.Equal(t, 58, record.Companion.MobTemplateID)
	store := module.store.(*fakeStore)
	assert.Equal(t, 1, store.saveCalls)
	assert.Equal(t, record, store.saved.Companies[7])
}

func TestCompanyDismissClearsSavedAndLiveState(t *testing.T) {
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companion: domain.Companion{MobTemplateID: 58}},
	}}, &fakeRuntime{})
	module.liveByLeader[7] = 101
	_, err := module.dismiss(7)
	require.NoError(t, err)
	_, saved := module.registry.Get(7)
	assert.False(t, saved)
	assert.NotContains(t, module.liveByLeader, 7)
}

func TestCompanyStatusReportsSavedCompanionAwaitingRestoration(t *testing.T) {
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companion: domain.Companion{MobTemplateID: 58}},
	}}, &fakeRuntime{})
	assert.Contains(t, module.status(7), "awaiting restoration")
}

func TestCompanyDismissSkipsStaleTrackedInstance(t *testing.T) {
	runtime := &fakeRuntime{live: map[int]bool{101: false}}
	store := &fakeStore{}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companion: domain.Companion{MobTemplateID: 58}},
	}}, runtime)
	module.store = store
	module.liveByLeader[7] = 101

	_, err := module.dismiss(7)
	require.NoError(t, err)
	_, saved := module.registry.Get(7)
	assert.False(t, saved)
	assert.NotContains(t, module.liveByLeader, 7)
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
	for _, command := range []string{"summon 58", "dismiss"} {
		t.Run(command, func(t *testing.T) {
			messages := captureCompanyMessages(t)
			runtime := &fakeRuntime{nextInstanceID: 101}
			module := newTestModule(*domain.NewRegistry(), runtime)
			store := module.store.(*fakeStore)
			if command == "dismiss" {
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
			assert.NotContains(t, strings.Join(*messages, ""), "Companion dismissed.")
			assert.Equal(t, before, module.registry)
			assert.Equal(t, beforeSaves+1, store.saveCalls)
			if command == "summon 58" {
				assert.Empty(t, module.liveByLeader)
				assert.False(t, runtime.IsLive(101))
			} else {
				assert.Equal(t, 101, module.liveByLeader[7])
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
				assert.True(t, runtime.IsLive(module.liveByLeader[7]))
			} else {
				assert.Contains(t, strings.Join(*messages, ""), "Companion dismissed.")
				assert.Empty(t, module.liveByLeader)
				assert.False(t, runtime.IsLive(101))
			}
		})
	}
}

func TestCompanyFailedLoadBlocksWritesUntilRecovery(t *testing.T) {
	runtime := &fakeRuntime{nextInstanceID: 101}
	module := newTestModule(*domain.NewRegistry(), runtime)
	store := &fakeStore{loadErr: errors.New("cannot read companies"), saved: domain.Registry{Companies: map[int]domain.Record{
		8: {LeaderUserID: 8, Companion: domain.Companion{MobTemplateID: 58}},
	}}}
	module.store = store
	module.load()
	assert.Empty(t, module.registry.Companies, "a failed/partial decode must not replace active state")
	for _, command := range []string{"summon 58", "dismiss"} {
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
				_, err := module.summon(7, 12, "58")
				require.NoError(t, err)
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
