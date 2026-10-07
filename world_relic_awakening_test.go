package main

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// Phase 67: the production hook registration reaches the place awakening,
// so a real RoomChange event, processed with every other listener, wakes the
// relic the leader wears.
func TestProductionHooksWakeAPlaceAwakeningOnARoomChange(t *testing.T) {
	partyWorldData(t)
	t.Cleanup(events.ClearListeners)
	hooks.RegisterListeners()

	const relicID = 99861
	items.SetTestItemSpec(&items.ItemSpec{ItemId: relicID, Name: "Test Pilgrim", Type: items.Weapon, Subtype: items.Slashing, Hands: 1, Tier: 5,
		Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 6},
		Relic: &items.RelicSpec{Signature: "Road", Effects: map[string]int{classes.Attack: 4}, ILvl: 20, Mob: 1, Chance: 5,
			Awakenings: []items.AwakeningSpec{
				{Name: "Far Walker", Kind: items.AwakenPlace, Zone: "Far Keep", Target: "Far Keep", Effects: map[string]int{classes.Attack: 3}},
			}}})
	t.Cleanup(func() { items.RemoveTestItemSpec(relicID) })

	road := &rooms.Room{RoomId: 934301, Title: "Road", Zone: "Old Road", Exits: map[string]exit.RoomExit{"north": {RoomId: 934302}}}
	keep := &rooms.Room{RoomId: 934302, Title: "Keep gate", Zone: "Far Keep", Exits: map[string]exit.RoomExit{}}
	t.Cleanup(rooms.SetTestZoneConfig(&rooms.ZoneConfig{Name: "Old Road"}))
	t.Cleanup(rooms.SetTestZoneConfig(&rooms.ZoneConfig{Name: "Far Keep"}))
	rooms.SetTestRoom(road)
	rooms.SetTestRoom(keep)
	t.Cleanup(func() { rooms.RemoveTestRoom(road.RoomId); rooms.RemoveTestRoom(keep.RoomId) })

	u := users.NewUserRecord(93431, 0)
	u.Password = "$2a$fixture"
	u.Character.RoomId = keep.RoomId
	u.Character.Health = 20
	u.Character.Equipment.Weapon = items.New(relicID)
	users.SetTestUser(u)
	t.Cleanup(func() { users.RemoveTestUser(u.UserId) })
	keep.AddPlayer(u.UserId)

	events.AddToQueue(events.RoomChange{UserId: u.UserId, FromRoomId: road.RoomId, ToRoomId: keep.RoomId})
	events.ProcessEvents()

	assert.True(t, u.Character.Equipment.Weapon.Awakened(0))
	assert.Equal(t, 7, u.Character.ClassEffects().Int(classes.Attack))
}
