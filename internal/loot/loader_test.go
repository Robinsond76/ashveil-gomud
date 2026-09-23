package loot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func TestLoadTablesOmitsUnknownItemsAndRefreshes(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 987654, Name: "test drop"})
	t.Cleanup(func() { items.RemoveTestItemSpec(987654) })
	dir := t.TempDir()
	path := filepath.Join(dir, "beast.yaml")
	data := []byte("category: beast\nentries:\n  - itemid: 987654\n    weight: 2\n  - itemid: 987655\n    weight: 1\n")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := loadLootDataFiles(dir); err != nil {
		t.Fatal(err)
	}
	table, ok := GetTable("beast")
	if !ok || len(table.Entries) != 1 || table.Entries[0].ItemID != 987654 {
		t.Fatalf("table = %+v, %v", table, ok)
	}
	if err := os.WriteFile(path, []byte("category: beast\nentries:\n  - itemid: 987655\n    weight: 1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := loadLootDataFiles(dir); err != nil {
		t.Fatal(err)
	}
	if _, ok := GetTable("beast"); ok {
		t.Fatal("table with no valid entries retained")
	}
}
