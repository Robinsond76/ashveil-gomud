package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func PurgeAlliance(e events.Event) events.ListenerReturn {
	evt := e.(events.UserPurged)
	if p := parties.Get(evt.UserId); p != nil {
		if !p.LeaveFor(evt.UserId, users.IsOnline) && parties.LastError() != nil {
			mudlog.Error("Alliance purge", "error", parties.LastError())
			return events.CancelAndRequeue
		}
	}
	return events.Continue
}
