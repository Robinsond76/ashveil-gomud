package loot

import (
	"math/rand"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tieredWeapon(id, tier int) items.ItemSpec {
	s := swordSpec()
	s.ItemId, s.Tier = id, tier
	return s
}

func withBases(t *testing.T, specs ...items.ItemSpec) {
	t.Helper()
	for i := range specs {
		items.SetTestItemSpec(&specs[i])
	}
	t.Cleanup(func() {
		for _, s := range specs {
			items.RemoveTestItemSpec(s.ItemId)
		}
	})
}

func TestZoneProfileHoldsItemLevelAndPicksTier(t *testing.T) {
	p := ZoneProfile{ILvl: Range{Low: 8, High: 12}}
	assert.Equal(t, 8, p.ItemLevel(3), "held to the zone's range")
	assert.Equal(t, 12, p.ItemLevel(40), "a farmed zone cannot produce runaway items")
	assert.Equal(t, 10, p.ItemLevel(10))
	assert.Equal(t, 40, ZoneProfile{}.ItemLevel(40), "no range: the source's own level")
	assert.Equal(t, MaxILvl, ZoneProfile{}.ItemLevel(400))
	assert.Equal(t, 1, p.BaseTier(9), "tier 1 at 1-9")
	assert.Equal(t, 2, p.BaseTier(10), "tier 2 at 10-19")
	assert.Equal(t, 3, ZoneProfile{Tier: 3}.BaseTier(1), "an explicit tier wins")
}

func TestPickBaseWeightsTheTierAndFallsBackToTheNearest(t *testing.T) {
	withBases(t, tieredWeapon(985101, 1), tieredWeapon(985102, 2), tieredWeapon(985103, 3))
	pick := func(roll int) int { return PickBase(2, &seq{vals: []int{roll, 0}}) }
	assert.Equal(t, 985102, pick(50), "the zone's tier most of the time")
	assert.Equal(t, 985101, pick(5), "one tier under sometimes")
	assert.Equal(t, 985103, pick(20), "one tier over sometimes")
	assert.Equal(t, 985103, PickBase(6, &seq{vals: []int{50, 0}}), "the nearest stocked tier when the choice has none")
}

func TestEquipmentRulesBySource(t *testing.T) {
	withBases(t, tieredWeapon(985111, 1))
	p := ZoneProfile{ILvl: Range{Low: 5, High: 9}}
	got, err := Equipment(Ordinary, 6, p, "wolf", &seq{vals: []int{99}})
	require.NoError(t, err)
	assert.Empty(t, got, "an ordinary foe drops gear 12% of the time")
	got, err = Equipment(Ordinary, 6, p, "wolf", &seq{vals: []int{5, 50, 0}})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, 985111, got[0].ItemId)
	assert.Equal(t, 6, got[0].Loot.ILvl, "item level is the source's, held to the zone")

	// A boss always drops two, and the first is Rare or better.
	for roll := 0; roll < 40; roll++ {
		got, err = Equipment(BossRoll, 9, p, "ogre", &seq{vals: []int{roll * 25, 50, roll * 7, 3, 50, 1}})
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.GreaterOrEqual(t, got[0].RollRarity().Rank(), items.RarityRare.Rank(), "roll %d", roll)
	}
}

func TestRarityBoostRaisesRareOdds(t *testing.T) {
	count := func(boost int) int {
		rare := 0
		rng := seeded{rand.New(rand.NewSource(7))}
		for i := 0; i < 20000; i++ {
			if weightedRarity(rng, boost, "").Rank() >= items.RarityRare.Rank() {
				rare++
			}
		}
		return rare
	}
	base, elite, boss := count(1), count(2), count(5)
	assert.InDelta(t, 79*20, base, 400, "7.9% of plain drops are Rare or better")
	assert.InDelta(t, 2*base, elite, 0.15*float64(2*base), "x2 for an elite")
	assert.InDelta(t, 5*base, boss, 0.15*float64(5*base), "x5 for a boss")
}

func TestGoodsAndCacheGold(t *testing.T) {
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 985121, Name: "pelt"})
	t.Cleanup(func() { items.RemoveTestItemSpec(985121) })
	table := Table{Category: "x", Entries: []WeightedLootEntry{{ItemID: 985121, Weight: 1}}}
	assert.Len(t, Goods(BossRoll, table, &seq{vals: []int{0}}), 3, "a boss makes three goods rolls")
	assert.Len(t, Goods(Cache, table, &seq{vals: []int{0}}), 1)
	assert.Len(t, Goods(Cache, table, &seq{vals: []int{1}}), 2, "a cache makes one or two")
	assert.Empty(t, Goods(Ordinary, table, &seq{vals: []int{0}}), "an ordinary foe's goods come from its own table")
	assert.Equal(t, 40, CacheGold(10, false, &seq{vals: []int{0}}))
	assert.Equal(t, 140, CacheGold(10, true, &seq{vals: []int{0}}), "a boss's cache holds much more")
}

func TestSpoilsLedgerIsReadOnceAndBounded(t *testing.T) {
	NoteSpoils(985131, "a sword", "5 gold")
	NoteSpoils(985131, "a pelt")
	assert.Equal(t, []string{"a sword", "5 gold", "a pelt"}, TakeSpoils(985131))
	assert.Empty(t, TakeSpoils(985131))
	for i := 0; i < 100; i++ {
		NoteSpoils(985132, "x")
	}
	assert.Len(t, TakeSpoils(985132), maxSpoils)
}
