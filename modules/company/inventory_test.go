package company

import (
	"strings"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mount"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCargo struct {
	load     encumbrance.Load
	stacks   []encumbrance.CargoStack
	consumed []int
	err      error
}

func (f *fakeCargo) CurrentLoad(int) (encumbrance.Load, bool) { return f.load, true }
func (f *fakeCargo) CargoContents(int) []encumbrance.CargoStack {
	return append([]encumbrance.CargoStack(nil), f.stacks...)
}
func (f *fakeCargo) ConsumeCargoUse(_ int, itemId int) error {
	if f.err != nil {
		return f.err
	}
	for i, s := range f.stacks {
		if s.ItemId == itemId {
			f.consumed = append(f.consumed, itemId)
			if f.stacks[i].Count--; f.stacks[i].Count == 0 {
				f.stacks = append(f.stacks[:i], f.stacks[i+1:]...)
			}
			return nil
		}
	}
	return encumbrance.ErrInsufficientCargo
}

type fakeHerd struct{ horses []mount.HorseView }

func (f fakeHerd) CapacityBonusGrams(int) int { return 0 }
func (f fakeHerd) Herd(int) []mount.HorseView { return f.horses }

func useCargo(t *testing.T, c *fakeCargo) {
	t.Helper()
	encumbrance.SetProvider(c)
	t.Cleanup(func() { encumbrance.SetProvider(nil) })
}

func spec(t *testing.T, s items.ItemSpec) items.Item {
	t.Helper()
	items.SetTestItemSpec(&s)
	t.Cleanup(func() { items.RemoveTestItemSpec(s.ItemId) })
	return items.Item{ItemId: s.ItemId, Uses: s.Uses}
}

// Phase 32f: one screen for everything the company carries, through the
// real `company inventory` command.
func TestCompanyInventory(t *testing.T) {
	messages := captureCompanyMessages(t)
	sword := spec(t, items.ItemSpec{ItemId: 989001, Name: "iron sword", Weight: 1500, Type: items.Weapon})
	satchel := spec(t, items.ItemSpec{ItemId: 989002, Name: "satchel", Weight: 600, CarryBonus: 5000})
	water := spec(t, items.ItemSpec{ItemId: 989003, Name: "waterskin", Weight: 1000, Uses: 5})
	meat := spec(t, items.ItemSpec{ItemId: 989004, Name: "seared meat", Weight: 300})
	half := water
	half.Uses = 3

	useCargo(t, &fakeCargo{
		load:   encumbrance.Load{PersonalGrams: 4000, CapacityGrams: 145000, MountCapacityGrams: 100000},
		stacks: []encumbrance.CargoStack{{ItemId: 989004, Count: 6}, {ItemId: 989003, Count: 1, Uses: 2}},
	})
	mount.SetProvider(fakeHerd{horses: []mount.HorseView{
		{ID: 1, Name: "pack horse", Kind: mount.KindPack, Saddle: "pack saddle", CapacityGrams: 100000},
		{ID: 2, Name: "riding horse", Kind: mount.KindRiding, Saddle: "riding saddle", CapacityGrams: 10000},
	}})
	t.Cleanup(func() { mount.SetProvider(nil) })

	carried := domain.MemberState{Level: 1, Items: []items.Item{satchel}}
	carried.Equipment.Weapon = sword
	fallen := domain.MemberState{Level: 1, Items: []items.Item{meat}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{
			{ID: 1, MobTemplateID: 58, Name: "Maren Vale", State: &carried},
			{ID: 2, MobTemplateID: 58, State: &fallen, Death: &domain.CompanionDeath{OpID: "x", Remaining: 60}},
		}},
	}}, &fakeRuntime{})
	user := users.NewUserRecord(7, 1)
	user.Character.Name = "Dain"
	user.Character.Items = []items.Item{half, meat, meat}

	handled, err := module.userCommand("inv", user, nil, 0)
	require.NoError(t, err)
	assert.True(t, handled)
	events.ProcessEvents()
	out := strings.Join(*messages, "\n")

	assert.Contains(t, out, "Company load: 4.0 kg / 145.0 kg (3%), of which horses 100.0 kg.")
	assert.Contains(t, out, "Dain (you)")
	assert.Contains(t, out, "#1 Maren Vale", "a generated recruit by its own name (32f review finding 3)")
	assert.Contains(t, out, "Carrying: waterskin (3 of 5), seared meat x2")
	assert.Contains(t, out, "pack: satchel (+5.0 kg)")
	assert.Contains(t, out, "Wearing: iron sword")
	assert.Contains(t, out, "2.1 kg", "the companion's gear weight")
	assert.Contains(t, out, "fallen; their gear is with the body")
	assert.Contains(t, out, "Horses: #1 pack horse (pack saddle, +100.0 kg); #2 riding horse (riding saddle, carries a rider)")
	assert.Contains(t, out, "Cargo (2.8 kg): seared meat x6, waterskin (2 of 5)")
}

func TestCompanyInventoryAlone(t *testing.T) {
	messages := captureCompanyMessages(t)
	module := newTestModule(*domain.NewRegistry(), &fakeRuntime{})
	user := users.NewUserRecord(7, 1)
	user.Character.Name = "Dain"

	_, err := module.userCommand("inventory", user, nil, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	out := strings.Join(*messages, "\n")
	assert.Contains(t, out, "Carrying: nothing")
	assert.Contains(t, out, "Horses: none")
	assert.Contains(t, out, "Cargo: empty")
}

// TestCompanyInventoryData (Phase 32g): the Inventory tab's data, read
// only: an out companion from its live mob (its record untouched), a
// stored one from its record, a fallen one with nothing, in ID order.
func TestCompanyInventoryData(t *testing.T) {
	spec(t, items.ItemSpec{ItemId: 989011, Name: "spear", Weight: 2000, Type: items.Weapon})
	spear := items.New(989011)
	recorded := domain.MemberState{Level: 1}
	stored := domain.MemberState{Level: 1, Items: []items.Item{spear}}
	live := domain.MemberState{Level: 1}
	live.Equipment.Weapon = spear
	runtime := &fakeRuntime{live: map[int]bool{501: true}, liveState: map[int]domain.MemberState{501: live}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{
			{ID: 1, MobTemplateID: 58, Name: "Maren", State: &recorded},
			{ID: 2, MobTemplateID: 58, Name: "Oswin", State: &stored},
			{ID: 3, MobTemplateID: 58, Name: "Ysolde", State: &stored, Death: &domain.CompanionDeath{OpID: "x", Remaining: 60}},
		}},
	}}, runtime)
	module.instances = map[int]map[int]int{7: {1: 501}}

	members, ok := module.CompanyInventory(7)
	require.True(t, ok)
	require.Len(t, members, 3)
	assert.Equal(t, "Maren", members[0].Name)
	require.Len(t, members[0].Worn, 1, "the live mob's gear")
	assert.Equal(t, "spear", members[0].Worn[0].Name)
	after, _ := module.registry.Get(7)
	assert.Nil(t, after.Companions[0].State.Equipment.Weapon.Spec, "reading never writes the record")
	assert.Zero(t, after.Companions[0].State.Equipment.Weapon.ItemId)
	require.Len(t, members[1].Carried, 1, "the stored record")
	assert.True(t, members[2].Fallen)
	assert.Empty(t, members[2].Carried, "the fallen's gear is with the body")

	none, ok := module.CompanyInventory(8)
	assert.True(t, ok)
	assert.Empty(t, none)
}

// TestCompanyInventoryDataReview (32g review findings 5 and 9): a
// companion's items carry no reference (none takes a command, and a
// template's would change every read), and one charmed away shows no
// gear, as the text view says.
func TestCompanyInventoryDataReview(t *testing.T) {
	spec(t, items.ItemSpec{ItemId: 989012, Name: "spear", Weight: 2000, Type: items.Weapon})
	stored := domain.MemberState{Level: 1, Items: []items.Item{items.New(989012)}}
	runtime := &fakeRuntime{live: map[int]bool{501: true}, stolen: map[int]bool{501: true}, liveState: map[int]domain.MemberState{501: stored}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{
			{ID: 1, MobTemplateID: 58, Name: "Maren", State: &stored},
			{ID: 2, MobTemplateID: 58, Name: "Oswin", State: &stored},
		}},
	}}, runtime)
	module.instances = map[int]map[int]int{7: {1: 501}}

	members, ok := module.CompanyInventory(7)
	require.True(t, ok)
	require.Len(t, members, 2)
	assert.Empty(t, members[0].Carried, "charmed away: not the company's gear")
	require.Len(t, members[1].Carried, 1)
	assert.Empty(t, members[1].Carried[0].Ref, "no reference on a companion's item")
	again, _ := module.CompanyInventory(7)
	assert.Equal(t, members, again, "the same state reads the same")
}
