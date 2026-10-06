package company

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// onUserPurged (Phase 32b) forgets a purged leader's company: the record
// (companions, formation, claims, service, the lost, recruit rosters),
// any companion mob
// still standing (the leader's logout detaches them already), the rescue
// clock, and tier-ups not yet announced.
func (m *CompanyModule) onUserPurged(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.UserPurged)
	if !ok {
		return events.Continue
	}
	leader := evt.UserId
	for companionID, instanceID := range m.instances[leader] {
		if m.runtime.IsLive(instanceID) {
			m.runtime.Detach(leader, instanceID)
		}
		m.clearInstance(leader, companionID)
	}
	delete(m.instances, leader)
	delete(m.anchors, leader)
	m.banter.forget(leader)
	for key, c := range m.pendingTierUps {
		if c.leaderUserID == leader {
			delete(m.pendingTierUps, key)
		}
	}
	if !m.registry.Remove(leader) {
		return events.Continue
	}
	if err := m.save(); err != nil {
		mudlog.Error("company: save after purge", "leader", leader, "error", err)
	}
	return events.Continue
}
