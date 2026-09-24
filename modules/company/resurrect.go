package company

// Phase 25b: resurrection. modules/death owns the rite (where, and before
// whom); this module owns what it does to the companion.

import (
	"strconv"
	"strings"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

var _ domain.ResurrectionProvider = (*CompanyModule)(nil)

// DeadCompanions implements company.ResurrectionProvider.
func (m *CompanyModule) DeadCompanions(leaderUserID int) []domain.DeadCompanionView {
	if m.persistenceAvailable() != nil {
		return nil
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return nil
	}
	var out []domain.DeadCompanionView
	for _, c := range record.Companions {
		if !c.Dead() {
			continue
		}
		view := domain.DeadCompanionView{ID: c.ID, Name: companionName(c), Remaining: c.Death.Remaining}
		if c.State != nil {
			view.Level = c.State.Level
		}
		out = append(out, view)
	}
	return out
}

// lostMatch reports whether selector names one of the record's lost
// companions.
func lostMatch(record domain.Record, selector string) bool {
	selector = strings.ToLower(strings.TrimSpace(selector))
	id, err := strconv.Atoi(strings.TrimPrefix(selector, "#"))
	for _, l := range record.Lost {
		if (err == nil && l.ID == id) || (err != nil && strings.Contains(strings.ToLower(l.Name), selector)) {
			return true
		}
	}
	return false
}

// ResurrectCompanion implements company.ResurrectionProvider: the leader's
// time is charged first, so a companion out of time is lost, not raised.
// Otherwise the companion loses a level (never below 1), its death ends,
// it returns to its old cell if that is free, and the record is saved; only
// then is it spawned into roomID. A failed save changes nothing. A failed
// spawn leaves it alive, awaiting restoration with the leader.
func (m *CompanyModule) ResurrectCompanion(leaderUserID int, selector string, roomID int) (domain.ResurrectionResult, error) {
	if err := m.persistenceAvailable(); err != nil {
		return domain.ResurrectionResult{}, err
	}
	if _, anchored := m.anchors[leaderUserID]; anchored {
		m.chargeLeader(leaderUserID)
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return domain.ResurrectionResult{}, domain.ErrUnknownMember
	}
	// The dead are matched first, so a name a living companion shares still
	// finds the dead one.
	dead := record
	dead.Companions = nil
	for _, candidate := range record.Companions {
		if candidate.Dead() {
			dead.Companions = append(dead.Companions, candidate)
		}
	}
	c, ok := resolveCompanion(dead, selector)
	if !ok {
		c, ok = resolveCompanion(record, selector)
	}
	if !ok {
		if lostMatch(record, selector) {
			return domain.ResurrectionResult{}, domain.ErrCompanionLost
		}
		return domain.ResurrectionResult{}, domain.ErrUnknownMember
	}
	if !c.Dead() {
		return domain.ResurrectionResult{}, domain.ErrNotDead
	}
	if c.Death.Remaining <= 0 {
		if err := m.expire(leaderUserID, c.ID); err != nil {
			return domain.ResurrectionResult{}, err
		}
		return domain.ResurrectionResult{}, domain.ErrCompanionLost
	}
	state := domain.MemberState{Level: 1}
	if c.State != nil {
		state = c.State.Clone()
	}
	state.Level = max(state.Level-1, 1)
	state.Experience = 0
	op := c.Death.OpID
	if err := m.registry.SetState(leaderUserID, c.ID, state); err != nil {
		return domain.ResurrectionResult{}, err
	}
	if err := m.registry.Revive(leaderUserID, c.ID); err != nil {
		m.registry.Put(record)
		return domain.ResurrectionResult{}, err
	}
	if err := m.save(); err != nil {
		m.registry.Put(record)
		return domain.ResurrectionResult{}, err
	}
	result := domain.ResurrectionResult{ID: c.ID, Name: companionName(c), Level: state.Level}
	mudlog.Info("company: companion resurrected", "leader", leaderUserID, "companion", c.ID, "op", op, "level", state.Level, "room", roomID)
	instanceID, err := m.runtime.Spawn(leaderUserID, roomID, c.MobTemplateID, &state)
	if err != nil {
		mudlog.Warn("company: resurrected companion awaits restoration", "leader", leaderUserID, "companion", c.ID, "error", err)
		return result, nil
	}
	m.setInstance(leaderUserID, c.ID, instanceID)
	m.applyInstanceAlignment(leaderUserID, c.ID, instanceID)
	result.Spawned = true
	return result, nil
}
