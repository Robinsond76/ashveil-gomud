package usercommands

import (
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
