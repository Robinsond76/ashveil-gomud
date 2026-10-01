package hooks

import (
	"fmt"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

//
// Some clean up
//

func HandleLeave(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PlayerDespawn)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "PlayerDespawn", "Actual Type", e.Type())
		return events.Cancel
	}

	user := users.GetByUserId(evt.UserId)
	if user == nil {
		mudlog.Error("HandleLeave", "error", fmt.Sprintf(`user %d not found`, evt.UserId))
		return events.Cancel
	}

	connId := user.ConnectionId()

	// Remove any link-dead tracking for the user since they've been despawned from the world.
	if users.IsLinkDeadConnection(connId) {
		users.RemoveLinkDeadUser(evt.UserId)
	}

	room := rooms.LoadRoom(user.Character.RoomId)

	if currentParty := parties.Get(evt.UserId); currentParty != nil {
		affected := append(currentParty.GetMembers(), currentParty.GetInvited()...)
		wasLeader := currentParty.IsLeader(evt.UserId)
		parties.Suspend(evt.UserId)
		if wasLeader {
			for _, uid := range currentParty.GetMembers() {
				if uid != evt.UserId && users.GetByUserId(uid) != nil {
					if !currentParty.Promote(uid) {
						user.SendText(parties.LastError().Error())
					}
					break
				}
			}
		}
		if wasLeader && currentParty.LeaderUserId != evt.UserId {
			for _, uid := range currentParty.GetMembers() {
				if u := users.GetByUserId(uid); u != nil {
					if currentParty.IsLeader(uid) {
						u.SendText(`You are now the leader of the party. Following is off for everyone.`)
					} else {
						u.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is now the party leader. Following is off; use party follow on to consent.`, users.CharacterName(currentParty.LeaderUserId)))
					}
				}
			}
		}
		events.AddToQueue(events.PartyUpdated{Action: `left`, UserIds: affected})
	}

	for _, mobInstId := range room.GetMobs(rooms.FindCharmed) {
		if mob := mobs.GetInstance(mobInstId); mob != nil {
			if mob.Character.IsCharmed(evt.UserId) {
				mob.Character.Charmed.Expire()
			}
		}
	}

	if _, ok := room.RemovePlayer(evt.UserId); ok {
		tplTxt, _ := templates.Process("player-despawn", user.Character.Name)
		room.SendText(tplTxt)
	}

	// Ashveil 32b: a hand-off keeps the connection for the next user
	// (UserHandOff, queued after this), so no goodbye and no hang-up.
	if !evt.HandOff {
		tplTxt, _ := templates.Process("goodbye", nil, evt.UserId)
		connections.SendTo([]byte(templates.AnsiParse(tplTxt)), connId)
	}

	if err := users.LogOutUserByConnectionId(connId); err != nil {
		mudlog.Error("Log Out Error", "connectionId", connId, "error", err)
	}
	if !evt.HandOff {
		connections.Remove(connId)
	}

	specialRooms := configs.GetSpecialRoomsConfig()
	testRoomId := rooms.GetOriginalRoom(user.Character.RoomId)
	for i := range specialRooms.TutorialRooms {
		roomId, _ := strconv.Atoi(specialRooms.TutorialRooms[i])
		if roomId == testRoomId {
			user.Character.RoomId = -1
			break
		}
	}

	users.SaveUser(*user)

	// Ashveil 32h: a character being deleted, now out of the world, is
	// purged with its login kept; on a hand-off the same user comes back on
	// the same connection, in the Void, to make a new character. Queued
	// from this final listener, both follow whatever the other despawn
	// listeners queued.
	if user.Deleting {
		events.AddToQueue(events.UserPurged{UserId: user.UserId, KeepAccount: true})
		if evt.HandOff {
			events.AddToQueue(events.UserHandOff{ConnectionId: uint64(connId), FromUserId: user.UserId, ToUserId: user.UserId})
		}
	}

	return events.Continue
}
