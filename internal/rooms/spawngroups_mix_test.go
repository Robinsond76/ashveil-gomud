package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestARoomsOneKindGroupMixesAndKeepsItsHeadCount drives the real spawn
// pass (Prepare, then FormSpawnGroups) in a room whose list spawns three of
// one kind and lists a second kind that is out wandering: the group comes mixed, and
// later passes neither respawn the swapped-out member nor grow the group
// (89 review).
func TestARoomsOneKindGroupMixesAndKeepsItsHeadCount(t *testing.T) {
	const roomID, skel, imp = 97001, 97101, 97102
	for id, name := range map[int]string{skel: "test skeleton", imp: "test imp"} {
		m := &mobs.Mob{MobId: mobs.MobId(id), Hostile: true}
		m.Character = *characters.New()
		m.Character.Name = name
		m.Character.Level = 3
		mobs.SetTestSpec(m)
	}
	// The listed imp is out wandering: its entry tracks a live mob elsewhere.
	away := mobs.NewMobById(imp, roomID+1, 3)
	require.NotNil(t, away)
	r := &Room{RoomId: roomID, Title: "Crypt", SpawnInfo: []SpawnInfo{
		{MobId: skel, RespawnRate: "1 year"},
		{MobId: skel, RespawnRate: "1 year"},
		{MobId: skel, RespawnRate: "1 year"},
		{MobId: imp, RespawnRate: "1 year", InstanceId: away.InstanceId},
	}}
	SetTestRoom(r)
	t.Cleanup(func() {
		for _, id := range r.GetMobs() {
			mobs.DestroyInstance(id)
		}
		mobs.DestroyInstance(away.InstanceId)
		RemoveTestRoom(roomID)
	})

	kinds := func() map[int]int {
		out := map[int]int{}
		groups := map[string]bool{}
		for _, id := range r.GetMobs() {
			m := mobs.GetInstance(id)
			require.NotNil(t, m)
			out[int(m.MobId)]++
			groups[m.SpawnGroup] = true
		}
		assert.Len(t, groups, 1, "one group")
		return out
	}

	r.Prepare(false)
	assert.Equal(t, map[int]int{skel: 2, imp: 1}, kinds(), "the row of skeletons takes an imp")
	tracked := 0
	for _, e := range r.SpawnInfo[:3] {
		if e.InstanceId > 0 {
			tracked++
			assert.NotNil(t, mobs.GetInstance(e.InstanceId), "each entry tracks a mob in the room")
		}
	}
	assert.Equal(t, 3, tracked, "the swapped entry tracks the imp")

	r.Prepare(false)
	r.Prepare(false)
	assert.Equal(t, map[int]int{skel: 2, imp: 1}, kinds(), "later passes leave the group as it is")
}
