package company

import (
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
