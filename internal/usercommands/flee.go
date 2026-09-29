package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Flee(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Phase 30a: a hobbled (or hamstrung) fighter cannot break away.
	if user.Character.HasBuffFlag("no-flee") {
		user.SendText(`Your legs will not carry you out of this. You cannot flee.`)
		return true, nil
	}

	if user.Character.Aggro == nil || user.Character.Aggro.Type != characters.Flee {
		user.SendText(`You attempt to flee...`)

		user.Character.Aggro = &characters.Aggro{}
		user.Character.Aggro.Type = characters.Flee
		events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})
	}

	return true, nil
}
