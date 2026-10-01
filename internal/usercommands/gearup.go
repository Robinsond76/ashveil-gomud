package usercommands

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/company"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Gearup(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if handled, text := company.ManageEquipment(user.UserId, "gearup", rest); handled {
		user.SendText(text)
		return true, nil
	}

	upgrades := user.Character.BestUpgrades()

	if len(upgrades) == 0 {
		wearableCount := 0
		for _, itm := range user.Character.GetAllBackpackItems() {
			spec := itm.GetSpec()
			if spec.Type == items.Weapon || spec.Subtype == items.Wearable {
				wearableCount++
			}
		}
		if wearableCount == 0 {
			user.SendText("You have nothing to wear.")
		} else {
			user.SendText("You're already wearing everything you can!")
		}
		return true, nil
	}

	for _, itm := range upgrades {
		user.Command(fmt.Sprintf(`wear !%d`, itm.ItemId), -1)
	}

	return true, nil
}
