package loot

// SetTestTable installs a category table for cross-package integration tests.
func SetTestTable(table Table) { allLootTables[table.Category] = table }

// RemoveTestTable removes a table installed by SetTestTable.
func RemoveTestTable(category string) { delete(allLootTables, category) }
