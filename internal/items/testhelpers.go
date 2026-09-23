package items

// Test helpers for cross-package testing.
// Since this is under internal/, it cannot be used outside the module.

// SetTestItemSpec registers an item spec directly in memory. For testing only.
func SetTestItemSpec(spec *ItemSpec) {
	items[spec.ItemId] = spec
}

// RemoveTestItemSpec removes an item spec registered with SetTestItemSpec.
// For testing only.
func RemoveTestItemSpec(itemId int) {
	delete(items, itemId)
}
