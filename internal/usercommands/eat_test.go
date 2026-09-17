package usercommands

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "low", "", false)
	os.Exit(m.Run())
}

type fakeProvisioner struct {
	result       survival.ProvisionResult
	err          error
	calls        int
	lastSelector string
	lastBenefit  survival.Benefit
	selectors    map[string]bool
}

func (f *fakeProvisioner) Provision(_ int, selector string, benefit survival.Benefit) (survival.ProvisionResult, error) {
	f.calls++
	f.lastSelector = selector
	f.lastBenefit = benefit
	return f.result, f.err
}

func (f *fakeProvisioner) IsMemberSelector(_ int, selector string) bool {
	return f.selectors[selector]
}

func memberSelectors(selectors ...string) map[string]bool {
	set := make(map[string]bool, len(selectors))
	for _, selector := range selectors {
		set[selector] = true
	}
	return set
}

func testRoom() *rooms.Room { return rooms.NewRoom("test") }

func userWithItem(t *testing.T, userId int, spec items.ItemSpec) *users.UserRecord {
	t.Helper()
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)

	user := users.NewUserRecord(userId, 1)
	user.Character.Name = "Tester"
	uses := spec.Uses
	if uses < 1 {
		uses = 1
	}
	user.Character.Items = append(user.Character.Items, items.Item{
		ItemId: spec.ItemId,
		UUID:   uuid.New(items.UUIDItem),
		Uses:   uses,
		Spec:   &spec,
	})
	users.SetTestUser(user)
	return user
}

func edibleSpec(name string, nutrition, hydration, uses int) items.ItemSpec {
	return items.ItemSpec{
		ItemId:     9001,
		Name:       name,
		NameSimple: name,
		Type:       items.Food,
		Subtype:    items.Edible,
		Nutrition:  nutrition,
		Hydration:  hydration,
		Uses:       uses,
	}
}

func useFakeProvisioner(t *testing.T, fake *fakeProvisioner) {
	t.Helper()
	survival.SetProvisioner(fake)
	t.Cleanup(func() { survival.SetProvisioner(nil) })
}

func TestEatProvisionsLeaderWithoutSelector(t *testing.T) {
	fake := &fakeProvisioner{result: survival.ProvisionResult{
		Member: survival.LeaderMemberKey,
		Name:   "Tester",
		Needs:  survival.Needs{Hunger: 70, Thirst: 100, Fatigue: 100},
	}}
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, edibleSpec("ration", 30, 0, 3))

	handled, err := Eat("ration", user, testRoom(), 0)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, 1, fake.calls)
	assert.Equal(t, "", fake.lastSelector)
	assert.Equal(t, survival.Benefit{Nutrition: 30}, fake.lastBenefit)
	require.Len(t, user.Character.Items, 1)
	assert.Equal(t, 2, user.Character.Items[0].Uses)
}

func TestEatForwardsCompanionSelectorForMultiWordItem(t *testing.T) {
	fake := &fakeProvisioner{result: survival.ProvisionResult{
		Member: survival.CompanionMemberKey(2),
		Name:   "Bear",
		Needs:  survival.Needs{Hunger: 60, Thirst: 100, Fatigue: 100},
		Hunger: survival.Change{Before: survival.BandLow, After: survival.BandSteady},
	}}
	fake.selectors = memberSelectors("#2")
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, edibleSpec("cheese sandwich", 35, 5, 3))

	_, err := Eat("cheese sandwich #2", user, testRoom(), 0)
	require.NoError(t, err)
	assert.Equal(t, 1, fake.calls)
	assert.Equal(t, "#2", fake.lastSelector)
	assert.Equal(t, survival.Benefit{Nutrition: 35, Hydration: 5}, fake.lastBenefit)
	assert.Equal(t, 2, user.Character.Items[0].Uses)
}

func TestEatDoesNotConsumeWhenSurvivalProvisionFails(t *testing.T) {
	fake := &fakeProvisioner{err: survival.ErrPersistenceUnavailable, selectors: memberSelectors("#2")}
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, edibleSpec("ration", 30, 0, 3))

	_, err := Eat("ration #2", user, testRoom(), 0)
	require.ErrorIs(t, err, survival.ErrPersistenceUnavailable)
	require.Len(t, user.Character.Items, 1)
	assert.Equal(t, 3, user.Character.Items[0].Uses)
	assert.False(t, user.Character.HasBuff(17))
}

func TestEatKeepsLegacyConsumptionForZeroMetadata(t *testing.T) {
	fake := &fakeProvisioner{}
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, edibleSpec("bread", 0, 0, 2))

	_, err := Eat("bread", user, testRoom(), 0)
	require.NoError(t, err)
	assert.Zero(t, fake.calls, "zero-metadata items must not call the survival provider")
	require.Len(t, user.Character.Items, 1)
	assert.Equal(t, 1, user.Character.Items[0].Uses)
}

func TestEatKeepsLegacyConsumptionForMultiWordItemWithoutSelector(t *testing.T) {
	fake := &fakeProvisioner{result: survival.ProvisionResult{Member: survival.LeaderMemberKey, Name: "Tester"}}
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, edibleSpec("cheese sandwich", 35, 0, 3))

	_, err := Eat("cheese sandwich", user, testRoom(), 0)
	require.NoError(t, err)
	assert.Equal(t, 1, fake.calls)
	assert.Equal(t, "", fake.lastSelector)
	require.Len(t, user.Character.Items, 1)
	assert.Equal(t, 2, user.Character.Items[0].Uses)
}

func TestEatRejectsNonEdibleWithoutProvisioning(t *testing.T) {
	fake := &fakeProvisioner{}
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, items.ItemSpec{
		ItemId:     9002,
		Name:       "stone",
		NameSimple: "stone",
		Type:       items.Junk,
		Subtype:    items.Mundane,
		Nutrition:  30,
		Uses:       1,
	})

	_, err := Eat("stone", user, testRoom(), 0)
	require.NoError(t, err)
	assert.Zero(t, fake.calls)
	require.Len(t, user.Character.Items, 1)
	assert.Equal(t, 1, user.Character.Items[0].Uses)
}

func TestEatReportsMissingItem(t *testing.T) {
	fake := &fakeProvisioner{}
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, edibleSpec("ration", 30, 0, 1))

	handled, err := Eat("nothing", user, testRoom(), 0)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Zero(t, fake.calls)
	require.Len(t, user.Character.Items, 1)
}

func TestEatKeepsPartialItemMatchWithValidTarget(t *testing.T) {
	fake := &fakeProvisioner{
		result:    survival.ProvisionResult{Member: survival.CompanionMemberKey(2), Name: "Bear"},
		selectors: memberSelectors("#2"),
	}
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, edibleSpec("ration pack", 20, 0, 2))

	_, err := Eat("rati #2", user, testRoom(), 0)
	require.NoError(t, err)
	assert.Equal(t, 1, fake.calls)
	assert.Equal(t, "#2", fake.lastSelector)
	assert.Equal(t, 1, user.Character.Items[0].Uses)
}

func TestEatTreatsNonMemberSuffixAsLegacyItemName(t *testing.T) {
	fake := &fakeProvisioner{
		result:    survival.ProvisionResult{Member: survival.LeaderMemberKey, Name: "Tester"},
		selectors: memberSelectors("#2"),
	}
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, edibleSpec("ration stranger", 20, 0, 2))

	_, err := Eat("ration stranger", user, testRoom(), 0)
	require.NoError(t, err)
	assert.Equal(t, 1, fake.calls, "a non-member suffix must fall back to the full-input item match")
	assert.Equal(t, "", fake.lastSelector)
	assert.Equal(t, 1, user.Character.Items[0].Uses)
}

func TestEatTreatsStaleNumericSuffixAsItemName(t *testing.T) {
	fake := &fakeProvisioner{result: survival.ProvisionResult{Member: survival.LeaderMemberKey, Name: "Tester"}}
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, edibleSpec("ration 2", 20, 0, 2))

	_, err := Eat("ration 2", user, testRoom(), 0)
	require.NoError(t, err)
	assert.Equal(t, 1, fake.calls)
	assert.Equal(t, "", fake.lastSelector, "a stale numeric suffix must not be stripped as a target")
	assert.Equal(t, 1, user.Character.Items[0].Uses)
}

func TestEatReportsMissingItemWhenSuffixIsNotAMember(t *testing.T) {
	fake := &fakeProvisioner{selectors: memberSelectors("#2")}
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, edibleSpec("ration", 20, 0, 2))

	_, err := Eat("ration stranger", user, testRoom(), 0)
	require.NoError(t, err)
	assert.Zero(t, fake.calls)
	assert.Equal(t, 2, user.Character.Items[0].Uses)
}
