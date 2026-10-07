package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/awakening"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// RelicPlaceAwakening is Phase 67's place source: a member of a company who
// crosses from one zone into another gives the relics they wear a step
// toward any awakening that names that zone. The leader is credited by their
// own step and each companion by its own (companions follow a moment after
// the leader, so the leader's step cannot see them arrive). A step from no
// room (logging in, being spawned) is not a crossing, and moving about
// inside a zone adds nothing. Sneaking in counts like walking in.
func RelicPlaceAwakening(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.RoomChange)
	if !ok || (evt.UserId == 0 && evt.MobInstanceId == 0) {
		return events.Continue
	}
	to := rooms.LoadRoom(evt.ToRoomId)
	from := rooms.LoadRoom(evt.FromRoomId)
	if to == nil || to.Zone == `` || from == nil || from.Zone == to.Zone {
		return events.Continue
	}
	if evt.UserId != 0 {
		awakening.Reached(evt.UserId, to.Zone)
	} else {
		awakening.CompanionReached(evt.MobInstanceId, to.Zone)
	}
	return events.Continue
}
