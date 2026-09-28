package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// tooHeavy refuses an item that would put the user's company over its
// carrying capacity (Phase 32f), telling them why. It reports true when
// refused. Moving things within the company never comes through here.
func tooHeavy(user *users.UserRecord, itm items.Item) bool {
	text, refuse := encumbrance.TooMuchToCarry(user.UserId, itm.Weight())
	if refuse {
		user.SendText(text)
	}
	return refuse
}

// canCarryStolen reports whether a pickpocketed item fits the thief's
// company (32f review finding 4); a full one leaves it where it was.
func canCarryStolen(user *users.UserRecord, itm items.Item) bool {
	if _, full := encumbrance.WouldExceed(user.UserId, itm.Weight()); full {
		user.SendText(fmt.Sprintf(`You feel a <ansi fg="itemname">%s</ansi>, but your company can't carry any more.`, itm.DisplayName()))
		return false
	}
	return true
}
