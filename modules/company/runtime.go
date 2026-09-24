package company

import (
	"fmt"
	"slices"

	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/uuid"
)

type nativeRuntime struct{}

func (nativeRuntime) ResolveTemplate(name string) (int, bool) {
	id := mobs.MobIdByName(name)
	return int(id), id > 0
}

// Spawn creates the companion's live mob. With a state (Phase 22b), the mob
// is spawned at the saved level and the template's minted gear is replaced
// by copies of the saved gear.
func (nativeRuntime) Spawn(leaderUserID, roomID, mobTemplateID int, state *domain.MemberState) (int, error) {
	leader := users.GetByUserId(leaderUserID)
	if leader == nil {
		return 0, fmt.Errorf("company: leader %d is unavailable", leaderUserID)
	}
	room := rooms.LoadRoom(roomID)
	if room == nil {
		return 0, fmt.Errorf("company: room %d is unavailable", roomID)
	}
	var mob *mobs.Mob
	if state != nil && state.Level > 0 {
		mob = mobs.NewMobById(mobs.MobId(mobTemplateID), roomID, state.Level)
	} else {
		mob = mobs.NewMobById(mobs.MobId(mobTemplateID), roomID)
	}
	if mob == nil {
		return 0, fmt.Errorf("company: mob template %d is unavailable", mobTemplateID)
	}
	if state != nil {
		applyState(mob, *state)
	}
	mob.Character.Charm(leaderUserID, -2, characters.CharmExpiredRevert)
	leader.Character.TrackCharmed(mob.InstanceId, true)
	room.AddMob(mob.InstanceId)
	return mob.InstanceId, nil
}

func (nativeRuntime) IsLive(instanceID int) bool { return mobs.MobInstanceExists(instanceID) }

func (nativeRuntime) IsAttached(leaderUserID, instanceID int) bool {
	leader := users.GetByUserId(leaderUserID)
	mob := mobs.GetInstance(instanceID)
	return leader != nil && mob != nil && mob.Character.IsCharmed(leaderUserID) &&
		mob.Character.Charmed.RoundsRemaining != 0 &&
		slices.Contains(leader.Character.GetCharmIds(), instanceID)
}

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

// applyState puts a saved state on a freshly spawned mob: its level (an
// elite roll can't change it), experience, and copies of its gear in place
// of the template's.
func applyState(mob *mobs.Mob, state domain.MemberState) {
	saved := state.Clone()
	if saved.Level > 0 {
		mob.Character.Level = saved.Level
	}
	if saved.Experience > 0 {
		mob.Character.Experience = saved.Experience
	}
	mob.Character.Equipment = saved.Equipment
	mob.Character.Items = saved.Items
	if mob.Character.Items == nil {
		mob.Character.Items = []items.Item{}
	}
	mob.Character.Validate(true)
	mob.Character.Health = mob.Character.HealthMax.Value
	mob.Character.Mana = mob.Character.ManaMax.Value
}

// Snapshot reads a live mob's level and gear.
func (nativeRuntime) Snapshot(instanceID int) (domain.MemberState, bool) {
	mob := mobs.GetInstance(instanceID)
	if mob == nil {
		return domain.MemberState{}, false
	}
	state := domain.MemberState{
		Level:      mob.Character.Level,
		Experience: mob.Character.Experience,
		Equipment:  mob.Character.Equipment,
		Items:      mob.Character.Items,
	}
	return state.Clone(), true
}

// TemplateState is the state a companion of this template starts with,
// derived from the spec without spawning (the legacy upgrade). Every item
// gets its own fresh UUID.
func (nativeRuntime) TemplateState(mobTemplateID int) (domain.MemberState, bool) {
	spec := mobs.GetMobSpec(mobs.MobId(mobTemplateID))
	if spec == nil {
		return domain.MemberState{}, false
	}
	state := domain.MemberState{
		Level:     spec.Character.Level,
		Equipment: spec.Character.Equipment,
		Items:     spec.Character.Items,
	}.Clone()
	if state.Level < 1 {
		state.Level = 1
	}
	for _, slot := range characters.AllSlots() {
		if itm := state.Equipment.Get(slot); itm != nil && itm.ItemId > 0 {
			itm.UUID = uuid.UUID{}
			itm.Validate()
		}
	}
	for i := range state.Items {
		state.Items[i].UUID = uuid.UUID{}
		state.Items[i].Validate()
	}
	return state, true
}
