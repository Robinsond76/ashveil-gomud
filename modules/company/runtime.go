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
	// Companions never roll elite: their level comes from the record (or
	// the template, for a new recruit), not from chance.
	level := 0
	if state != nil {
		level = state.Level
	}
	mob := mobs.NewMobByIdNoElite(mobs.MobId(mobTemplateID), roomID, level)
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

func (nativeRuntime) Relocate(instanceID, roomID int) bool {
	mob := mobs.GetInstance(instanceID)
	if mob == nil || mob.Character.Health < 1 {
		return false
	}
	to := rooms.LoadRoom(roomID)
	if to == nil {
		return false
	}
	mob.Character.Aggro = nil
	if mob.Character.RoomId == roomID {
		return true
	}
	if from := rooms.LoadRoom(mob.Character.RoomId); from != nil {
		from.RemoveMob(instanceID)
	}
	to.AddMob(instanceID)
	return true
}

// applyState puts a saved state on a freshly spawned mob: its level,
// experience, gold, and copies of its gear in place of the template's.
func applyState(mob *mobs.Mob, state domain.MemberState) {
	saved := state.Clone()
	if saved.Level > 0 {
		mob.Character.Level = saved.Level
	}
	if saved.Experience > 0 {
		mob.Character.Experience = saved.Experience
	}
	mob.Character.Gold = saved.Gold
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
		Gold:       mob.Character.Gold,
	}
	return state.Clone(), true
}

// CharmedByOther reports whether a live mob is now charmed by someone other
// than the leader (befriended away). An uncharmed companion, such as one
// whose charm expired when its leader left, is still the company's.
func (nativeRuntime) CharmedByOther(leaderUserID, instanceID int) bool {
	mob := mobs.GetInstance(instanceID)
	return mob != nil && mob.Character.IsCharmed() && !mob.Character.IsCharmed(leaderUserID)
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
		Gold:      spec.Character.Gold,
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

// Vitals reads a live mob's health.
func (nativeRuntime) Vitals(instanceID int) (int, int, bool) {
	mob := mobs.GetInstance(instanceID)
	if mob == nil {
		return 0, 0, false
	}
	return mob.Character.Health, mob.Character.HealthMax.Value, true
}
