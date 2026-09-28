package company

import (
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/uuid"
	"github.com/stretchr/testify/assert"
)

func member(key survival.MemberKey, name string, hunger, thirst int) survival.MemberNeeds {
	return survival.MemberNeeds{Key: key, Name: name, Needs: survival.Needs{Hunger: hunger, Thirst: thirst, Fatigue: 100}}
}

func food(source larderSource, owner survival.MemberKey, name string, nutrition, hydration, uses int) larderItem {
	return larderItem{Source: source, Owner: owner, Name: name, Edible: true, Nutrition: nutrition, Hydration: hydration, Uses: uses}
}

func water(source larderSource, owner survival.MemberKey, name string, hydration, uses int) larderItem {
	return larderItem{Source: source, Owner: owner, Name: name, Drinkable: true, Hydration: hydration, Uses: uses}
}

func eaten(plan mealPlan, larder []larderItem) map[string]string {
	out := map[string]string{}
	for _, s := range plan.Steps {
		out[s.Member.Name] += larder[s.Food].Name + ";"
	}
	return out
}

var (
	you    = survival.LeaderMemberKey
	tamsin = survival.CompanionMemberKey(1)
	oswin  = survival.CompanionMemberKey(2)
)

// Phase 32f: the meal planner, pure.
func TestPlanMealOrderAndSkip(t *testing.T) {
	needs := []survival.MemberNeeds{member(you, "Dain", 60, 100), member(tamsin, "Tamsin", 20, 100), member(oswin, "Oswin", 90, 100)}
	larder := []larderItem{food(fromCargo, "", "meat", 30, 0, 1)}
	plan := planMeal(needs, larder, mealEat)
	assert.Equal(t, map[string]string{"Tamsin": "meat;"}, eaten(plan, larder), "the most in need first; the one meat")
	assert.Equal(t, []string{"Dain"}, plan.Hungry, "Oswin is well fed and skipped")
}

func TestPlanMealSourceOrder(t *testing.T) {
	needs := []survival.MemberNeeds{member(you, "Dain", 40, 100), member(tamsin, "Tamsin", 30, 100), member(oswin, "Oswin", 50, 100)}
	larder := []larderItem{
		food(fromLeaderPack, you, "cheese", 30, 0, 3),
		food(fromOwnPack, tamsin, "tamsin's bread", 30, 0, 1),
		food(fromOwnPack, oswin, "oswin's bread", 30, 0, 1),
		food(fromCargo, "", "cargo meat", 30, 0, 1),
	}
	plan := planMeal(needs, larder, mealEat)
	assert.Equal(t, map[string]string{
		"Tamsin": "cargo meat;",    // most in need: the cargo first
		"Dain":   "cheese;",        // cargo gone; the leader's own pack
		"Oswin":  "oswin's bread;", // their own pack before the leader's
	}, eaten(plan, larder))
	assert.Empty(t, plan.Hungry)
}

func TestPlanMealSmallestThatCovers(t *testing.T) {
	larder := []larderItem{
		food(fromCargo, "", "feast", 80, 0, 5),
		food(fromCargo, "", "apple", 10, 0, 5),
		food(fromCargo, "", "meat", 30, 0, 5),
	}
	plan := planMeal([]survival.MemberNeeds{member(you, "Dain", 75, 100)}, larder, mealEat)
	assert.Equal(t, map[string]string{"Dain": "meat;"}, eaten(plan, larder), "need 25: the meat covers it, the feast wastes")
	plan = planMeal([]survival.MemberNeeds{member(you, "Dain", 5, 100)}, larder, mealEat)
	assert.Equal(t, map[string]string{"Dain": "feast;"}, eaten(plan, larder), "nothing covers 95: the largest")
}

func TestPlanMealFoodThatWaters(t *testing.T) {
	larder := []larderItem{food(fromCargo, "", "stew", 30, 30, 2), water(fromCargo, "", "waterskin", 40, 5)}
	needs := []survival.MemberNeeds{member(you, "Dain", 50, 60), member(tamsin, "Tamsin", 100, 40)}
	plan := planMeal(needs, larder, mealBoth)
	assert.Equal(t, map[string]string{"Dain": "stew;", "Tamsin": "waterskin;"}, eaten(plan, larder),
		"the stew leaves Dain at 90 thirst, so only Tamsin drinks")
	assert.False(t, plan.Steps[0].Drink)
	assert.True(t, plan.Steps[1].Drink)

	plan = planMeal(needs, larder, mealDrink)
	assert.Equal(t, map[string]string{"Tamsin": "waterskin;", "Dain": "waterskin;"}, eaten(plan, larder), "drink never serves stew")
}

func TestPlanMealUsesRunOut(t *testing.T) {
	larder := []larderItem{water(fromCargo, "", "waterskin", 40, 2)}
	needs := []survival.MemberNeeds{member(you, "Dain", 100, 50), member(tamsin, "Tamsin", 100, 30), member(oswin, "Oswin", 100, 10)}
	plan := planMeal(needs, larder, mealDrink)
	assert.Len(t, plan.Steps, 2)
	assert.Equal(t, []string{"Dain"}, plan.Thirsty)
	assert.Equal(t, 2, larder[0].Uses, "the caller's larder is not changed")
}

func TestLarderEntry(t *testing.T) {
	_, ok := larderEntry(items.ItemSpec{ItemId: 1, Subtype: items.Edible, Nutrition: 20, BuffIds: []int{17}})
	assert.True(t, ok, "Well Fed is a meal's buff")
	_, ok = larderEntry(items.ItemSpec{ItemId: 2, Subtype: items.Drinkable, Hydration: 20, BuffIds: []int{34, 5}})
	assert.False(t, ok, "another buff: a choice made by hand")
	_, ok = larderEntry(items.ItemSpec{ItemId: 3, Subtype: items.Drinkable, BuffIds: []int{9}})
	assert.False(t, ok, "a potion with no water")
	_, ok = larderEntry(items.ItemSpec{ItemId: 4, Name: "rock"})
	assert.False(t, ok)
}

// A companion who isn't out eats from its record, which is saved; a failed
// save puts the item back.
func TestUseCompanionItemFromRecord(t *testing.T) {
	skin := spec(t, items.ItemSpec{ItemId: 989101, Name: "waterskin", Uses: 5, Subtype: items.Drinkable, Hydration: 40})
	skin.Uses = 2
	skin.UUID = uuid.New(items.UUIDItem)
	state := domain.MemberState{Level: 1, Items: []items.Item{skin}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58, State: &state}}},
	}}, &fakeRuntime{})
	store := module.store.(*fakeStore)

	assert.True(t, module.useCompanionItem(7, 1, skin))
	saved, _ := store.saved.Get(7)
	assert.Equal(t, 1, saved.Companions[0].State.Items[0].Uses)

	store.saveErr = assert.AnError
	assert.False(t, module.useCompanionItem(7, 1, skin))
	record, _ := module.registry.Get(7)
	assert.Equal(t, 1, record.Companions[0].State.Items[0].Uses, "rolled back")

	store.saveErr = nil
	assert.True(t, module.useCompanionItem(7, 1, skin))
	record, _ = module.registry.Get(7)
	assert.Empty(t, record.Companions[0].State.Items, "the last use empties it")
	assert.False(t, module.useCompanionItem(7, 1, skin), "gone")
}
