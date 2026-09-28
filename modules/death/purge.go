package death

import "github.com/GoMudEngine/GoMud/internal/events"

// onUserPurged (Phase 32b) forgets when a purged user was last returned
// to a church. Everything else about a death is on the character.
func (m *DeathModule) onUserPurged(e events.Event) events.ListenerReturn {
	if evt, ok := e.(events.UserPurged); ok {
		m.mu.Lock()
		delete(m.returned, evt.UserId)
		m.mu.Unlock()
	}
	return events.Continue
}
