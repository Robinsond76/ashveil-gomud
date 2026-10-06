package loot

import (
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const (
	relicBossID    = 970001
	relicGlaiveID  = 970101
	relicHelmID    = 970102
	relicChestID   = 970103
	plainGlaiveID  = 970104
	relicOtherBoss = 970002
	relicOtherID   = 970105
)

// relicWorld registers a boss's relics: a Legendary glaive (5%) and two set
// pieces (4% each), and another boss's one relic.
func relicWorld(t *testing.T) {
	t.Helper()
	specs := []*items.ItemSpec{
		{ItemId: relicGlaiveID, Name: "Test Reaper", Type: items.Weapon, Subtype: items.Slashing, Hands: 2, Tier: 6, Value: 500, Weight: 3000,
			Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 12},
			Relic:  &items.RelicSpec{Signature: "Reaping", Effects: map[string]int{classes.Wounded: 20}, ILvl: 35, Mob: relicBossID, Chance: 5}},
		{ItemId: relicHelmID, Name: "Test Helm", Type: items.Head, Subtype: items.Wearable, Tier: 5, Value: 300, Weight: 1000, DamageReduction: 6,
			Relic: &items.RelicSpec{Set: "loottest", ILvl: 35, Mob: relicBossID, Chance: 4}},
		{ItemId: relicChestID, Name: "Test Chest", Type: items.Body, Subtype: items.Wearable, Tier: 5, Value: 400, Weight: 6000, DamageReduction: 14,
			Relic: &items.RelicSpec{Set: "loottest", ILvl: 35, Mob: relicBossID, Chance: 4}},
		{ItemId: relicOtherID, Name: "Other Relic", Type: items.Weapon, Subtype: items.Slashing, Hands: 1, Tier: 4, Value: 200, Weight: 1500,
			Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 8},
			Relic:  &items.RelicSpec{Signature: "Other", Effects: map[string]int{classes.Damage: 2}, ILvl: 27, Mob: relicOtherBoss, Chance: 50}},
		{ItemId: plainGlaiveID, Name: "plain glaive", Type: items.Weapon, Subtype: items.Slashing, Hands: 2, Tier: 6, Value: 100, Weight: 3000,
			Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 10}},
	}
	for _, s := range specs {
		items.SetTestItemSpec(s)
	}
	t.Cleanup(func() {
		for _, s := range specs {
			items.RemoveTestItemSpec(s.ItemId)
		}
	})
	SetAffixes(shippedSet(t))
}

// An authored relic rolls as its own rarity at its own item level and
// standard workmanship, whatever the caller asks: a Legendary has two
// affixes to read and no generated name; a set piece is fixed and has
// nothing hidden.
func TestRelicsRollAsTheirOwnRarityAtTheirOwnLevel(t *testing.T) {
	relicWorld(t)
	rng := seeded{rand.New(rand.NewSource(36))}
	for i := 0; i < 50; i++ {
		leg, err := Roll(relicGlaiveID, Options{ILvl: 5, Rarity: items.RarityCommon, Quality: items.QualityCrude, Source: "the lich"}, rng)
		require.NoError(t, err)
		assert.Equal(t, items.RarityLegendary, leg.Loot.Rarity)
		assert.Equal(t, 35, leg.Loot.ILvl)
		assert.Equal(t, items.QualityStandard, leg.Loot.Quality, "a relic's numbers are the ones in its file")
		assert.Len(t, leg.Loot.Affixes, 2, "a legendary rolls two affixes, the signature is authored")
		assert.False(t, leg.Loot.Identified, "its affixes need reading")
		assert.Empty(t, leg.Loot.Name, "its own name stands")
		assert.Equal(t, 30, leg.LevelRequirement())
		assert.Equal(t, "the lich", leg.Loot.Source)
		assert.Equal(t, 1, leg.GetSpec().Damage.DiceCount, "numbers are the file's")
		assert.Equal(t, 6, leg.Loot.Tier)
		for _, a := range leg.Loot.Affixes {
			assert.False(t, a.Major, "no rolled major affix: the signature is the major effect")
		}

		piece, err := Roll(relicHelmID, Options{ILvl: 5}, rng)
		require.NoError(t, err)
		assert.Equal(t, items.RaritySet, piece.Loot.Rarity)
		assert.Empty(t, piece.Loot.Affixes, "a set piece is fixed")
		assert.True(t, piece.Loot.Identified, "nothing hidden to read")
		assert.Equal(t, 6, piece.GetSpec().DamageReduction, "protection is the file's")
	}
}

// The plain drop tables never pick a relic's base type as an ordinary item.
func TestOrdinaryDropsNeverDrawRelicBases(t *testing.T) {
	relicWorld(t)
	rng := seeded{rand.New(rand.NewSource(3))}
	for i := 0; i < 500; i++ {
		id := PickBase(6, rng)
		assert.Equal(t, plainGlaiveID, id, "only the plain tier 6 base is drawn")
	}
	for tier, ids := range bases() {
		for _, id := range ids {
			assert.Nil(t, items.GetItemSpec(id).Relic, "tier %d base %d", tier, id)
		}
	}
}

func TestRelicsOfAndBosses(t *testing.T) {
	relicWorld(t)
	got := RelicsOf(relicBossID)
	require.Len(t, got, 3)
	assert.Equal(t, relicGlaiveID, got[0].ItemId, "ascending by id")
	assert.Len(t, RelicsOf(relicOtherBoss), 1)
	assert.Empty(t, RelicsOf(424242))
	assert.Contains(t, RelicBosses(), relicBossID)
}

// One roll per kill picks at most one relic by cumulative chance (5, 4, 4):
// 0-4 the glaive, 5-8 the helm, 9-12 the chest, the rest nothing.
func TestRelicRollPicksByChanceOneAtMost(t *testing.T) {
	relicWorld(t)
	for roll, want := range map[int]int{0: relicGlaiveID, 4: relicGlaiveID, 5: relicHelmID, 8: relicHelmID, 9: relicChestID, 12: relicChestID, 13: 0, 99: 0} {
		// First Intn is the chance roll; later draws (affixes, quality) are free.
		itm, missed, err := RelicRoll(relicBossID, 3, "the boss", &seq{vals: []int{roll, 0}})
		require.NoError(t, err)
		if want == 0 {
			assert.Nil(t, itm, "roll %d", roll)
			assert.Equal(t, 4, missed, "a miss counts toward bad luck")
			continue
		}
		require.NotNil(t, itm, "roll %d", roll)
		assert.Equal(t, want, itm.ItemId, "roll %d", roll)
		assert.Zero(t, missed, "a drop resets the count")
	}
	itm, missed, err := RelicRoll(424242, 7, "x", &seq{vals: []int{0}})
	require.NoError(t, err)
	assert.Nil(t, itm)
	assert.Equal(t, 7, missed, "a mob with no relics changes nothing")
}

// Bad-luck protection: the 20th kill without a relic is guaranteed one, from
// the boss's own table; the count then starts over.
func TestRelicBadLuckGuaranteesTheTwentiethKill(t *testing.T) {
	relicWorld(t)
	rng := &seq{vals: []int{99}} // the chance roll always misses
	missed := 0
	for kill := 1; kill < BadLuckKills; kill++ {
		var itm *items.Item
		var err error
		itm, missed, err = RelicRoll(relicBossID, missed, "the boss", rng)
		require.NoError(t, err)
		require.Nil(t, itm, "kill %d", kill)
	}
	assert.Equal(t, BadLuckKills-1, missed)
	itm, missed, err := RelicRoll(relicBossID, missed, "the boss", &seq{vals: []int{99, 12, 0}})
	require.NoError(t, err)
	require.NotNil(t, itm, "the 20th kill is guaranteed a relic")
	assert.Zero(t, missed)
	assert.Equal(t, relicBossID, items.GetItemSpec(itm.ItemId).Relic.Mob)
	assert.Equal(t, relicChestID, itm.ItemId, "the guaranteed pick is weighted by chance: 12 of 13")

	// Real randomness: every boss eventually pays, never later than the 20th.
	real := seeded{rand.New(rand.NewSource(1))}
	for trial := 0; trial < 200; trial++ {
		missed, kills := 0, 0
		for {
			kills++
			itm, m, err := RelicRoll(relicBossID, missed, "boss", real)
			require.NoError(t, err)
			missed = m
			if itm != nil {
				break
			}
			require.Less(t, kills, BadLuckKills, "a relic by the 20th kill")
		}
	}
}

// Treasure pays: a relic breaks down at a smith like any other find.
func TestRelicsCanBeSalvaged(t *testing.T) {
	relicWorld(t)
	assert.NotNil(t, SalvageYield(items.New(relicGlaiveID)))
	assert.NotNil(t, SalvageYield(items.New(plainGlaiveID)), "the plain tier 6 glaive still is")
}

// The salvage material values held in code are the shipped item files'.
func TestMaterialWorthMatchesTheItemFiles(t *testing.T) {
	root := "../../_datafiles/world/default/items"
	found := 0
	require.NoError(t, filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return err
		}
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		spec := items.ItemSpec{}
		require.NoError(t, yaml.Unmarshal(data, &spec))
		if want, ok := materialWorth[spec.ItemId]; ok {
			found++
			assert.Equal(t, want, spec.Value, "%s", spec.Name)
		}
		return nil
	}))
	assert.Equal(t, len(materialWorth), found)
}
