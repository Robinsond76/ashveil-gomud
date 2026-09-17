package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func drinkableSpec(name string, hydration, uses int) items.ItemSpec {
	return items.ItemSpec{
		ItemId:     9003,
		Name:       name,
		NameSimple: name,
		Type:       items.Drink,
		Subtype:    items.Drinkable,
		Hydration:  hydration,
		Uses:       uses,
	}
}

func TestDrinkProvisionsHydrationForLeader(t *testing.T) {
	fake := &fakeProvisioner{result: survival.ProvisionResult{
		Member: survival.LeaderMemberKey,
		Name:   "Tester",
		Needs:  survival.Needs{Hunger: 100, Thirst: 60, Fatigue: 100},
		Thirst: survival.Change{Before: survival.BandCritical, After: survival.BandSteady},
	}}
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, drinkableSpec("waterskin", 40, 5))

	handled, err := Drink("waterskin", user, testRoom(), 0)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, 1, fake.calls)
	assert.Equal(t, "", fake.lastSelector)
	assert.Equal(t, survival.Benefit{Hydration: 40}, fake.lastBenefit)
	require.Len(t, user.Character.Items, 1)
	assert.Equal(t, 4, user.Character.Items[0].Uses)
}

func TestDrinkForwardsCompanionSelector(t *testing.T) {
	fake := &fakeProvisioner{result: survival.ProvisionResult{
		Member: survival.CompanionMemberKey(2),
		Name:   "Bear",
		Needs:  survival.Needs{Hunger: 100, Thirst: 80, Fatigue: 100},
	}}
	fake.selectors = memberSelectors("#2")
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, drinkableSpec("mug of ale", 20, 2))

	_, err := Drink("mug of ale #2", user, testRoom(), 0)
	require.NoError(t, err)
	assert.Equal(t, 1, fake.calls)
	assert.Equal(t, "#2", fake.lastSelector)
	assert.Equal(t, survival.Benefit{Hydration: 20}, fake.lastBenefit)
	assert.Equal(t, 1, user.Character.Items[0].Uses)
}

func TestDrinkDoesNotConsumeWhenSurvivalProvisionFails(t *testing.T) {
	fake := &fakeProvisioner{err: survival.ErrPersistenceUnavailable, selectors: memberSelectors("#2")}
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, drinkableSpec("waterskin", 40, 5))

	_, err := Drink("waterskin #2", user, testRoom(), 0)
	require.ErrorIs(t, err, survival.ErrPersistenceUnavailable)
	require.Len(t, user.Character.Items, 1)
	assert.Equal(t, 5, user.Character.Items[0].Uses)
}

func TestDrinkKeepsLegacyConsumptionForZeroMetadata(t *testing.T) {
	fake := &fakeProvisioner{}
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, drinkableSpec("stale water", 0, 2))

	_, err := Drink("stale water", user, testRoom(), 0)
	require.NoError(t, err)
	assert.Zero(t, fake.calls)
	require.Len(t, user.Character.Items, 1)
	assert.Equal(t, 1, user.Character.Items[0].Uses)
}

func TestDrinkKeepsPartialItemMatchWithValidTarget(t *testing.T) {
	fake := &fakeProvisioner{
		result:    survival.ProvisionResult{Member: survival.CompanionMemberKey(2), Name: "Bear"},
		selectors: memberSelectors("#2"),
	}
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, drinkableSpec("waterskin", 40, 5))

	_, err := Drink("water #2", user, testRoom(), 0)
	require.NoError(t, err)
	assert.Equal(t, 1, fake.calls)
	assert.Equal(t, "#2", fake.lastSelector)
	assert.Equal(t, 4, user.Character.Items[0].Uses)
}

func TestDrinkKeepsNumberedItemMatchWithValidTarget(t *testing.T) {
	fake := &fakeProvisioner{
		result:    survival.ProvisionResult{Member: survival.CompanionMemberKey(2), Name: "Bear"},
		selectors: memberSelectors("#2"),
	}
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, drinkableSpec("waterskin", 40, 5))
	spec := drinkableSpec("waterskin", 40, 5)
	spec.ItemId = 9005
	user.Character.Items = append(user.Character.Items, items.Item{
		ItemId: spec.ItemId,
		UUID:   uuid.New(items.UUIDItem),
		Uses:   5,
		Spec:   &spec,
	})

	_, err := Drink("waterskin#2 #2", user, testRoom(), 0)
	require.NoError(t, err)
	assert.Equal(t, 1, fake.calls)
	assert.Equal(t, "#2", fake.lastSelector)
	assert.Equal(t, 5, user.Character.Items[0].Uses, "the first waterskin must be untouched")
	assert.Equal(t, 4, user.Character.Items[1].Uses, "the numbered match must be consumed")
}

func TestDrinkTreatsStaleNumericSuffixAsItemName(t *testing.T) {
	fake := &fakeProvisioner{result: survival.ProvisionResult{Member: survival.LeaderMemberKey, Name: "Tester"}}
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, drinkableSpec("waterskin 2", 40, 5))

	_, err := Drink("waterskin 2", user, testRoom(), 0)
	require.NoError(t, err)
	assert.Equal(t, 1, fake.calls)
	assert.Equal(t, "", fake.lastSelector, "a stale numeric suffix must not be stripped as a target")
	assert.Equal(t, 4, user.Character.Items[0].Uses)
}

func TestDrinkRejectsNonDrinkableWithoutProvisioning(t *testing.T) {
	fake := &fakeProvisioner{}
	useFakeProvisioner(t, fake)
	user := userWithItem(t, 17, items.ItemSpec{
		ItemId:     9004,
		Name:       "stone",
		NameSimple: "stone",
		Type:       items.Junk,
		Subtype:    items.Mundane,
		Hydration:  40,
		Uses:       1,
	})

	_, err := Drink("stone", user, testRoom(), 0)
	require.NoError(t, err)
	assert.Zero(t, fake.calls)
	require.Len(t, user.Character.Items, 1)
	assert.Equal(t, 1, user.Character.Items[0].Uses)
}
