package loot

import (
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

var allLootTables = map[string]Table{}

// LoadLootDataFiles refreshes category tables after item specs load.
func LoadLootDataFiles() {
	start := time.Now()
	if err := loadLootDataFiles(configs.GetFilePathsConfig().DataFiles.String() + "/loot"); err != nil {
		panic(err)
	}
	mudlog.Info("loot.LoadLootDataFiles()", "loadedCount", len(allLootTables), "Time Taken", time.Since(start))
}

func loadLootDataFiles(path string) error {
	loaded, err := fileloader.LoadAllFlatFiles[string, *Table](path)
	if err != nil {
		return err
	}
	valid := make(map[string]Table, len(loaded))
	for category, table := range loaded {
		entries := make([]WeightedLootEntry, 0, len(table.Entries))
		for _, entry := range table.Entries {
			if items.GetItemSpec(entry.ItemID) == nil {
				mudlog.Warn("loot.LoadLootDataFiles()", "category", category, "unknownItemID", entry.ItemID)
				continue
			}
			entries = append(entries, entry)
		}
		if len(entries) == 0 {
			continue
		}
		valid[category] = Table{Category: category, Entries: entries}
	}
	allLootTables = valid
	return nil
}

func GetTable(category string) (Table, bool) {
	table, ok := allLootTables[category]
	return table, ok
}
