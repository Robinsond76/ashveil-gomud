package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/loot"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Appraise(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	for _, mobId := range room.GetMobs(rooms.FindMerchant) {

		mob := mobs.GetInstance(mobId)
		if mob == nil {
			continue
		}

		if rest == "" {

			mob.Command(`say I will appraise items for 20 gold, and read unknown gear for more: 60 gold for a rare, 150 for an epic, 400 for a legendary.`)

			return true, nil
		}

		item, found := user.Character.FindInBackpack(rest)
		if !found {
			user.SendText("You don't have that item.")
			return true, nil
		}

		itemSpec := item.GetSpec()
		if itemSpec.ItemId < 1 {
			return true, nil
		}

		appraisePrice := 20

		// Phase 36c: a merchant also reads unidentified gear, for a fee that
		// grows with its rarity (an economy sink; help identify).
		reading := item.IsRolled() && !item.IsIdentified()
		if reading {
			appraisePrice = loot.IdentifyFee(item.RollRarity())
		}

		if appraisePrice > user.Character.Gold {

			mob.Command(fmt.Sprintf("say That costs %d gold to appraise, which you don't seem to have.", appraisePrice))

			return true, nil
		}

		user.Character.Gold -= appraisePrice
		mob.Character.Gold += appraisePrice

		events.AddToQueue(events.EquipmentChange{
			UserId:     user.UserId,
			GoldChange: appraisePrice,
		})

		if reading {
			read := item
			read.Identify()
			user.Character.UpdateItem(item, read)
			item = read
			itemSpec = item.GetSpec()
			user.SendText(fmt.Sprintf(`You give <ansi fg="mobname">%s</ansi> %d gold to read <ansi fg="itemname">%s</ansi>, and its properties are laid bare.`, mob.Character.Name, appraisePrice, item.DisplayName()))
		} else {
			user.SendText(fmt.Sprintf(`You give <ansi fg="mobname">%s</ansi> %d gold to appraise <ansi fg="itemname">%s</ansi>.`, mob.Character.Name, appraisePrice, itemSpec.Name))
		}
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> appraises <ansi fg="itemname">%s</ansi>.`, user.Character.Name, itemSpec.Name), user.UserId)

		user.SendText(buildInspectPanel(3, &item, &itemSpec))

		break
	}

	return true, nil
}
