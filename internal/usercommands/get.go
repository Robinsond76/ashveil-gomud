package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Get(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) == 0 {
		user.SendText("Get what?")
		return true, nil
	}

	if args[0] == "all" && len(args) == 1 {
		if room.Gold > 0 {
			Get(`gold`, user, room, flags)
		}

		if len(room.Items) > 0 {
			iCopies := append([]items.Item{}, room.Items...)

			left := 0
			for _, item := range iCopies {
				// One line for everything left behind, not one each
				// (32f review).
				if !fits(user, item) {
					left++
					continue
				}
				Get(item.Name(), user, room, flags)
			}
			leftBehind(user, left)
		}

		return true, nil
	}

	getFromStash := false
	containerName := ``
	petUserId := 0
	var corpseRef *rooms.Corpse

	if len(args) >= 2 {
		// Detect "stash" or "from stash" at end and remove it
		if args[len(args)-1] == "stash" {
			getFromStash = true
			if args[len(args)-2] == "from" {
				rest = strings.Join(args[0:len(args)-2], " ")
			} else {
				rest = strings.Join(args[0:len(args)-1], " ")
			}
		}

		if args[len(args)-1] == "ground" {
			getFromStash = false
			if args[len(args)-2] == "from" {
				rest = strings.Join(args[0:len(args)-2], " ")
			} else {
				rest = strings.Join(args[0:len(args)-1], " ")
			}
		}

		containerName = room.FindContainerByName(args[len(args)-1])
		if containerName != `` {
			getFromStash = false
			if args[len(args)-2] == "from" {
				rest = strings.Join(args[0:len(args)-2], " ")
			} else {
				rest = strings.Join(args[0:len(args)-1], " ")
			}
		}

		//
		// Look for any pets in the room
		//
		petUserId = room.FindByPetName(args[len(args)-1])
		if petUserId == 0 && args[len(args)-1] == `pet` && user.Character.Pet.Exists() && !user.Character.Pet.IsMissing() {
			petUserId = user.UserId
		}
		if petUserId > 0 {

			if petUserId != user.UserId {
				user.SendText(`You can't do that!`)
				return true, nil
			}

			getFromStash = false
			if petUser := users.GetByUserId(petUserId); petUser != nil {

				if args[len(args)-2] == "from" {
					rest = strings.Join(args[0:len(args)-2], " ")
				} else {
					rest = strings.Join(args[0:len(args)-1], " ")
				}
			}
		}

		// Look for a corpse as the source when CorpseItems is enabled, or
		// for a claimed shared-battle corpse (Phase 33d) in any world.
		if containerName == `` && petUserId == 0 {
			if c, ok := room.FindCorpseByRef(args[len(args)-1], func(c *rooms.Corpse) bool { return corpseLootable(c, user.UserId) }); ok {
				corpseRef = c
				if len(args) >= 2 && args[len(args)-2] == "from" {
					rest = strings.Join(args[0:len(args)-2], " ")
				} else {
					rest = strings.Join(args[0:len(args)-1], " ")
				}
			}
		}
	}

	if petUserId == user.UserId {

		matchItem, found := user.Character.Pet.FindItem(rest)
		if !found {
			user.SendText(fmt.Sprintf(`You don't see a %s carried by %s.`, rest, user.Character.Pet.DisplayName()))
		} else {

			// A pet's pouch already counts in its owner's load (Phase
			// 32f), so taking from it needs no capacity check.
			if user.Character.Pet.RemoveItem(matchItem) {
				if !user.Character.StoreItem(matchItem) {
					user.Character.Pet.StoreItem(matchItem)
				} else {

					events.AddToQueue(events.ItemOwnership{
						UserId: user.UserId,
						Item:   matchItem,
						Gained: true,
					})

					events.AddToQueue(events.PetItemChange{
						UserId: user.UserId,
					})
				}

				user.SendText(
					fmt.Sprintf(`You remove a <ansi fg="itemname">%s</ansi> from %s.`, matchItem.DisplayName(), user.Character.Pet.DisplayName()),
				)
				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> removes a <ansi fg="itemname">%s</ansi> from %s...`, user.Character.Name, matchItem.DisplayName(), user.Character.Pet.DisplayName()),
					user.UserId,
				)

			}
		}

		return true, nil

	}

	// Handle getting from a corpse
	if corpseRef != nil {
		if !configs.GetGamePlayConfig().Death.CorpseItems && corpseRef.ClaimUserId == 0 {
			// Only a claimed corpse holds loot in a floor-drop world; the
			// body's own gear stays on it (Phase 33d review).
			user.SendText(`There is nothing to take from the corpse. Loot falls to the ground.`)
			return true, nil
		}
		if corpseRef.ClaimUserId != 0 && corpseRef.ClaimUserId != user.UserId {
			user.SendText(`This battle loot is claimed by another company.`)
			return true, nil
		}
		// Gear still worn on the body is loot only in CorpseItems worlds:
		// elsewhere it failed its drop roll (or is perma-gear) and stays.
		wornLoot := bool(configs.GetGamePlayConfig().Death.CorpseItems)

		corpseColor := `mob-corpse`
		if corpseRef.UserId > 0 {
			corpseColor = `user-corpse`
		}
		corpseName := fmt.Sprintf(`<ansi fg="%s">%s corpse</ansi>`, corpseColor, corpseRef.Character.Name)

		// "get all <corpse>"
		if rest == "all" {
			tookSomething := false
			left := 0

			if corpseRef.Gold > 0 {
				goldAmt := corpseRef.Gold
				user.Character.Gold += goldAmt
				corpseRef.Gold = 0

				events.AddToQueue(events.EquipmentChange{
					UserId:     user.UserId,
					GoldChange: -goldAmt,
				})

				user.SendText(fmt.Sprintf(`You take <ansi fg="gold">%d gold</ansi> from the %s.`, goldAmt, corpseName))
				room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> takes some <ansi fg="gold">gold</ansi> from the %s.`, user.Character.Name, corpseName), user.UserId)
				tookSomething = true
			}

			// Backpack items on corpse
			allCorpseItems := append([]items.Item{}, corpseRef.Items...)
			for _, item := range allCorpseItems {
				if !fits(user, item) {
					left++
					continue
				}
				if user.Character.StoreItem(item) {
					corpseRef.RemoveItem(item)
					events.AddToQueue(events.ItemOwnership{
						UserId: user.UserId,
						Item:   item,
						Gained: true,
					})
					user.SendText(fmt.Sprintf(`You take the <ansi fg="itemname">%s</ansi> from the %s.`, item.DisplayName(), corpseName))
					room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> takes the <ansi fg="itemname">%s</ansi> from the %s.`, user.Character.Name, item.DisplayName(), corpseName), user.UserId)
					tookSomething = true
				} else {
					user.SendText(fmt.Sprintf(`You can't carry the <ansi fg="itemname">%s</ansi>.`, item.DisplayName()))
				}
			}

			// Worn items on corpse character
			var allWorn []items.Item
			if wornLoot {
				allWorn = corpseRef.Character.GetAllWornItems()
			}
			for _, item := range allWorn {
				if !fits(user, item) {
					left++
					continue
				}
				if user.Character.StoreItem(item) {
					corpseRef.Character.RemoveFromBody(item)
					events.AddToQueue(events.ItemOwnership{
						UserId: user.UserId,
						Item:   item,
						Gained: true,
					})
					user.SendText(fmt.Sprintf(`You take the <ansi fg="itemname">%s</ansi> from the %s.`, item.DisplayName(), corpseName))
					room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> takes the <ansi fg="itemname">%s</ansi> from the %s.`, user.Character.Name, item.DisplayName(), corpseName), user.UserId)
					tookSomething = true
				} else {
					user.SendText(fmt.Sprintf(`You can't carry the <ansi fg="itemname">%s</ansi>.`, item.DisplayName()))
				}
			}

			leftBehind(user, left)
			if !tookSomething && left == 0 {
				user.SendText(fmt.Sprintf(`There is nothing to take from the %s.`, corpseName))
			}

			return true, nil
		}

		// "get gold <corpse>"
		goldName := `gold`
		if args[0] == goldName || (len(args[0]) >= 1 && len(args[0]) < 5 && goldName[0:len(args[0])] == args[0]) {
			if corpseRef.Gold < 1 {
				user.SendText(fmt.Sprintf(`There is no gold on the %s.`, corpseName))
			} else {
				user.Character.CancelBuffsWithFlag("hidden")

				goldAmt := corpseRef.Gold
				user.Character.Gold += goldAmt
				corpseRef.Gold = 0

				events.AddToQueue(events.EquipmentChange{
					UserId:     user.UserId,
					GoldChange: -goldAmt,
				})

				user.SendText(fmt.Sprintf(`You take <ansi fg="gold">%d gold</ansi> from the %s.`, goldAmt, corpseName))
				room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> takes some <ansi fg="gold">gold</ansi> from the %s.`, user.Character.Name, corpseName), user.UserId)
			}
			return true, nil
		}

		// "get <item> <corpse>" - search backpack items first, then worn items
		matchItem, found := corpseRef.FindItem(rest)
		if !found && wornLoot {
			// Also search worn equipment on the corpse character
			wornItems := corpseRef.Character.GetAllWornItems()
			closeMatch, exactMatch := items.FindMatchIn(rest, wornItems...)
			if exactMatch.ItemId != 0 {
				matchItem = exactMatch
				found = true
			} else if closeMatch.ItemId != 0 {
				matchItem = closeMatch
				found = true
			}
		}

		if !found {
			user.SendText(fmt.Sprintf(`You don't see a %s on the %s.`, rest, corpseName))
			return true, nil
		}

		if tooHeavy(user, matchItem) {
			return true, nil
		}

		user.Character.CancelBuffsWithFlag("hidden")

		if !user.Character.StoreItem(matchItem) {
			user.SendText(fmt.Sprintf(`You can't carry the <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()))
			return true, nil
		}

		events.AddToQueue(events.ItemOwnership{
			UserId: user.UserId,
			Item:   matchItem,
			Gained: true,
		})

		// Remove from whichever location it came from
		corpseRef.RemoveItem(matchItem)
		corpseRef.Character.RemoveFromBody(matchItem)

		user.SendText(fmt.Sprintf(`You take the <ansi fg="itemname">%s</ansi> from the %s.`, matchItem.DisplayName(), corpseName))
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> takes the <ansi fg="itemname">%s</ansi> from the %s.`, user.Character.Name, matchItem.DisplayName(), corpseName), user.UserId)

		return true, nil
	}

	if containerName != `` {
		container := room.Containers[containerName]

		goldName := `gold`
		if args[0] == goldName || (len(args[0]) >= 1 && len(args[0]) < 5 && goldName[0:len(args[0])] == args[0]) {

			if container.Gold < 1 {
				user.SendText("There's no gold to grab.")
			} else {

				user.Character.CancelBuffsWithFlag("hidden") // No longer sneaking

				goldAmt := container.Gold
				user.Character.Gold += goldAmt
				container.Gold -= goldAmt
				room.Containers[containerName] = container

				events.AddToQueue(events.EquipmentChange{
					UserId:     user.UserId,
					GoldChange: -goldAmt,
				})

				user.SendText(
					fmt.Sprintf(`You pick up <ansi fg="gold">%d gold</ansi> from the <ansi fg="container">%s</ansi>.`, goldAmt, containerName),
				)
				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> picks up some <ansi fg="gold">gold</ansi> from the <ansi fg="container">%s</ansi>.`, user.Character.Name, containerName),
					user.UserId,
				)
			}

			return true, nil
		}

		matchItem, found := container.FindItem(rest)

		if !found {
			user.SendText(fmt.Sprintf(`You don't see a %s in the <ansi fg="container">%s</ansi>.`, rest, containerName))
		} else {

			if tooHeavy(user, matchItem) {
				return true, nil
			}

			user.Character.CancelBuffsWithFlag("hidden") // No longer sneaking

			// Trigger onFound event
			if user.Character.StoreItem(matchItem) {

				events.AddToQueue(events.ItemOwnership{
					UserId: user.UserId,
					Item:   matchItem,
					Gained: true,
				})

				// Swap the item location
				container.RemoveItem(matchItem)
				room.Containers[containerName] = container

				user.SendText(
					fmt.Sprintf(`You take the <ansi fg="itemname">%s</ansi> from the <ansi fg="container">%s</ansi>.`, matchItem.DisplayName(), containerName),
				)
				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> picks up the <ansi fg="itemname">%s</ansi> from the <ansi fg="container">%s</ansi>...`, user.Character.Name, matchItem.DisplayName(), containerName),
					user.UserId,
				)

				return true, nil

			} else {
				user.SendText(
					fmt.Sprintf(`You can't carry the <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()),
				)
			}

		}

	} else {

		goldName := `gold`
		if args[0] == goldName || (len(args[0]) >= 1 && len(args[0]) < 5 && goldName[0:len(args[0])] == args[0]) {

			if room.Gold < 1 {
				user.SendText("There's no gold to grab.")
			} else {

				user.Character.CancelBuffsWithFlag("hidden") // No longer sneaking

				goldAmt := room.Gold
				user.Character.Gold += goldAmt
				room.Gold -= goldAmt

				events.AddToQueue(events.EquipmentChange{
					UserId:     user.UserId,
					GoldChange: -goldAmt,
				})

				user.SendText(
					fmt.Sprintf(`You pick up <ansi fg="gold">%d gold</ansi>.`, goldAmt),
				)
				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> picks up some <ansi fg="gold">gold</ansi>.`, user.Character.Name),
					user.UserId,
				)
			}

			return true, nil
		}

		// Check whether the user has an item in their inventory that matches
		matchItem, found := room.FindOnFloor(rest, getFromStash)

		// Check if user is specifying an item they stashed
		if !found && !getFromStash {
			stashItemMatch, stashFound := room.FindOnFloor(rest, true)
			if stashFound && stashItemMatch.StashedBy == user.UserId {
				found = true
				getFromStash = true
				matchItem = stashItemMatch
			}
		}

		if found {

			if matchItem.HasAdjective(`exploding`) {
				user.SendText(`You can't pick that up, it's about to explode!`)
				return true, nil
			}

			if tooHeavy(user, matchItem) {
				return true, nil
			}

			user.Character.CancelBuffsWithFlag("hidden") // No longer sneaking

			// If it was in the stash, remove the stash owner tag
			if getFromStash {
				matchItem.StashedBy = 0
			}

			if user.Character.StoreItem(matchItem) {

				// Swap the item location
				room.RemoveItem(matchItem, getFromStash)

				events.AddToQueue(events.ItemOwnership{
					UserId: user.UserId,
					Item:   matchItem,
					Gained: true,
				})

				if getFromStash {
					user.SendText(
						fmt.Sprintf(`You dig out the <ansi fg="itemname">%s</ansi> from where it was stashed.`, matchItem.DisplayName()),
					)
					room.SendText(
						fmt.Sprintf(`<ansi fg="username">%s</ansi> digs around in the area and picks something up...`, user.Character.Name),
						user.UserId,
					)
				} else {
					user.SendText(
						fmt.Sprintf(`You pick up the <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()),
					)
					room.SendText(
						fmt.Sprintf(`<ansi fg="username">%s</ansi> picks up the <ansi fg="itemname">%s</ansi>...`, user.Character.Name, matchItem.DisplayName()),
						user.UserId,
					)
				}

			} else {
				user.SendText(
					fmt.Sprintf(`You can't carry the <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()),
				)
			}

			return true, nil
		}

		//
		// Look for any nouns in the room info
		//
		foundNoun, _ := room.FindNoun(rest)
		if len(foundNoun) > 0 {

			user.SendText(fmt.Sprintf(`You can't get the <ansi fg="noun">%s</ansi>`, foundNoun))
			room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is grasping at the air.`, user.Character.Name), user.UserId)

			return true, nil
		}

	}

	if _, corpseFound := room.FindCorpse(rest); corpseFound {
		user.SendText(`You can't pick up corpses. What would people think?`)
		return true, nil
	}

	containerName = room.FindContainerByName(rest)
	if containerName != `` {
		user.SendText(fmt.Sprintf(`You can't pick up the <ansi fg="container">%s</ansi>. Try looking at it.`, containerName))
	} else {
		user.SendText(fmt.Sprintf("You don't see a %s around.", rest))
	}

	return true, nil
}

// corpseLootable reports whether userId may take something from the corpse
// now: claimed loot for its claimant, or any loot in a CorpseItems world.
func corpseLootable(c *rooms.Corpse, userId int) bool {
	if c.ClaimUserId != 0 {
		return c.ClaimUserId == userId && c.HasItems()
	}
	return bool(configs.GetGamePlayConfig().Death.CorpseItems) && (c.HasItems() || len(c.Character.GetAllWornItems()) > 0)
}
