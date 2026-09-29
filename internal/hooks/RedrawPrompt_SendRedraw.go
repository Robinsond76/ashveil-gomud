package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Checks whether their level is too high for a guide
func RedrawPrompt_SendRedraw(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.RedrawPrompt)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "RedrawPrompt", "Actual Type", e.Type())
		return events.Cancel
	}

	if user := users.GetByUserId(evt.UserId); user != nil {

		newCmdPrompt := user.GetCommandPrompt()
		// Phase 29f: while combat lines are held, the prompt shows the
		// round's start, so it never runs ahead of the narration.
		if held, ok := heldPrompt(user.UserId); ok {
			newCmdPrompt = held
		}

		if evt.OnlyIfChanged {

			oldCmdPrompt := user.GetTempData(`cmdprompt`)

			// If the prompt hasn't changed, skip redrawing
			if oldCmdPrompt != nil && oldCmdPrompt.(string) == newCmdPrompt {
				return events.Continue
			}

			// save the new prompt for next time we want to check
			user.SetTempData(`cmdprompt`, newCmdPrompt)

		}

		writePrompt(user, newCmdPrompt)

	}

	return events.Continue
}

// writePrompt sends a prompt to a player's connection; tests replace it
// (Phase 29f).
var writePrompt = func(user *users.UserRecord, prompt string) {
	connections.SendTo([]byte(templates.AnsiParse(prompt)), user.ConnectionId())
}

// SetWritePromptForTest captures prompt redraws and returns a restore func.
func SetWritePromptForTest(fn func(userId int, prompt string)) (restore func()) {
	prev := writePrompt
	writePrompt = func(user *users.UserRecord, prompt string) { fn(user.UserId, prompt) }
	return func() { writePrompt = prev }
}
