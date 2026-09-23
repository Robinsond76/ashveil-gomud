package main

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const waymarkInnRoomId = 2003

// TestShippedCookingContent loads the real world and checks the Phase 18b slice end to
// end: the cooking skill and cook profession load, the Waymark Inn hearth's recipes use
// shipped 18a ingredients and produce edible food, the inn trains cooking, and the real
// use command cooks there.
func TestShippedCookingContent(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	if err := configs.ReloadConfig(); err != nil {
		t.Fatal(err)
	}
	loadAllDataFiles(false)

	if !skills.SkillExists("cooking") {
		t.Fatal("cooking skill not loaded")
	}
	if p := skills.GetProfessionSpec("cook"); p == nil || len(p.Skills) != 1 || p.Skills[0] != "cooking" {
		t.Fatalf("cook profession = %+v", p)
	}

	room := rooms.LoadRoomTemplate(waymarkInnRoomId)
	if room == nil {
		t.Fatal("Waymark Inn not loaded")
	}
	if _, ok := room.SkillTraining["cooking"]; !ok {
		t.Fatal("Waymark Inn does not train cooking")
	}
	hearth, ok := room.Containers["hearth"]
	if !ok || len(hearth.Recipes) < 2 {
		t.Fatalf("Waymark Inn hearth recipes = %+v", hearth.Recipes)
	}

	for outputId, inputs := range hearth.Recipes {
		out := items.GetItemSpec(outputId)
		if out == nil || out.Type != items.Food || out.Subtype != items.Edible || out.Nutrition < 1 {
			t.Errorf("recipe output %d is not edible food: %+v", outputId, out)
		}
		for _, inputId := range inputs {
			if items.GetItemSpec(inputId) == nil {
				t.Errorf("recipe %d input %d has no item spec", outputId, inputId)
			}
		}
		req, gated := hearth.RecipeRequirements[outputId]
		if !gated || req.SkillId != "cooking" || req.MinLevel > skills.MaxSkillLevel("cooking") {
			t.Errorf("recipe %d requirement = %+v, gated=%v", outputId, req, gated)
		}
	}

	// Cook through the real command with every recipe's ingredients present.
	hearth.Items = nil
	for _, inputId := range []int{29, 29, 30018} {
		hearth.AddItem(items.New(inputId))
	}
	room.Containers = map[string]rooms.Container{"hearth": hearth}

	user := users.NewUserRecord(4242, 1)
	user.Character.Name = "Cook"
	user.Character.Skills = map[string]int{"cooking": 3}
	if _, err := usercommands.Use("hearth", user, room, 0); err != nil {
		t.Fatal(err)
	}

	got := room.Containers["hearth"].Items
	if len(got) != 1 || got[0].ItemId != 30019 {
		t.Fatalf("hearth contents after cooking = %+v, want one hunter's stew", got)
	}
}
