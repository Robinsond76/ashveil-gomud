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

var _ domain.LoyaltyAdjuster = (*CompanyModule)(nil)

// AdjustLoyaltyOnce implements company.LoyaltyAdjuster (Phase 60): the
// story events' loyalty outcome. Each listed living companion with a
// disposition moves by delta, kept within 0..100; op is saved in the same
// record, so a retry changes nothing, and a failed save rolls back. Like
// the vigil it saves the company file, so the gear snapshots in memory ride
// along on the same seam the vigil already uses.
func (m *CompanyModule) AdjustLoyaltyOnce(leaderUserID int, op string, companionIDs []int, delta int) ([]string, error) {
	if err := m.persistenceAvailable(); err != nil {
		return nil, err
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok || delta == 0 || record.HasApplied(op) {
		return nil, nil
	}
	want := map[int]bool{}
	for _, id := range companionIDs {
		want[id] = true
	}
	var changed []string
	for i, c := range record.Companions {
		if !want[c.ID] || c.Dead() || c.Disposition == nil {
			continue
		}
		d := *c.Disposition
		d.Loyalty = min(100, max(0, d.Loyalty+delta))
		if d.Loyalty == c.Disposition.Loyalty {
			continue
		}
		record.Companions[i].Disposition = &d
		changed = append(changed, companionName(c))
	}
	record.MarkApplied(op)
	before := m.registry.Clone()
	m.registry.Put(record)
	if err := m.save(); err != nil {
		m.registry = before
		return nil, err
	}
	return changed, nil
}
