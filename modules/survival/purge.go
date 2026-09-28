package survival

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// onUserPurged (Phase 32b) forgets a purged leader's company needs and
// the bookkeeping that keeps their drains and recoveries idempotent.
func (m *SurvivalModule) onUserPurged(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.UserPurged)
	if !ok {
		return events.Continue
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r := &m.registry
	_, a := r.Leaders[evt.UserId]
	_, b := r.ReservedNextCompanionIDs[evt.UserId]
	_, c := r.AppliedExertion[evt.UserId]
	_, d := r.AppliedRestOperation[evt.UserId]
	if !a && !b && !c && !d {
		return events.Continue
	}
	delete(r.Leaders, evt.UserId)
	delete(r.ReservedNextCompanionIDs, evt.UserId)
	delete(r.AppliedExertion, evt.UserId)
	delete(r.AppliedRestOperation, evt.UserId)
	if err := m.save(); err != nil {
		mudlog.Error("survival: save after purge", "leader", evt.UserId, "error", err)
	}
	return events.Continue
}
