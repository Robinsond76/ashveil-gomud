package walking

import (
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/walking"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Wiring: these tests go through the real user Go command, the real
// internal/walking step seam, the real room manager and users, and the
// module's native companion lookup (survival roster + company formation
// seam + a live charmed mob instance). Only survival's drain is faked.

type rosterStub struct{ refs map[int][]survival.MemberRef }

func (r rosterStub) Roster(leader int) []survival.MemberRef { return r.refs[leader] }

type formationStub struct{ instances map[int]int }

func (f formationStub) FormationFor(int) (company.Formation, bool) { return company.Formation{}, false }
func (f formationStub) InstanceFor(_ int, companionID int) (int, bool) {
	id, ok := f.instances[companionID]
	return id, ok
}
func (f formationStub) LeaderAndKeyForInstance(int) (int, company.MemberKey, bool) {
	return 0, "", false
}

type drainRecorder struct{ calls []drainCall }

func (d *drainRecorder) ApplyMemberDrain(leader int, key survival.MemberKey, cost survival.Exertion) (survival.ExertionResult, error) {
	d.calls = append(d.calls, drainCall{leader, key, cost})
	return survival.ExertionResult{Member: key}, nil
}

func (d *drainRecorder) total() map[survival.MemberKey]int {
	out := map[survival.MemberKey]int{}
	for _, c := range d.calls {
		out[c.key] += c.cost.Fatigue
	}
	return out
}

type startStub struct{ calls int }

func (s *startStub) StartTravel(expedition.StartRequest) (bool, error) {
	s.calls++
	return true, nil
}

type wiringWorld struct {
	user   *users.UserRecord
	bran   *mobs.Mob
	drains *drainRecorder
	module *WalkingModule
}

// Rooms: 95001 (forest) <-> 95002 (forest), 95003 (city) <-> 95004 (city),
// 95005 (forest) has a travel exit to 95006.
var aliasesOnce sync.Once

// loadAliases loads the shipped world's direction aliases once per binary
// (configs.AddOverlayOverrides only applies a key the first time).
func loadAliases(t *testing.T) {
	t.Helper()
	aliasesOnce.Do(func() {
		_, thisFile, _, _ := runtime.Caller(0)
		dataDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
		require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir}))
		keywords.LoadAliases()
	})
}

func setupWiring(t *testing.T) *wiringWorld {
	t.Helper()
	setup(t) // test biomes, buffs, flags
	loadAliases(t)
	mk := func(id int, biome string, exits map[string]exit.RoomExit) {
		r := &rooms.Room{RoomId: id, Zone: "WiringWood", Title: "Room", Biome: biome, Exits: exits}
		rooms.SetTestRoom(r)
		t.Cleanup(func() { rooms.RemoveTestRoom(id) })
	}
	mk(95001, "forest", map[string]exit.RoomExit{"north": {RoomId: 95002}})
	mk(95002, "forest", map[string]exit.RoomExit{"south": {RoomId: 95001}})
	mk(95003, "city", map[string]exit.RoomExit{"north": {RoomId: 95004}})
	mk(95004, "city", map[string]exit.RoomExit{"south": {RoomId: 95003}})
	mk(95005, "forest", map[string]exit.RoomExit{"north": {RoomId: 95006, TravelProfile: "oak-road"}})
	mk(95006, "forest", map[string]exit.RoomExit{"south": {RoomId: 95005}})

	w := &wiringWorld{drains: &drainRecorder{}}
	w.user = users.NewUserRecord(7, 7)
	w.user.Character.Name = "Hero"
	w.user.Character.RoomId = 95001
	w.user.Character.ActionPoints = 100
	w.user.Character.Validate()
	users.SetTestUser(w.user)

	w.bran = &mobs.Mob{InstanceId: 95901, Character: *characters.New()}
	w.bran.Character.Name = "Bran"
	w.bran.Character.RoomId = 95001
	w.bran.Character.Charm(7, characters.CharmPermanent, "")
	mobs.SetTestInstance(w.bran)
	t.Cleanup(func() { mobs.RemoveTestInstance(95901) })

	survival.SetRosterProvider(rosterStub{refs: map[int][]survival.MemberRef{7: {
		{Key: survival.LeaderMemberKey, Name: "Hero"},
		{Key: survival.CompanionMemberKey(1), Name: "Bran"},
	}}})
	company.SetFormationProvider(formationStub{instances: map[int]int{1: 95901}})
	survival.SetMemberDrainService(w.drains)
	t.Cleanup(func() {
		survival.SetRosterProvider(nil)
		company.SetFormationProvider(nil)
		survival.SetMemberDrainService(nil)
	})

	// The real module with its native seams (rooms, users, roster, drain).
	w.module = newModule()
	walking.SetStepProvider(w.module)
	t.Cleanup(func() { walking.SetStepProvider(nil) })
	return w
}

func (w *wiringWorld) place(roomId int) {
	w.user.Character.RoomId = roomId
	w.bran.Character.RoomId = roomId
	w.user.Character.ActionPoints = 100
}

func TestGoInForestChargesLeaderAndCharmedCompanion(t *testing.T) {
	w := setupWiring(t)
	w.place(95001)

	handled, err := usercommands.Go("north", w.user, rooms.LoadRoom(95001), 0)
	require.NoError(t, err)
	require.True(t, handled)
	require.Equal(t, 95002, w.user.Character.RoomId, "the move itself happens")
	handled, err = usercommands.Go("south", w.user, rooms.LoadRoom(95002), 0)
	require.NoError(t, err)
	require.True(t, handled)

	// Two forest steps of 50 strain: one whole fatigue point each.
	assert.Equal(t, map[survival.MemberKey]int{
		survival.LeaderMemberKey:       1,
		survival.CompanionMemberKey(1): 1,
	}, w.drains.total())
}

func TestGoInCityChargesNothing(t *testing.T) {
	w := setupWiring(t)
	w.place(95003)
	for i := 0; i < 10; i++ {
		from, dir := 95003, "north"
		if i%2 == 1 {
			from, dir = 95004, "south"
		}
		_, err := usercommands.Go(dir, w.user, rooms.LoadRoom(from), 0)
		require.NoError(t, err)
	}
	assert.Equal(t, 95003, w.user.Character.RoomId)
	assert.Empty(t, w.drains.calls)
	assert.Empty(t, w.module.registry.Carry)
}

// TestRouteStartAndArrivalChargeNoWalkingStrain: a profiled exit starts a
// journey instead of stepping, and a route arrival moves the leader through
// the room manager directly, never through Go, so neither is charged.
func TestRouteStartAndArrivalChargeNoWalkingStrain(t *testing.T) {
	w := setupWiring(t)
	w.place(95005)
	starter := &startStub{}
	expedition.SetStartProvider(starter)
	t.Cleanup(func() { expedition.SetStartProvider(nil) })

	for i := 0; i < 4; i++ {
		_, err := usercommands.Go("north", w.user, rooms.LoadRoom(95005), 0)
		require.NoError(t, err)
	}
	assert.Equal(t, 4, starter.calls)
	assert.Equal(t, 95005, w.user.Character.RoomId)

	// The expedition module's arrival path.
	require.NoError(t, rooms.MoveToRoom(7, 95006))
	assert.Equal(t, 95006, w.user.Character.RoomId)
	assert.Empty(t, w.drains.calls)
	assert.Empty(t, w.module.registry.Carry)
}
