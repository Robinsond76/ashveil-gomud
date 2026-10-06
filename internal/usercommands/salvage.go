package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/loot"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Salvage (Phase 36c) asks a smith to break gear down into materials.
func Salvage(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	var smith *mobs.Mob
	for _, mobId := range room.GetMobs(rooms.FindMerchant) {
		if mob := mobs.GetInstance(mobId); mob != nil && mob.IsSmith() {
			smith = mob
			break
		}
	}
	if smith == nil {
		user.SendText("There is no smith here. Salvage gear at a smith, such as the armorer in Frostfang (<ansi fg=\"command\">help salvage</ansi>).")
		return true, nil
	}

	if strings.TrimSpace(rest) == "" {
		smith.Command(`say Bring me gear you've no use for and I'll break it down into good metal, hide and cloth. <ansi fg="command">salvage [item]</ansi>.`)
		return true, nil
	}

	item, found := user.Character.FindInBackpack(rest)
	if !found {
		user.SendText("You don't have that item.")
		return true, nil
	}

	spec := item.GetSpec()
	switch {
	case spec.QuestToken != ``:
		user.SendText("Quest items cannot be salvaged!")
		return true, nil
	case len(item.Blob) > 0:
		smith.Command(`say I'll not break that up; it's more than it seems.`)
		return true, nil
	}

	parts := loot.SalvageYield(item)
	if len(parts) == 0 {
		smith.Command(`say There's nothing in that I could use.`)
		return true, nil
	}

	made := make([]items.Item, 0, len(parts))
	grams := 0
	names := make([]string, 0, len(parts))
	for _, p := range parts {
		for n := 0; n < p.Count; n++ {
			mat := items.New(p.ItemID)
			if mat.ItemId == 0 {
				continue
			}
			made = append(made, mat)
			grams += mat.Weight()
		}
		if mat := items.New(p.ItemID); mat.ItemId != 0 {
			names = append(names, fmt.Sprintf(`%d <ansi fg="itemname">%s</ansi>`, p.Count, mat.Name()))
		}
	}
	if len(made) == 0 {
		smith.Command(`say There's nothing in that I could use.`)
		return true, nil
	}

	if text, refuse := encumbrance.TooMuchToCarry(user.UserId, grams-item.Weight()); refuse {
		user.SendText(text)
		return true, nil
	}

	user.Character.CancelBuffsWithFlag("hidden")
	user.Character.RemoveItem(item)
	events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: item, Gained: false})
	for _, mat := range made {
		user.Character.StoreItem(mat)
		events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: mat, Gained: true})
	}

	user.EventLog.Add(`shop`, fmt.Sprintf(`Had <ansi fg="mobname">%s</ansi> salvage your <ansi fg="itemname">%s</ansi> into %s`, smith.Character.Name, item.DisplayName(), strings.Join(names, ", ")))
	user.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> breaks your <ansi fg="itemname">%s</ansi> down into %s.`, smith.Character.Name, item.DisplayName(), strings.Join(names, ", ")))
	room.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> breaks down <ansi fg="username">%s</ansi>'s <ansi fg="itemname">%s</ansi>.`, smith.Character.Name, user.Character.Name, item.DisplayName()), user.UserId)
	return true, nil
}
