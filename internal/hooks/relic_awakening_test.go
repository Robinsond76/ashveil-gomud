package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
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
	RelicPlaceAwakening(events.RoomChange{UserId: u.UserId, FromRoomId: 0, ToRoomId: keepGate.RoomId})
	assert.Zero(t, relic.AwakeningProgress(0), "logging in inside the zone is not crossing into it")
	u.Character.RoomId = keepGate.RoomId
	RelicPlaceAwakening(events.RoomChange{UserId: u.UserId, FromRoomId: road.RoomId, ToRoomId: keepGate.RoomId, Unseen: true})
	assert.True(t, relic.Awakened(0), "sneaking in counts like walking in")
	assert.Equal(t, 7, u.Character.ClassEffects().Int(classes.Attack), "4 from the signature, 3 awakened")
}

type relicCompanionLookup struct{ leader, instance int }

func (relicCompanionLookup) FormationFor(int) (company.Formation, bool) {
	return company.Formation{}, false
}
func (l relicCompanionLookup) InstanceFor(int, int) (int, bool) { return l.instance, true }
func (l relicCompanionLookup) LeaderAndKeyForInstance(id int) (int, company.MemberKey, bool) {
	return l.leader, company.CompanionMemberKey(1), id == l.instance
}

// Review fix: companions follow their leader a moment later, as their own
// queued step, so the leader's crossing never saw them arrive. A companion
// is credited by its own arrival (the RoomChange the real room.AddMob
// queues), while its leader is in the zone.
func TestACompanionFollowingIntoAZoneWakesItsPlaceAwakening(t *testing.T) {
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
	road, keep := mk(99861, "Old Road"), mk(99862, "Far Keep")

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	u := users.NewUserRecord(99860, 0)
	u.Character.Health, u.Character.RoomId = 10, keep.RoomId // the leader went first
	users.SetTestUser(u)

	const inst = 99865
	mate := &mobs.Mob{MobId: 77, InstanceId: inst}
	mate.Character = *characters.New()
	mate.Character.Name, mate.Character.Health, mate.Character.RoomId = "Tobin", 10, road.RoomId
	mate.Character.Equipment.Weapon = items.New(placeRelicID)
	mobs.SetTestInstance(mate)
	t.Cleanup(func() { mobs.RemoveTestInstance(inst) })
	company.SetFormationProvider(relicCompanionLookup{leader: u.UserId, instance: inst})
	t.Cleanup(func() { company.SetFormationProvider(nil) })

	events.ClearListeners()
	t.Cleanup(events.ClearListeners)
	events.RegisterListener(events.RoomChange{}, RelicPlaceAwakening)
	keep.AddMob(inst)
	events.ProcessEvents()

	assert.True(t, mate.Character.Equipment.Weapon.Awakened(0), "the companion's own step into the zone counts")
	assert.Equal(t, 7, mate.Character.ClassEffects().Int(classes.Attack))
}
