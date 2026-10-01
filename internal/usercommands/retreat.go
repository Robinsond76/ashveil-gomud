package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/withdrawal"
	"strings"
)

func Retreat(rest string, u *users.UserRecord, r *rooms.Room, flags events.EventFlag) (bool, error) {
	if a := u.Character.Aggro; a != nil && a.Type == characters.Retreat {
		u.SendText("Your company is already withdrawing. The order is unchanged.")
		return true, nil
	}
	if !InBattle(u) {
		u.SendText("You are not in a battle. Use go [exit] to travel.")
		return true, nil
	}
	name, _, err := withdrawal.Route(u, r, strings.TrimSpace(rest))
	if err != nil {
		u.SendText(err.Error())
		return true, nil
	}
	req := withdrawal.Capture(u, r, name)
	if _, err := withdrawal.Present(u, req); err != nil {
		u.SendText(err.Error())
		return true, nil
	}
	old := u.Character.Aggro
	u.Character.Aggro = &characters.Aggro{Type: characters.Retreat, RetreatInfo: req, RoundsWaiting: 1}
	if old != nil {
		u.Character.Aggro.UserId = old.UserId
		u.Character.Aggro.MobInstanceId = old.MobInstanceId
	}
	u.SendText("Your company begins an ordered retreat " + name + ". One round to prepare, then an escape attempt.")
	events.AddToQueue(events.AggroChanged{UserId: u.UserId, RoomId: r.RoomId})
	return true, nil
}
