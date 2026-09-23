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
	for _, reload := range []bool{false, true} {
		loadAllDataFiles(reload)
		for _, category := range []string{"beast", "humanoid"} {
			table, ok := loot.GetTable(category)
			if !ok || len(table.Entries) == 0 {
				t.Fatalf("reload=%v category=%s table=%+v", reload, category, table)
			}
		}
	}
}
