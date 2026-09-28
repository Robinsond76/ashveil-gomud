package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Password(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Get if already exists, otherwise create new
	cmdPrompt, _ := user.StartPrompt(`password`, rest)

	if !user.HasPlaintextPassword() {
		// Ashveil 32h: three wrong passwords in a row lock this and
		// `delete character` until the next login.
		if PasswordLocked(user) {
			user.SendText(`<ansi fg="alert-5">` + deleteLockedMsg + `</ansi>`)
			user.ClearPrompt()
			return true, nil
		}
		question := cmdPrompt.Ask(`What is your current password?`, []string{})
		question.Masked = true // Ashveil 32h
		if !question.Done {
			return true, nil
		}

		if !CheckPassword(user, question.Response, `password`) {
			user.SendText(`<ansi fg="alert-5">Sorry, your password was incorrect.</ansi>`)
			user.ClearPrompt()
			return true, nil
		}
	}

	question := cmdPrompt.Ask(`What new password would you like?`, []string{})
	question.Masked = true // Ashveil 32h
	if !question.Done {
		return true, nil
	}

	newPW := question.Response

	question = cmdPrompt.Ask(`Confirm the change by entered the new password one more time.`, []string{})
	question.Masked = true // Ashveil 32h
	if !question.Done {
		return true, nil
	}

	newPWConfirm := question.Response

	if newPW != newPWConfirm {
		user.SendText(`<ansi fg="alert-5">Sorry, your new password and the confirmation password did not match.</ansi>`)
		user.ClearPrompt()
		return true, nil
	}

	if err := user.SetPassword(newPW); err != nil {
		user.SendText(`<ansi fg="alert-5">` + err.Error() + `</ansi>`)
		user.ClearPrompt()
		return true, nil
	}

	users.SaveUser(*user)

	user.SendText(`<ansi fg="alert-1">Your password has been changed!</ansi>`)

	return true, nil
}
