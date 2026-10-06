package items

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTierNames(t *testing.T) {
	for tier, want := range map[int]string{0: "Common", 1: "Common", 2: "Steel", 3: "Tempered", 4: "Masterwork", 5: "Runeforged", 6: "Relic"} {
		assert.Equal(t, want, TierName(tier), "tier %d", tier)
	}
	assert.Empty(t, TierName(7))
	assert.Empty(t, TierName(-1))
}

func TestValidateChecksTierAndGoods(t *testing.T) {
	good := func() *ItemSpec {
		return &ItemSpec{ItemId: 988200, Name: "pelt", Type: Commodity, Subtype: Mundane, Value: 5, Weight: 1000, Goods: " Trophy "}
	}
	spec := good()
	require.NoError(t, spec.Validate())
	assert.Equal(t, GoodsTrophy, spec.Goods, "the category is normalised")

	spec = good()
	spec.Goods = "loot"
	assert.ErrorContains(t, spec.Validate(), "unknown goods category")

	spec = good()
	spec.Type = Weapon
	assert.ErrorContains(t, spec.Validate(), "goods cannot be equipment")

	spec = good()
	spec.Type, spec.Subtype = Body, Wearable
	assert.ErrorContains(t, spec.Validate(), "goods cannot be equipment")

	spec = good()
	spec.Tier = 7
	assert.ErrorContains(t, spec.Validate(), "tier 7")
	spec.Tier = -1
	assert.ErrorContains(t, spec.Validate(), "tier -1")
	spec.Tier = MaxTier
	assert.NoError(t, spec.Validate())

	spec = &ItemSpec{ItemId: 988201, Name: "glaive", Type: Weapon, Subtype: Slashing, Hands: 2, Family: " Glaive ", Tier: 2}
	require.NoError(t, spec.Validate())
	assert.Equal(t, "glaive", spec.Family)
}

func TestCatalogDescriptions(t *testing.T) {
	glaive := &ItemSpec{ItemId: 988202, Name: "steel glaive", Description: "A glaive.", Type: Weapon, Subtype: Slashing, Hands: 2, Reach: true, Tier: 2, Family: "glaive"}
	require.NoError(t, glaive.Validate())
	SetTestItemSpec(glaive)
	t.Cleanup(func() { RemoveTestItemSpec(988202) })
	itm := New(988202)
	assert.Contains(t, itm.GetLongDescription(), "Tier 2 (Steel) glaive. See help equipmenttiers.")

	untiered := &ItemSpec{ItemId: 988203, Name: "stick", Description: "A stick.", Type: Weapon, Subtype: Bludgeoning}
	require.NoError(t, untiered.Validate())
	SetTestItemSpec(untiered)
	t.Cleanup(func() { RemoveTestItemSpec(988203) })
	bare := New(988203)
	assert.NotContains(t, bare.GetLongDescription(), "Tier")

	pelt := &ItemSpec{ItemId: 988204, Name: "bear hide", Description: "A hide.", Type: Commodity, Subtype: Mundane, Goods: GoodsTrophy, Value: 24, Weight: 3500}
	require.NoError(t, pelt.Validate())
	SetTestItemSpec(pelt)
	t.Cleanup(func() { RemoveTestItemSpec(988204) })
	hide := New(988204)
	assert.Contains(t, hide.GetLongDescription(), "Trade good (trophy): worth about 7 gold a kg. See help goods.")
	assert.NotContains(t, hide.GetLongDescription(), "Tier")

	assert.Zero(t, (ItemSpec{Value: 10}).GoodsValuePerKg(), "no weight, no density")
	assert.Empty(t, (ItemSpec{}).GoodsDescription())
	assert.Empty(t, (ItemSpec{}).TierDescription())
}
