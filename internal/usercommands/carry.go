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
	text, refuse := encumbrance.TooMuchToCarry(user.UserId, encumbrance.AddedGrams(user.UserId, user.Character.Items, itm))
	if refuse {
		user.SendText(text)
	}
	return refuse
}

// fits reports, quietly, whether an item would fit the user's company;
// `get all` counts what doesn't and says so once (leftBehind).
func fits(user *users.UserRecord, itm items.Item) bool {
	_, full := encumbrance.WouldExceed(user.UserId, encumbrance.AddedGrams(user.UserId, user.Character.Items, itm))
	return !full
}

// leftBehind is `get all`'s one line for what the company couldn't carry.
func leftBehind(user *users.UserRecord, count int) {
	switch {
	case count == 1:
		user.SendText(`You leave one thing behind: your company can't carry any more (<ansi fg="command">help cargo</ansi>).`)
	case count > 1:
		user.SendText(fmt.Sprintf(`You leave %d things behind: your company can't carry any more (<ansi fg="command">help cargo</ansi>).`, count))
	}
}
