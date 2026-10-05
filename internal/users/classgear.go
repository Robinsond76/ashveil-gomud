package users

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// SettleClassGear moves a held shield or weapon the user's class may not
// use (Phase 35a2: clerics and rangers' shields, clerics' weapons) from the
// hands to carried items and saves the record with save (SaveUserAtomic
// when nil). It returns the
// one-time notice, or "" when nothing moved, so it is safe on every login,
// copyover and archetype choice: a second run finds nothing to move.
func SettleClassGear(u *UserRecord, save func(*UserRecord) error) string {
	if u == nil || u.Character == nil {
		return ""
	}
	u.Character.SetUserId(u.UserId)
	moved := u.Character.UnequipDisallowed()
	if len(moved) == 0 {
		return ""
	}
	if save == nil {
		save = func(r *UserRecord) error { return SaveUserAtomic(*r) }
	}
	if err := save(u); err != nil {
		// The move stays in memory and reaches disk with the next save.
		mudlog.Error("SettleClassGear", "user", u.UserId, "error", err)
	}
	names := make([]string, len(moved))
	for i, itm := range moved {
		names[i] = fmt.Sprintf(`<ansi fg="item">%s</ansi>`, itm.DisplayName())
	}
	_, reason := u.Character.CanWield(moved[0])
	return fmt.Sprintf(`%s You put your %s away with your carried things (see <ansi fg="command">help shields</ansi>).`, reason, strings.Join(names, ` and `))
}
