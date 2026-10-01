package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
Protection Skill
Level 2 - Front/Backrank
*/
func Rank(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(`protection`)

	if skillLevel < 1 {
		user.SendText("You don't know how to change your combat rank.")
		return true, fmt.Errorf("you don't know how to change your combat rank.")
	}

	party := parties.Get(user.UserId)
	if party == nil || !party.IsMember(user.UserId) {
		user.SendText("You must be in a party to change your combat rank.")
		return true, fmt.Errorf("you must be in a party to change your combat rank.")
	}

	if rest == `back` {
		party.SetRank(user.UserId, `back`)
	} else if rest == `front` {
		party.SetRank(user.UserId, `front`)
	} else {
		party.SetRank(user.UserId, `middle`)
	}

	// Fire an event that a skill has been used
	events.AddToQueue(events.SkillUsed{UserId: user.UserId, Skill: `protection`, Details: `rank`})

	user.SendText(fmt.Sprintf(`Your legacy party display rank is now <ansi fg="magenta">%s</ansi> rank.`, party.GetRank(user.UserId)))
	user.SendText(`Party ranks do not affect targeting. Use formation for your company grid.`)
	room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is now fighting from the <ansi fg="magenta">%s</ansi> rank.`, user.Character.Name, party.GetRank(user.UserId)), user.UserId)

	return true, nil
}
