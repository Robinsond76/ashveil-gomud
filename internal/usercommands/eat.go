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

// findConsumable resolves an optional trailing member selector without breaking
// multi-word item names. It first treats the final token as a member selector
// and requires the remaining words to fully match an item; when that fails it
// falls back to the whole input as the legacy item name.
func findConsumable(rest string, user *users.UserRecord) (items.Item, string, bool) {
	tokens := util.SplitButRespectQuotes(rest)
	if len(tokens) >= 2 {
		itemName := strings.Join(tokens[:len(tokens)-1], " ")
		if _, full := items.FindMatchIn(itemName, user.Character.Items...); full.ItemId != 0 {
			return full, tokens[len(tokens)-1], true
		}
	}
	if matchItem, found := user.Character.FindInBackpack(rest); found {
		return matchItem, "", true
	}
	return items.Item{}, "", false
}

// provisionSuffix renders the target and band-crossing text appended to the
// personal consumption message after a successful survival provision.
func provisionSuffix(result survival.ProvisionResult, explicit bool) string {
	if !result.Crossed() {
		if explicit {
			return fmt.Sprintf(` You provision <ansi fg="username">%s</ansi>.`, result.Name)
		}
		return ""
	}
	parts := []string{}
	if result.Hunger.Crossed() {
		parts = append(parts, fmt.Sprintf("hunger %s", survival.HungerLabel(result.Needs.Hunger)))
	}
	if result.Thirst.Crossed() {
		parts = append(parts, fmt.Sprintf("thirst %s", survival.ThirstLabel(result.Needs.Thirst)))
	}
	if result.Fatigue.Crossed() {
		parts = append(parts, fmt.Sprintf("fatigue %s", survival.FatigueLabel(result.Needs.Fatigue)))
	}
	subject := "You are"
	if explicit {
		subject = fmt.Sprintf(`<ansi fg="username">%s</ansi> is`, result.Name)
	}
	return fmt.Sprintf(" %s now %s.", subject, strings.Join(parts, ", "))
}

func Eat(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Check whether the user has an item in their inventory that matches
	matchItem, selector, found := findConsumable(rest, user)

	if !found {
		user.SendText(fmt.Sprintf(`You don't have a "%s" to eat.`, rest))
	} else {

		itemSpec := matchItem.GetSpec()

		if itemSpec.Subtype != items.Edible {
			user.SendText(
				fmt.Sprintf(`You can't eat <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()),
			)
			return true, nil
		}

		suffix := ""
		if itemSpec.Nutrition > 0 || itemSpec.Hydration > 0 {
			result, err := survival.Provision(user.UserId, selector, survival.Benefit{
				Nutrition: itemSpec.Nutrition,
				Hydration: itemSpec.Hydration,
			})
			if err != nil {
				return true, err
			}
			suffix = provisionSuffix(result, selector != "")
		}

		user.Character.CancelBuffsWithFlag("hidden")

		user.SendText(fmt.Sprintf(`You eat some of the <ansi fg="itemname">%s</ansi>.%s`, matchItem.DisplayName(), suffix))
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> eats some <ansi fg="itemname">%s</ansi>.`, user.Character.Name, matchItem.DisplayName()), user.UserId)

		// If no more uses, will be lost, so trigger event
		if usesLeft := user.Character.UseItem(matchItem); usesLeft < 1 {

			events.AddToQueue(events.ItemOwnership{
				UserId: user.UserId,
				Item:   matchItem,
				Gained: false,
			})

		}

		for _, buffId := range itemSpec.BuffIds {
			user.AddBuff(buffId, `food`)
		}

	}

	return true, nil
}
