package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
)

// TestNpcGroupsNameTheGroupsMembers: Room.Info's Npcs carry their group's
// name (Phase 32c); a lone mob carries none.
func TestNpcGroupsNameTheGroupsMembers(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	add := func(id int, name, group string) {
		m := &mobs.Mob{InstanceId: id, SpawnGroup: group}
		m.Character.Name = name
		m.Character.Health, m.Character.HealthMax.Value = 10, 10
		mobs.SetTestInstance(m)
		t.Cleanup(func() { mobs.RemoveTestInstance(id) })
	}
	add(9201, "ruffian", "g")
	add(9202, "rat", "g")
	add(9203, "ruffian", "g")
	add(9204, "cave troll", "")
	r := &rooms.Room{RoomId: 990201}
	r.SetTestOccupants(nil, []int{9201, 9202, 9203, 9204})
	rooms.SetTestRoom(r)
	t.Cleanup(func() { rooms.RemoveTestRoom(990201) })

	assert.Equal(t, map[int]string{9201: "a band of ruffians", 9202: "a band of ruffians", 9203: "a band of ruffians"}, npcGroups(r))
}
