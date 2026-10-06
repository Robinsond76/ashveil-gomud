package camping

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// onUserPurged (Phase 32b) forgets a purged leader: their camp and inn
// stay, with any rest still running (nothing is granted or applied; the
// user is gone), their timers, and every tier or grant owed to them.
func (m *CampingModule) onUserPurged(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.UserPurged)
	if !ok {
		return events.Continue
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.refreshLitRoomsLocked()
	if !m.purgeLocked(evt.UserId) {
		return events.Continue
	}
	if err := m.saveLocked(); err != nil {
		mudlog.Error("camping: save after purge", "leader", evt.UserId, "error", err)
	}
	return events.Continue
}

// purgeLocked drops a leader's state and reports whether any was held.
func (m *CampingModule) purgeLocked(leaderUserID int) bool {
	m.stopTimerLocked(leaderUserID)
	m.stopInnTimerLocked(leaderUserID)
	delete(m.timerGeneration, leaderUserID)
	delete(m.innTimerGeneration, leaderUserID)
	_, camp := m.camps[leaderUserID]
	_, applied := m.recoveryApplied[leaderUserID]
	_, stay := m.stays[leaderUserID]
	_, innApplied := m.innRecoveryApplied[leaderUserID]
	_, wellRested := m.wellRestedPending[leaderUserID]
	_, rested := m.restedPending[leaderUserID]
	_, owed := m.owed[leaderUserID]
	_, sharpen := m.autoSharpen[leaderUserID]
	_, poisonPlan := m.poisonPlans[leaderUserID]
	_, rewards := m.campRewards[leaderUserID]
	_, dutied := m.restedDuties[leaderUserID]
	_, tented := m.restedTents[leaderUserID]
	delete(m.camps, leaderUserID)
	delete(m.recoveryApplied, leaderUserID)
	delete(m.stays, leaderUserID)
	delete(m.innRecoveryApplied, leaderUserID)
	delete(m.wellRestedPending, leaderUserID)
	delete(m.restedPending, leaderUserID)
	delete(m.owed, leaderUserID)
	delete(m.autoSharpen, leaderUserID)
	delete(m.poisonPlans, leaderUserID)
	delete(m.campRewards, leaderUserID)
	delete(m.lastRewards, leaderUserID)
	delete(m.restedDuties, leaderUserID)
	delete(m.restedTents, leaderUserID)
	delete(m.raiders, leaderUserID)
	return camp || applied || stay || innApplied || wellRested || rested || owed || sharpen || poisonPlan || rewards || dutied || tented
}
