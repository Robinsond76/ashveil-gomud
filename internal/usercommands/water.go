package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 40a: water sources. A room with the `water` resource lets you drink
// from it (`drink water`, `drink source`) and refill a refillable
// container (`fill`). None of it spends supplies or advances game time.

const (
	// WaterSourceHydration is what one drink from a source gives: the same
	// as one glug of the waterskin.
	WaterSourceHydration = 40
	// WaterSourceBuff is Hydrated, which a waterskin glug also gives.
	WaterSourceBuff = 34
	// RefillWater is the `refillable` value of items filled from water.
	RefillWater = `water`

	noWaterHere = `There is no fresh water here.`
)

// HasWater reports whether the room is a water source.
func HasWater(room *rooms.Room) bool {
	return room != nil && room.HasResource(rooms.ResourceWater)
}

// RefillableUses is the uses a water container is filled back to, and
// whether the item is one at all.
func RefillableUses(spec items.ItemSpec) (int, bool) {
	if !strings.EqualFold(spec.Refillable, RefillWater) {
		return 0, false
	}
	// Phase 43a: an empty container is filled into its full item, so it
	// fills to that item's uses.
	if spec.FilledItemId > 0 {
		if full := items.GetItemSpec(spec.FilledItemId); full != nil && full.Uses > 0 {
			return full.Uses, true
		}
		return 0, false
	}
	if spec.Uses < 1 {
		return 0, false
	}
	return spec.Uses, true
}

// drinkFromSource handles `drink water` and `drink source`. handled is
// false when the words name something else, so Drink carries on with the
// pack lookup.
func drinkFromSource(rest string, user *users.UserRecord, room *rooms.Room) (handled bool, err error) {
	tokens := util.SplitButRespectQuotes(rest)
	if len(tokens) == 0 {
		return false, nil
	}
	word := strings.ToLower(tokens[0])
	if word != `water` && word != `source` && !(word == `from` && len(tokens) > 1) {
		return false, nil
	}
	if word == `from` {
		tokens = tokens[1:]
		word = strings.ToLower(tokens[0])
		if word != `water` && word != `source` {
			return false, nil
		}
	}
	selector := ``
	switch len(tokens) {
	case 1:
	case 2:
		if !survival.IsMemberSelector(user.UserId, tokens[1]) {
			return false, nil
		}
		selector = tokens[1]
	default:
		return false, nil
	}
	if word == `water` {
		// An item actually named "water" in the pack is drunk first, and
		// away from a source "drink water" keeps its old meaning.
		if itm, found := user.Character.FindInBackpack(`water`); found && strings.EqualFold(itm.GetSpec().Name, `water`) {
			return false, nil
		}
		if !HasWater(room) {
			return false, nil
		}
	}
	if !HasWater(room) {
		user.SendText(noWaterHere)
		return true, nil
	}
	result, err := survival.Provision(user.UserId, selector, survival.Benefit{Hydration: WaterSourceHydration})
	if err != nil {
		return true, err
	}
	user.Character.CancelBuffsWithFlag(`hidden`)
	if result.Member == survival.LeaderMemberKey {
		user.AddBuff(WaterSourceBuff, `drink`)
		user.SendText(`You drink deeply from the water here.` + drinkSuffix(result, false))
	} else {
		user.SendText(fmt.Sprintf(`You give <ansi fg="username">%s</ansi> a drink from the water here.`, result.Name) + drinkSuffix(result, true))
	}
	room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> drinks from the water here.`, user.Character.Name), user.UserId)
	return true, nil
}

// Fill refills a water container at a water source: `fill` tops up every
// refillable container in your pack, `fill [container]` one of them.
// fillHandlers (light gear) let a module fill what water can't: a lantern
// with lamp oil. Each returns whether it handled the command.
var fillHandlers []func(rest string, user *users.UserRecord) bool

// RegisterFillHandler adds a handler `fill` tries before water.
func RegisterFillHandler(h func(rest string, user *users.UserRecord) bool) {
	fillHandlers = append(fillHandlers, h)
}

func Fill(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	for _, h := range fillHandlers {
		if h(strings.TrimSpace(rest), user) {
			return true, nil
		}
	}
	if InBattle(user) {
		user.SendText(BattleUnderWay)
		return true, nil
	}
	rest = strings.TrimSpace(rest)
	if !HasWater(room) {
		user.SendText(noWaterHere + ` Look for water on the map or in "look".`)
		return true, nil
	}
	var targets []items.Item
	if rest == `` {
		for _, itm := range user.Character.Items {
			if _, ok := RefillableUses(itm.GetSpec()); ok {
				targets = append(targets, itm)
			}
		}
		if len(targets) == 0 {
			user.SendText(`You carry nothing that can be filled with water.`)
			return true, nil
		}
	} else {
		itm, found := user.Character.FindInBackpack(rest)
		if !found {
			user.SendText(fmt.Sprintf(`You don't have a "%s" to fill.`, rest))
			return true, nil
		}
		if _, ok := RefillableUses(itm.GetSpec()); !ok {
			user.SendText(fmt.Sprintf(`You can't fill <ansi fg="itemname">%s</ansi>.`, itm.DisplayName()))
			return true, nil
		}
		targets = append(targets, itm)
	}

	filled := []string{}
	full := []string{}
	for _, itm := range targets {
		max, _ := RefillableUses(itm.GetSpec())
		if itm.Uses >= max {
			full = append(full, itm.DisplayName())
			continue
		}
		for i := range user.Character.Items {
			if user.Character.Items[i].Equals(itm) {
				user.Character.Items[i] = user.Character.Items[i].Refilled(max)
				break
			}
		}
		filled = append(filled, itm.DisplayName())
	}
	switch {
	case len(filled) > 0:
		user.SendText(fmt.Sprintf(`You fill <ansi fg="itemname">%s</ansi> from the water here.`, strings.Join(filled, `</ansi>, <ansi fg="itemname">`)))
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> fills a container from the water here.`, user.Character.Name), user.UserId)
	default:
		user.SendText(fmt.Sprintf(`<ansi fg="itemname">%s</ansi> is already full.`, strings.Join(full, `</ansi>, <ansi fg="itemname">`)))
	}
	return true, nil
}
