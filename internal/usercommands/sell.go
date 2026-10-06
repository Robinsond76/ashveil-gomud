package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Sell(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Phase 36c: `sell junk` sells everything marked as junk in one go
	// (plain junk-type items included, such as the shipped "junk").
	if strings.EqualFold(strings.TrimSpace(rest), `junk`) {
		return sellJunk(user, room)
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

	if itemSpec.QuestToken != `` {
		user.SendText("Quest items cannot be sold!")
		return true, nil
	}

	for _, mobId := range room.GetMobs(rooms.FindMerchant) {

		mob := mobs.GetInstance(mobId)
		if mob == nil {
			continue
		}

		user.Character.CancelBuffsWithFlag("hidden")

		if item.IsSpecialForSale() {

			mob.Command(`say I'm afraid I don't buy those.`)

			continue
		}

		sellValue := mob.GetSellPrice(item)

		if sellValue <= 0 {
			mob.Command(`say I'm not interested in that.`)
			continue
		}

		completeSale(user, room, mob, item, sellValue)

		user.EventLog.Add(`shop`, fmt.Sprintf(`Sold your <ansi fg="itemname">%s</ansi> to <ansi fg="mobname">%s</ansi> for <ansi fg="gold">%d gold</ansi>`, item.DisplayName(), mob.Character.Name, sellValue))

		user.SendText(
			fmt.Sprintf(`You sell a <ansi fg="itemname">%s</ansi> for <ansi fg="gold">%d gold</ansi>.`, item.DisplayName(), sellValue),
		)
		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> sells a <ansi fg="itemname">%s</ansi>.`, user.Character.Name, item.DisplayName()),
			user.UserId,
		)

		break
	}

	return true, nil

}

// completeSale moves an item and its gold between the seller and a merchant.
func completeSale(user *users.UserRecord, room *rooms.Room, mob *mobs.Mob, item items.Item, sellValue int) {
	user.Character.Gold += sellValue
	user.Character.RemoveItem(item)

	events.AddToQueue(events.ItemOwnership{
		UserId: user.UserId,
		Item:   item,
		Gained: false,
	})

	events.AddToQueue(events.EquipmentChange{
		UserId:     user.UserId,
		GoldChange: sellValue,
	})

	mob.Character.Shop.StockItem(item.ItemId)
}

// sellJunk (Phase 36c) sells every carried item marked as junk, and every
// plain junk-type item, to the first merchant here who will buy it, in one
// transaction with one summary. Items nobody here wants stay in the pack.
func sellJunk(user *users.UserRecord, room *rooms.Room) (bool, error) {
	merchants := []*mobs.Mob{}
	for _, mobId := range room.GetMobs(rooms.FindMerchant) {
		if mob := mobs.GetInstance(mobId); mob != nil {
			merchants = append(merchants, mob)
		}
	}
	if len(merchants) == 0 {
		user.SendText(`There is no merchant here to buy your junk.`)
		return true, nil
	}

	candidates := []items.Item{}
	for _, item := range user.Character.Items {
		if item.ItemId < 1 || item.GetSpec().QuestToken != `` {
			continue
		}
		if item.Junk || item.IsAutoJunk() {
			candidates = append(candidates, item)
		}
	}
	if len(candidates) == 0 {
		user.SendText(`You have no junk to sell. Mark items with <ansi fg="command">mark [item] junk</ansi> (<ansi fg="command">help mark</ansi>).`)
		return true, nil
	}

	user.Character.CancelBuffsWithFlag("hidden")

	total, sold := 0, 0
	kept := []string{}
	lines := []string{}
	for _, item := range candidates {
		done := false
		for _, mob := range merchants {
			if item.IsSpecialForSale() {
				break
			}
			value := mob.GetSellPrice(item)
			if value <= 0 {
				continue
			}
			completeSale(user, room, mob, item, value)
			user.EventLog.Add(`shop`, fmt.Sprintf(`Sold your junk <ansi fg="itemname">%s</ansi> to <ansi fg="mobname">%s</ansi> for <ansi fg="gold">%d gold</ansi>`, item.DisplayName(), mob.Character.Name, value))
			lines = append(lines, fmt.Sprintf(`  <ansi fg="itemname">%s</ansi> for <ansi fg="gold">%d gold</ansi>`, item.DisplayName(), value))
			total += value
			sold++
			done = true
			break
		}
		if !done {
			kept = append(kept, item.DisplayName())
		}
	}

	if sold == 0 {
		user.SendText(`No one here wants your junk.`)
		return true, nil
	}

	out := fmt.Sprintf("You sell %d junk item(s) for <ansi fg=\"gold\">%d gold</ansi>:\n%s", sold, total, strings.Join(lines, "\n"))
	if len(kept) > 0 {
		out += fmt.Sprintf("\nNo one here would buy: %s.", strings.Join(kept, ", "))
	}
	user.SendText(out)
	room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> sells some junk.`, user.Character.Name), user.UserId)
	return true, nil
}
