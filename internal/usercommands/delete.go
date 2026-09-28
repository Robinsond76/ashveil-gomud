package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/death"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Ashveil Phase 32h: `delete character`. The player confirms with their
// password (masked) and then the character's name; the record is flagged
// and saved, and the character leaves the world with a hand-off despawn.
// The engine's final leave listener then purges every module's state with
// the login kept, resets the record to a new character, and logs the same
// user back in on the same connection, in the Void, where creation runs.

// deleteFailsKey counts wrong passwords since login (temp data, so a new
// login starts it empty). Three lock the command until the next login.
const (
	deleteFailsKey  = `delete-character-fails`
	deleteMaxFails  = 3
	DeleteNothing   = `Nothing was deleted.`
	deleteLockedMsg = `You've given the wrong password too many times. Log in again to try once more.`
)

func Delete(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if strings.ToLower(strings.TrimSpace(rest)) != `character` {
		user.SendText(`To delete your character for good, type <ansi fg="command">delete character</ansi>. See <ansi fg="command">help delete</ansi>.`)
		return true, nil
	}

	// Refusals are checked on every step: a fight may start while the
	// player is typing their password.
	if why := deleteRefusal(user); why != `` {
		user.SendText(why)
		user.ClearPrompt()
		return true, nil
	}
	if fails, _ := user.GetTempData(deleteFailsKey).(int); fails >= deleteMaxFails {
		user.SendText(deleteLockedMsg)
		user.ClearPrompt()
		return true, nil
	}

	name := user.Character.Name
	cmdPrompt, isNew := user.StartPrompt(`delete`, rest)
	if isNew {
		user.SendText(fmt.Sprintf("This deletes <ansi fg=\"username\">%s</ansi> for good: their level, gear, gold, company, camp, and everything else. Your login stays, and you'll make a new character. This can't be undone.", name))
	}

	question := cmdPrompt.Ask(fmt.Sprintf(`Type your password to delete %s, or anything else to cancel:`, name), []string{})
	question.Masked = true
	if !question.Done {
		return true, nil
	}
	if !user.PasswordMatches(question.Response) {
		fails, _ := user.GetTempData(deleteFailsKey).(int)
		fails++
		user.SetTempData(deleteFailsKey, fails)
		mudlog.Warn("delete character", "userId", user.UserId, "result", "wrong password", "fails", fails)
		user.ClearPrompt()
		if fails >= deleteMaxFails {
			user.SendText(DeleteNothing + ` ` + deleteLockedMsg)
			return true, nil
		}
		user.SendText(DeleteNothing)
		return true, nil
	}

	question = cmdPrompt.Ask(fmt.Sprintf(`Type %s to confirm, or anything else to cancel:`, name), []string{})
	if !question.Done {
		return true, nil
	}
	user.ClearPrompt()
	if !strings.EqualFold(strings.TrimSpace(question.Response), name) {
		user.SendText(DeleteNothing)
		return true, nil
	}

	user.Deleting = true
	if err := users.SaveUser(*user); err != nil {
		user.Deleting = false
		mudlog.Error("delete character", "userId", user.UserId, "error", err)
		user.SendText(DeleteNothing + ` Something went wrong saving; please try again.`)
		return true, nil
	}
	mudlog.Info("delete character", "userId", user.UserId, "character", name, "result", "deleting")
	user.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is gone.`, name))

	events.AddToQueue(events.PlayerDespawn{
		UserId:        user.UserId,
		RoomId:        user.Character.RoomId,
		Username:      user.Username,
		CharacterName: name,
		TimeOnline:    user.GetOnlineInfo().OnlineTimeStr,
		HandOff:       true,
	})
	return true, nil
}

// deleteRefusal is why the character can't be deleted now, or "".
func deleteRefusal(user *users.UserRecord) string {
	if user.IsReplay() {
		return `That's a practice character. Leave the course first.`
	}
	if user.Character.RoomId == -1 {
		return `You haven't finished making a character yet, so there's nothing to delete.`
	}
	if user.Character.Health < 1 {
		return `You can't do that while you're down.`
	}
	if p, ok := death.Active(); ok && p.Pending(user.UserId) {
		return `You can't do that while you're down.`
	}
	if user.Character.Aggro != nil {
		return `You're too busy fighting to do that right now.`
	}
	if _, ok := battle.Current(user.UserId); ok {
		return `You're too busy fighting to do that right now.`
	}
	return ``
}
