package walking

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// onUserPurged (Phase 32b) forgets a purged user's walking fatigue.
func (m *WalkingModule) onUserPurged(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.UserPurged)
	if !ok {
		return events.Continue
	}
	m.mu.Lock()
	_, held := m.registry.Carry[evt.UserId]
	if held {
		delete(m.registry.Carry, evt.UserId)
		m.dirty = true
	}
	m.mu.Unlock()
	if !held {
		return events.Continue
	}
	if err := m.save(); err != nil {
		mudlog.Error("walking: save after purge", "user", evt.UserId, "error", err)
	}
	return events.Continue
}
