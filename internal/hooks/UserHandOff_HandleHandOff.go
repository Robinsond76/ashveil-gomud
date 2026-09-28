package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// HandleUserHandOff (Ashveil 32b) logs the next user in on a connection
// the previous one has just left with a hand-off despawn, and brings them
// into the world as a login does (world.enterWorld): PlayerSpawn, then
// placed in their saved room. When that can't be done, the connection is
// closed as a logout would, so the player is never left with no one.
func HandleUserHandOff(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.UserHandOff)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "UserHandOff", "Actual Type", e.Type())
		return events.Cancel
	}

	connId := evt.ConnectionId
	cd := connections.Get(connId)
	if cd == nil || cd.State() == connections.LinkDead {
		// The player hung up in between; the next user stays offline.
		mudlog.Info("HandOff", "result", "connection gone", "connectionId", connId, "toUserId", evt.ToUserId)
		if cd != nil {
			connections.Remove(connId)
		}
		dropUnused(evt.ToUserId)
		return events.Continue
	}

	if current := users.GetByConnectionId(connId); current != nil {
		mudlog.Error("HandOff", "error", "connection still has a user", "connectionId", connId, "userId", current.UserId, "toUserId", evt.ToUserId)
		dropUnused(evt.ToUserId)
		return events.Continue
	}

	next, err := users.LoadUserFile(evt.ToUserId)
	if err != nil {
		mudlog.Error("HandOff", "error", err, "connectionId", connId, "toUserId", evt.ToUserId)
		hangUp(connId, ``)
		return events.Continue
	}

	loggedIn, msg, err := users.LoginUser(next, connId)
	if err != nil {
		mudlog.Warn("HandOff", "error", err, "connectionId", connId, "toUserId", evt.ToUserId)
		hangUp(connId, msg)
		dropUnused(evt.ToUserId)
		return events.Continue
	}

	events.AddToQueue(events.PlayerSpawn{
		UserId:        loggedIn.UserId,
		ConnectionId:  loggedIn.ConnectionId(),
		RoomId:        loggedIn.Character.RoomId,
		Username:      loggedIn.Username,
		CharacterName: loggedIn.Character.Name,
	})

	if err := rooms.MoveToRoom(loggedIn.UserId, loggedIn.Character.RoomId, true); err != nil {
		mudlog.Warn("HandOff", "move", err, "userId", loggedIn.UserId, "roomId", loggedIn.Character.RoomId)
	}

	return events.Continue
}

// dropUnused purges a tutorial replay that a refused hand-off never
// brought online; a real user just stays offline.
func dropUnused(userId int) {
	if userId < users.ReplayUserIdBase || users.GetByUserId(userId) != nil {
		return
	}
	events.AddToQueue(events.UserPurged{UserId: userId})
}

// hangUp says why, then goodbye, and closes a connection with no user.
func hangUp(connId uint64, why string) {
	if why != `` {
		connections.SendTo([]byte(why+"\r\n"), connId)
	}
	tplTxt, _ := templates.Process("goodbye", nil)
	connections.SendTo([]byte(templates.AnsiParse(tplTxt)), connId)
	connections.Remove(connId)
}
