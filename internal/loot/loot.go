// Package loot resolves shared, weighted category drops without owning randomness.
package loot

import (
	"fmt"
	"math"
	"strings"
)

type WeightedLootEntry struct {
	ItemID   int  `yaml:"itemid"`
	Weight   uint `yaml:"weight"`
	MinCount int  `yaml:"mincount,omitempty"`
	MaxCount int  `yaml:"maxcount,omitempty"`
}

type Table struct {
	Category string              `yaml:"category"`
	Entries  []WeightedLootEntry `yaml:"entries"`
}

func (t Table) Id() string       { return t.Category }
func (t Table) Filepath() string { return t.Category + ".yaml" }

func (t Table) Validate() error {
	if strings.TrimSpace(t.Category) == "" || t.Category != strings.ToLower(strings.TrimSpace(t.Category)) {
		return fmt.Errorf("loot category must be a nonempty lowercase name")
	}
	if len(t.Entries) == 0 {
		return fmt.Errorf("loot category %q has no entries", t.Category)
	}
	var total uint64
	for _, e := range t.Entries {
		if e.Weight == 0 {
			return fmt.Errorf("loot category %q has a zero weight", t.Category)
		}
		min, max := e.countBounds()
		if e.MinCount < 0 || e.MaxCount < 0 || max < min {
			return fmt.Errorf("loot category %q has invalid count bounds", t.Category)
		}
		if uint64(e.Weight) > math.MaxUint64-total {
			return fmt.Errorf("loot category %q weight sum overflows", t.Category)
		}
		total += uint64(e.Weight)
	}
	return nil
}

func (t Table) Resolve(roll uint64) (WeightedLootEntry, bool) {
	if t.Validate() != nil {
		return WeightedLootEntry{}, false
	}
	var total uint64
	for _, e := range t.Entries {
		total += uint64(e.Weight)
	}
	position := roll % total
	for _, e := range t.Entries {
		if position < uint64(e.Weight) {
			return e, true
		}
		position -= uint64(e.Weight)
	}
	return WeightedLootEntry{}, false
}

func (e WeightedLootEntry) countBounds() (int, int) {
	min := e.MinCount
	if min == 0 {
		min = 1
	}
	max := e.MaxCount
	if max == 0 {
		max = min
	}
	return min, max
}

func (e WeightedLootEntry) RollCount(roll uint64) int {
	min, max := e.countBounds()
	if min < 1 || max < min {
		return 0
	}
	return min + int(roll%uint64(max-min+1))
}
