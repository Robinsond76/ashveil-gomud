package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// HandlePurge (Ashveil 32b) is the last word on a purged user: every
// module has dropped its state in its own listener, and now the user file
// goes. An online user is never removed; a file already gone is fine.
func HandlePurge(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.UserPurged)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "UserPurged", "Actual Type", e.Type())
		return events.Cancel
	}
	// Ashveil 32h: a deleted character keeps its login and starts again.
	if evt.KeepAccount {
		if err := users.ResetDeletedCharacter(evt.UserId); err != nil {
			mudlog.Error("HandlePurge", "userId", evt.UserId, "keepAccount", true, "error", err)
			return events.Continue
		}
		mudlog.Info("HandlePurge", "userId", evt.UserId, "result", "character reset")
		return events.Continue
	}
	if err := users.RemoveUserFile(evt.UserId); err != nil {
		mudlog.Error("HandlePurge", "userId", evt.UserId, "error", err)
		return events.Continue
	}
	mudlog.Info("HandlePurge", "userId", evt.UserId, "result", "removed")
	return events.Continue
}
