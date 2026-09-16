package company

import (
	"errors"
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
}

func (f *fakeRuntime) ResolveTemplate(name string) (int, bool) {
	id, ok := f.resolved[name]
	return id, ok
}
func (f *fakeRuntime) Spawn(_ int, roomID int, templateID int) (int, error) {
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
func (f *fakeRuntime) IsLive(instanceID int) bool { return f.live[instanceID] }
func (f *fakeRuntime) Detach(_ int, _ int)        {}

type fakeStore struct {
	saved            domain.Registry
	loadErr, saveErr error
}

func (f *fakeStore) Load(*domain.Registry) error    { return f.loadErr }
func (f *fakeStore) Save(reg domain.Registry) error { f.saved = reg; return f.saveErr }

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
