package company

import (
	"strings"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/uuid"
	"github.com/stretchr/testify/assert"
)

type exactMealCargo struct {
	*fakeCargo
	user *users.UserRecord
}

func (c *exactMealCargo) ConsumeCargoItemUse(_ int, itm items.Item) error {
	if c.err != nil {
		return c.err
	}
	for _, current := range c.user.Character.Items {
		if current.Equals(itm) {
			c.user.Character.UseItem(current)
			return nil
		}
	}
	return encumbrance.ErrInsufficientCargo
}

func TestSharedMealUsesExactLarderOnceAndRefusesFailedSave(t *testing.T) {
	m, prov, user := mealSetup(t)
	user.Character.CompanyCargo = true
	meal := spec(t, items.ItemSpec{ItemId: 989298, Name: "shared jerky", Uses: 1, Subtype: items.Edible, Nutrition: 90})
	meal.UUID = uuid.New(items.UUIDItem)
	user.Character.Items = []items.Item{meal}
	cargo := &exactMealCargo{fakeCargo: &fakeCargo{err: assert.AnError}, user: user}
	encumbrance.SetProvider(cargo)
	t.Cleanup(func() { encumbrance.SetProvider(nil) })
	assert.Contains(t, m.mealView(user, nil, mealEat), "couldn't get at")
	assert.Empty(t, prov.fed)
	assert.Len(t, user.Character.Items, 1)
	cargo.err = nil
	m.mealView(user, nil, mealEat)
	assert.Len(t, prov.fed, 1, "only one physical serving")
	assert.Empty(t, user.Character.Items)
}

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

type fakeProvisioner struct{ fed []string }

func (f *fakeProvisioner) Provision(_ int, selector string, _ survival.Benefit) (survival.ProvisionResult, error) {
	f.fed = append(f.fed, selector)
	return survival.ProvisionResult{Name: selector, Needs: survival.Needs{Hunger: 100, Thirst: 100}}, nil
}
func (f *fakeProvisioner) IsMemberSelector(int, string) bool { return true }

type fakeNeeds struct{ needs []survival.MemberNeeds }

func (f fakeNeeds) ApplyCompanyExertion(int, string, survival.Exertion) ([]survival.ExertionResult, error) {
	return nil, nil
}
func (f fakeNeeds) ApplyCompanyRestRecovery(int, string, int) ([]survival.ExertionResult, error) {
	return nil, nil
}
func (f fakeNeeds) CompanyNeeds(int) []survival.MemberNeeds { return f.needs }

// mealSetup is a leader with companions #1 (out, beside them) and #2 (out,
// elsewhere), everyone hungry, and a fake survival service.
func mealSetup(t *testing.T) (*CompanyModule, *fakeProvisioner, *users.UserRecord) {
	t.Helper()
	runtime := &fakeRuntime{live: map[int]bool{101: true, 102: true}, away: map[int]bool{102: true}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}, {ID: 2, MobTemplateID: 58}}},
	}}, runtime)
	module.instances = map[int]map[int]int{7: {1: 101, 2: 102}}
	prov := &fakeProvisioner{}
	survival.SetProvisioner(prov)
	survival.SetCompanyService(fakeNeeds{needs: []survival.MemberNeeds{
		member(you, "Dain", 10, 100), member(tamsin, "Tamsin", 10, 100), member(oswin, "Oswin", 10, 100),
	}})
	t.Cleanup(func() {
		survival.SetProvisioner(nil)
		survival.SetCompanyService(nil)
	})
	user := users.NewUserRecord(7, 1)
	user.Character.Name = "Dain"
	return module, prov, user
}

// TestCompanionsWithLeader (32f review): out, living, the company's, and
// in the leader's room.
func TestCompanionsWithLeader(t *testing.T) {
	module, _, _ := mealSetup(t)
	assert.Equal(t, []int{1}, module.CompanionsWithLeader(7))
}

// TestMealFeedsOnlyThosePresent (32f review finding 6): a companion away
// from the leader isn't fed from the company's food.
func TestMealFeedsOnlyThosePresent(t *testing.T) {
	module, prov, user := mealSetup(t)
	useCargo(t, &fakeCargo{stacks: []encumbrance.CargoStack{{ItemId: 989201, Count: 5}}})
	spec(t, items.ItemSpec{ItemId: 989201, Name: "jerky", Subtype: items.Edible, Nutrition: 90})

	messages := captureCompanyMessages(t)
	out := module.mealView(user, rooms.NewEmptyRoom(), mealEat)
	assert.ElementsMatch(t, []string{"", "#1"}, prov.fed, "the leader and #1, not #2")
	assert.NotContains(t, out, "Oswin")
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*messages, "\n"), "Dain</ansi>'s company eats.", "the room sees one line")
}

// TestMealSpendsBeforeFeeding (32f review finding 7): food that can't be
// spent feeds no one.
func TestMealSpendsBeforeFeeding(t *testing.T) {
	module, prov, user := mealSetup(t)
	useCargo(t, &fakeCargo{stacks: []encumbrance.CargoStack{{ItemId: 989201, Count: 5}}, err: assert.AnError})
	spec(t, items.ItemSpec{ItemId: 989201, Name: "jerky", Subtype: items.Edible, Nutrition: 90})

	out := module.mealView(user, nil, mealEat)
	assert.Empty(t, prov.fed)
	assert.Contains(t, out, "couldn't get at the jerky")
}

// TestUseCompanionItemLiveRecordsGear (32f review finding 7): a live
// companion's meal updates its record at once.
func TestUseCompanionItemLiveRecordsGear(t *testing.T) {
	skin := spec(t, items.ItemSpec{ItemId: 989102, Name: "waterskin", Uses: 5, Subtype: items.Drinkable, Hydration: 40})
	skin.Uses = 2
	skin.UUID = uuid.New(items.UUIDItem)
	state := domain.MemberState{Level: 1, Items: []items.Item{skin}}
	runtime := &fakeRuntime{live: map[int]bool{101: true}, liveState: map[int]domain.MemberState{101: state.Clone()}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58, State: &state}}},
	}}, runtime)
	module.instances = map[int]map[int]int{7: {1: 101}}

	assert.True(t, module.useCompanionItem(7, 1, skin))
	record, _ := module.registry.Get(7)
	assert.Equal(t, 1, record.Companions[0].State.Items[0].Uses, "the record follows the live mob")
}
