package mount

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mount"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
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

const (
	packSaddleID   = 988401
	ridingSaddleID = 988402
)

func testSpecs() map[string]mount.MountSpec {
	return map[string]mount.MountSpec{
		"pack-horse":   {Type: "pack-horse", Kind: mount.KindPack, Description: "A sturdy pack horse.", Price: 120, BareCapacityGrams: 40000, SaddledCapacityGrams: 100000},
		"riding-horse": {Type: "riding-horse", Kind: mount.KindRiding, Description: "A riding horse.", Price: 150, BareCapacityGrams: 10000, SaddledCapacityGrams: 10000, TravelDurationPct: 90, FatiguePct: 75},
	}
}

func newTestModule(store Store) *MountModule {
	return &MountModule{
		store:     store,
		members:   func(int) int { return 1 },
		specs:     testSpecs(),
		stableTag: defaultStableTag,
		herds:     map[int]mount.Herd{},
	}
}

func testUser(t *testing.T, userId int) *users.UserRecord {
	t.Helper()
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(userId, 1)
	user.Character.Gold = 1000
	users.SetTestUser(user)
	return user
}

func stableRoom() *rooms.Room {
	room := rooms.NewRoom("test")
	room.Tags = []string{"stable"}
	return room
}

func saddles(t *testing.T) {
	t.Helper()
	items.SetTestItemSpec(&items.ItemSpec{ItemId: packSaddleID, Name: "pack saddle", NameSimple: "saddle", Type: items.Object, Weight: 9000, Saddle: items.SaddlePack})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: ridingSaddleID, Name: "riding saddle", NameSimple: "saddle", Type: items.Object, Weight: 7000, Saddle: items.SaddleRiding})
	t.Cleanup(func() {
		items.RemoveTestItemSpec(packSaddleID)
		items.RemoveTestItemSpec(ridingSaddleID)
	})
}

func carrying(user *users.UserRecord, itemID int) int {
	n := 0
	for _, itm := range user.Character.Items {
		if itm.ItemId == itemID {
			n++
		}
	}
	return n
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

// Phase 32f: horses are bought at a stable, for gold, one riding and one
// pack horse per member.
func TestStableBuysAHorseAtAStable(t *testing.T) {
	user := testUser(t, 7)
	store := &fakeStore{}
	module := newTestModule(store)

	assert.Contains(t, module.stable(user, rooms.NewRoom("test"), "pack-horse"), "no stable here")
	assert.Contains(t, module.stable(user, nil, "pack-horse"), "no stable here")
	assert.Empty(t, module.herds)

	msg := module.stable(user, stableRoom(), "pack-horse")
	assert.Contains(t, msg, "You buy a pack horse (#1) for")
	assert.Equal(t, 880, user.Character.Gold)
	require.Contains(t, store.saved.Herds, 7, "stable persists")
	assert.Equal(t, []mount.Horse{{ID: 1, Type: "pack-horse"}}, store.saved.Herds[7].Horses)

	assert.Contains(t, module.stable(user, stableRoom(), "pack-horse"), "one pack horse per member: 1 already for 1")
	assert.Equal(t, 880, user.Character.Gold, "a refusal costs nothing")
	assert.Contains(t, module.stable(user, stableRoom(), "riding-horse"), "You buy a riding horse (#2)")

	module.members = func(int) int { return 2 }
	assert.Contains(t, module.stable(user, stableRoom(), "pack-horse"), "(#3)")
	assert.Contains(t, module.stable(user, stableRoom(), "pack-horse"), "2 already for 2")

	module.members = func(int) int { return 1 } // a companion left
	assert.Len(t, module.herds[7].Horses, 3, "a company that shrinks keeps its horses")
	assert.Contains(t, module.stable(user, stableRoom(), "riding-horse"), "1 already for 1")
}

func TestStableRefusals(t *testing.T) {
	user := testUser(t, 7)
	module := newTestModule(&fakeStore{})

	assert.Contains(t, module.stable(user, stableRoom(), "dragon"), "no horse type")
	user.Character.Gold = 100
	assert.Contains(t, module.stable(user, stableRoom(), "pack-horse"), "don't have enough")
	assert.Equal(t, 100, user.Character.Gold)
	assert.Empty(t, module.herds)

	list := module.stable(user, stableRoom(), "")
	assert.Contains(t, list, "pack-horse")
	assert.Contains(t, list, "120 gold")
	assert.Less(t, indexOf(list, "pack-horse"), indexOf(list, "riding-horse"), "cheapest first")
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestStableRollsBackOnFailedSave(t *testing.T) {
	user := testUser(t, 7)
	module := newTestModule(&fakeStore{saveErr: assert.AnError})

	msg := module.stable(user, stableRoom(), "pack-horse")
	assert.Contains(t, msg, assert.AnError.Error())
	assert.Empty(t, module.herds, "a failed save leaves no horse")
	assert.Equal(t, 1000, user.Character.Gold, "and takes no gold")
}

func TestSaddleFitsMatchingKindAndSwaps(t *testing.T) {
	saddles(t)
	user := testUser(t, 7)
	store := &fakeStore{}
	module := newTestModule(store)
	module.stable(user, stableRoom(), "pack-horse")
	user.Character.Items = []items.Item{items.New(packSaddleID), items.New(ridingSaddleID), items.New(packSaddleID)}

	assert.Contains(t, module.saddle(user, "pack", "riding saddle"), "needs a pack saddle")
	assert.Equal(t, 40000, module.CapacityBonusGrams(7))

	msg := module.saddle(user, "#1", "pack saddle")
	assert.Contains(t, msg, "It carries 100.0 kg")
	assert.Equal(t, 100000, module.CapacityBonusGrams(7))
	assert.Equal(t, 1, carrying(user, packSaddleID), "the saddle left the pack")
	assert.Equal(t, packSaddleID, store.saved.Herds[7].Horses[0].SaddleItemId)

	msg = module.saddle(user, "horse", "pack saddle")
	assert.Contains(t, msg, "You keep the pack saddle it wore")
	assert.Equal(t, 1, carrying(user, packSaddleID), "one on, one back")

	assert.Contains(t, module.saddle(user, "#9", "pack saddle"), "no horse")
	assert.Contains(t, module.saddle(user, "#1", "rope"), "don't have")
}

func TestUnsaddleAndReleaseReturnTheSaddle(t *testing.T) {
	saddles(t)
	user := testUser(t, 7)
	module := newTestModule(&fakeStore{})
	module.stable(user, stableRoom(), "pack-horse")
	module.stable(user, stableRoom(), "riding-horse")
	user.Character.Items = []items.Item{items.New(packSaddleID), items.New(ridingSaddleID)}
	module.saddle(user, "pack", "pack saddle")
	module.saddle(user, "riding", "riding saddle")
	assert.Empty(t, user.Character.Items)

	assert.Contains(t, module.unsaddle(user, "pack"), "take the pack saddle off")
	assert.Equal(t, 1, carrying(user, packSaddleID))
	assert.Contains(t, module.unsaddle(user, "pack"), "has no saddle")

	assert.Contains(t, module.release(user, "riding"), "keeping its riding saddle")
	assert.Equal(t, 1, carrying(user, ridingSaddleID))
	assert.Len(t, module.herds[7].Horses, 1)
	assert.Contains(t, module.release(user, "#1"), "You release your pack horse (#1).")
	assert.Contains(t, module.release(user, "#1"), "no horses to release")
}

func TestSaddleRollsBackOnFailedSave(t *testing.T) {
	saddles(t)
	user := testUser(t, 7)
	store := &fakeStore{}
	module := newTestModule(store)
	module.stable(user, stableRoom(), "pack-horse")
	user.Character.Items = []items.Item{items.New(packSaddleID)}
	store.saveErr = assert.AnError

	assert.Contains(t, module.saddle(user, "pack", "pack saddle"), assert.AnError.Error())
	assert.Equal(t, 1, carrying(user, packSaddleID), "the saddle stays in the pack")
	assert.False(t, module.herds[7].Horses[0].Saddled())
	assert.Contains(t, module.release(user, "pack"), assert.AnError.Error())
	assert.Len(t, module.herds[7].Horses, 1)
}

func TestCapacityBonusGramsRequiresKnownType(t *testing.T) {
	user := testUser(t, 7)
	module := newTestModule(&fakeStore{})
	assert.Zero(t, module.CapacityBonusGrams(7), "no horses yet")
	module.stable(user, stableRoom(), "pack-horse")
	assert.Equal(t, 40000, module.CapacityBonusGrams(7), "bare")
	module.specs = map[string]mount.MountSpec{}
	assert.Zero(t, module.CapacityBonusGrams(7), "type removed from config")
}

func TestStatusListsHorses(t *testing.T) {
	saddles(t)
	user := testUser(t, 7)
	module := newTestModule(&fakeStore{})
	assert.Contains(t, module.status(7), "You have no horses")

	module.stable(user, stableRoom(), "pack-horse")
	module.stable(user, stableRoom(), "riding-horse")
	user.Character.Items = []items.Item{items.New(ridingSaddleID)}
	module.saddle(user, "riding", "riding saddle")
	status := module.status(7)
	assert.Contains(t, status, "Your horses (riding 1 of 1, pack 1 of 1)")
	assert.Contains(t, status, "#1 pack horse, no saddle: carries 40.0 kg.")
	assert.Contains(t, status, "#2 riding horse, riding saddle: carries a rider and 10.0 kg.")

	views := module.Herd(7)
	require.Len(t, views, 2)
	assert.Equal(t, mount.HorseView{ID: 2, Name: "riding horse", Kind: mount.KindRiding, Saddle: "riding saddle", CapacityGrams: 10000}, views[1])
}

func TestUserCommandDispatchAndUsage(t *testing.T) {
	user := testUser(t, 7)
	module := newTestModule(&fakeStore{})
	messages := captureMessages(t)
	run := func(rest string, room *rooms.Room) string {
		*messages = nil
		_, err := module.userCommand(rest, user, room, 0)
		require.NoError(t, err)
		events.ProcessEvents()
		return joinMessages(*messages)
	}
	assert.Contains(t, run("bogus", nil), mountUsage)
	assert.Contains(t, run("saddle pack", nil), mountUsage)
	assert.Contains(t, run("stable pack-horse", stableRoom()), "You buy a pack horse")
	assert.Contains(t, run("", nil), "#1 pack horse")
}

// Phase 32f: an old save's single mount becomes a herd of one, with the
// legacy pack saddle fitted, so it carries what it did.
func TestLoadMigratesSingleMountToHerd(t *testing.T) {
	data := []byte("mounts:\n  7:\n    leaderuserid: 7\n    type: pack-horse\n  8:\n    leaderuserid: 8\n    type: dragon\n")
	var legacy Registry
	require.NoError(t, decodeRegistry(data, &legacy))
	store := &fakeStore{saved: legacy}
	module := newTestModule(store)
	module.legacySaddle = packSaddleID

	module.load()

	require.Contains(t, module.herds, 7)
	assert.Equal(t, []mount.Horse{{ID: 1, Type: "pack-horse", SaddleItemId: packSaddleID}}, module.herds[7].Horses)
	assert.Equal(t, 100000, module.CapacityBonusGrams(7), "the same 100 kg as before")
	require.Contains(t, module.herds, 8, "an unknown type is retained for repair")
	assert.Zero(t, module.CapacityBonusGrams(8))

	require.NoError(t, module.save())
	assert.Empty(t, store.saved.Mounts, "the old shape is never written again")
	module.load()
	assert.Equal(t, packSaddleID, module.herds[7].Horses[0].SaddleItemId, "reloading keeps the herd")
}

func TestHerdSurvivesAReload(t *testing.T) {
	saddles(t)
	user := testUser(t, 7)
	store := &fakeStore{}
	module := newTestModule(store)
	module.stable(user, stableRoom(), "riding-horse")
	user.Character.Items = []items.Item{items.New(ridingSaddleID)}
	module.saddle(user, "riding", "riding saddle")

	fresh := newTestModule(store)
	fresh.load()
	assert.Equal(t, module.herds[7], fresh.herds[7])
	_, riders := fresh.Relief(7)
	assert.Equal(t, 1, riders)
}

func TestParseMountSpecs(t *testing.T) {
	specs := parseMountSpecs([]any{
		map[any]any{"Type": "Pack-Horse", "Kind": "pack", "Price": 120, "BareCapacityKg": 40, "SaddledCapacityKg": 100.0},
		map[any]any{"Type": "riding-horse", "Kind": "riding", "SaddledCapacityKg": 10, "TravelDurationPct": 90, "FatiguePct": 75},
		map[string]any{"type": "", "kind": "pack"},
		map[string]any{"type": "cart", "kind": "wagon"},
		map[string]any{"type": "broken-mule", "kind": "pack", "barecapacitykg": -5.0},
		map[string]any{"type": "bad-fatigue", "kind": "riding", "fatiguepct": 500},
		map[any]any{"Type": "pack-horse", "Kind": "pack"},
	})
	require.Len(t, specs, 2, "malformed entries and a duplicate are rejected")
	assert.Equal(t, mount.MountSpec{Type: "pack-horse", Kind: mount.KindPack, Price: 120, BareCapacityGrams: 40000, SaddledCapacityGrams: 100000}, specs["pack-horse"])
	assert.Equal(t, 75, specs["riding-horse"].FatiguePct)
}

// TestReliefThroughRegisteredSeam reads relief the way modules/walking and
// modules/expedition do: through internal/mount's package functions.
func TestReliefThroughRegisteredSeam(t *testing.T) {
	saddles(t)
	user := testUser(t, 7)
	module := newTestModule(&fakeStore{})
	module.members = func(int) int { return 2 }
	mount.SetProvider(module)
	t.Cleanup(func() { mount.SetProvider(nil) })

	pct, riders := mount.Relief(7)
	assert.Equal(t, 100, pct, "no horses: no relief")
	assert.Zero(t, riders)

	module.stable(user, stableRoom(), "riding-horse")
	module.stable(user, stableRoom(), "riding-horse")
	user.Character.Items = []items.Item{items.New(ridingSaddleID), items.New(ridingSaddleID)}
	module.saddle(user, "#1", "riding saddle")
	pct, riders = mount.Relief(7)
	assert.Equal(t, 75, pct)
	assert.Equal(t, 1, riders)
	assert.Equal(t, 100, mount.TravelDurationPct(7), "one of two walks: walking pace")

	module.saddle(user, "#2", "riding saddle")
	_, riders = mount.Relief(7)
	assert.Equal(t, 2, riders)
	assert.Equal(t, 90, mount.TravelDurationPct(7), "everyone rides")
	assert.Equal(t, 20000, mount.CapacityBonus(7))
	assert.Len(t, mount.HerdOf(7), 2)
}
