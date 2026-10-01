package archetype

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// retireSkills (Ashveil 33f1) refunds and removes any retired skill (or
// retired top level) a player still holds, at every spawn, and saves the user so the refund is
// kept. The refund and the removal are one change to the user record, so
// it happens once: a later spawn finds nothing to refund.
func (m *ArchetypeModule) retireSkills(user *users.UserRecord) {
	if user == nil || user.Character == nil {
		return
	}
	points, removed := user.Character.RetireSkills()
	if len(removed) == 0 {
		return
	}
	if m.saveUser != nil {
		if err := m.saveUser(user); err != nil {
			mudlog.Error("archetype: saving retired-skill refund", "user", user.UserId, "error", err)
		}
	}
	user.SendText(fmt.Sprintf(`<ansi fg="yellow">No longer part of the world: %s. You get back the %d training %s you spent.</ansi>`,
		strings.Join(removed, ", "), points, plural(points, "point", "points")))
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
