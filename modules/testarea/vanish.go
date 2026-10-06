package testarea

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Whatever lies on the floor of a test room vanishes: the area is a place to
// try gear, and a dropped item or heap of gold must never pile up there. The
// trip's return already restores the user's own goods; this keeps the rooms
// themselves empty.

// onItemDropped removes a dropped item at once. ItemOwnership (Gained false)
// is queued by a drop, so by the time it runs the item is on the floor.
func onItemDropped(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.ItemOwnership)
	if !ok || evt.Gained || evt.UserId == 0 {
		return events.Continue
	}
	user := userInArea(evt.UserId)
	if user == nil {
		return events.Continue
	}
	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		return events.Continue
	}
	before := len(room.Items)
	room.RemoveItem(evt.Item, false)
	if len(room.Items) < before {
		say(user, fmt.Sprintf(`The <ansi fg="item">%s</ansi> vanishes in a puff of smoke.`, evt.Item.DisplayName()))
	}
	return events.Continue
}

// onNewRound sweeps what the drop hook cannot see (dropped gold, and any
// loot a foe left) from every test room.
func onNewRound(e events.Event) events.ListenerReturn {
	for id := FirstRoom; id <= LastRoom; id++ {
		if room := rooms.LoadRoom(id); room != nil && (len(room.Items) > 0 || room.Gold > 0) {
			room.Items = nil
			room.Gold = 0
		}
	}
	return events.Continue
}

func userInArea(userID int) *users.UserRecord {
	user := users.GetByUserId(userID)
	if user == nil || user.Character == nil || !IsAreaRoom(user.Character.RoomId) {
		return nil
	}
	return user
}
