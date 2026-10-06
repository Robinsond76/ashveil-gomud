package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/cookbook"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Use(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Ashveil Phase 32d: a battle plays out as it was set up.
	if InBattle(user) {
		user.SendText(BattleUnderWay)
		return true, nil
	}

	containerName := room.FindContainerByName(rest)
	if containerName != `` {

		container := room.Containers[containerName]

		if len(container.Recipes) > 0 {

			if container.Lock.IsLocked() {
				user.SendText(``)
				user.SendText(fmt.Sprintf(`The <ansi fg="container">%s</ansi> is locked.`, containerName))
				user.SendText(``)
				return true, nil
			}

			var known func(int) bool
			if container.IsHearth(containerName) {
				known = func(finalItemId int) bool {
					return cookbook.Knows(user.Character, HearthRecipe(container, finalItemId))
				}
			}
			recipeReadyItemId, blocked := container.SelectRecipe(user.Character.GetSkillLevel, known)

			if recipeReadyItemId == 0 && blocked.MinLevel > 0 {
				user.SendText("")
				user.SendText(fmt.Sprintf(`You need at least level %d in <ansi fg="skill">%s</ansi> to make anything with the <ansi fg="container">%s</ansi>.`, blocked.MinLevel, recipeSkillName(blocked.SkillId), containerName))
				user.SendText("")
				return true, nil
			}

			if recipeReadyItemId == 0 {
				user.SendText("")
				if known != nil {
					user.SendText(fmt.Sprintf(`The <ansi fg="container">%s</ansi> seems to be missing something, or you don't know a dish to make from it. To try a new combination, <ansi fg="command">cook</ansi> it (<ansi fg="command">help recipes</ansi>).`, containerName))
				} else {
					user.SendText(fmt.Sprintf(`The <ansi fg="container">%s</ansi> seems to be missing something.`, containerName))
				}
				user.SendText("")
				return true, nil
			}

			for _, removeItem := range container.Recipes[recipeReadyItemId] {
				if matchItem, found := container.FindItemById(removeItem); found {
					container.RemoveItem(matchItem)
				}
			}

			newItem := items.New(recipeReadyItemId)

			container.AddItem(newItem)
			room.Containers[containerName] = container

			room.PlaySound(`change`, `other`)

			user.SendText(``)
			user.SendText(fmt.Sprintf(`The <ansi fg="container">%s</ansi> produces a <ansi fg="itemname">%s</ansi>!`, containerName, newItem.DisplayName()))
			user.SendText(``)

			room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> does something with the <ansi fg="container">%s</ansi>.`, user.Character.Name, containerName), user.UserId)

			return true, nil

		}

	}

	// Check whether the user has an item in their inventory that matches
	matchItem, found := user.Character.FindInBackpack(rest)

	if !found {
		user.SendText(fmt.Sprintf(`You don't have a "%s" to use.`, rest))
	} else {

		itemSpec := matchItem.GetSpec()

		if itemSpec.Subtype != items.Usable {
			user.SendText(
				fmt.Sprintf(`You can't use <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()))
			return true, nil
		}

		user.Character.CancelBuffsWithFlag("hidden")

		// Phase 56: a recipe page teaches its dish.
		if itemSpec.Recipe > 0 {
			dish := items.GetItemSpec(itemSpec.Recipe)
			if dish == nil {
				user.SendText(fmt.Sprintf(`The <ansi fg="itemname">%s</ansi> is too smudged to read.`, matchItem.DisplayName()))
				return true, nil
			}
			if usesLeft := user.Character.UseItem(matchItem); usesLeft < 1 {
				events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: matchItem, Gained: false})
			}
			if cookbook.Learn(user.Character, itemSpec.Recipe) {
				user.SendText(fmt.Sprintf(`You study the <ansi fg="itemname">%s</ansi> and learn to cook <ansi fg="itemname">%s</ansi>. It is in your recipe book (<ansi fg="command">recipes</ansi>).`, matchItem.DisplayName(), dish.Name))
			} else {
				user.SendText(fmt.Sprintf(`You already know how to cook <ansi fg="itemname">%s</ansi>; the page crumbles in your hands.`, dish.Name))
			}
			return true, nil
		}

		user.SendText(fmt.Sprintf(`You use the <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()))
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> uses their <ansi fg="itemname">%s</ansi>.`, user.Character.Name, matchItem.DisplayName()), user.UserId)

		// If no more uses, will be lost, so trigger event
		if usesLeft := user.Character.UseItem(matchItem); usesLeft < 1 {

			events.AddToQueue(events.ItemOwnership{
				UserId: user.UserId,
				Item:   matchItem,
				Gained: false,
			})

		}

		for _, buffId := range itemSpec.BuffIds {
			user.AddBuff(buffId, `item`)
		}
	}

	return true, nil
}

// recipeSkillName returns a skill's display name, falling back to its id.
func recipeSkillName(skillId string) string {
	if s := skills.GetSkill(skillId); s != nil && s.Name != `` {
		return s.Name
	}
	return skillId
}

// HearthRecipe is a hearth container's recipe for an output item as the
// cookbook sees it (Phase 56): ungated dishes are basic.
func HearthRecipe(container rooms.Container, finalItemId int) cookbook.Recipe {
	r := cookbook.Recipe{Output: finalItemId, Inputs: container.Recipes[finalItemId]}
	if req, gated := container.RecipeRequirements[finalItemId]; gated {
		r.Skill, r.MinLevel = req.SkillId, req.MinLevel
	}
	return r
}
