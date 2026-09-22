package mount

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mount"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "low", "", false)
	os.Exit(m.Run())
}

type fakeStore struct {
	saved   Registry
	loadErr error
	saveErr error
}

func (f *fakeStore) Load(registry *Registry) error {
	if f.loadErr != nil {
		return f.loadErr
	}
	*registry = f.saved.Clone()
	return nil
}

func (f *fakeStore) Save(registry Registry) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = registry.Clone()
	return nil
}

func packHorseSpecs() map[string]mount.MountSpec {
	return map[string]mount.MountSpec{
		"pack-horse": {Type: "pack-horse", Description: "A sturdy pack horse.", CargoCapacityBonusGrams: 100000, TravelDurationPct: 90},
	}
}

func newTestModule(store Store) *MountModule {
	return &MountModule{
		store:  store,
		specs:  packHorseSpecs(),
		mounts: map[int]mount.Mount{},
	}
}

func testUser(t *testing.T, userId int) *users.UserRecord {
	t.Helper()
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(userId, 1)
	users.SetTestUser(user)
	return user
}

func captureMessages(t *testing.T) *[]string {
	t.Helper()
	messages := []string{}
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		messages = append(messages, e.(events.Message).Text)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	return &messages
}

func joinMessages(messages []string) string {
	out := ""
	for _, m := range messages {
		out += m
	}
	return out
}

func TestStableAssignsConfiguredMountType(t *testing.T) {
	user := testUser(t, 7)
	store := &fakeStore{}
	module := newTestModule(store)

	msg := module.stable(user, "pack-horse")
	assert.Contains(t, msg, "stable a pack-horse")
	require.Contains(t, module.mounts, 7)
	assert.Equal(t, "pack-horse", module.mounts[7].Type)
	assert.Contains(t, store.saved.Mounts, 7, "stable must persist")
}

func TestStableRefusesUnknownType(t *testing.T) {
	user := testUser(t, 7)
	module := newTestModule(&fakeStore{})

	msg := module.stable(user, "dragon")
	assert.Contains(t, msg, "no mount type")
	assert.Empty(t, module.mounts)
}

func TestReleaseRemovesAssignment(t *testing.T) {
	user := testUser(t, 7)
	store := &fakeStore{}
	module := newTestModule(store)
	module.stable(user, "pack-horse")

	msg := module.release(7)
	assert.Contains(t, msg, "release")
	assert.NotContains(t, module.mounts, 7)
	assert.NotContains(t, store.saved.Mounts, 7)

	msg = module.release(7)
	assert.Contains(t, msg, "no mount to release")
}

func TestStableRollsBackOnFailedSave(t *testing.T) {
	user := testUser(t, 7)
	store := &fakeStore{saveErr: assert.AnError}
	module := newTestModule(store)

	msg := module.stable(user, "pack-horse")
	assert.Contains(t, msg, assert.AnError.Error())
	assert.Empty(t, module.mounts, "a failed save must not leave an in-memory mount")
}

func TestReleaseRollsBackOnFailedSave(t *testing.T) {
	user := testUser(t, 7)
	store := &fakeStore{}
	module := newTestModule(store)
	module.stable(user, "pack-horse")
	store.saveErr = assert.AnError

	msg := module.release(7)
	assert.Contains(t, msg, assert.AnError.Error())
	assert.Contains(t, module.mounts, 7, "a failed save must roll the release back")
}

func TestCapacityBonusGramsRequiresKnownMountType(t *testing.T) {
	user := testUser(t, 7)
	module := newTestModule(&fakeStore{})

	assert.Equal(t, 0, module.CapacityBonusGrams(7), "no mount yet")

	module.stable(user, "pack-horse")
	assert.Equal(t, 100000, module.CapacityBonusGrams(7))

	module.specs = map[string]mount.MountSpec{}
	assert.Equal(t, 0, module.CapacityBonusGrams(7), "type removed from config")
}

func TestStatusRendersAssignmentOrPrompt(t *testing.T) {
	user := testUser(t, 7)
	module := newTestModule(&fakeStore{})

	assert.Contains(t, module.status(7), "no mount")

	module.stable(user, "pack-horse")
	status := module.status(7)
	assert.Contains(t, status, "pack-horse")
	assert.Contains(t, status, "100.0 kg")
}

func TestUserCommandDispatchAndUsage(t *testing.T) {
	user := testUser(t, 7)
	module := newTestModule(&fakeStore{})
	messages := captureMessages(t)

	_, err := module.userCommand("bogus", user, nil, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, joinMessages(*messages), mountUsage)

	*messages = nil
	_, err = module.userCommand("stable pack-horse", user, nil, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, joinMessages(*messages), "stable a pack-horse")
}

func TestRecoveryRetainsInvalidAndUnknownTypeMountsForRepair(t *testing.T) {
	saved := Registry{Mounts: map[int]mount.Mount{
		7: {LeaderUserID: 7, Type: "dragon"},
	}}
	store := &fakeStore{saved: saved}
	module := newTestModule(store)

	module.load()

	require.Contains(t, module.mounts, 7, "unknown mount type is retained, not dropped")
	assert.Equal(t, "dragon", module.mounts[7].Type)
}

func TestParseMountSpecsRejectsMalformedEntries(t *testing.T) {
	raw := []any{
		map[string]any{"type": "pack-horse", "description": "Horse.", "cargocapacitybonuskg": 100.0, "traveldurationpct": 90},
		map[string]any{"type": "", "cargocapacitybonuskg": 50.0},
		map[string]any{"type": "broken-mule", "cargocapacitybonuskg": -5.0},
	}

	specs := parseMountSpecs(raw)

	require.Contains(t, specs, "pack-horse")
	assert.Equal(t, 100000, specs["pack-horse"].CargoCapacityBonusGrams)
	assert.NotContains(t, specs, "")
	assert.NotContains(t, specs, "broken-mule")
}
