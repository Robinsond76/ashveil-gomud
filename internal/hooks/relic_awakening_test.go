package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

const placeRelicID = 99841

// Phase 67: crossing into a zone wakes the place awakening of a worn relic,
// once; moving about inside the zone, or back and forth, adds nothing.
func TestCrossingIntoAZoneWakesAPlaceAwakening(t *testing.T) {
	items.SetTestItemSpec(&items.ItemSpec{ItemId: placeRelicID, Name: "Test Pilgrim", Type: items.Weapon, Subtype: items.Slashing, Hands: 1, Tier: 5,
		Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 6},
		Relic: &items.RelicSpec{Signature: "Road", Effects: map[string]int{classes.Attack: 4}, ILvl: 20, Mob: 1, Chance: 5,
			Awakenings: []items.AwakeningSpec{
				{Name: "Far Walker", Kind: items.AwakenPlace, Zone: "Far Keep", Target: "Far Keep", Effects: map[string]int{classes.Attack: 3}},
			}}})
	t.Cleanup(func() { items.RemoveTestItemSpec(placeRelicID) })
	mk := func(id int, zone string) *rooms.Room {
		r := rooms.NewEmptyRoom()
		r.RoomId, r.Zone = id, zone
		rooms.SetTestRoom(r)
		t.Cleanup(func() { rooms.RemoveTestRoom(id) })
		return r
	}
	road, keepGate, keepHall := mk(99851, "Old Road"), mk(99852, "Far Keep"), mk(99853, "Far Keep")

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	u := users.NewUserRecord(99840, 0)
	u.Character.Health, u.Character.RoomId = 10, keepGate.RoomId
	u.Character.Equipment.Weapon = items.New(placeRelicID)
	users.SetTestUser(u)
	relic := &u.Character.Equipment.Weapon

	move := func(from, to *rooms.Room) {
		u.Character.RoomId = to.RoomId
		RelicPlaceAwakening(events.RoomChange{UserId: u.UserId, FromRoomId: from.RoomId, ToRoomId: to.RoomId})
	}
	move(keepGate, keepHall)
	assert.Zero(t, relic.AwakeningProgress(0), "inside the zone is not entering it")
	move(road, road)
	assert.Zero(t, relic.AwakeningProgress(0), "another zone")
	RelicPlaceAwakening(events.RoomChange{MobInstanceId: 5, FromRoomId: road.RoomId, ToRoomId: keepGate.RoomId})
	assert.Zero(t, relic.AwakeningProgress(0), "a mob's move is not the company's")
	move(road, keepGate)
	assert.True(t, relic.Awakened(0))
	assert.Equal(t, 7, u.Character.ClassEffects().Int(classes.Attack), "4 from the signature, 3 awakened")
}
