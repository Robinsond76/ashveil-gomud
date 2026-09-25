package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/death"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// lastLossLine is the experience line for the most recent level lost to
// death (Phase 26a), or "" when there is none.
func lastLossLine(user *users.UserRecord) string {
	from, to, ok := death.LastLoss(user.Character)
	if !ok {
		return ``
	}
	if from > to {
		return fmt.Sprintf(`<ansi fg="black-bold">Your last death cost you a level: %d to %d.</ansi>`, from, to) + "\n"
	}
	return fmt.Sprintf(`<ansi fg="black-bold">Your last death cost you your progress toward level %d.</ansi>`, from+1) + "\n"
}
