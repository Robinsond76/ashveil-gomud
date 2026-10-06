package company

import (
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 40a: company water at a source.

const skinID = 989401

func skinItem(t *testing.T, uses int) items.Item {
	t.Helper()
	itm := spec(t, items.ItemSpec{ItemId: skinID, Name: "waterskin", Subtype: items.Drinkable, Hydration: 40, Uses: 5, Refillable: "water", BuffIds: []int{34}})
	itm.UUID = uuid.New(items.UUIDItem)
	itm.Uses = uses
	return itm
}

func thirstyCompany(t *testing.T) {
	t.Helper()
	survival.SetCompanyService(fakeNeeds{needs: []survival.MemberNeeds{
		member(you, "Dain", 100, 20), member(tamsin, "Tamsin", 100, 20), member(oswin, "Oswin", 100, 20),
	}})
}

func waterRoom() *rooms.Room {
	room := rooms.NewEmptyRoom()
	room.Resources = []string{rooms.ResourceWater}
	return room
}

type keeperCargo struct {
	*fakeCargo
	withdrawn, deposited []encumbrance.CargoStack
}

func (c *keeperCargo) DepositCargo(_ int, _ string, d []encumbrance.CargoStack) error {
	c.deposited = append(c.deposited, d...)
	return nil
}
func (c *keeperCargo) WithdrawCargo(_, itemID, count int) error {
	c.withdrawn = append(c.withdrawn, encumbrance.CargoStack{ItemId: itemID, Count: count})
	return nil
}

func TestCompanyDrinkAtASourceWatersEveryoneAndSpendsNothing(t *testing.T) {
	m, prov, user := mealSetup(t)
	thirstyCompany(t)
	cargo := &fakeCargo{stacks: []encumbrance.CargoStack{{ItemId: skinID, Count: 2, Uses: 5}}}
	useCargo(t, cargo)
	skinItem(t, 5)

	out := m.mealView(user, waterRoom(), mealDrink)
	assert.ElementsMatch(t, []string{"", "#1"}, prov.fed, "the leader and #1 drink from the source")
	assert.Empty(t, cargo.consumed, "the cargo's waterskins are untouched")
	assert.Contains(t, out, "drinks from the water here")
	assert.NotContains(t, out, "still thirsty")

	// Away from a source it spends cargo as it always did.
	prov.fed = nil
	m.mealView(user, rooms.NewEmptyRoom(), mealDrink)
	assert.Len(t, prov.fed, 2)
	assert.Len(t, cargo.consumed, 2, "two waterskin uses spent without a source")
}

func TestCompanyMealAtASourceStillFeedsButWatersFromTheSource(t *testing.T) {
	m, prov, user := mealSetup(t)
	survival.SetCompanyService(fakeNeeds{needs: []survival.MemberNeeds{member(you, "Dain", 10, 20)}})
	cargo := &fakeCargo{stacks: []encumbrance.CargoStack{{ItemId: 989402, Count: 3}, {ItemId: skinID, Count: 1, Uses: 5}}}
	useCargo(t, cargo)
	spec(t, items.ItemSpec{ItemId: 989402, Name: "jerky", Subtype: items.Edible, Nutrition: 90})
	skinItem(t, 5)

	m.mealView(user, waterRoom(), mealBoth)
	assert.Equal(t, []int{989402}, cargo.consumed, "food is eaten, the waterskin is not drunk")
	assert.Len(t, prov.fed, 2, "one meal, one drink from the source")
}

func TestCompanyFillRefillsPackCompanionAndCargo(t *testing.T) {
	m, _, user := mealSetup(t)
	thirstyCompany(t)
	cargo := &keeperCargo{fakeCargo: &fakeCargo{stacks: []encumbrance.CargoStack{
		{ItemId: skinID, Count: 2, Uses: 2}, // two part-used skins
		{ItemId: skinID, Count: 1},          // a full one is left alone
	}}}
	encumbrance.SetProvider(cargo)
	t.Cleanup(func() { encumbrance.SetProvider(nil) })

	mine := skinItem(t, 1)
	user.Character.Items = []items.Item{mine}
	theirs := skinItem(t, 2)
	runtime := m.runtime.(*fakeRuntime)
	runtime.liveState = map[int]domain.MemberState{101: {Level: 1, Items: []items.Item{theirs}}}

	out := m.fillView(user, waterRoom())
	assert.Contains(t, out, "fills 4 water containers")
	assert.Equal(t, 5, user.Character.Items[0].Uses, "the leader's pack")
	assert.Equal(t, 5, runtime.liveState[101].Items[0].Uses, "the companion's pack")
	assert.Equal(t, []encumbrance.CargoStack{{ItemId: skinID, Count: 2}}, cargo.withdrawn, "part-used cargo comes out")
	assert.Equal(t, []encumbrance.CargoStack{{ItemId: skinID, Count: 2}}, cargo.deposited, "and goes back full")
}

func TestCompanyFillIsRefusedAwayFromWater(t *testing.T) {
	m, _, user := mealSetup(t)
	mine := skinItem(t, 1)
	user.Character.Items = []items.Item{mine}
	assert.Contains(t, m.fillView(user, rooms.NewEmptyRoom()), "no fresh water")
	assert.Equal(t, 1, user.Character.Items[0].Uses)
	require.Contains(t, companyUsage, "company fill")
}

// thirstTracker raises thirst by each drink's hydration, like the real
// provisioner, so repeat drinks can be counted.
type thirstTracker struct {
	thirst map[string]int
	drinks map[string]int
}

func (f *thirstTracker) Provision(_ int, selector string, b survival.Benefit) (survival.ProvisionResult, error) {
	f.thirst[selector] = min(100, f.thirst[selector]+b.Hydration)
	f.drinks[selector]++
	return survival.ProvisionResult{Name: selector, Needs: survival.Needs{Hunger: 100, Thirst: f.thirst[selector]}}, nil
}
func (f *thirstTracker) IsMemberSelector(int, string) bool { return true }

// Review fix: the source is free, so a parched member drinks until no
// longer thirsty instead of one glug per "company drink".
func TestCompanyDrinkAtASourceDrinksUntilFull(t *testing.T) {
	m, _, user := mealSetup(t)
	survival.SetCompanyService(fakeNeeds{needs: []survival.MemberNeeds{member(you, "Dain", 100, 0), member(tamsin, "Tamsin", 100, 50)}})
	tracker := &thirstTracker{thirst: map[string]int{"": 0, "#1": 50}, drinks: map[string]int{}}
	survival.SetProvisioner(tracker)

	out := m.mealView(user, waterRoom(), mealDrink)
	assert.Equal(t, 2, tracker.drinks[""], "0 -> 40 -> 80: two drinks reach full")
	assert.Equal(t, 1, tracker.drinks["#1"], "50 -> 90: one drink")
	assert.NotContains(t, out, "still thirsty")
}
