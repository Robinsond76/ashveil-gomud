package company

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"slices"

	domain "github.com/GoMudEngine/GoMud/internal/company"
)

var _ domain.RelocationProvider = (*CompanyModule)(nil)

// RelocateCompany implements company.RelocationProvider (Phase 25a): when the
// leader dies, every companion whose mob is live, attached, and alive goes
// with them to the church, companions by number. Only live mobs move: the
// record, formation, gear, alignment, and service are unchanged, and a crash
// needs no recovery because companions respawn with the leader on login. A
// dead companion has no live mob and stays behind.
func (m *CompanyModule) RelocateCompany(leaderUserID, roomID int) int {
	byCompanion := m.instances[leaderUserID]
	companionIDs := make([]int, 0, len(byCompanion))
	for companionID := range byCompanion {
		companionIDs = append(companionIDs, companionID)
	}
	slices.Sort(companionIDs)
	moved := 0
	for _, companionID := range companionIDs {
		instanceID := byCompanion[companionID]
		if !m.runtime.IsAttached(leaderUserID, instanceID) {
			continue
		}
		if m.runtime.Relocate(instanceID, roomID) {
			moved++
		}
	}
	return moved
}

// RelocateWithdrawal preflights every selected member before moving any. It
// owns no durable separation: dead and already-fled members are not selected.
func (m *CompanyModule) RelocateWithdrawal(uid, origin, destination int, ids []int) error {
	if rooms.LoadRoom(destination) == nil {
		return domain.ErrUnknownMember
	}
	for _, id := range ids {
		leader, _, ok := m.LeaderAndKeyForInstance(id)
		mob := mobs.GetInstance(id)
		if !ok || leader != uid || mob == nil || mob.Character.RoomId != origin || mob.Character.Health <= 0 || !m.runtime.IsAttached(uid, id) {
			return domain.ErrUnknownMember
		}
	}
	moved := []int{}
	for _, id := range ids {
		if !m.runtime.Relocate(id, destination) {
			for _, previous := range moved {
				m.runtime.Relocate(previous, origin)
			}
			return domain.ErrUnknownMember
		}
		moved = append(moved, id)
	}
	return nil
}
