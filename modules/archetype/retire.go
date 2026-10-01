package archetype

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// retireSkills (Ashveil 33f1) refunds and removes any retired skill a
// player still holds, at every spawn, and saves the user so the refund is
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
	user.SendText(fmt.Sprintf(`<ansi fg="yellow">The %s %s no longer part of the world. You get back the %d training %s you spent on %s.</ansi>`,
		strings.Join(removed, ", "), plural(len(removed), "skill is", "skills are"), points, plural(points, "point", "points"), plural(len(removed), "it", "them")))
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
