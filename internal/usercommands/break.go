package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/engagement"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Break(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Ashveil Phase 32d: a battle plays out as it was set up.
	if InBattle(user) {
		user.SendText(BattleOnlyFlee)
		return true, nil
	}

	// Ashveil Phase 29a: the company's engagement upkeep leaves a player
	// who broke off out of the fight until they attack again. That holds
	// even between blows (no target just then), while the company fights.
	engagement.StandDown(user.UserId)

	if user.Character.Aggro != nil {
		user.Character.Aggro = nil
		events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})
		user.SendText(`You break off combat.`)
		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> breaks off combat.`, user.Character.Name),
			user.UserId,
		)
	} else {
		user.SendText(`You aren't in combat!`)
	}

	return true, nil
}
