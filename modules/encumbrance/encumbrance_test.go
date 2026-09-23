package encumbrance

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mount"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "low", "", false)
	os.Exit(m.Run())
}

type fakeStore struct {
	saved   Registry
	loadErr error
	saveErr error
}

func (f *fakeStore) Load(registry *Registry) error {
	if f.loadErr != nil {
		return f.loadErr
	}
	*registry = f.saved.Clone()
	return nil
}

func (f *fakeStore) Save(registry Registry) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = registry.Clone()
	return nil
}

// rockId weighs 500g; featherId weighs 0g (unweighted).
const rockId = 100
const featherId = 200

func fakeItemSpec(itemId int) (items.ItemSpec, bool) {
	switch itemId {
	case rockId:
		return items.ItemSpec{ItemId: rockId, Name: "rock", Weight: 500}, true
	case featherId:
		return items.ItemSpec{ItemId: featherId, Name: "feather", Weight: 0}, true
	}
	return items.ItemSpec{}, false
}

// testItem builds a real Item carrying its spec as a per-instance override
// (items.Item.Spec), so tests don't depend on the live global item
// registry, matching internal/usercommands/eat_test.go's established
// pattern.
func testItem(itemId int) items.Item {
	spec, _ := fakeItemSpec(itemId)
	return items.Item{ItemId: itemId, UUID: uuid.New(items.UUIDItem), Spec: &spec}
}

func testUser(t *testing.T, userId int) *users.UserRecord {
	t.Helper()
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(userId, 1)
	users.SetTestUser(user)
	return user
}

func newTestModule(store Store, user *users.UserRecord) *EncumbranceModule {
	return &EncumbranceModule{
		store:         store,
		itemSpec:      fakeItemSpec,
		userLookup:    func(userId int) *users.UserRecord { return user },
		capacityGrams: 10000,
		cargo:         map[int]encumbrance.Cargo{},
	}
}

func TestCurrentLoadUntrackedWithoutCapacity(t *testing.T) {
	user := testUser(t, 7)
	module := newTestModule(&fakeStore{}, user)
	module.capacityGrams = 0

	_, ok := module.CurrentLoad(7)
	assert.False(t, ok, "an unconfigured capacity must be untracked, not a guessed default")
}

func TestCurrentLoadComputesPersonalAndCargoWeight(t *testing.T) {
	user := testUser(t, 7)
	user.Character.Items = []items.Item{testItem(rockId), testItem(featherId)}
	module := newTestModule(&fakeStore{}, user)
	module.cargo[7] = encumbrance.Cargo{LeaderUserID: 7, Stacks: []encumbrance.CargoStack{{ItemId: rockId, Count: 2}}}

	load, ok := module.CurrentLoad(7)
	require.True(t, ok)
	assert.Equal(t, 500, load.PersonalGrams, "rock weighs 500g, feather weighs 0g")
	assert.Equal(t, 1000, load.CargoGrams, "two rocks in cargo weigh 1000g")
	assert.Equal(t, 10000, load.CapacityGrams)
}

func TestPutMovesItemFromBackpackToCargoWithoutChangingTotalWeight(t *testing.T) {
	user := testUser(t, 7)
	user.Character.Items = []items.Item{testItem(rockId)}
	store := &fakeStore{}
	module := newTestModule(store, user)

	before, _ := module.CurrentLoad(7)

	msg := module.put(user, "rock")
	assert.Contains(t, msg, "stow")
	assert.Empty(t, user.Character.Items, "the rock left the backpack")
	require.Contains(t, module.cargo, 7)
	assert.Equal(t, 1, module.cargo[7].Stacks[0].Count)
	assert.Contains(t, store.saved.Cargo, 7, "put must persist")

	after, _ := module.CurrentLoad(7)
	assert.Equal(t, before.TotalGrams(), after.TotalGrams(), "moving between backpack and cargo must not change total party weight")
}

func TestPutRefusesWhenItemNotInBackpack(t *testing.T) {
	user := testUser(t, 7)
	store := &fakeStore{}
	module := newTestModule(store, user)

	msg := module.put(user, "rock")
	assert.Contains(t, msg, "don't have")
	assert.Empty(t, module.cargo)
	assert.Empty(t, store.saved.Cargo)
}

func TestTakeMovesItemFromCargoToBackpack(t *testing.T) {
	user := testUser(t, 7)
	store := &fakeStore{}
	module := newTestModule(store, user)
	module.cargo[7] = encumbrance.Cargo{LeaderUserID: 7, Stacks: []encumbrance.CargoStack{{ItemId: rockId, Count: 2}}}

	msg := module.take(user, "rock")
	assert.Contains(t, msg, "take")
	require.Len(t, user.Character.Items, 1)
	assert.Equal(t, rockId, user.Character.Items[0].ItemId)
	assert.Equal(t, 1, module.cargo[7].Stacks[0].Count, "one rock remains in cargo")
	assert.Equal(t, 1, store.saved.Cargo[7].Stacks[0].Count)
}

func TestTakeRefusesWhenCargoEmptyOrNoMatch(t *testing.T) {
	user := testUser(t, 7)
	module := newTestModule(&fakeStore{}, user)

	msg := module.take(user, "rock")
	assert.Contains(t, msg, "empty")

	module.cargo[7] = encumbrance.Cargo{LeaderUserID: 7, Stacks: []encumbrance.CargoStack{{ItemId: rockId, Count: 1}}}
	msg = module.take(user, "feather")
	assert.Contains(t, msg, "no")
}

func TestPutRollsBackOnFailedSave(t *testing.T) {
	user := testUser(t, 7)
	user.Character.Items = []items.Item{testItem(rockId)}
	store := &fakeStore{saveErr: assert.AnError}
	module := newTestModule(store, user)

	msg := module.put(user, "rock")
	assert.Contains(t, msg, assert.AnError.Error())
	assert.NotEmpty(t, user.Character.Items, "a failed save must not remove the item from the backpack")
	assert.Empty(t, module.cargo, "a failed save must not leave an in-memory cargo entry")
}

func TestTakeRollsBackOnFailedSave(t *testing.T) {
	user := testUser(t, 7)
	store := &fakeStore{saveErr: assert.AnError}
	module := newTestModule(store, user)
	module.cargo[7] = encumbrance.Cargo{LeaderUserID: 7, Stacks: []encumbrance.CargoStack{{ItemId: rockId, Count: 1}}}

	msg := module.take(user, "rock")
	assert.Contains(t, msg, assert.AnError.Error())
	assert.Empty(t, user.Character.Items, "a failed save must not add the item to the backpack")
	assert.Equal(t, 1, module.cargo[7].Stacks[0].Count, "a failed save must roll cargo back")
}

func TestStatusRendersLoadAndCargo(t *testing.T) {
	user := testUser(t, 7)
	user.Character.Items = []items.Item{testItem(rockId)}
	module := newTestModule(&fakeStore{}, user)
	module.capacityGrams = 1000
	module.bands = []encumbrance.LoadBand{{MinRatio: 0.4, TravelDurationPct: 110, FatiguePct: 108}}
	module.cargo[7] = encumbrance.Cargo{LeaderUserID: 7, Stacks: []encumbrance.CargoStack{{ItemId: rockId, Count: 1}}}

	status := module.status(7)
	assert.Contains(t, status, "1.0 kg / 1.0 kg (100%)")
	assert.Contains(t, status, "Travel duration modifier")
	assert.Contains(t, status, "rock x1")
}

func TestParseConfigRejectsMalformedBands(t *testing.T) {
	bandsRaw := []any{
		map[string]any{"minratio": 0.75, "traveldurationpct": 110, "fatiguepct": 108},
		map[string]any{"minratio": -1, "traveldurationpct": 999, "fatiguepct": 999},
		map[string]any{"minratio": 0.5, "traveldurationpct": 105, "fatiguepct": 103},
	}

	capacityGrams, bands := parseConfig(200.0, bandsRaw)

	assert.Equal(t, 200000, capacityGrams)
	require.Len(t, bands, 2, "the negative-ratio band is rejected")
	assert.Equal(t, 0.5, bands[0].MinRatio, "bands must be sorted ascending")
	assert.Equal(t, 0.75, bands[1].MinRatio)
}

func TestParseConfigHandlesMissingBands(t *testing.T) {
	capacityGrams, bands := parseConfig(150.0, nil)
	assert.Equal(t, 150000, capacityGrams)
	assert.Empty(t, bands)
}

type fakeMountProvider struct{ bonusGrams int }

func (f fakeMountProvider) CapacityBonusGrams(_ int) int { return f.bonusGrams }

func TestCurrentLoadAddsMountCapacityBonus(t *testing.T) {
	user := testUser(t, 7)
	module := newTestModule(&fakeStore{}, user)
	module.capacityGrams = 10000

	mount.SetProvider(fakeMountProvider{bonusGrams: 50000})
	t.Cleanup(func() { mount.SetProvider(nil) })

	load, ok := module.CurrentLoad(7)
	require.True(t, ok)
	assert.Equal(t, 60000, load.CapacityGrams, "the mount's bonus must add to the configured base capacity")
}

func TestCurrentLoadWithoutMountProviderIsUnaffected(t *testing.T) {
	user := testUser(t, 7)
	module := newTestModule(&fakeStore{}, user)
	module.capacityGrams = 10000
	mount.SetProvider(nil)

	load, ok := module.CurrentLoad(7)
	require.True(t, ok)
	assert.Equal(t, 10000, load.CapacityGrams)
}

// TestCurrentBandResolvesConfiguredBandForRealLoad goes through the
// registered internal/encumbrance seam, as walking and travel do.
func TestCurrentBandResolvesConfiguredBandForRealLoad(t *testing.T) {
	user := testUser(t, 7)
	module := newTestModule(&fakeStore{}, user)
	module.capacityGrams = 1000
	module.bands = []encumbrance.LoadBand{
		{MinRatio: 0.75, TravelDurationPct: 110, FatiguePct: 108},
		{MinRatio: 0.9, TravelDurationPct: 125, FatiguePct: 115},
	}
	encumbrance.SetProvider(module)
	t.Cleanup(func() { encumbrance.SetProvider(nil) })

	band, ok := encumbrance.CurrentBand(7)
	require.True(t, ok)
	assert.Equal(t, encumbrance.LoadBand{TravelDurationPct: 100, FatiguePct: 100}, band, "an empty pack is unladen")

	module.cargo[7] = encumbrance.Cargo{LeaderUserID: 7, Stacks: []encumbrance.CargoStack{{ItemId: rockId, Count: 1}}}
	user.Character.Items = []items.Item{testItem(rockId)} // 1000g of 1000g
	band, ok = encumbrance.CurrentBand(7)
	require.True(t, ok)
	assert.Equal(t, 115, band.FatiguePct)
	assert.Equal(t, 125, band.TravelDurationPct)

	module.capacityGrams = 0
	band, ok = encumbrance.CurrentBand(7)
	assert.False(t, ok, "an untracked load is neutral")
	assert.Equal(t, 100, band.FatiguePct)
}

func TestCurrentBandNeutralWithoutProvider(t *testing.T) {
	encumbrance.SetProvider(nil)
	band, ok := encumbrance.CurrentBand(7)
	assert.False(t, ok)
	assert.Equal(t, encumbrance.LoadBand{TravelDurationPct: 100, FatiguePct: 100}, band)
}
