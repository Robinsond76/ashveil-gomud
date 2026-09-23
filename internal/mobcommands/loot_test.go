package mobcommands

import (
	"reflect"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/loot"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func TestSuicideCategoryLootUsesCorpseOrFloor(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	for _, id := range []int{987654, 987653, 987652} {
		items.SetTestItemSpec(&items.ItemSpec{ItemId: id, Name: "test item"})
	}
	t.Cleanup(func() {
		for _, id := range []int{987654, 987653, 987652} {
			items.RemoveTestItemSpec(id)
		}
		loot.RemoveTestTable("test-beast")
	})
	loot.SetTestTable(loot.Table{Category: "test-beast", Entries: []loot.WeightedLootEntry{{ItemID: 987654, Weight: 1, MinCount: 2}}})
	for _, tc := range []struct {
		name        string
		corpse      bool
		category    string
		missingItem bool
		wornChance  int
		locked      bool
		wantWorn    bool
		wantExtra   int
	}{
		{"floor with worn gear", false, "test-beast", false, 100, false, true, 2},
		{"uncategorized keeps worn roll", false, "", false, 100, false, true, 0},
		{"zero worn chance keeps category drop", false, "test-beast", false, 0, false, false, 2},
		{"locked worn gear stays put", false, "test-beast", false, 100, true, false, 2},
		{"corpse", true, "test-beast", false, 100, false, true, 2},
		{"item spec removed after load", false, "test-beast", true, 100, false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gameplay := configs.GetGamePlayConfig()
			gameplay.Death.CorpsesEnabled = configs.ConfigBool(tc.corpse)
			gameplay.Death.CorpseItems = configs.ConfigBool(tc.corpse)
			t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
			room := rooms.NewEmptyRoom()
			mob := &mobs.Mob{MobId: 999, LootCategory: tc.category, ItemDropChance: tc.wornChance}
			mob.Character.Name = "test beast"
			mob.Character.Items = []items.Item{items.New(987653)}
			worn := items.New(987652)
			worn.CanNeverBeRemoved = tc.locked
			mob.Character.Equipment.Head = worn
			mob.Character.Gold = 7
			if tc.missingItem {
				items.RemoveTestItemSpec(987654)
			}
			drops := map[int]int{}
			listener := events.RegisterListener(events.MobItemDrop{}, func(event events.Event) events.ListenerReturn {
				drops[event.(events.MobItemDrop).ItemId]++
				return events.Continue
			})
			ok, err := Suicide("", mob, room)
			if err != nil || !ok {
				t.Fatalf("suicide: %v, %v", ok, err)
			}
			events.ProcessEvents()
			events.UnregisterListener(events.MobItemDrop{}, listener)
			wantIDs := map[int]int{987653: 1}
			if tc.wantWorn {
				wantIDs[987652] = 1
			}
			if tc.wantExtra > 0 {
				wantIDs[987654] = tc.wantExtra
			}
			actualIDs := map[int]int{}
			if tc.corpse {
				if len(room.Corpses) != 1 || room.Corpses[0].Gold != 7 {
					t.Fatalf("corpse = %+v", room.Corpses)
				}
				for _, item := range room.Corpses[0].Items {
					actualIDs[item.ItemId]++
				}
				if len(drops) != 0 {
					t.Fatalf("corpse emitted drop events: %v", drops)
				}
			} else {
				for _, item := range room.Items {
					actualIDs[item.ItemId]++
				}
				if room.Gold != 7 {
					t.Fatalf("floor gold = %d", room.Gold)
				}
				if !reflect.DeepEqual(drops, wantIDs) {
					t.Fatalf("drop events = %v; want %v", drops, wantIDs)
				}
			}
			if !reflect.DeepEqual(actualIDs, wantIDs) {
				t.Fatalf("item IDs = %v; want %v", actualIDs, wantIDs)
			}
		})
	}
}
