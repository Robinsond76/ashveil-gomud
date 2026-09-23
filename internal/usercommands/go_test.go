package usercommands

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/walking"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeStarter struct {
	handled bool
	err     error
	last    expedition.StartRequest
	calls   int
}

func (f *fakeStarter) StartTravel(req expedition.StartRequest) (bool, error) {
	f.calls++
	f.last = req
	return f.handled, f.err
}

type fakeViewer struct {
	handled bool
	err     error
	last    int
	calls   int
}

func (f *fakeViewer) RenderTravelView(leaderUserID int) (bool, error) {
	f.calls++
	f.last = leaderUserID
	return f.handled, f.err
}

type fakeMovementProvider struct {
	blocked bool
	message string
	calls   int
}

func (f *fakeMovementProvider) MovementBlocked(int) (bool, string) {
	f.calls++
	return f.blocked, f.message
}

func str(n int) string { return strconv.Itoa(n) }

func roomYAML(roomId int, zone, title, exits string) string {
	return "roomid: " + str(roomId) + "\nzone: " + zone + "\ntitle: " + title + "\ndescription: " + title + "\nexits:\n" + exits
}

func bidirectionalExits(originId, destId int, originExit string) map[string]string {
	return map[string]string{
		str(originId): roomYAML(originId, "traveltest", "Origin", originExit),
		str(destId):   roomYAML(destId, "traveltest", "Dest", "  south:\n    roomid: "+str(originId)+"\n"),
	}
}

// loadTravelTestWorld writes one disposable world for the whole test and loads
// it once. configs.AddOverlayOverrides only applies a key the first time, so a
// second call in the same test binary would be silently ignored.
func loadTravelTestWorld(t *testing.T, fixtures map[string]string) {
	t.Helper()
	dataDir := t.TempDir()
	require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir}))
	for path, data := range fixtures {
		fullPath := filepath.Join(dataDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(data), 0600))
	}
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	keywords.LoadAliases()
}

func travelTestUser(t *testing.T, userId, roomId int) *users.UserRecord {
	t.Helper()
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(userId, 1)
	user.Character.Name = "Tester"
	user.Character.RoomId = roomId
	user.Character.ActionPoints = 100
	users.SetTestUser(user)
	return user
}

func TestGoTravelInterception(t *testing.T) {
	fixtures := map[string]string{
		"biomes/default.yaml":               "biomeid: default\nname: Test\nsymbol: '.'\ndarkarea: true\n",
		"keywords.yaml":                     "direction-aliases: {}\n",
		"rooms/traveltest/zone-config.yaml": "name: traveltest\nroomid: 920001\n",
	}
	for path, data := range bidirectionalExits(920001, 920002, "  north:\n    roomid: 920002\n    travel_profile: oak-road\n") {
		fixtures["rooms/traveltest/"+path+".yaml"] = data
	}
	for path, data := range bidirectionalExits(920011, 920012, "  north:\n    roomid: 920012\n") {
		fixtures["rooms/traveltest/"+path+".yaml"] = data
	}
	for path, data := range bidirectionalExits(920021, 920022, "  north:\n    roomid: 920022\n    travel_profile: oak-road\n") {
		fixtures["rooms/traveltest/"+path+".yaml"] = data
	}
	for path, data := range bidirectionalExits(920031, 920032, "  north:\n    roomid: 920032\n    travel_profile: oak-road\n") {
		fixtures["rooms/traveltest/"+path+".yaml"] = data
	}
	loadTravelTestWorld(t, fixtures)

	t.Run("starts profiled travel before action points", func(t *testing.T) {
		origin := rooms.LoadRoom(920001)
		require.NotNil(t, origin)
		user := travelTestUser(t, 7, origin.RoomId)

		starter := &fakeStarter{handled: true}
		expedition.SetStartProvider(starter)
		t.Cleanup(func() { expedition.SetStartProvider(nil) })

		handled, err := Go("north", user, origin, 0)
		require.NoError(t, err)
		assert.True(t, handled)
		require.Equal(t, 1, starter.calls)
		assert.Equal(t, expedition.StartRequest{
			LeaderUserID:      7,
			OriginRoomID:      920001,
			DestinationRoomID: 920002,
			ExitName:          "north",
			ProfileName:       "oak-road",
		}, starter.last)
		assert.Equal(t, 920001, user.Character.RoomId, "travel start must not move the leader")
		assert.Equal(t, 100, user.Character.ActionPoints, "travel start must not spend action points")
	})

	t.Run("leaves unmarked exits instant", func(t *testing.T) {
		origin := rooms.LoadRoom(920011)
		require.NotNil(t, origin)
		user := travelTestUser(t, 8, origin.RoomId)

		starter := &fakeStarter{handled: true}
		expedition.SetStartProvider(starter)
		t.Cleanup(func() { expedition.SetStartProvider(nil) })

		handled, err := Go("north", user, origin, 0)
		require.NoError(t, err)
		assert.True(t, handled)
		assert.Zero(t, starter.calls, "unmarked exits must never consult the travel provider")
		assert.Equal(t, 920012, user.Character.RoomId)
		assert.Equal(t, 90, user.Character.ActionPoints)
	})

	t.Run("propagates travel start error", func(t *testing.T) {
		origin := rooms.LoadRoom(920021)
		require.NotNil(t, origin)
		user := travelTestUser(t, 9, origin.RoomId)

		starter := &fakeStarter{handled: true, err: assert.AnError}
		expedition.SetStartProvider(starter)
		t.Cleanup(func() { expedition.SetStartProvider(nil) })

		handled, err := Go("north", user, origin, 0)
		require.ErrorIs(t, err, assert.AnError)
		assert.True(t, handled)
		assert.Equal(t, 920021, user.Character.RoomId)
		assert.Equal(t, 100, user.Character.ActionPoints)
	})

	t.Run("falls back when provider absent", func(t *testing.T) {
		origin := rooms.LoadRoom(920031)
		require.NotNil(t, origin)
		user := travelTestUser(t, 10, origin.RoomId)

		expedition.SetStartProvider(nil)

		handled, err := Go("north", user, origin, 0)
		require.NoError(t, err)
		assert.True(t, handled)
		assert.Equal(t, 920032, user.Character.RoomId, "without a provider a marked exit stays instant")
	})

	// The walking seam hears about an instant move, and never about a route
	// start or a refused move.
	t.Run("reports only successful ordinary steps to walking", func(t *testing.T) {
		stepper := &recordingStepper{}
		walking.SetStepProvider(stepper)
		t.Cleanup(func() { walking.SetStepProvider(nil) })

		origin := rooms.LoadRoom(920011)
		require.NotNil(t, origin)
		user := travelTestUser(t, 12, origin.RoomId)
		_, err := Go("north", user, origin, 0)
		require.NoError(t, err)
		assert.Equal(t, [][3]int{{12, 920011, 920012}}, stepper.calls)

		blocker := &fakeMovementProvider{blocked: true, message: "no"}
		expedition.SetMovementProvider(blocker)
		_, err = Go("south", user, rooms.LoadRoom(920012), 0)
		require.NoError(t, err)
		expedition.SetMovementProvider(nil)
		assert.Len(t, stepper.calls, 1, "a refused move reports nothing")

		starter := &fakeStarter{handled: true}
		expedition.SetStartProvider(starter)
		t.Cleanup(func() { expedition.SetStartProvider(nil) })
		routeOrigin := rooms.LoadRoom(920001)
		require.NotNil(t, routeOrigin)
		user = travelTestUser(t, 13, routeOrigin.RoomId)
		_, err = Go("north", user, routeOrigin, 0)
		require.NoError(t, err)
		assert.Equal(t, 1, starter.calls)
		assert.Len(t, stepper.calls, 1, "a route start reports nothing")
	})

	t.Run("refuses ordinary movement while travelling", func(t *testing.T) {
		origin := rooms.LoadRoom(920011)
		require.NotNil(t, origin)
		user := travelTestUser(t, 11, origin.RoomId)

		blocker := &fakeMovementProvider{blocked: true, message: "You are already travelling."}
		expedition.SetMovementProvider(blocker)
		t.Cleanup(func() { expedition.SetMovementProvider(nil) })

		handled, err := Go("north", user, origin, 0)
		require.NoError(t, err)
		assert.True(t, handled)
		assert.Equal(t, 1, blocker.calls)
		assert.Equal(t, 920011, user.Character.RoomId, "ordinary movement must be refused during travel")
		assert.Equal(t, 100, user.Character.ActionPoints)
	})
}

type recordingStepper struct{ calls [][3]int }

func (r *recordingStepper) Stepped(userID, from, to int) {
	r.calls = append(r.calls, [3]int{userID, from, to})
}
