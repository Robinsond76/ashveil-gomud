package main

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/cookbook"
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
	if p := skills.GetProfessionSpec("cook"); p == nil || len(p.Skills) == 0 || p.Skills[0] != "cooking" {
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
	cookbook.Learn(user.Character, 30019) // Phase 56: the stew is learned, not common knowledge
	if _, err := usercommands.Use("hearth", user, room, 0); err != nil {
		t.Fatal(err)
	}

	got := room.Containers["hearth"].Items
	if len(got) != 1 || got[0].ItemId != 30019 {
		t.Fatalf("hearth contents after cooking = %+v, want one hunter's stew", got)
	}
}

// TestShippedRecipeDiscoveryContent (Phase 56): the real gathered foods
// are ingredients, a cooked meal is not, every recipe page teaches a shipped
// hearth dish, and nothing the cook makes can be sold back.
func TestShippedRecipeDiscoveryContent(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	if err := configs.ReloadConfig(); err != nil {
		t.Fatal(err)
	}
	loadAllDataFiles(false)

	for _, id := range []int{29, 30018, 43} {
		if !cookbook.IsIngredient(items.GetItemSpec(id)) {
			t.Errorf("item %d should be an ingredient", id)
		}
	}
	for _, id := range []int{30019, 30020, 30021, 30024, items.MakeshiftMealItemId, 30063, 30064} {
		if cookbook.IsIngredient(items.GetItemSpec(id)) {
			t.Errorf("item %d must not be an ingredient", id)
		}
	}

	room := rooms.LoadRoomTemplate(waymarkInnRoomId)
	hearth := room.Containers["hearth"]
	meal := items.GetItemSpec(items.MakeshiftMealItemId)
	if meal == nil || meal.Type != items.Food || meal.Subtype != items.Edible || meal.Nutrition < 1 || meal.Meal != "" {
		t.Fatalf("makeshift meal = %+v, want plain edible food with no buff", meal)
	}
	if best := items.GetItemSpec(30021); meal.Nutrition >= best.Nutrition {
		t.Errorf("makeshift meal nutrition %d should be below seared game meat %d", meal.Nutrition, best.Nutrition)
	}
	m := items.New(items.MakeshiftMealItemId)
	if !m.IsSpecialForSale() {
		t.Error("a makeshift meal must never be bought back")
	}

	for _, pageId := range []int{30063, 30064} {
		page := items.GetItemSpec(pageId)
		if page == nil || page.Recipe == 0 {
			t.Fatalf("page %d = %+v", pageId, page)
		}
		if _, ok := hearth.Recipes[page.Recipe]; !ok {
			t.Errorf("page %d teaches %d, which the Waymark hearth does not make", pageId, page.Recipe)
		}
		if req := hearth.RecipeRequirements[page.Recipe]; req.MinLevel <= cookbook.BasicLevel {
			t.Errorf("page %d teaches a common dish", pageId)
		}
		p := items.New(pageId)
		if !p.IsSpecialForSale() {
			t.Errorf("page %d must never be bought back", pageId)
		}
	}

	// Reading a page through the real use command teaches its dish once.
	user := users.NewUserRecord(4243, 1)
	user.Character.Name = "Reader"
	user.Character.StoreItem(items.New(30064))
	if _, err := usercommands.Use("page", user, room, 0); err != nil {
		t.Fatal(err)
	}
	if got := cookbook.Learned(user.Character); len(got) != 1 || got[0] != 30019 {
		t.Fatalf("book after reading the stew page = %v, want [30019]", got)
	}
	if _, found := user.Character.FindInBackpack("page"); found {
		t.Error("the page should be used up")
	}
}
