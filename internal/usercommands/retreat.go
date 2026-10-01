package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/withdrawal"
	"strings"
)

// Retreat orders the company out of its fight (Phase 33c): one round to
// prepare, then one escape attempt. flee is the same order.
func Retreat(rest string, u *users.UserRecord, r *rooms.Room, flags events.EventFlag) (bool, error) {
	if a := u.Character.Aggro; a != nil && a.Type == characters.Retreat {
		u.SendText("Your company is already withdrawing. The order is unchanged.")
		return true, nil
	}
	if !InBattle(u) && !inPlayerFight(u, r) {
		u.SendText("You are not in a battle. Use go [exit] to travel.")
		return true, nil
	}
	if !withdrawal.Eligible(u.Character) {
		u.SendText(withdrawal.LeaderPinned)
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
	who := "Your company begins"
	if len(req.Members) == 0 {
		who = "You begin"
	}
	u.SendText(who + " an ordered retreat " + name + ". One round to prepare, then an escape attempt.")
	events.AddToQueue(events.AggroChanged{UserId: u.UserId, RoomId: r.RoomId})
	return true, nil
}

// inPlayerFight reports a fight with another player, which is no battle:
// the player aims at someone, or someone here aims at them.
func inPlayerFight(u *users.UserRecord, r *rooms.Room) bool {
	if a := u.Character.Aggro; a != nil && a.UserId > 0 {
		return true
	}
	if r == nil {
		return false
	}
	for _, uid := range r.GetPlayers() {
		if other := users.GetByUserId(uid); other != nil && uid != u.UserId && other.Character.Aggro != nil && other.Character.Aggro.UserId == u.UserId {
			return true
		}
	}
	return false
}
