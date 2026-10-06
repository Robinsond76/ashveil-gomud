package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func Gearup(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if mob.Character.HasBuffFlag("perma-gear") {
		mob.Command(`emote struggles with their gear for a while, then gives up.`)
		return true, nil
	}

	if rest != `` {
		matchItem, found := mob.Character.FindInBackpack(rest)
		if found {
			matchSpec := matchItem.GetSpec()
			for _, itm := range mob.Character.Equipment.GetAllItems() {
				itmSpec := itm.GetSpec()
				if itmSpec.Type == matchSpec.Type && matchSpec.Value > itmSpec.Value {
					mob.Command(fmt.Sprintf(`wear !%d`, matchItem.ItemId))
					mob.Command(fmt.Sprintf(`drop !%d`, itm.ItemId))
				}
			}
		}
		return true, nil
	}

	wornItems := map[items.ItemType]items.Item{}
	for _, itm := range mob.Character.Equipment.GetAllItems() {
		wornItems[itm.GetSpec().Type] = itm
	}

	upgrades := mob.Character.BestUpgrades()

	isCharmed := mob.Character.IsCharmed()
	for _, itm := range upgrades {
		mob.Command(fmt.Sprintf(`wear !%d:%s`, itm.ItemId, itm.UUID))
		if isCharmed {
			if oldItm, ok := wornItems[itm.GetSpec().Type]; ok {
				mob.Command(fmt.Sprintf(`drop !%d:%s`, oldItm.ItemId, oldItm.UUID))
			}
		}
	}

	return true, nil
}
