package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
)

func PurgeAlliance(e events.Event) events.ListenerReturn {
	evt := e.(events.UserPurged)
	if p := parties.Get(evt.UserId); p != nil {
		if !p.Leave(evt.UserId) && parties.LastError() != nil {
			mudlog.Error("Alliance purge", "error", parties.LastError())
			return events.CancelAndRequeue
		}
	}
	return events.Continue
}
