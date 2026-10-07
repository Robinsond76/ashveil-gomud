package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Imbue (Phase 71) asks an enchanter to work a creature trophy into a weapon
// or piece of armor for a fee. The item then grants the trophy's effects
// while worn (help enchanting). It changes only what the item grants: never
// its worth to a merchant.
func Imbue(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	var enchanter *mobs.Mob
	for _, mobId := range room.GetMobs() {
		if mob := mobs.GetInstance(mobId); mob != nil && mob.IsEnchanter() && mob.Character.Health > 0 {
			enchanter = mob
			break
		}
	}
	if enchanter == nil {
		user.SendText(`There is no enchanter here. Find one in town (<ansi fg="command">help enchanting</ansi>).`)
		return true, nil
	}

	rest = strings.TrimSpace(rest)
	if rest == `` {
		imbueMenu(user, enchanter)
		return true, nil
	}

	itemWords, trophyWords, ok := strings.Cut(rest, ` with `)
	itemWords, trophyWords = strings.TrimSpace(itemWords), strings.TrimSpace(trophyWords)
	if !ok || itemWords == `` || trophyWords == `` {
		enchanter.Command(`say Tell me which piece and which trophy: ` + `<ansi fg="command">imbue [item] with [trophy]</ansi>.`)
		return true, nil
	}

	item, inPack := user.Character.FindInBackpack(itemWords)
	if !inPack {
		var worn bool
		if item, worn = user.Character.FindOnBody(itemWords); !worn {
			user.SendText("You don't have that item.")
			return true, nil
		}
	}
	trophy, found := user.Character.FindInBackpack(trophyWords)
	if !found || trophy.GetSpec().Trophy == nil {
		user.SendText("You don't carry a trophy by that name. Trophies fall to hunted creatures (help enchanting).")
		return true, nil
	}

	spec := item.GetSpec()
	switch {
	case !items.Enchantable(&spec):
		enchanter.Command(`say I can only work a trophy into a weapon or something worn.`)
		return true, nil
	case item.IsTrophyEnchanted():
		enchanter.Command(`say That already carries a trophy. A piece takes one, and it can't be undone.`)
		return true, nil
	case len(item.Blob) > 0:
		enchanter.Command(`say I'll not touch that; it's more than it seems.`)
		return true, nil
	}

	fee := item.EnchantFee()
	if user.Character.Gold < fee {
		enchanter.Command(fmt.Sprintf(`say That work costs %d gold. Come back when you have it.`, fee))
		return true, nil
	}

	// Take the trophy, then work it in; refuse and change nothing if the
	// item is not where it was found.
	wornSlot := items.ItemType(``)
	if inPack {
		updated := item
		if err := updated.EnchantWithTrophy(trophy.ItemId); err != nil {
			enchanter.Command(`say I can't do that one.`)
			return true, nil
		}
		user.Character.RemoveItem(trophy)
		events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: trophy, Gained: false})
		user.Character.UpdateItem(item, updated)
		item = updated
	} else {
		for _, slot := range characters.AllSlots() {
			if slot == items.Pack {
				continue
			}
			if on := user.Character.Equipment.Get(slot); on.ItemId > 0 && on.Equals(item) {
				if err := on.EnchantWithTrophy(trophy.ItemId); err != nil {
					enchanter.Command(`say I can't do that one.`)
					return true, nil
				}
				wornSlot, item = slot, *on
				break
			}
		}
		if wornSlot == `` {
			user.SendText("You don't have that item.")
			return true, nil
		}
		user.Character.RemoveItem(trophy)
		events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: trophy, Gained: false})
	}
	user.Character.Gold -= fee
	enchanter.Character.Gold += fee

	trophySpec := trophy.GetSpec()
	effects := strings.Join(classes.DescribeGearEffects(trophySpec.Trophy.Effects), "; ")
	user.EventLog.Add(`shop`, fmt.Sprintf(`Had <ansi fg="mobname">%s</ansi> work <ansi fg="itemname">%s</ansi> into your <ansi fg="itemname">%s</ansi> for <ansi fg="gold">%d gold</ansi>`, enchanter.Character.Name, trophy.DisplayName(), item.DisplayName(), fee))
	user.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> works your <ansi fg="itemname">%s</ansi> into your <ansi fg="itemname">%s</ansi> for <ansi fg="gold">%d gold</ansi>. While worn it gives: %s.`, enchanter.Character.Name, trophy.DisplayName(), item.DisplayName(), fee, effects))
	room.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> works a trophy into <ansi fg="username">%s</ansi>'s <ansi fg="itemname">%s</ansi>.`, enchanter.Character.Name, user.Character.Name, item.DisplayName()), user.UserId)
	events.AddToQueue(events.EquipmentChange{UserId: user.UserId, GoldChange: -fee})
	events.AddToQueue(events.CompanyAssetsChanged{UserId: user.UserId}) // the gear window and company inventory show the enchant
	return true, nil
}

// imbueMenu says what the enchanter will do: the trophies carried, and the
// pieces they may go into, with the fee.
func imbueMenu(user *users.UserRecord, enchanter *mobs.Mob) {
	var trophies []string
	for _, it := range user.Character.Items {
		if spec := it.GetSpec(); spec.Trophy != nil {
			trophies = append(trophies, fmt.Sprintf(` - <ansi fg="itemname">%s</ansi>: %s`, it.DisplayName(), strings.Join(classes.DescribeGearEffects(spec.Trophy.Effects), "; ")))
		}
	}
	enchanter.Command(`say Bring me a trophy from something you hunted and a weapon or armor to put it in. I ask 25 gold for each tier of the piece. <ansi fg="command">imbue [item] with [trophy]</ansi>.`)
	if len(trophies) == 0 {
		user.SendText(`You carry no trophies. They fall to hunted creatures, by kind (<ansi fg="command">help enchanting</ansi>).`)
		return
	}
	user.SendText("Trophies you carry:\n" + strings.Join(trophies, "\n"))
}
