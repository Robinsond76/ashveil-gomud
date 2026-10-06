package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Mark (Phase 36c) marks a carried item as junk, or clears the mark, for
// `sell junk`. With no arguments it lists what is marked.
func Mark(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		listed := []string{}
		for _, item := range user.Character.Items {
			if item.Junk {
				listed = append(listed, fmt.Sprintf(`  <ansi fg="itemname">%s</ansi>`, item.DisplayName()))
			}
		}
		if len(listed) == 0 {
			user.SendText(`You have nothing marked as junk. Try <ansi fg="command">mark [item] junk</ansi>; see <ansi fg="command">help mark</ansi>.`)
			return true, nil
		}
		user.SendText("Marked as junk (<ansi fg=\"command\">sell junk</ansi> sells these):\n" + strings.Join(listed, "\n"))
		return true, nil
	}

	words := strings.Fields(rest)
	state := words[len(words)-1]
	junk := true
	switch strings.ToLower(state) {
	case "junk":
	case "keep", "clear":
		junk = false
	default:
		user.SendText(`Mark it how? Try <ansi fg="command">mark [item] junk</ansi> or <ansi fg="command">mark [item] keep</ansi>.`)
		return true, nil
	}
	name := strings.TrimSpace(strings.Join(words[:len(words)-1], " "))
	if name == "" {
		user.SendText(`Mark what? Try <ansi fg="command">mark [item] junk</ansi>.`)
		return true, nil
	}

	item, found := user.Character.FindInBackpack(name)
	if !found {
		user.SendText("You don't have that item.")
		return true, nil
	}
	if item.GetSpec().QuestToken != `` {
		user.SendText("Quest items cannot be marked as junk!")
		return true, nil
	}
	if item.Junk == junk {
		if junk {
			user.SendText(fmt.Sprintf(`The <ansi fg="itemname">%s</ansi> is already marked as junk.`, item.DisplayName()))
		} else {
			user.SendText(fmt.Sprintf(`The <ansi fg="itemname">%s</ansi> isn't marked as junk.`, item.DisplayName()))
		}
		return true, nil
	}

	marked := item
	marked.Junk = junk
	user.Character.UpdateItem(item, marked)
	item = marked
	if junk {
		user.SendText(fmt.Sprintf(`You mark the <ansi fg="itemname">%s</ansi> as junk. <ansi fg="command">sell junk</ansi> at a merchant sells it.`, item.DisplayName()))
	} else {
		user.SendText(fmt.Sprintf(`You take the junk mark off the <ansi fg="itemname">%s</ansi>.`, item.DisplayName()))
	}
	return true, nil
}
