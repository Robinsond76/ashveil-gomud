package hooks

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/wounds"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/users"
)

//
// Execute on join commands
//

func HandleJoin(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PlayerSpawn)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "PlayerSpawn", "Actual Type", e.Type())
		return events.Cancel
	}

	user := users.GetByUserId(evt.UserId)
	if user == nil {
		mudlog.Error("HandleJoin", "error", fmt.Sprintf(`user %d not found`, evt.UserId))
		return events.Cancel
	}

	user.EventLog.Add(`conn`, fmt.Sprintf(`<ansi fg="username">%s</ansi> entered the world`, user.Character.Name))

	users.RemoveLinkDeadUser(evt.UserId)

	// Phase 30b: no fight survives a logout or a restart, so neither does a
	// light wound (the combat round's stray pass would close it too late if
	// a new fight opened in the player's first round back).
	user.Character.Wounds = wounds.CloseLight(user.Character.Wounds)

	room := rooms.LoadRoom(user.Character.RoomId)
	sendResetMessage := false

	// If an onreset room is set, send to it, then clear it.
	// This way, if they were in an ephemeral chunk and it has been cleared, we
	// handle sending them to whatever "reset" room was defined (if any)
	if room == nil && user.Character.RoomIdOnReset != 0 {
		mudlog.Warn("HandleJoin", "msg", fmt.Sprintf("room %d not found, trying RoomIdOnReset %d", user.Character.RoomId, user.Character.RoomIdOnReset))
		room = rooms.LoadRoom(user.Character.RoomIdOnReset)

		if room != nil {
			sendResetMessage = true
			user.Character.RoomId = user.Character.RoomIdOnReset
		}

		user.Character.RoomIdOnReset = 0
	}

	// If still no room was found, send to the zero room (default world start room)
	if room == nil {
		mudlog.Error("EnterWorld", "error", fmt.Sprintf(`room %d not found`, user.Character.RoomId))

		if err := rooms.MoveToRoom(user.UserId, 0); err != nil {
			mudlog.Error("EnterWorld", "msg", "could not move to room 0", "error", err)
		}

		// Load whatever default room they have been assigned
		room = rooms.LoadRoom(user.Character.RoomId)
		sendResetMessage = true
	}

	// TODO HERE
	if user.HasPlaintextPassword() {
		user.SendText(`<ansi fg="alert-5">You must change your password before doing anything else.</ansi>`)
		user.SendText(`<ansi fg="yellow">Type <ansi fg="yellow-bold">password</ansi> to set a new password.</ansi>`)
	} else {
		loginCmds := configs.GetConfig().Server.OnLoginCommands
		if len(loginCmds) > 0 {

			for _, cmd := range loginCmds {

				events.AddToQueue(events.Input{
					UserId:    evt.UserId,
					InputText: cmd,
					ReadyTurn: 0, // No delay between execution of commands
				})

			}

		}
	}

	if room != nil {

		if sendResetMessage {
			user.SendText("A portal opens before you and you feel an intense pulling... you can't escape it... You are transported elsewhere!")
		}

		if doLook, err := scripting.TryRoomScriptEvent(`onEnter`, user.UserId, user.Character.RoomId); err != nil || doLook {
			user.CommandFlagged(`look`, events.CmdSecretly) // Do a secret look.
		}
	}

	return events.Continue
}
