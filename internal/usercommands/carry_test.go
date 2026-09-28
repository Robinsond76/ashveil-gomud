package usercommands

import (
	"strings"
	"testing"

	"maps"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 32f: weight is the only limit, and a full company takes on nothing
// more. The fake load is each leader's own carried items against a set
// capacity.

const (
	carryAnvil  = 988301 // 4 kg
	carryPebble = 988302 // 100 g
)

type packLoad struct{ capacity map[int]int }

func (p packLoad) CurrentLoad(leaderUserID int) (encumbrance.Load, bool) {
	capacity, ok := p.capacity[leaderUserID]
	if !ok {
		return encumbrance.Load{}, false
	}
	carried := 0
	if user := users.GetByUserId(leaderUserID); user != nil {
		for i := range user.Character.Items {
			carried += user.Character.Items[i].Weight()
		}
	}
	return encumbrance.Load{PersonalGrams: carried, CapacityGrams: capacity}, true
}

func setupCarry(t *testing.T, capacity map[int]int) {
	t.Helper()
	items.SetTestItemSpec(&items.ItemSpec{ItemId: carryAnvil, Name: "anvil", NameSimple: "anvil", Type: items.Object, Weight: 4000, Value: 5})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: carryPebble, Name: "pebble", NameSimple: "pebble", Type: items.Object, Weight: 100, Value: 1})
	encumbrance.SetProvider(packLoad{capacity: capacity})
	users.ResetActiveUsers()
	t.Cleanup(func() {
		items.RemoveTestItemSpec(carryAnvil)
		items.RemoveTestItemSpec(carryPebble)
		encumbrance.SetProvider(nil)
		users.ResetActiveUsers()
	})
}

func carrier(t *testing.T, userID int, name string, room *rooms.Room) *users.UserRecord {
	t.Helper()
	user := users.NewUserRecord(userID, 1)
	user.Character.Name = name
	user.Character.RoomId = room.RoomId
	user.Character.Gold = 100
	users.SetTestUser(user)
	room.AddPlayer(userID)
	return user
}

func newItem(id int) items.Item { return items.Item{ItemId: id, UUID: uuid.New(items.UUIDItem)} }

func holds(user *users.UserRecord, id int) bool {
	for _, itm := range user.Character.Items {
		if itm.ItemId == id {
			return true
		}
	}
	return false
}

func TestGetRefusedWhenFull(t *testing.T) {
	setupCarry(t, map[int]int{7: 3000})
	room := testRoom()
	user := carrier(t, 7, "Dain", room)

	room.AddItem(newItem(carryAnvil), false)
	out := captureUserText(t, func() {
		_, err := Get("anvil", user, room, 0)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "too much for your company to carry")
	assert.False(t, holds(user, carryAnvil))
	_, onFloor := room.FindOnFloor("anvil", false)
	assert.True(t, onFloor, "it stays where it was")

	room.AddItem(newItem(carryPebble), false)
	_, err := Get("pebble", user, room, 0)
	require.NoError(t, err)
	assert.True(t, holds(user, carryPebble), "a light thing still fits")
}

func TestGetFromContainerRefusedWhenFull(t *testing.T) {
	setupCarry(t, map[int]int{7: 3000})
	room := testRoom()
	room.Containers = map[string]rooms.Container{"chest": {Items: []items.Item{newItem(carryAnvil)}}}
	user := carrier(t, 7, "Dain", room)

	out := captureUserText(t, func() {
		_, err := Get("anvil chest", user, room, 0)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "too much for your company to carry")
	assert.False(t, holds(user, carryAnvil))
	assert.Len(t, room.Containers["chest"].Items, 1)
}

func TestBuyRefusedWhenFullAndNoGoldTaken(t *testing.T) {
	setupCarry(t, map[int]int{7: 3000})
	room := testRoom()
	buyer := carrier(t, 7, "Dain", room)
	seller := carrier(t, 8, "Mira", room)
	seller.Character.Shop = characters.Shop{{ItemId: carryAnvil, Quantity: 1, QuantityMax: 1, Price: 5}}

	out := captureUserText(t, func() {
		assert.False(t, tryPurchase("anvil", buyer, room, nil, seller))
	})
	assert.Contains(t, out, "too much for your company to carry")
	assert.Equal(t, 100, buyer.Character.Gold, "no gold changes hands")
	assert.Equal(t, 1, seller.Character.Shop[0].Quantity, "no stock changes hands")
	assert.False(t, holds(buyer, carryAnvil))
}

func TestGiveToFullPlayerRefused(t *testing.T) {
	setupCarry(t, map[int]int{7: 100000, 8: 3000})
	room := testRoom()
	giver := carrier(t, 7, "Dain", room)
	receiver := carrier(t, 8, "Mira", room)
	giver.Character.Items = []items.Item{newItem(carryAnvil)}

	out := captureUserText(t, func() {
		_, err := Give("anvil mira", giver, room, 0)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "can't carry any more")
	assert.True(t, holds(giver, carryAnvil), "the giver keeps it")
	assert.False(t, holds(receiver, carryAnvil))
}

func TestGetFromCorpseRefusedWhenFull(t *testing.T) {
	setupCarry(t, map[int]int{7: 3000})
	before := configs.Flatten(configs.GetOverrides())
	after := maps.Clone(before)
	after["GamePlay.Death.CorpseItems"] = true
	require.NoError(t, configs.RestoreOverrides(after))
	t.Cleanup(func() { _ = configs.RestoreOverrides(before) })

	room := testRoom()
	corpse := rooms.Corpse{MobId: 5, Items: []items.Item{newItem(carryAnvil), newItem(carryPebble)}}
	corpse.Character.Name = "wolf"
	room.AddCorpse(corpse)
	user := carrier(t, 7, "Dain", room)

	out := captureUserText(t, func() {
		_, err := Get("anvil wolf", user, room, 0)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "too much for your company to carry")
	assert.False(t, holds(user, carryAnvil))

	_, err := Get("pebble wolf", user, room, 0)
	require.NoError(t, err)
	assert.True(t, holds(user, carryPebble), "a light thing still fits")
}

// TestPickpocketLeavesWhatWontFit (32f review finding 4): a stolen item
// that would overload the thief's company stays with its owner.
func TestPickpocketLeavesWhatWontFit(t *testing.T) {
	setupCarry(t, map[int]int{7: 3000, 8: 100000})
	skills.SetTestData([]*skills.Skill{{SkillId: "skulduggery", Name: "Skulduggery", Description: "Sneak.", MaxLevel: 4}}, nil)
	t.Cleanup(func() { skills.SetTestData(nil, nil) })
	before := configs.Flatten(configs.GetOverrides())
	after := maps.Clone(before)
	after["GamePlay.PVP.Enabled"] = "enabled"
	after["GamePlay.PVP.MinimumLevel"] = 0
	require.NoError(t, configs.RestoreOverrides(after))
	t.Cleanup(func() { _ = configs.RestoreOverrides(before) })
	room := testRoom()
	room.RoomId = 988300 // not the death-recovery room, where no one fights
	thief := carrier(t, 7, "Dain", room)
	mark := carrier(t, 8, "Mira", room)
	mark.Character.Gold = 0
	mark.Character.Items = []items.Item{newItem(carryAnvil)}
	thief.Character.Skills = map[string]int{"skulduggery": 4}
	thief.Character.Stats.Speed.ValueAdj = 500
	thief.Character.Stats.Smarts.ValueAdj = 500
	thief.Character.Stats.Perception.ValueAdj = 500

	out := captureUserText(t, func() {
		_, err := Pickpocket("mira", thief, room, 0)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "can't carry any more")
	assert.False(t, holds(thief, carryAnvil))
	assert.True(t, holds(mark, carryAnvil), "the mark keeps it")
}

// TestGiveToFullPlayersPetRefused (32f review finding 4): a pet's pouch is
// its owner's load, so a full company can't take it through the pet.
func TestGiveToFullPlayersPetRefused(t *testing.T) {
	setupCarry(t, map[int]int{7: 100000, 8: 3000})
	room := testRoom()
	giver := carrier(t, 7, "Dain", room)
	owner := carrier(t, 8, "Mira", room)
	owner.Character.Pet.Type = "dog"
	owner.Character.Pet.Name = "rex"
	giver.Character.Items = []items.Item{newItem(carryAnvil)}

	out := captureUserText(t, func() {
		_, err := Give("anvil rex", giver, room, 0)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "can't carry any more")
	assert.True(t, holds(giver, carryAnvil), "the giver keeps it")
	assert.Empty(t, owner.Character.Pet.Items)
}

type ownCompanion struct{}

func (ownCompanion) FormationFor(int) (company.Formation, bool) { return company.Formation{}, false }
func (ownCompanion) InstanceFor(int, int) (int, bool)           { return 988399, true }
func (ownCompanion) LeaderAndKeyForInstance(instanceId int) (int, company.MemberKey, bool) {
	if instanceId == 988399 {
		return 7, company.MemberKey("companion:1"), true
	}
	return 0, "", false
}

// TestGiveToOwnCompanionAtCapacity (32f review test gap): handing a thing
// to your own companion keeps it in the company, so a full company may.
func TestGiveToOwnCompanionAtCapacity(t *testing.T) {
	setupCarry(t, map[int]int{7: 4000}) // the anvil alone fills it
	company.SetFormationProvider(ownCompanion{})
	t.Cleanup(func() { company.SetFormationProvider(nil) })
	room := testRoom()
	giver := carrier(t, 7, "Dain", room)
	giver.Character.Items = []items.Item{newItem(carryAnvil)}
	tamsin := &mobs.Mob{InstanceId: 988399}
	tamsin.Character.Name = "Tamsin"
	tamsin.Character.RoomId = room.RoomId
	mobs.SetTestInstance(tamsin)
	t.Cleanup(func() { mobs.RemoveTestInstance(988399) })
	room.AddMob(988399)

	out := captureUserText(t, func() {
		_, err := Give("anvil tamsin", giver, room, 0)
		require.NoError(t, err)
	})
	assert.NotContains(t, out, "can't carry any more")
	assert.False(t, holds(giver, carryAnvil))
	require.Len(t, tamsin.Character.Items, 1, "Tamsin carries it now")
}

// TestGetPackWhenFull (32f review nit): a pack picked up counts the room it
// makes, so a full company can still take on a satchel that pays for itself.
func TestGetPackWhenFull(t *testing.T) {
	setupCarry(t, map[int]int{7: 3000})
	const satchel = 988303
	items.SetTestItemSpec(&items.ItemSpec{ItemId: satchel, Name: "satchel", NameSimple: "satchel", Type: items.Object, Weight: 600, CarryBonus: 5000})
	t.Cleanup(func() { items.RemoveTestItemSpec(satchel) })
	room := testRoom()
	user := carrier(t, 7, "Dain", room)
	user.Character.Items = []items.Item{newItem(carryPebble), newItem(carryPebble)}
	for len(user.Character.Items) < 30 {
		user.Character.Items = append(user.Character.Items, newItem(carryPebble))
	} // 3.0 kg: full

	room.AddItem(newItem(satchel), false)
	_, err := Get("satchel", user, room, 0)
	require.NoError(t, err)
	assert.True(t, holds(user, satchel), "it makes more room than it weighs")
}

// TestGetAllSaysOnceWhenFull (32f review nit): `get all` takes what fits
// and names the rest in one line.
func TestGetAllSaysOnceWhenFull(t *testing.T) {
	setupCarry(t, map[int]int{7: 3000})
	room := testRoom()
	user := carrier(t, 7, "Dain", room)
	room.AddItem(newItem(carryAnvil), false)
	room.AddItem(newItem(carryAnvil), false)
	room.AddItem(newItem(carryPebble), false)

	out := captureUserText(t, func() {
		_, err := Get("all", user, room, 0)
		require.NoError(t, err)
	})
	assert.True(t, holds(user, carryPebble))
	assert.Equal(t, 1, strings.Count(out, "can't carry any more"), out)
	assert.Contains(t, out, "You leave 2 things behind")
	assert.NotContains(t, out, "too much for your company to carry")
}

// TestPeepShowsWeight (32f review nit): peep weighs what someone carries.
func TestPeepShowsWeight(t *testing.T) {
	setupCarry(t, map[int]int{})
	c := characters.New()
	c.Items = []items.Item{newItem(carryAnvil), newItem(carryPebble)}
	out := buildPeepInventoryPanel(c, []string{"anvil", "pebble"})
	assert.Contains(t, out, "Weight:   4.1 kg")
}
