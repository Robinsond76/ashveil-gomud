package death

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/death"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "low", "", false)
	os.Exit(m.Run())
}

// testWorld is a death module on fake seams: a room map, a move that only
// sets the character's room, and journeys and a company that log calls.
type testWorld struct {
	module    *DeathModule
	user      *users.UserRecord
	rooms     map[int]*rooms.Room
	calls     *[]string
	moveErr   error
	travelErr error
	campErr   error
	companion int // how many companions relocate reports
	messages  *[]string
}

func newTestWorld(t *testing.T) *testWorld {
	t.Helper()
	w := &testWorld{calls: &[]string{}, rooms: map[int]*rooms.Room{
		18:   {RoomId: 18, Zone: "Frostfang", Title: "The Sanctuary of the Benevolent Heart", Tags: []string{"church"}},
		17:   {RoomId: 17, Zone: "Frostfang", Title: "Temple Row"},
		2001: {RoomId: 2001, Zone: "Dunmar", Title: "Dunmar West Gate"},
		2007: {RoomId: 2007, Zone: "Dunmar", Title: "The Chapel of the Wayfarer", Tags: []string{"church"}},
		2002: {RoomId: 2002, Zone: "Old Kings Road", Title: "Fork at the Black Oak"},
		2005: {RoomId: 2005, Zone: "Old Kings Road", Title: "Trappers' Post", Tags: []string{"shaman"}},
		3001: {RoomId: 3001, Zone: "Bleakmoor", Title: "Bleakmoor Square"},
		3002: {RoomId: 3002, Zone: "Bleakmoor", Title: "Bleakmoor Hall"}, // registered, but no church tag
	}}
	m := newModule()
	m.lookupUser = func(id int) *users.UserRecord {
		if w.user != nil && w.user.UserId == id {
			return w.user
		}
		return nil
	}
	m.loadRoom = func(id int) *rooms.Room { return w.rooms[id] }
	m.moveToRoom = func(userID, roomID int) error {
		*w.calls = append(*w.calls, "move")
		if w.moveErr != nil {
			return w.moveErr
		}
		w.user.Character.RoomId = roomID
		w.user.Character.Zone = w.rooms[roomID].Zone
		return nil
	}
	m.abandonTravel = func(int) error {
		*w.calls = append(*w.calls, "travel")
		return w.travelErr
	}
	m.abandonCamp = func(int) error {
		*w.calls = append(*w.calls, "camp")
		return w.campErr
	}
	m.relocate = func(_, roomID int) int {
		*w.calls = append(*w.calls, "company")
		return w.companion
	}
	m.round = func() uint64 { return 1400000 }
	m.cfg = parseSettings(configMap(map[string]any{
		"Settlements": []any{
			map[string]any{"Zone": "Frostfang", "Kind": "city", "ServiceRoomId": 18},
			map[string]any{"Zone": "Dunmar", "Kind": "city", "ServiceRoomId": 2007},
			map[string]any{"Zone": "Old Kings Road", "Kind": "village", "ServiceRoomId": 2005},
			map[string]any{"Zone": "Bleakmoor", "Kind": "city", "ServiceRoomId": 3002},
		},
		"FallbackRoomId":   18,
		"RespawnVitalsPct": 50,
	}))
	w.module = m

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	w.user = users.NewUserRecord(7, 1)
	w.user.Character.Name = "Wren"
	w.user.Character.Level = 6
	w.user.Character.Experience = w.user.Character.XPTL(5) + 40
	w.user.Character.RoomId = 2002
	w.user.Character.Validate()
	w.user.Character.Health = -10
	users.SetTestUser(w.user)

	messages := []string{}
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if msg := e.(events.Message); msg.UserId == w.user.UserId {
			messages = append(messages, msg.Text)
		}
		return events.Continue
	})
	t.Cleanup(func() {
		events.ProcessEvents()
		events.UnregisterListener(events.Message{}, id)
	})
	w.messages = &messages
	return w
}

func configMap(values map[string]any) func(string) any {
	return func(key string) any { return values[key] }
}

// text drains the event queue and returns the messages the user got since
// the last call.
func (w *testWorld) text() string {
	events.ProcessEvents()
	out := strings.Join(*w.messages, "\n")
	*w.messages = nil
	return out
}

func (w *testWorld) enter(roomID int) {
	w.user.Character.RoomId = roomID
	w.module.onRoomChange(events.RoomChange{UserId: w.user.UserId, ToRoomId: roomID})
}

func TestParseConfig(t *testing.T) {
	s := parseSettings(configMap(map[string]any{}))
	assert.Equal(t, defaultFallbackRoomID, s.fallbackID)
	assert.Equal(t, defaultVitalsPct, s.vitalsPct)
	assert.Empty(t, s.registry.Settlements())

	s = parseSettings(configMap(map[string]any{
		"Settlements": []any{
			map[any]any{"Zone": "Dunmar", "Kind": "city", "ServiceRoomId": 2007},
			"not a map",
			map[string]any{"Zone": "Dunmar", "Kind": "city", "ServiceRoomId": 9},
			map[string]any{"Zone": "Ashford", "Kind": "hamlet", "ServiceRoomId": 9},
			map[string]any{"Zone": "Old Kings Road", "Kind": "village", "ServiceRoomId": "2005"},
		},
		"FallbackRoomId":   0,
		"RespawnVitalsPct": 150,
	}))
	assert.Equal(t, []domain.Settlement{
		{Zone: "Dunmar", Kind: domain.City, ServiceRoomID: 2007},
		{Zone: "Old Kings Road", Kind: domain.Village, ServiceRoomID: 2005},
	}, s.registry.Settlements())
	assert.Equal(t, defaultFallbackRoomID, s.fallbackID, "a room below 1 uses the default")
	assert.Equal(t, defaultVitalsPct, s.vitalsPct, "out of range uses the default")

	s = parseSettings(configMap(map[string]any{"FallbackRoomId": 2007, "RespawnVitalsPct": 100}))
	assert.Equal(t, 2007, s.fallbackID)
	assert.Equal(t, 100, s.vitalsPct)
}

func TestCheckpointSetOnEnteringCity(t *testing.T) {
	w := newTestWorld(t)

	w.enter(2001)

	assert.Equal(t, 2007, checkpoint(w.user.Character), "any room of the city")
	assert.Contains(t, w.text(), "Should you fall, you will wake in The Chapel of the Wayfarer.")

	w.enter(17)
	assert.Equal(t, 18, checkpoint(w.user.Character), "the last city visited")
}

func TestCheckpointIgnoresVillageAndChurchless(t *testing.T) {
	w := newTestWorld(t)
	w.enter(2001)
	w.text()

	w.enter(2005) // a village with a shaman
	w.enter(2002) // a village road
	w.enter(3001) // a registered city whose church lacks the tag
	w.module.onRoomChange(events.RoomChange{MobInstanceId: 5, ToRoomId: 17})

	assert.Equal(t, 2007, checkpoint(w.user.Character))
	assert.Empty(t, w.text())
}

func TestCheckpointToldOncePerChange(t *testing.T) {
	w := newTestWorld(t)
	w.enter(2001)
	w.enter(2007)
	w.module.onPlayerSpawn(events.PlayerSpawn{UserId: w.user.UserId})

	assert.Equal(t, 1, strings.Count(w.text(), "Should you fall"))
}

func TestCheckpointSetOnSpawn(t *testing.T) {
	w := newTestWorld(t)
	w.user.Character.RoomId = 17

	w.module.onPlayerSpawn(events.PlayerSpawn{UserId: w.user.UserId})

	assert.Equal(t, 18, checkpoint(w.user.Character))
}

func TestRespawnAtCheckpoint(t *testing.T) {
	w := newTestWorld(t)
	w.enter(2001)
	w.user.Character.RoomId = 2002
	w.companion = 2
	w.text()

	w.module.Respawn(w.user.UserId, true)

	c := w.user.Character
	assert.Equal(t, []string{"travel", "camp", "move", "company"}, *w.calls, "journeys end before the move; the company follows it")
	assert.Equal(t, 2007, c.RoomId)
	assert.Equal(t, 5, c.Level)
	assert.Equal(t, c.XPTL(4), c.Experience)
	assert.Equal(t, c.HealthMax.Value*50/100, c.Health)
	assert.Equal(t, c.ManaMax.Value*50/100, c.Mana)
	assert.False(t, w.module.Pending(w.user.UserId), "the mark is cleared")
	assert.Nil(t, c.GetMiscData(domain.PendingKey))
	text := w.text()
	assert.Contains(t, text, "You lose a level (now level")
	assert.Contains(t, text, "You wake before the altar of The Chapel of the Wayfarer.")
	assert.Contains(t, text, "Your company is with you.")
}

func TestRespawnAloneSaysNothingOfTheCompany(t *testing.T) {
	w := newTestWorld(t)
	w.enter(2001)

	w.module.Respawn(w.user.UserId, true)

	assert.NotContains(t, w.text(), "Your company")
}

func TestRespawnInvalidCheckpointUsesFallback(t *testing.T) {
	w := newTestWorld(t)
	w.enter(2001)
	w.rooms[2007].Tags = nil // the chapel lost its church

	w.module.Respawn(w.user.UserId, true)

	assert.Equal(t, 18, w.user.Character.RoomId)
	assert.False(t, w.module.Pending(w.user.UserId))

	w2 := newTestWorld(t)
	w2.module.Respawn(w2.user.UserId, true)
	assert.Equal(t, 18, w2.user.Character.RoomId, "no checkpoint at all")
}

func TestRespawnNoChurchStaysPending(t *testing.T) {
	w := newTestWorld(t)
	w.enter(2001)
	w.rooms[2007].Tags = nil
	fallback := w.rooms[18]
	delete(w.rooms, 18)
	w.text()

	w.module.Respawn(w.user.UserId, true)

	c := w.user.Character
	assert.True(t, w.module.Pending(w.user.UserId))
	assert.Equal(t, "death-7-1400000", c.GetMiscData(domain.PendingKey))
	assert.Equal(t, 5, c.Level, "the level is taken")
	assert.Equal(t, 2001, c.RoomId, "not moved anywhere")
	assert.Equal(t, -10, c.Health, "so the engine retries")
	assert.Empty(t, *w.calls, "no journey is ended for nothing")
	assert.Contains(t, w.text(), "The way back is closed to you.")

	w.module.Respawn(w.user.UserId, false)
	assert.Equal(t, 5, c.Level, "a retry takes no second level")
	assert.Empty(t, w.text(), "and says nothing new")

	w.rooms[18] = fallback // repaired
	w.module.Respawn(w.user.UserId, false)
	assert.Equal(t, 5, c.Level)
	assert.Equal(t, 18, c.RoomId)
	assert.False(t, w.module.Pending(w.user.UserId))
	text := w.text()
	assert.NotContains(t, text, "You lose a level")
	assert.Contains(t, text, "You wake before the altar")
}

func TestRespawnAbandonFailureStaysPending(t *testing.T) {
	for name, fail := range map[string]func(*testWorld){
		"travel": func(w *testWorld) { w.travelErr = errors.New("travel store down") },
		"camp":   func(w *testWorld) { w.campErr = errors.New("camp store down") },
		"move":   func(w *testWorld) { w.moveErr = errors.New("room gone") },
	} {
		t.Run(name, func(t *testing.T) {
			w := newTestWorld(t)
			w.enter(2001)
			w.user.Character.RoomId = 2002
			fail(w)

			w.module.Respawn(w.user.UserId, true)

			assert.True(t, w.module.Pending(w.user.UserId))
			assert.Equal(t, 2002, w.user.Character.RoomId)
			assert.Equal(t, -10, w.user.Character.Health)
			assert.NotContains(t, *w.calls, "company", "the company waits with the leader")

			w.travelErr, w.campErr, w.moveErr = nil, nil, nil
			w.module.Respawn(w.user.UserId, false)
			assert.False(t, w.module.Pending(w.user.UserId))
			assert.Equal(t, 5, w.user.Character.Level, "one level for the one death")
			assert.Equal(t, 2007, w.user.Character.RoomId)
		})
	}
}

func TestRespawnLevelOne(t *testing.T) {
	w := newTestWorld(t)
	w.user.Character.Level = 1
	w.user.Character.Experience = 300
	w.user.Character.Validate()

	w.module.Respawn(w.user.UserId, true)

	assert.Equal(t, 1, w.user.Character.Level)
	assert.Equal(t, 1, w.user.Character.Experience)
	assert.Contains(t, w.text(), "You lose what you had learned toward level 2.")
}

func TestRespawnUnknownUser(t *testing.T) {
	w := newTestWorld(t)
	w.module.Respawn(99, true)
	assert.False(t, w.module.Pending(99))
	assert.Empty(t, *w.calls)
}

// useAsProvider registers the test world's module as the death provider,
// so the real suicide command reaches it.
func (w *testWorld) useAsProvider(t *testing.T) {
	t.Helper()
	previous, _ := domain.Active()
	domain.SetProvider(w.module)
	t.Cleanup(func() { domain.SetProvider(previous) })
}

// deaths counts the PlayerDeath events announced from now on.
func deaths(t *testing.T) *int {
	t.Helper()
	count := 0
	id := events.RegisterListener(events.PlayerDeath{}, func(events.Event) events.ListenerReturn {
		count++
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.PlayerDeath{}, id) })
	return &count
}

// suicide runs the real suicide command where the player stands.
func (w *testWorld) suicide(t *testing.T) {
	t.Helper()
	handled, err := usercommands.Suicide("", w.user, w.rooms[w.user.Character.RoomId], 0)
	assert.NoError(t, err)
	assert.True(t, handled)
	events.ProcessEvents()
}

// TestDoubleQueuedSuicideIsOneDeath: the combat loop and AutoHeal can both
// queue a suicide for one death in the same round. The second finds the
// player already back at the church and is ignored.
func TestDoubleQueuedSuicideIsOneDeath(t *testing.T) {
	w := newTestWorld(t)
	w.useAsProvider(t)
	w.enter(2001)
	died := deaths(t)

	w.suicide(t)
	require.Equal(t, 2007, w.user.Character.RoomId)
	w.suicide(t)

	assert.Equal(t, 5, w.user.Character.Level, "one level for one death")
	assert.Equal(t, 1, *died, "one death announced")

	// Two rounds later, a suicide is a new, deliberate death.
	w.module.round = func() uint64 { return 1400002 }
	w.suicide(t)
	assert.Equal(t, 4, w.user.Character.Level)
}

// TestHealedPendingPlayerDiesAgain: a pending player healed back up and
// killed again has died twice, and pays twice.
func TestHealedPendingPlayerDiesAgain(t *testing.T) {
	w := newTestWorld(t)
	w.useAsProvider(t)
	w.enter(2001)
	w.rooms[2007].Tags = nil
	fallback := w.rooms[18]
	delete(w.rooms, 18)
	died := deaths(t)

	w.suicide(t)
	require.True(t, w.module.Pending(w.user.UserId))
	require.Equal(t, 5, w.user.Character.Level)

	w.user.Character.Health = 30 // someone gave aid
	w.suicide(t)
	assert.Equal(t, 4, w.user.Character.Level, "a second death")
	assert.Equal(t, 2, *died)
	assert.True(t, w.module.Pending(w.user.UserId))

	// While down, a retry costs nothing and leaves no corpse.
	w.user.Character.KillerMobName = "wolf"
	w.user.Character.KillerMobInstanceId = 12
	w.rooms[18] = fallback
	w.suicide(t)
	assert.Equal(t, 4, w.user.Character.Level)
	assert.Equal(t, 18, w.user.Character.RoomId)
	assert.Equal(t, 2, *died, "a retry announces nothing")
	assert.Empty(t, w.user.Character.KillerMobName, "a retry clears the stale killer")
	assert.Zero(t, w.user.Character.KillerMobInstanceId)
}

// TestPendingSurvivesReload: the mark is a string in MiscData, so a pending
// death saved and loaded again is still pending and costs no second level.
func TestPendingSurvivesReload(t *testing.T) {
	w := newTestWorld(t)
	w.enter(2001)
	w.rooms[2007].Tags = nil
	fallback := w.rooms[18]
	delete(w.rooms, 18)
	w.module.Respawn(w.user.UserId, true)
	require.Equal(t, 5, w.user.Character.Level)

	data, err := yaml.Marshal(w.user)
	require.NoError(t, err)
	reloaded := &users.UserRecord{}
	require.NoError(t, yaml.Unmarshal(data, reloaded))
	reloaded.UserId = w.user.UserId
	w.user = reloaded
	assert.True(t, w.module.Pending(w.user.UserId))
	assert.Equal(t, 2007, checkpoint(w.user.Character))

	w.rooms[18] = fallback
	w.module.Respawn(w.user.UserId, false)
	assert.Equal(t, 5, w.user.Character.Level)
	assert.Equal(t, 18, w.user.Character.RoomId)
	assert.False(t, w.module.Pending(w.user.UserId))
}

func TestRetryWithNothingOwedDoesNothing(t *testing.T) {
	w := newTestWorld(t)
	w.module.Respawn(w.user.UserId, false)
	assert.Equal(t, 6, w.user.Character.Level)
	assert.Empty(t, *w.calls)
}

func TestHoldClearsAggro(t *testing.T) {
	w := newTestWorld(t)
	w.moveErr = errors.New("room gone")
	w.user.Character.Aggro = &characters.Aggro{MobInstanceId: 4}

	w.module.Respawn(w.user.UserId, true)

	assert.Nil(t, w.user.Character.Aggro, "so AutoHeal retries")
}

// TestRespawnRecordsLastLoss (Phase 26a): a new death records the levels
// before and after, for experience; a retry doesn't change it, and it
// survives a YAML reload.
func TestRespawnRecordsLastLoss(t *testing.T) {
	w := newTestWorld(t)
	w.enter(2001)
	w.module.Respawn(w.user.UserId, true)
	from, to, ok := domain.LastLoss(w.user.Character)
	require.True(t, ok)
	assert.Equal(t, [2]int{6, 5}, [2]int{from, to})

	data, err := yaml.Marshal(w.user)
	require.NoError(t, err)
	reloaded := &users.UserRecord{}
	require.NoError(t, yaml.Unmarshal(data, reloaded))
	from, to, ok = domain.LastLoss(reloaded.Character)
	require.True(t, ok)
	assert.Equal(t, [2]int{6, 5}, [2]int{from, to})

	fresh := users.NewUserRecord(9, 1)
	_, _, ok = domain.LastLoss(fresh.Character)
	assert.False(t, ok)
}
