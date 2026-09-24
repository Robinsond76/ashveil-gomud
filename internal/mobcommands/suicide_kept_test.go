package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// keptOnDeath kills a mob wearing a helmet and a weapon and carrying one
// item and 7 gold, and returns the MobDeath event and the IDs that reached
// the floor.
func keptOnDeath(t *testing.T, dropChance int) (events.MobDeath, map[int]int) {
	t.Helper()
	mudlog.SetupLogger(nil, "low", "", false)
	for _, id := range []int{987641, 987642, 987643} {
		items.SetTestItemSpec(&items.ItemSpec{ItemId: id, Name: "test item"})
	}
	t.Cleanup(func() {
		for _, id := range []int{987641, 987642, 987643} {
			items.RemoveTestItemSpec(id)
		}
	})
	gameplay := configs.GetGamePlayConfig()
	gameplay.Death.CorpsesEnabled = false
	gameplay.Death.CorpseItems = false
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))

	room := rooms.NewEmptyRoom()
	mob := &mobs.Mob{MobId: 999, ItemDropChance: dropChance}
	mob.Character.Name = "test companion"
	mob.Character.Level = 4
	mob.Character.Equipment.Head = items.New(987641)
	mob.Character.Equipment.Weapon = items.New(987642)
	mob.Character.Items = []items.Item{items.New(987643)}
	mob.Character.Gold = 7

	var death events.MobDeath
	listener := events.RegisterListener(events.MobDeath{}, func(e events.Event) events.ListenerReturn {
		death = e.(events.MobDeath)
		return events.Continue
	})
	defer events.UnregisterListener(events.MobDeath{}, listener)
	ok, err := Suicide("", mob, room)
	if err != nil || !ok {
		t.Fatalf("suicide: %v, %v", ok, err)
	}
	events.ProcessEvents()
	floor := map[int]int{}
	for _, itm := range room.Items {
		floor[itm.ItemId]++
	}
	return death, floor
}

// TestMobDeathKeptWornDropChanceZero: a drop chance of 0 keeps every worn
// item on the body (Phase 25b), and the event says so; carried items and
// gold still drop.
func TestMobDeathKeptWornDropChanceZero(t *testing.T) {
	death, floor := keptOnDeath(t, 0)
	if death.KeptWorn[items.Head].ItemId != 987641 || death.KeptWorn[items.Weapon].ItemId != 987642 || len(death.KeptWorn) != 2 {
		t.Fatalf("kept worn = %+v", death.KeptWorn)
	}
	if len(death.KeptItems) != 0 || death.KeptGold != 0 {
		t.Fatalf("carried and gold drop: %+v, %d", death.KeptItems, death.KeptGold)
	}
	if floor[987643] != 1 || floor[987641] != 0 || floor[987642] != 0 {
		t.Fatalf("floor = %v", floor)
	}
}

// TestMobDeathKeptWornDropChanceAll: at 100 every worn item drops and none
// is reported kept.
func TestMobDeathKeptWornDropChanceAll(t *testing.T) {
	death, floor := keptOnDeath(t, 100)
	if len(death.KeptWorn) != 0 {
		t.Fatalf("kept worn = %+v", death.KeptWorn)
	}
	if floor[987641] != 1 || floor[987642] != 1 || floor[987643] != 1 {
		t.Fatalf("floor = %v", floor)
	}
}

// TestMobDeathPermaGearKeepsEverything: a perma-gear mob drops nothing, so
// everything is reported kept.
func TestMobDeathPermaGearKeepsEverything(t *testing.T) {
	mob := &mobs.Mob{MobId: 999, ItemDropChance: 100}
	mob.Character.Equipment.Head = items.Item{ItemId: 987641}
	mob.Character.Equipment.Weapon = items.Item{ItemId: 987642}
	mob.Character.Items = []items.Item{{ItemId: 987643}}
	mob.Character.Gold = 7
	body := bodyKeeps(mob, true)
	if len(body.dropWorn) != 0 || len(body.keptWorn) != 2 || len(body.keptItems) != 1 || body.keptGold != 7 {
		t.Fatalf("body = %+v", body)
	}
	// Remove-locked worn gear never drops, whatever the chance.
	locked := items.Item{ItemId: 987641, CanNeverBeRemoved: true}
	mob.Character.Equipment.Head = locked
	body = bodyKeeps(mob, false)
	if body.keptWorn[items.Head].ItemId != 987641 || len(body.dropWorn) != 1 || body.dropWorn[0].ItemId != 987642 {
		t.Fatalf("body = %+v", body)
	}
	if len(body.keptItems) != 0 || body.keptGold != 0 {
		t.Fatalf("carried and gold drop: %+v", body)
	}
}
