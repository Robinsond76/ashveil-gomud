package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/awakening"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// RelicPlaceAwakening is Phase 67's place source: a leader who crosses into
// another zone gives the relics the company wears a step toward any
// awakening that names that zone. Leaving and returning counts again (a
// place awakening asks once), so walking back and forth earns nothing.
func RelicPlaceAwakening(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.RoomChange)
	if !ok || evt.UserId == 0 || evt.Unseen {
		return events.Continue
	}
	to := rooms.LoadRoom(evt.ToRoomId)
	if to == nil || to.Zone == `` {
		return events.Continue
	}
	if from := rooms.LoadRoom(evt.FromRoomId); from != nil && from.Zone == to.Zone {
		return events.Continue
	}
	awakening.Reached(evt.UserId, to.Zone)
	return events.Continue
}
