package company

import (
	"github.com/GoMudEngine/GoMud/internal/userstate"
)

// stateContributor lets the admin test area snapshot a leader's whole
// company: the record (companions, formation, gear, claims, rosters) and
// the live companions it spawns. The company module has no lock of its own;
// like its commands, this runs on the game loop.
type stateContributor struct{ m *CompanyModule }

func (stateContributor) Name() string { return "company" }

func (c stateContributor) Capture(userID int) ([]byte, error) {
	m := c.m
	if err := m.persistenceAvailable(); err != nil {
		return nil, err
	}
	// Live companions' gear is only in memory until a snapshot seam.
	for companionID := range m.instances[userID] {
		m.refreshSnapshot(userID, companionID)
	}
	return userstate.Maps{m.registry.Companies}.Capture(userID)
}

func (c stateContributor) Restore(userID, roomID int, data []byte) error {
	m := c.m
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	// Whatever stands beside the leader now is the area's: remove it
	// without recording it, then rebuild from the saved record.
	for companionID, instanceID := range m.instances[userID] {
		if m.runtime.IsLive(instanceID) {
			m.runtime.Detach(userID, instanceID)
		}
		m.clearInstance(userID, companionID)
	}
	delete(m.instances, userID)
	delete(m.anchors, userID)
	for key, t := range m.pendingTierUps {
		if t.leaderUserID == userID {
			delete(m.pendingTierUps, key)
		}
	}
	m.forgetEquipmentView(userID)
	// Banter remembers lines and falls from the area; it is flavor held in
	// memory, so the return just forgets it.
	m.banter.forget(userID)
	if m.registry.Companies == nil {
		return nil
	}
	if err := (userstate.Maps{m.registry.Companies}).Apply(userID, data); err != nil {
		return err
	}
	if err := m.save(); err != nil {
		return err
	}
	return m.restoreForLeader(userID, roomID)
}
