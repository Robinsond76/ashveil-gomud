package main

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/loot"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func TestLoadAllDataFilesLoadsShippedLootOnStartupAndReload(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	if err := configs.ReloadConfig(); err != nil {
		t.Fatal(err)
	}
	loadAllDataFiles(false)
	for _, category := range []string{"beast", "humanoid"} {
		table, ok := loot.GetTable(category)
		if !ok || len(table.Entries) == 0 {
			t.Fatalf("startup category=%s table=%+v", category, table)
		}
	}
	loot.SetTestTable(loot.Table{Category: "beast", Entries: []loot.WeightedLootEntry{{ItemID: 987654, Weight: 1}}})
	loot.SetTestTable(loot.Table{Category: "stale", Entries: []loot.WeightedLootEntry{{ItemID: 987654, Weight: 1}}})
	loadAllDataFiles(true)
	beast, ok := loot.GetTable("beast")
	if !ok || len(beast.Entries) == 0 || beast.Entries[0].ItemID == 987654 {
		t.Fatalf("reload left stale beast table: %+v", beast)
	}
	if _, ok := loot.GetTable("stale"); ok {
		t.Fatal("reload retained a removed category")
	}
}
