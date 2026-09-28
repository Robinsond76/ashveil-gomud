package archetype

import (
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// onUserPurged (Phase 32b) forgets a purged user: their archetype, their
// autoskill toggles, their owed kit, and this session's free trap senses.
func (m *ArchetypeModule) onUserPurged(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.UserPurged)
	if !ok {
		return events.Continue
	}
	if m.purge(evt.UserId) {
		if err := m.save(); err != nil {
			mudlog.Error("archetype: save after purge", "user", evt.UserId, "error", err)
		}
	}
	return events.Continue
}

// purge drops a user's state and reports whether the registry changed.
func (m *ArchetypeModule) purge(userID int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	prefix := strconv.Itoa(userID) + "|"
	for key := range m.pickSensed {
		if strings.HasPrefix(key, prefix) {
			delete(m.pickSensed, key)
		}
	}
	if m.registry == nil {
		return false
	}
	_, a := m.registry.Players[userID]
	_, b := m.registry.Autoskill[userID]
	_, c := m.registry.Kits[userID]
	delete(m.registry.Players, userID)
	delete(m.registry.Autoskill, userID)
	delete(m.registry.Kits, userID)
	return a || b || c
}
