package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// SweepDeletions (Ashveil 32h) finishes character deletions a restart or
// crash interrupted: the flag is on the saved record, so an offline user
// gets the purge (and the reset after it), and one still online (a
// copyover kept them) gets the whole sequence again. Call it at boot,
// after the plugins load.
func SweepDeletions() {
	for _, id := range users.DeletingUserIds() {
		if u := users.GetByUserId(id); u != nil {
			u.Deleting = true
			mudlog.Info("SweepDeletions", "userId", id, "state", "online")
			events.AddToQueue(events.PlayerDespawn{UserId: id, RoomId: u.Character.RoomId, Username: u.Username, CharacterName: u.Character.Name, HandOff: true})
			continue
		}
		mudlog.Info("SweepDeletions", "userId", id, "state", "offline")
		events.AddToQueue(events.UserPurged{UserId: id, KeepAccount: true})
	}
}
