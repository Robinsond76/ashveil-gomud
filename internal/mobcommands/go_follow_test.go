package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// TestCompanionDropsAMootFollow: found live (Phase 44), companions printed
// "looks a little confused (gate )" after the tutorial's gate, because the
// graduation had already moved them beside their leader and the gate their
// queued follow named was not in the new room. A companion told to follow
// that is already with its leader drops the command quietly; one left
// behind, or a mob that is not a companion, still fails as before.
func TestCompanionDropsAMootFollow(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	leader := users.NewUserRecord(7, 700)
	users.SetTestUser(leader)

	room := rooms.NewEmptyRoom()
	room.RoomId = 987650
	leader.Character.RoomId = room.RoomId

	mob := &mobs.Mob{MobId: 999, InstanceId: 424243}
	mob.Character.Name = "Tamsin"
	mob.Character.RoomId = room.RoomId
	mob.Character.Charmed = &characters.CharmInfo{UserId: 7, Companion: true}
	mob.CompanyMoveTo = room.RoomId

	assert.True(t, companionAlreadyWithLeader(mob), "already beside the leader: nothing to do")
	assert.Zero(t, mob.CompanyMoveTo)
	assert.False(t, companionAlreadyWithLeader(mob), "an unmarked move is not a follow")

	mob.CompanyMoveTo = room.RoomId
	leader.Character.RoomId = room.RoomId + 1
	assert.False(t, companionAlreadyWithLeader(mob), "left behind: the follow is not moot")

	leader.Character.RoomId = room.RoomId
	mob.Character.Charmed.Companion = false
	assert.False(t, companionAlreadyWithLeader(mob), "a charmed pet is not a company member")
}
