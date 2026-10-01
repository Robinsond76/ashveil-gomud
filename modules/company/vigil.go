package company

import (
	domain "github.com/GoMudEngine/GoMud/internal/company"
)

var _ domain.LoyaltyKeeper = (*CompanyModule)(nil)

// RaiseLoyaltyOnce implements company.LoyaltyKeeper (Phase 33f3 Vigil): each
// listed living companion with a disposition gains delta loyalty, up to
// cap (never lowered), and op is saved in the same record, so a retried
// vigil changes nothing. A failed save rolls back.
func (m *CompanyModule) RaiseLoyaltyOnce(leaderUserID int, op string, companionIDs []int, delta, cap int) ([]string, error) {
	if err := m.persistenceAvailable(); err != nil {
		return nil, err
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok || delta <= 0 || record.HasApplied(op) {
		return nil, nil
	}
	want := map[int]bool{}
	for _, id := range companionIDs {
		want[id] = true
	}
	var raised []string
	for i, c := range record.Companions {
		if !want[c.ID] || c.Dead() || c.Disposition == nil || c.Disposition.Loyalty >= cap {
			continue
		}
		d := *c.Disposition
		d.Loyalty = min(cap, d.Loyalty+delta)
		record.Companions[i].Disposition = &d
		raised = append(raised, c.Name)
	}
	record.MarkApplied(op)
	before := m.registry.Clone()
	m.registry.Put(record)
	if err := m.save(); err != nil {
		m.registry = before
		return nil, err
	}
	return raised, nil
}
