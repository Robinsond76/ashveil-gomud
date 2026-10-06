package camping

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 40a: forage and shelter rooms.

// forageFinds is how many raw game meat a level-4 forager brings back from
// a rest at a room with the given resources.
func forageFinds(t *testing.T, resources []string) int {
	t.Helper()
	forageSpecs(t)
	w := newRaidWorld(t, 0)
	w.room.Resources = resources
	w.m.campCfg.Forage["Road"] = []forageFind{{ItemID: 29, Weight: 1}}
	w.m.specialist = specialistsAt(t, 100, map[string]archetypes.Specialist{
		archetypes.UtilityForage: {Name: "Mira", Level: 4},
	})
	cargo := newFakeCargo(100000)
	useCargo(t, cargo)
	w.rest(t)
	w.at(camping.RestDuration + time.Second)
	w.sched.fireLatest()
	w.m.onNewRound(events.NewRound{})
	events.ProcessEvents()
	return cargo.stacks[29]
}

func TestForageRoomGivesOneMoreFind(t *testing.T) {
	assert.Equal(t, 3, forageFinds(t, nil), "1 + level 4 / 2")
	assert.Equal(t, 4, forageFinds(t, []string{rooms.ResourceForage}), "a forage room adds one")
	assert.Equal(t, 3, forageFinds(t, []string{rooms.ResourceWater}), "other resources add nothing")
}

func TestForageRoomWithoutAZoneTableFindsNothingExtra(t *testing.T) {
	forageSpecs(t)
	w := newRaidWorld(t, 0)
	w.room.Resources = []string{rooms.ResourceForage}
	w.m.specialist = specialistsAt(t, 100, map[string]archetypes.Specialist{
		archetypes.UtilityForage: {Name: "Mira", Level: 4},
	})
	text, err := w.m.forage(w.user, w.room, "op")
	require.NoError(t, err)
	assert.Empty(t, text)
}

func TestShelterRoomHalvesTheWeatherPenaltyAndLocksIt(t *testing.T) {
	w := newRaidWorld(t, 0)
	w.m.weatherIn = func(string) (weather.Condition, bool) {
		return weather.Condition{Name: "storm", RestRecoveryPct: 50}, true
	}
	w.rest(t)
	assert.Equal(t, 10, w.camp().Rest.Recovery, "20 x 50% with no shelter")
	w.at(camping.RestDuration + time.Second)
	w.sched.fireLatest()

	sheltered := newRaidWorld(t, 0)
	sheltered.room.Resources = []string{rooms.ResourceShelter}
	sheltered.m.weatherIn = w.m.weatherIn
	text := sheltered.m.startRest(sheltered.user, sheltered.room)
	assert.Equal(t, 15, sheltered.camp().Rest.Recovery, "50% moves halfway to 100%: 75% of 20")
	assert.Contains(t, text, "The shelter here softens it.")
	assert.True(t, strings.Contains(text, "storm"))

	clear := newRaidWorld(t, 0)
	clear.room.Resources = []string{rooms.ResourceShelter}
	clear.m.weatherIn = func(string) (weather.Condition, bool) {
		return weather.Condition{Name: "clear", RestRecoveryPct: 100}, true
	}
	clear.m.startRest(clear.user, clear.room)
	assert.Equal(t, 20, clear.camp().Rest.Recovery, "shelter never improves on a full rest")
}

// Review fix: the camping module tells rooms where a camp can be made, so
// forage and shelter markers only show in rooms tagged for camping.
func TestCampResourcesShowOnlyInCampableRooms(t *testing.T) {
	camp := &rooms.Room{RoomId: 1, Tags: []string{defaultRoomTag}, Resources: []string{rooms.ResourceWater, rooms.ResourceShelter}}
	cave := &rooms.Room{RoomId: 2, Resources: []string{rooms.ResourceWater, rooms.ResourceShelter}}
	assert.Equal(t, []string{rooms.ResourceWater, rooms.ResourceShelter}, camp.ShownResources())
	assert.Equal(t, []string{rooms.ResourceWater}, cave.ShownResources())
}
