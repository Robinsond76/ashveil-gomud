package expedition

import (
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// encounterFoes lists the living mobs in a room with the encounter's group
// (or the lead alone), and clears them when the test ends.
func encounterFoes(t *testing.T, room *rooms.Room, lead int) []*mobs.Mob {
	t.Helper()
	leader := mobs.GetInstance(lead)
	require.NotNil(t, leader)
	var out []*mobs.Mob
	for _, id := range room.GetMobs() {
		m := mobs.GetInstance(id)
		if m == nil {
			continue
		}
		if id == lead || (leader.SpawnGroup != "" && m.SpawnGroup == leader.SpawnGroup) {
			out = append(out, m)
		}
	}
	t.Cleanup(func() {
		for _, m := range out {
			room.RemoveMob(m.InstanceId)
			mobs.DestroyInstance(m.InstanceId)
		}
	})
	return out
}

// It lives in a wiring_ file so it runs after expedition_test.go, whose
// tests expect no world loaded.

// TestEncounterSpawnsAPair: a travel encounter's foe comes with a second of
// its kind, one group of their own; the encounter stays active while either
// stands. A solitary foe comes alone. Uses the shipped mob files.
func TestEncounterSpawnsAPair(t *testing.T) {
	dataDir := filepath.Join("..", "..", "_datafiles", "world", "default")
	require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir}))
	races.LoadDataFiles()
	items.LoadDataFiles()
	rooms.LoadDataFiles()
	mobs.LoadDataFiles()
	room := rooms.LoadRoom(2002)
	require.NotNil(t, room)

	spawner := nativeMobSpawner{}
	lead, err := spawner.SpawnHostileEncounter(2002, 28, 7) // a ruffian
	require.NoError(t, err)
	foes := encounterFoes(t, room, lead)
	require.Len(t, foes, 2, "a pair")
	assert.Equal(t, "encounter:2002:"+strconv.Itoa(lead), foes[0].SpawnGroup)
	assert.Equal(t, foes[0].SpawnGroup, foes[1].SpawnGroup)
	for _, m := range foes {
		assert.Equal(t, "a band of ruffians", m.GroupName, "the pair is named (32c)")
		assert.True(t, m.Hostile)
		assert.Zero(t, m.MaxWander)
	}

	assert.True(t, spawner.EncounterActive(lead, 2002))
	mobs.GetInstance(lead).Character.Health = 0
	assert.True(t, spawner.EncounterActive(lead, 2002), "the second still stands")
	for _, m := range foes {
		m.Character.Health = 0
	}
	assert.False(t, spawner.EncounterActive(lead, 2002))

	lich, err := spawner.SpawnHostileEncounter(2002, 14, 7)
	require.NoError(t, err)
	alone := encounterFoes(t, room, lich)
	require.Len(t, alone, 1, "a solitary foe comes alone")
	assert.Empty(t, alone[0].SpawnGroup)
	assert.Empty(t, alone[0].GroupName, "a lone foe goes by its own name")
}

// TestShippedBossesStandAlone: the shipped lair of the lich spawns the lich
// alone: a solitary boss isn't grouped or topped up.
func TestShippedBossesStandAlone(t *testing.T) {
	dataDir := filepath.Join("..", "..", "_datafiles", "world", "default")
	require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir}))
	races.LoadDataFiles()
	items.LoadDataFiles()
	rooms.LoadDataFiles()
	mobs.LoadDataFiles()

	for _, id := range []int{14, 25, 34, 37} { // lich, abyssal creeper, ent, spider queen
		spec := mobs.GetMobSpec(mobs.MobId(id))
		require.NotNil(t, spec, id)
		assert.True(t, spec.Solitary, spec.Character.Name)
	}

	lair := rooms.LoadRoom(138)
	require.NotNil(t, lair)
	lair.Prepare(false)
	var hostiles []*mobs.Mob
	for _, id := range lair.GetMobs() {
		if m := mobs.GetInstance(id); m != nil && m.Hostile {
			hostiles = append(hostiles, m)
		}
	}
	t.Cleanup(func() {
		for _, m := range hostiles {
			lair.RemoveMob(m.InstanceId)
			mobs.DestroyInstance(m.InstanceId)
		}
	})
	require.Len(t, hostiles, 1, "the lich alone")
	assert.Equal(t, "lich", hostiles[0].Character.Name)
	assert.Empty(t, hostiles[0].SpawnGroup)
}

// TestTravelTimerRunsOnTheGameLoop: the real scheduler's timer only queues
// its callback; the callback runs when the game loop processes events, so
// an ambush spawn never touches the world from the timer's goroutine.
func TestTravelTimerRunsOnTheGameLoop(t *testing.T) {
	// The module's init registered onTravelTimerDue.
	var ran atomic.Int32
	realScheduler{}.AfterFunc(0, func() { ran.Add(1) })
	time.Sleep(50 * time.Millisecond) // the timer has fired, and queued
	assert.Zero(t, ran.Load(), "not run on the timer's goroutine")
	events.ProcessEvents()
	assert.Equal(t, int32(1), ran.Load(), "run once, on the loop")
}
