package company

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

type nativeRuntime struct{}

func (nativeRuntime) ResolveTemplate(name string) (int, bool) {
	id := mobs.MobIdByName(name)
	return int(id), id > 0
}

func (nativeRuntime) Spawn(leaderUserID, roomID, mobTemplateID int) (int, error) {
	leader := users.GetByUserId(leaderUserID)
	if leader == nil {
		return 0, fmt.Errorf("company: leader %d is unavailable", leaderUserID)
	}
	room := rooms.LoadRoom(roomID)
	if room == nil {
		return 0, fmt.Errorf("company: room %d is unavailable", roomID)
	}
	mob := mobs.NewMobById(mobs.MobId(mobTemplateID), roomID)
	if mob == nil {
		return 0, fmt.Errorf("company: mob template %d is unavailable", mobTemplateID)
	}
	mob.Character.Charm(leaderUserID, -2, characters.CharmExpiredRevert)
	leader.Character.TrackCharmed(mob.InstanceId, true)
	room.AddMob(mob.InstanceId)
	return mob.InstanceId, nil
}

func (nativeRuntime) IsLive(instanceID int) bool { return mobs.MobInstanceExists(instanceID) }

func (nativeRuntime) Detach(leaderUserID, instanceID int) {
	if leader := users.GetByUserId(leaderUserID); leader != nil {
		leader.Character.TrackCharmed(instanceID, false)
	}
	if mob := mobs.GetInstance(instanceID); mob != nil {
		if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
			room.RemoveMob(instanceID)
		}
	}
	mobs.DestroyInstance(instanceID)
}
