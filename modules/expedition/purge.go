package expedition

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// onUserPurged (Phase 32b) forgets a purged leader's journey and stops
// its timer. Nothing moves: the user is gone.
func (m *ExpeditionModule) onUserPurged(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.UserPurged)
	if !ok {
		return events.Continue
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopTimerLocked(evt.UserId)
	delete(m.timerGeneration, evt.UserId)
	if _, held := m.sessions[evt.UserId]; !held {
		return events.Continue
	}
	delete(m.sessions, evt.UserId)
	if err := m.saveLocked(); err != nil {
		mudlog.Error("expedition: save after purge", "leader", evt.UserId, "error", err)
	}
	return events.Continue
}
