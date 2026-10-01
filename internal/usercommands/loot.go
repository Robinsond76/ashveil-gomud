package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Loot uses Get's existing corpse/claim/capacity checks. The numeric corpse
// reference makes same-named corpses deterministic, including when one is empty.
func Loot(rest string, u *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if InBattle(u) {
		u.SendText(BattleUnderWay)
		return true, nil
	}
	onlyOwn := strings.EqualFold(strings.TrimSpace(rest), "own")
	if strings.TrimSpace(rest) != "" && !onlyOwn {
		return Get("all "+rest, u, room, flags)
	}
	n := 0
	for i := range room.Corpses {
		c := &room.Corpses[i]
		if onlyOwn && c.ClaimUserId != u.UserId {
			continue
		}
		if !c.HasItems() || !c.CanLoot(u.UserId, util.GetRoundCount()) {
			continue
		}
		if !corpseLootable(c, u.UserId) {
			continue
		}
		_, err := Get("all corpse#"+strconv.Itoa(i+1), u, room, flags)
		if err != nil {
			return true, err
		}
		n++
	}
	if n == 0 {
		u.SendText("No eligible battle loot is here.")
	}
	return true, nil
}

func AutoLoot(rest string, u *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	switch strings.TrimSpace(strings.ToLower(rest)) {
	case "on":
		u.Character.AutoLoot = true
	case "off":
		u.Character.AutoLoot = false
	case "":
	default:
		u.SendText("Usage: autoloot on|off")
		return true, nil
	}
	u.SendText(fmt.Sprintf("Automatic loot: %t. Loot is collected after your battle ends.", u.Character.AutoLoot))
	return true, nil
}
