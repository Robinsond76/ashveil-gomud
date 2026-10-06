package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/loot"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
* Role Permissions:
* spawn				(All)
 */
func Spawn(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) == 0 {
		// send some sort of help info?
		infoOutput, _ := templates.Process("admincommands/help/command.spawn", nil, user.UserId)
		user.SendText(infoOutput)
		return true, nil
	}

	spawnType := args[0]
	args = args[1:]

	if spawnType == `loot` {
		user.SendText(spawnLoot(args, user, room))
		return true, nil
	}

	spawnTarget := ``
	if len(args) == 1 {
		spawnTarget = args[0]
		args = args[1:]
	} else {
		spawnTarget = strings.Join(args, ` `)
		args = []string{}

	}

	if len(spawnTarget) > 0 {

		if spawnType == `container` {

			containerName := room.SpawnTempContainer(spawnTarget, "3 rounds", 0)

			user.SendText(
				fmt.Sprintf(`You wave your hands around and <ansi fg="container">%s</ansi> appears from thin air and falls to the ground.`, containerName),
			)
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> waves their hands around and <ansi fg="container">%s</ansi> appears from thin air and falls to the ground.`, user.Character.Name, containerName),
				user.UserId,
			)

			return true, nil
		}

		if spawnType == `gold` || spawnTarget == `gold` {

			goldAmt := 0
			if spawnType == `gold` {
				goldAmt, _ = strconv.Atoi(spawnTarget)
			} else {
				goldAmt, _ = strconv.Atoi(spawnType)
			}

			if goldAmt < 1 {
				goldAmt = 1
			}

			room.Gold += goldAmt

			user.SendText(
				fmt.Sprintf(`You wave your hands around and <ansi fg="gold">%d gold</ansi> appears from thin air and falls to the ground.`, goldAmt),
			)
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> waves their hands around and <ansi fg="gold">%d gold</ansi> appears from thin air and falls to the ground.`, user.Character.Name, goldAmt),
				user.UserId,
			)

			return true, nil
		}

	}

	user.SendText(
		"You wave your hands around pathetically.",
	)
	room.SendText(
		fmt.Sprintf(`<ansi fg="username">%s</ansi> waves their hands around pathetically.`, user.Character.Name),
		user.UserId,
	)

	return true, nil
}

// spawnLoot is "spawn loot [item] [ilvl] [rarity] [quality]" (Phase 36a):
// rolls one piece of gear with the loot generator and drops it in the room.
// The item is an id or name; the other words are told apart by what they
// are (a number is the item level).
func spawnLoot(args []string, user *users.UserRecord, room *rooms.Room) string {
	const usage = `Usage: spawn loot [item id or name] [ilvl] [rarity] [quality]`
	if len(args) == 0 {
		return usage
	}
	itemID := items.FindItem(args[0])
	if itemID == 0 {
		return fmt.Sprintf(`No item matches "%s". %s`, args[0], usage)
	}
	opts := loot.Options{ILvl: 10, Source: `an admin's hand`}
	for _, word := range args[1:] {
		if n, err := strconv.Atoi(word); err == nil {
			opts.ILvl = n
		} else if r := items.Rarity(word); r.Valid() {
			opts.Rarity = r
		} else if q := items.Quality(word); q.Valid() {
			opts.Quality = q
		} else {
			return fmt.Sprintf(`"%s" is not an item level, rarity or quality. %s`, word, usage)
		}
	}
	itm, err := loot.Roll(itemID, opts, loot.GameSource())
	if err != nil {
		return err.Error()
	}
	room.AddItem(itm, false)
	room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> waves their hands around and %s appears from thin air and falls to the ground.`, user.Character.Name, itm.DisplayName()), user.UserId)
	return fmt.Sprintf(`You wave your hands around and %s appears from thin air and falls to the ground.`, itm.DisplayName())
}
