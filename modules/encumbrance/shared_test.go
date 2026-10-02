package encumbrance

import (
	"errors"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	domain "github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestSharedCargoMigrationAtomicRetryAndUses(t *testing.T) {
	u := testUser(t, 7)
	store := &fakeStore{}
	m := newTestModule(store, u)
	legacy, _ := domain.Established(7)
	legacy, _ = legacy.DepositUses(waterId, 2, 1)
	legacy = legacy.MarkApplied("forage:1")
	m.cargo[7] = legacy
	original := testItem(rockId)
	original.Enchantments = 3
	original.Sharpen(1, 6)
	u.Character.Items = []items.Item{original}
	u.Character.CargoApplied = []string{"existing"}
	m.saveUser = func(*users.UserRecord) error { return errors.New("unavailable") }
	require.Error(t, m.UnifyCargo(7))
	assert.False(t, u.Character.CompanyCargo)
	assert.Equal(t, []string{"existing"}, u.Character.CargoApplied)
	require.Len(t, u.Character.Items, 1)
	var saved []byte
	m.saveUser = func(user *users.UserRecord) error { var err error; saved, err = yaml.Marshal(user); return err }
	store.saveErr = errors.New("cleanup unavailable")
	require.NoError(t, m.UnifyCargo(7))
	require.Len(t, u.Character.Items, 2)
	assert.Equal(t, original.UUID, u.Character.Items[0].UUID)
	assert.Equal(t, uint8(3), u.Character.Items[0].Enchantments)
	assert.Equal(t, 6, u.Character.Items[0].SharpStrikes)
	assert.Equal(t, 2, u.Character.Items[1].Uses)
	// Restart while old stacks remain: the marker prevents a second import.
	var restored users.UserRecord
	require.NoError(t, yaml.Unmarshal(saved, &restored))
	*u = restored
	require.NoError(t, m.UnifyCargo(7))
	require.Len(t, u.Character.Items, 2)
	require.NoError(t, m.sharedDeposit(u, "forage:1", []domain.CargoStack{{ItemId: rockId, Count: 1}}))
	require.Len(t, u.Character.Items, 2)
	before := u.Character.Items[1]
	m.saveUser = func(*users.UserRecord) error { return errors.New("unavailable") }
	require.Error(t, m.ConsumeCargoItemUse(7, before))
	assert.Equal(t, 2, u.Character.Items[1].Uses)
	m.saveUser = func(*users.UserRecord) error { return nil }
	require.NoError(t, m.ConsumeCargoItemUse(7, before))
	assert.Equal(t, 1, u.Character.Items[1].Uses)
	require.NoError(t, m.ConsumeCargoItemUse(7, before))
	require.Len(t, u.Character.Items, 1)
}

func TestSharedTradeMeasuresLossOfCountedPackCapacity(t *testing.T) {
	u := testUser(t, 7)
	u.Character.CompanyCargo = true
	m := newTestModule(&fakeStore{}, u)
	m.companionCarry = func(int) []company.MemberCarry { return []company.MemberCarry{{}} }
	packSpec := items.ItemSpec{ItemId: 989699, Name: "trade pack", Weight: 500, CarryBonus: 5000}
	items.SetTestItemSpec(&packSpec)
	t.Cleanup(func() { items.RemoveTestItemSpec(packSpec.ItemId) })
	pack, other := items.New(packSpec.ItemId), items.New(packSpec.ItemId)
	u.Character.Items = []items.Item{pack, other}
	grams, ok := m.CargoExchangeGrams(7, []items.Item{pack}, []items.Item{testItem(rockId)})
	assert.True(t, ok)
	assert.Zero(t, grams, "loose packs provide no capacity; trade measures cargo weight only")
	u.Character.Items = append(u.Character.Items, items.New(packSpec.ItemId))
	grams, _ = m.CargoExchangeGrams(7, []items.Item{pack}, []items.Item{testItem(rockId)})
	assert.Zero(t, grams, "a spare pack replaces the removed physical pack")
}

func TestUnknownLegacyCargoDoesNotPartlyMigrate(t *testing.T) {
	u := testUser(t, 7)
	m := newTestModule(&fakeStore{}, u)
	c, _ := domain.Established(7)
	c, _ = c.Deposit(rockId, 1)
	c, _ = c.Deposit(9876543, 1)
	m.cargo[7] = c
	require.Error(t, m.UnifyCargo(7))
	assert.Empty(t, u.Character.Items)
	assert.False(t, u.Character.CargoMigrated)
}

func TestSharedCargoWeightAndPacksCountOnce(t *testing.T) {
	u := testUser(t, 7)
	specs := []items.ItemSpec{{ItemId: 989610, Name: "test sword", Weight: 2000}, {ItemId: 989611, Name: "test pack", Weight: 500, CarryBonus: 5000}}
	for i := range specs {
		s := specs[i]
		items.SetTestItemSpec(&s)
		t.Cleanup(func() { items.RemoveTestItemSpec(s.ItemId) })
	}
	m := newTestModule(&fakeStore{}, u)
	m.strengthGrams = 500
	m.companionCarry = func(int) []company.MemberCarry { return []company.MemberCarry{{Strength: 2}} }
	u.Character.CompanyCargo = true
	u.Character.Equipment.Weapon = items.New(989610)
	u.Character.Items = []items.Item{items.New(989611), items.New(989611), items.New(989611)}
	load, ok := m.CurrentLoad(7)
	require.True(t, ok)
	assert.Zero(t, load.PersonalGrams)
	assert.Equal(t, 1500, load.CargoGrams)
	assert.Equal(t, 2000, u.Character.PersonalGrams(), "cargo is not the leader's burden")
	assert.Zero(t, load.CapacityGrams, "only assigned packs add capacity")
}

func TestSharedRecipeReplacesInstancesAtomically(t *testing.T) {
	u := testUser(t, 7)
	u.Character.CompanyCargo = true
	m := newTestModule(&fakeStore{}, u)
	input := testItem(waterId)
	input.Uses = 2
	input.Enchantments = 2
	output := testItem(rockId)
	u.Character.Items = []items.Item{input}
	m.saveUser = func(*users.UserRecord) error { return errors.New("save unavailable") }
	require.Error(t, m.TransformCargo(7, []items.Item{input}, []items.Item{output}))
	require.Len(t, u.Character.Items, 1)
	assert.Equal(t, input.UUID, u.Character.Items[0].UUID)
	assert.Equal(t, 2, u.Character.Items[0].Uses)
	m.saveUser = func(*users.UserRecord) error { return nil }
	require.NoError(t, m.TransformCargo(7, []items.Item{input}, []items.Item{output}))
	require.Len(t, u.Character.Items, 1)
	assert.Equal(t, output.UUID, u.Character.Items[0].UUID)
	require.Error(t, m.TransformCargo(7, []items.Item{input}, []items.Item{output}))
	require.Len(t, u.Character.Items, 1)
}
