package usercommands

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureLookMessages(t *testing.T) *[]string {
	t.Helper()
	messages := []string{}
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		messages = append(messages, e.(events.Message).Text)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	return &messages
}

// TestLookUsesPerViewerVisibility drives the real look command in a dark
// cave: without a light the player sees nothing; with their own torch they
// see the room but can't see beyond its exits.
func TestLookUsesPerViewerVisibility(t *testing.T) {
	// look resolves exit names through keywords.yaml, so load the real
	// config and keywords from the repo root.
	_, thisFile, _, _ := runtime.Caller(0)
	t.Chdir(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	require.NoError(t, configs.ReloadConfig())
	keywords.LoadAliases()

	buffs.SetTestFlag(rooms.FlagLightSource)
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 9101, Name: "torch", TriggerCount: 1, Flags: []string{rooms.FlagLightSource}})
	t.Cleanup(func() { buffs.RemoveTestBuffSpec(9101) })
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "testcave", Name: "Test Cave", Symbol: "C", DarkArea: true, Indoor: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("testcave") })

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 1)
	users.SetTestUser(user)
	room := &rooms.Room{RoomId: 91001, Zone: "Deep", Biome: "testcave", Exits: map[string]exit.RoomExit{"north": {RoomId: 91002}}}
	room.SetTestOccupants([]int{7}, nil)

	messages := captureLookMessages(t)
	handled, err := Look("", user, room, events.CmdSecretly)
	require.NoError(t, err)
	assert.True(t, handled)
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*messages, "\n"), "You can't see anything!")

	require.True(t, user.Character.Buffs.AddBuff(9101, true))
	*messages = nil
	handled, err = Look("north", user, room, events.CmdSecretly)
	require.NoError(t, err)
	assert.True(t, handled)
	events.ProcessEvents()
	out := strings.Join(*messages, "\n")
	assert.NotContains(t, out, "You can't see anything!", "a carried torch lets its bearer see")
	assert.Contains(t, out, "too dark to see anything in that direction", "a torch alone doesn't light beyond the exits")
}

type fogProvider struct{ zone string }

func (f fogProvider) CurrentCondition(zone string) (weather.Condition, bool) {
	if zone != f.zone {
		return weather.Condition{}, false
	}
	return weather.Condition{Name: "thick-fog", Description: "Thick fog.", TravelDurationPct: 100, ExertionPct: 100, RestRecoveryPct: 100, CloudCover: 3, VisibilityMod: -2}, true
}

// TestLookThroughExitRespectsFogInLitBiome is a regression test: look <exit>
// used to let any lit biome see through exits even when fog had made the
// light dim.
func TestLookThroughExitRespectsFogInLitBiome(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	t.Chdir(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	require.NoError(t, configs.ReloadConfig())
	keywords.LoadAliases()

	saved := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(saved) })
	day := uint64(0)
	for gametime.GetDate(day).Night {
		day += 5
	}
	util.SetRoundCount(day)

	weather.SetProvider(fogProvider{zone: "Fogtown"})
	t.Cleanup(func() { weather.SetProvider(nil) })
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "teststreet", Name: "Test Street", Symbol: "S", LitArea: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("teststreet") })

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(8, 1)
	users.SetTestUser(user)
	room := &rooms.Room{RoomId: 91011, Zone: "Fogtown", Biome: "teststreet", Exits: map[string]exit.RoomExit{"north": {RoomId: 91012}}}
	room.SetTestOccupants([]int{8}, nil)
	require.Equal(t, 1, room.VisibilityForUser(user), "thick fog by day leaves a lit street dim")

	messages := captureLookMessages(t)
	handled, err := Look("north", user, room, events.CmdSecretly)
	require.NoError(t, err)
	assert.True(t, handled)
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*messages, "\n"), "too dark to see anything in that direction")
}
