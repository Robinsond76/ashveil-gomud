package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Drink(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Check whether the user has an item in their inventory that matches
	matchItem, selector, found := findConsumable(rest, user)

	if !found {
		user.SendText(fmt.Sprintf(`You don't have a "%s" to drink.`, rest))
	} else {

		itemSpec := matchItem.GetSpec()

		if itemSpec.Subtype != items.Drinkable {
			user.SendText(
				fmt.Sprintf(`You can't drink <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()),
			)
			return true, nil
		}

		suffix := ""
		if itemSpec.Hydration > 0 {
			result, err := survival.Provision(user.UserId, selector, survival.Benefit{
				Hydration: itemSpec.Hydration,
			})
			if err != nil {
				return true, err
			}
			suffix = provisionSuffix(result, selector != "")
		}

		user.Character.CancelBuffsWithFlag("hidden")

		user.Character.UseItem(matchItem)

		user.SendText(fmt.Sprintf(`You drink the <ansi fg="itemname">%s</ansi>.%s`, matchItem.DisplayName(), suffix))
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> drinks <ansi fg="itemname">%s</ansi>.`, user.Character.Name, matchItem.DisplayName()), user.UserId)

		for _, buffId := range itemSpec.BuffIds {
			user.AddBuff(buffId, `drink`)

		}
	}

	return true, nil
}
