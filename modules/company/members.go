package company

// Phase 26a: the company's members for the information surfaces
// (internal/companyview). Read only.

import (
	domain "github.com/GoMudEngine/GoMud/internal/company"
)

var _ domain.MemberViewProvider = (*CompanyModule)(nil)

// CompanyMembers implements company.MemberViewProvider: each companion by
// ID, present (with its live health), awaiting restoration, or dead (with
// its rescue time). ok is false while the company can't be read.
func (m *CompanyModule) CompanyMembers(leaderUserID int) ([]domain.MemberView, bool) {
	if m.persistenceAvailable() != nil {
		return nil, false
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return []domain.MemberView{}, true
	}
	out := make([]domain.MemberView, 0, len(record.Companions))
	for _, c := range record.Companions {
		view := domain.MemberView{ID: c.ID, Name: companionName(c), Archetype: c.Archetype, Level: companionLevelNumber(c), Status: domain.MemberAwaiting}
		view.Row, view.Col, view.Placed = record.Formation.Find(domain.CompanionMemberKey(c.ID))
		if !view.Placed {
			view.Row, view.Col = 0, 0
		}
		switch {
		case c.Dead():
			view.Status = domain.MemberDead
			view.RescueSeconds = c.Death.Remaining
		default:
			if instanceID, tracked := m.instance(leaderUserID, c.ID); tracked && m.runtime.IsAttached(leaderUserID, instanceID) {
				view.Status = domain.MemberPresent
				view.HP, view.HPMax, _ = m.runtime.Vitals(instanceID)
			}
		}
		out = append(out, view)
	}
	return out, true
}
