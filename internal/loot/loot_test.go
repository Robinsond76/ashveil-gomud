package loot

import "testing"

func TestTableValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table Table
	}{
		{"empty entries", Table{Category: "beast"}},
		{"zero weight", Table{Category: "beast", Entries: []WeightedLootEntry{{ItemID: 1}}}},
		{"reversed count", Table{Category: "beast", Entries: []WeightedLootEntry{{ItemID: 1, Weight: 1, MinCount: 3, MaxCount: 2}}}},
		{"missing category", Table{Entries: []WeightedLootEntry{{ItemID: 1, Weight: 1}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.table.Validate(); err == nil {
				t.Fatal("expected invalid table")
			}
		})
	}
}

func TestTableResolveWeightedBands(t *testing.T) {
	table := Table{Category: "beast", Entries: []WeightedLootEntry{{ItemID: 1, Weight: 3}, {ItemID: 2, Weight: 1}}}
	for roll, want := range []int{1, 1, 1, 2, 1, 1, 1, 2} {
		entry, ok := table.Resolve(uint64(roll))
		if !ok || entry.ItemID != want {
			t.Fatalf("roll %d = %+v, %v; want %d", roll, entry, ok, want)
		}
	}
	if _, ok := (Table{}).Resolve(0); ok {
		t.Fatal("empty table resolved")
	}
}

func TestRollCountDefaultsAndBounds(t *testing.T) {
	if got := (WeightedLootEntry{}).RollCount(99); got != 1 {
		t.Fatalf("default count = %d", got)
	}
	if got := (WeightedLootEntry{MinCount: 2}).RollCount(99); got != 2 {
		t.Fatalf("default maximum = %d", got)
	}
	entry := WeightedLootEntry{MinCount: 2, MaxCount: 4}
	for roll, want := range []int{2, 3, 4, 2, 3, 4} {
		if got := entry.RollCount(uint64(roll)); got != want {
			t.Fatalf("roll %d = %d; want %d", roll, got, want)
		}
	}
}
