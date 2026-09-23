package mobcommands

import (
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
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 987654, Name: "test meat"})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 987653, Name: "old gear"})
	t.Cleanup(func() {
		items.RemoveTestItemSpec(987654)
		items.RemoveTestItemSpec(987653)
		loot.RemoveTestTable("test-beast")
	})
	loot.SetTestTable(loot.Table{Category: "test-beast", Entries: []loot.WeightedLootEntry{{ItemID: 987654, Weight: 1, MinCount: 2}}})
	for _, tc := range []struct {
		name        string
		corpse      bool
		category    string
		missingItem bool
		wantExtra   int
	}{
		{"floor", false, "test-beast", false, 2},
		{"uncategorized", false, "", false, 0},
		{"corpse", true, "test-beast", false, 2},
		{"item spec removed after load", false, "test-beast", true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gameplay := configs.GetGamePlayConfig()
			gameplay.Death.CorpsesEnabled = configs.ConfigBool(tc.corpse)
			gameplay.Death.CorpseItems = configs.ConfigBool(tc.corpse)
			t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
			room := rooms.NewEmptyRoom()
			mob := &mobs.Mob{MobId: 999, LootCategory: tc.category}
			mob.Character.Name = "test beast"
			mob.Character.Items = []items.Item{items.New(987653)}
			mob.Character.Gold = 7
			if tc.missingItem {
				items.RemoveTestItemSpec(987654)
			}
			drops := 0
			listener := events.RegisterListener(events.MobItemDrop{}, func(event events.Event) events.ListenerReturn { drops++; return events.Continue })
			ok, err := Suicide("", mob, room)
			if err != nil || !ok {
				t.Fatalf("suicide: %v, %v", ok, err)
			}
			events.ProcessEvents()
			events.UnregisterListener(events.MobItemDrop{}, listener)
			if drops != 1+tc.wantExtra && !tc.corpse {
				t.Fatalf("drop events = %d; want %d", drops, 1+tc.wantExtra)
			}
			if tc.corpse && drops != 0 {
				t.Fatalf("corpse emitted %d item drop events", drops)
			}
			if tc.corpse {
				if len(room.Corpses) != 1 || len(room.Corpses[0].Items) != 1+tc.wantExtra || room.Corpses[0].Gold != 7 {
					t.Fatalf("corpse = %+v", room.Corpses)
				}
			} else if len(room.Items) != 1+tc.wantExtra || room.Gold != 7 {
				t.Fatalf("floor items = %+v; gold = %d", room.Items, room.Gold)
			}
		})
	}
}
