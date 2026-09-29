package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Checks whether their level is too high for a guide
func Message_SendMessage(e events.Event) events.ListenerReturn {

	message, typeOk := e.(events.Message)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "Message", "Actual Type", e.Type())
		return events.Continue
	}

	if message.UserId > 0 {

		if user := users.GetByUserId(message.UserId); user != nil {
			// Phase 29f: a combat round's text may be held and paced.
			sendOrHold(user, message)
		}
	}

	if message.RoomId > 0 {

		room := rooms.LoadRoom(message.RoomId)
		if room == nil {
			return events.Continue
		}

		for _, userId := range room.GetPlayers() {
			skip := false

			if message.UserId == userId {
				continue
			}

			exLen := len(message.ExcludeUserIds)
			if exLen > 0 {
				for _, excludeId := range message.ExcludeUserIds {
					if excludeId == userId {
						skip = true
						break
					}
				}
			}

			if skip {
				continue
			}

			if user := users.GetByUserId(userId); user != nil {

				// If this is a quiet message, make sure the player can hear it
				if message.IsQuiet {
					if !user.Character.HasBuffFlag("superhearing") {
						continue
					}
				}

				sendOrHold(user, message)

			}
		}

	}
	return events.Continue

}
