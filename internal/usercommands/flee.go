package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Flee is another name for retreat (Phase 33c owner review): there is one
// way out of a fight, an ordered withdrawal of the whole company. Wimpy's
// automatic flee issues the same order.
func Flee(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	return Retreat(rest, user, room, flags)
}
