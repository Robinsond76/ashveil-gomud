package walkto

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// showcase loads the shipped Alderbrook rooms and the Trappers' Post that
// leads to them into the room manager, with the player at start.
func showcase(t *testing.T, start int) *world {
	t.Helper()
	w := newWorld(t)
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default", "rooms")
	for _, id := range []string{"forest", "road", "land", "city", "house", "cliffs", "mountains", "cave", "farmland", "shore"} {
		rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: id, Name: id, Symbol: "x"})
		id := id
		t.Cleanup(func() { rooms.RemoveTestBiome(id) })
	}
	files, err := filepath.Glob(filepath.Join(root, "alderbrook", "21*.yaml"))
	require.NoError(t, err)
	files = append(files, filepath.Join(root, "old_kings_road", "2005.yaml"))
	for _, f := range files {
		data, err := os.ReadFile(f)
		require.NoError(t, err)
		room := &rooms.Room{}
		require.NoError(t, yaml.Unmarshal(data, room), f)
		rooms.SetTestZoneRoom(room)
		t.Cleanup(func() { rooms.RemoveTestZoneRoom(room) })
		w.user.Character.MarkVisitedRoom(room.RoomId, room.Zone, nil)
	}
	w.user.Character.RoomId = start
	return w
}

func TestWalktoOverTheShippedShowcase(t *testing.T) {
	w := showcase(t, 2005)

	// From the Trappers' Post to the Waystone: out the new exit, along the
	// road, through the hamlet and up the hill, crossing a zone boundary.
	assert.Contains(t, w.walk("obelisk"), "don't know a place", "a landmark word only searches the zone you stand in")
	out := w.walk("2117")
	assert.Contains(t, out, "You set out for Waystone Hill, 8 steps away")
	for len(w.timers) > 0 && w.m.Active(7) {
		w.tick()
	}
	assert.Equal(t, 2117, w.user.Character.RoomId)
	assert.False(t, w.m.Active(7))
	assert.Len(t, w.steps.steps, 8, "eight ordinary moves, each through Go")
}

func TestWalktoShowcaseLakeLoopAndLandmarks(t *testing.T) {
	w := showcase(t, 2122) // the lakeshore ring's north-west corner

	// The ring is dense: across the lake's west side is two steps, and the
	// same walk the long way round is never chosen.
	out := w.walk("2135")
	assert.Contains(t, out, "2 steps away")
	for w.m.Active(7) && len(w.timers) > 0 {
		w.tick()
	}
	assert.Equal(t, 2135, w.user.Character.RoomId)

	// A landmark word picks the nearest visited room of that legend.
	for _, word := range []string{"bridge", "hermit", "cave", "rocks", "village"} {
		w.said = nil
		w.walk(word)
		assert.True(t, w.m.Active(7) || w.user.Character.RoomId != 2135, word)
		w.walk("stop")
		w.user.Character.RoomId = 2122
		users.GetByUserId(7).Character.RoomId = 2122
	}
}

func TestWalktoShowcaseLockedLoftAndUnvisitedCave(t *testing.T) {
	w := showcase(t, 2115) // Hollin's Farmhouse

	// The loft is behind a bolted door and the player has no key.
	assert.Contains(t, w.walk("loft"), "locked door is in the way")
	assert.False(t, w.m.Active(7))

	// Rooms never visited are not places you know, even by number.
	w.user.Character.ZonesVisited = nil
	w.user.Character.MarkVisitedRoom(2115, "Alderbrook", nil)
	assert.Contains(t, w.walk("2102"), "haven't been there")
	assert.Contains(t, w.walk("bat gallery"), "don't know a place")
}
