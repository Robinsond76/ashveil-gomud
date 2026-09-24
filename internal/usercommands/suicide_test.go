package usercommands

import (
	"maps"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/death"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDeathProvider struct {
	pending      bool
	justReturned bool
	respawns     []int
	newDeaths    []bool
}

func (f *fakeDeathProvider) Pending(int) bool      { return f.pending }
func (f *fakeDeathProvider) JustReturned(int) bool { return f.justReturned }
func (f *fakeDeathProvider) Respawn(userID int, newDeath bool) {
	f.respawns = append(f.respawns, userID)
	f.newDeaths = append(f.newDeaths, newDeath)
}

// deathConfig overrides the death config in memory for one test: the
// engine's level XP penalty, no protection levels, permadeath with no extra
// lives, every item dropped, and no corpse items.
func deathConfig(t *testing.T) {
	t.Helper()
	before := configs.Flatten(configs.GetOverrides())
	after := maps.Clone(before)
	after["GamePlay.Death.XPPenalty"] = "level"
	after["GamePlay.Death.ProtectionLevels"] = 0
	after["GamePlay.Death.PermaDeath"] = true
	after["GamePlay.Death.EquipmentDropChance"] = 1.0
	after["GamePlay.Death.CorpsesEnabled"] = true
	after["GamePlay.Death.CorpseItems"] = false
	after["GamePlay.Death.AlwaysDropBackpack"] = false
	after["SpecialRooms.DeathRecoveryRoom"] = 9402
	require.NoError(t, configs.RestoreOverrides(after))
	t.Cleanup(func() { _ = configs.RestoreOverrides(before) })
	require.Equal(t, "level", string(configs.GetGamePlayConfig().Death.XPPenalty))
	require.True(t, bool(configs.GetGamePlayConfig().Death.PermaDeath))
}

type suicideWorld struct {
	user   *users.UserRecord
	room   *rooms.Room
	deaths *[]events.PlayerDeath
}

func newSuicideWorld(t *testing.T) suicideWorld {
	t.Helper()
	deathConfig(t)
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)

	room := &rooms.Room{RoomId: 9401, Zone: "suicidetest", Title: "Killing Floor"}
	rooms.SetTestRoom(room)
	t.Cleanup(func() { rooms.RemoveTestRoom(room.RoomId) })
	require.Equal(t, 9402, int(configs.GetSpecialRoomsConfig().DeathRecoveryRoom))
	recovery := &rooms.Room{RoomId: 9402, Zone: "Shadow Realm", Title: "Waiting room"}
	rooms.SetTestRoom(recovery)
	t.Cleanup(func() { rooms.RemoveTestRoom(recovery.RoomId) })

	user := users.NewUserRecord(9401, 1)
	user.Character.Name = "Doomed"
	user.Character.Level = 10
	user.Character.Experience = user.Character.XPTL(9) + 50
	user.Character.Gold = 40
	user.Character.RoomId = room.RoomId
	user.Character.Zone = room.Zone
	user.Character.Validate()
	user.Character.Health = -10
	users.SetTestUser(user)
	room.SetTestOccupants([]int{user.UserId}, nil)

	deaths := []events.PlayerDeath{}
	id := events.RegisterListener(events.PlayerDeath{}, func(e events.Event) events.ListenerReturn {
		deaths = append(deaths, e.(events.PlayerDeath))
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.PlayerDeath{}, id) })
	return suicideWorld{user: user, room: room, deaths: &deaths}
}

func (w suicideWorld) die(t *testing.T) {
	t.Helper()
	handled, err := Suicide(``, w.user, w.room, 0)
	require.NoError(t, err)
	require.True(t, handled)
	events.ProcessEvents()
}

func TestSuicideWithoutProviderUsesShadowRealm(t *testing.T) {
	death.SetProvider(nil)
	w := newSuicideWorld(t)
	// Extra lives keep permadeath from resetting the character.
	w.user.Character.ExtraLives = 1

	w.die(t)

	assert.Equal(t, 10, w.user.Character.Level, "the engine never takes a level")
	assert.Equal(t, w.user.Character.XPTL(9), w.user.Character.Experience, "the engine's level XP penalty")
	assert.Equal(t, -10, w.user.Character.Health)
	assert.Equal(t, "Shadow Realm", w.user.Character.Zone)
	assert.Equal(t, 0, w.user.Character.ExtraLives, "permadeath spent a life")
	require.Len(t, *w.deaths, 1)
	assert.False(t, (*w.deaths)[0].Permanent)
}

func TestSuicideWithProviderRespawnsOnce(t *testing.T) {
	provider := &fakeDeathProvider{}
	death.SetProvider(provider)
	t.Cleanup(func() { death.SetProvider(nil) })
	w := newSuicideWorld(t)
	experience := w.user.Character.Experience

	w.die(t)

	assert.Equal(t, []int{w.user.UserId}, provider.respawns)
	assert.Equal(t, []bool{true}, provider.newDeaths)
	assert.Equal(t, "Doomed", w.user.Character.Name, "no permadeath")
	assert.Equal(t, 10, w.user.Character.Level, "the provider takes the level")
	assert.Equal(t, experience, w.user.Character.Experience, "no engine XP penalty")
	assert.Equal(t, w.room.RoomId, w.user.Character.RoomId, "the provider moves the player, not the engine")
	assert.NotEqual(t, "Shadow Realm", w.user.Character.Zone)
	require.Len(t, *w.deaths, 1)
	assert.False(t, (*w.deaths)[0].Permanent, "an Ashveil death is never permanent")
	assert.Zero(t, w.user.Character.Gold, "the death config's drops still apply")
	assert.Equal(t, 40, w.room.Gold)
	assert.Len(t, w.room.Corpses, 1)
}

func TestSuicidePendingGoesStraightToRespawn(t *testing.T) {
	provider := &fakeDeathProvider{pending: true}
	death.SetProvider(provider)
	t.Cleanup(func() { death.SetProvider(nil) })
	w := newSuicideWorld(t)
	messages := captureLookMessages(t)
	broadcasts := 0
	id := events.RegisterListener(events.Broadcast{}, func(events.Event) events.ListenerReturn {
		broadcasts++
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Broadcast{}, id) })

	w.user.Character.KillerMobName = "wolf"
	w.die(t)

	assert.Equal(t, []int{w.user.UserId}, provider.respawns)
	assert.Equal(t, []bool{false}, provider.newDeaths, "a retry")
	assert.Empty(t, w.user.Character.KillerMobName, "stale killer cleared")
	assert.Empty(t, *w.deaths, "no second death event")
	assert.Empty(t, *messages, "no second announcement")
	assert.Zero(t, broadcasts, "no second broadcast")
	assert.Equal(t, 40, w.user.Character.Gold, "no second drop")
	assert.Zero(t, w.room.Gold)
	assert.Empty(t, w.room.Corpses, "no second corpse")
}

func TestSuicideJustReturnedIsIgnored(t *testing.T) {
	provider := &fakeDeathProvider{justReturned: true}
	death.SetProvider(provider)
	t.Cleanup(func() { death.SetProvider(nil) })
	w := newSuicideWorld(t)
	w.user.Character.Health = 20

	w.die(t)

	assert.Empty(t, provider.respawns)
	assert.Empty(t, *w.deaths)
	assert.Equal(t, 40, w.user.Character.Gold)
}

func TestSuicidePendingButHealedIsANewDeath(t *testing.T) {
	provider := &fakeDeathProvider{pending: true}
	death.SetProvider(provider)
	t.Cleanup(func() { death.SetProvider(nil) })
	w := newSuicideWorld(t)
	w.user.Character.Health = 20

	w.die(t)

	assert.Equal(t, []bool{true}, provider.newDeaths, "a full death, not a retry")
	assert.Len(t, *w.deaths, 1)
}
