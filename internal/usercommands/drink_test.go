package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/survival"
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
	fake := &fakeProvisioner{err: survival.ErrPersistenceUnavailable}
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
