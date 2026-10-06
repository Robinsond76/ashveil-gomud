package main

import (
	"slices"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
)

// TestShippedMealBuffs loads the real world (Phase 50): every hearth dish
// gives a meal buff, every item's `meal:` names a known kind, and each kind
// is served by some shipped item.
func TestShippedMealBuffs(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	if err := configs.ReloadConfig(); err != nil {
		t.Fatal(err)
	}
	loadAllDataFiles(false)

	served := map[string]bool{}
	for _, spec := range items.GetAllItemSpecs() {
		if spec.Meal == "" {
			continue
		}
		if !slices.Contains(survival.MealKinds(), spec.Meal) {
			t.Errorf("item %d (%s) has unknown meal %q", spec.ItemId, spec.Name, spec.Meal)
		}
		if spec.Type != items.Food || spec.Nutrition < 1 {
			t.Errorf("item %d (%s) gives a meal buff but is not food", spec.ItemId, spec.Name)
		}
		served[spec.Meal] = true
	}
	for _, kind := range survival.MealKinds() {
		if !served[kind] {
			t.Errorf("no shipped item serves meal %q", kind)
		}
	}
	room := rooms.LoadRoomTemplate(waymarkInnRoomId)
	if room == nil {
		t.Fatal("Waymark Inn not loaded")
	}
	for outputId := range room.Containers["hearth"].Recipes {
		if out := items.GetItemSpec(outputId); out == nil || out.Meal == "" {
			t.Errorf("hearth dish %d gives no meal buff", outputId)
		}
	}
}
