package company

import (
	domain "github.com/GoMudEngine/GoMud/internal/company"
)

var _ domain.InventoryProvider = (*CompanyModule)(nil)

// CompanyInventory implements company.InventoryProvider (Phase 32g): each
// companion's gear for the web client, read the way `company inventory`
// reads it (the live mob's when it is out, else its record or template;
// a fallen one's gear stays with the body) but never writing: the web
// client asks every round.
func (m *CompanyModule) CompanyInventory(leaderUserID int) ([]domain.InventoryMember, bool) {
	if m.persistenceAvailable() != nil {
		return nil, false
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return []domain.InventoryMember{}, true
	}
	out := make([]domain.InventoryMember, 0, len(record.Companions))
	for _, c := range record.Companions {
		key := domain.CompanionMemberKey(c.ID)
		name := nameOf(c, "")
		if c.Dead() {
			out = append(out, domain.InventoryMember{Key: key, Name: name, Fallen: true, Worn: []domain.InventoryItem{}, Carried: []domain.InventoryItem{}})
			continue
		}
		var state *domain.MemberState
		if instanceID, tracked := m.instance(leaderUserID, c.ID); tracked && m.runtime.IsLive(instanceID) {
			if m.runtime.CharmedByOther(leaderUserID, instanceID) {
				// Charmed away: its gear is no longer the company's, as
				// the text view says (32g review finding 9).
				out = append(out, domain.InventoryMember{Key: key, Name: name, Worn: []domain.InventoryItem{}, Carried: []domain.InventoryItem{}})
				continue
			}
			if live, ok := m.runtime.Snapshot(instanceID); ok {
				state = &live
			}
		}
		if state == nil {
			state = c.State
		}
		if state == nil {
			if template, ok := m.runtime.TemplateState(c.MobTemplateID); ok {
				state = &template
			}
		}
		if state == nil {
			out = append(out, domain.InventoryMember{Key: key, Name: name, Unrecorded: true, Worn: []domain.InventoryItem{}, Carried: []domain.InventoryItem{}})
			continue
		}
		member := domain.InventoryMemberOf(key, name, *state)
		// A companion's items take no commands, so they carry no
		// reference; a template's items get fresh UUIDs on every read,
		// which would resend the payload every round (32g review finding 5).
		for i := range member.Worn {
			member.Worn[i].Ref = ""
		}
		for i := range member.Carried {
			member.Carried[i].Ref = ""
		}
		out = append(out, member)
	}
	return out, true
}
