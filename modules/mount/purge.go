package mount

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// onUserPurged (Phase 32b) forgets a purged user's mount.
func (m *MountModule) onUserPurged(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.UserPurged)
	if !ok {
		return events.Continue
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, held := m.mounts[evt.UserId]; !held {
		return events.Continue
	}
	delete(m.mounts, evt.UserId)
	if err := m.saveLocked(); err != nil {
		mudlog.Error("mount: save after purge", "user", evt.UserId, "error", err)
	}
	return events.Continue
}
