package items

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Phase 36c: what a merchant prices and refuses.

func TestSaleBaseValueByQualityAndRarity(t *testing.T) {
	rollTestSpecs(t)

	plain := New(testSwordID)
	assert.Equal(t, 100, plain.SaleBaseValue(), "an unrolled item is its spec's value")

	fine := New(testSwordID)
	fine.ApplyRoll(Rolled{Version: RollVersion, Quality: QualityFine, Rarity: RarityCommon, Identified: true})
	assert.Equal(t, 160, fine.SaleBaseValue(), "quality scales value")

	read := New(testSwordID)
	read.ApplyRoll(testRoll(QualityStandard, RarityRare, true))
	unread := New(testSwordID)
	unread.ApplyRoll(testRoll(QualityStandard, RarityRare, false))
	assert.Greater(t, read.SaleBaseValue(), unread.SaleBaseValue(), "an unread item is discounted against a read one")
	assert.Equal(t, 100*180/100*UnidentifiedPricePct/100, unread.SaleBaseValue(), "an unread Rare is priced by its rarity")

	epic := New(testSwordID)
	epic.ApplyRoll(testRoll(QualityStandard, RarityEpic, false))
	legendary := New(testSwordID)
	legendary.ApplyRoll(testRoll(QualityStandard, RarityLegendary, false))
	assert.Less(t, unread.SaleBaseValue(), epic.SaleBaseValue())
	assert.Less(t, epic.SaleBaseValue(), legendary.SaleBaseValue())

	// An uncommon item shows its affix from the start, so it is "identified".
	uncommon := New(testSwordID)
	uncommon.ApplyRoll(testRoll(QualityStandard, RarityUncommon, true))
	assert.Equal(t, uncommon.GetSpec().Value, uncommon.SaleBaseValue())
}

func TestIsSpecialForSale(t *testing.T) {
	rollTestSpecs(t)

	assert.False(t, (&Item{ItemId: testSwordID}).IsSpecialForSale())

	rolled := New(testSwordID)
	rolled.ApplyRoll(testRoll(QualityFine, RarityRare, false))
	assert.True(t, rolled.IsSpecial(), "36a: rolled gear is special to everything else")
	assert.False(t, rolled.IsSpecialForSale(), "36c: but a merchant buys it")

	blob := rolled
	blob.SetBlob("x")
	assert.True(t, blob.IsSpecialForSale(), "a blob still keeps an item from sale")

	edited := New(testSwordID)
	spec := edited.GetSpec()
	edited.Spec = &spec
	assert.True(t, edited.IsSpecialForSale(), "a hand-edited override is not a roll")
}

func TestJunkMarkPersistsAndShows(t *testing.T) {
	rollTestSpecs(t)
	item := New(testSwordID)
	assert.False(t, item.IsJunkMarked())
	assert.NotContains(t, item.AttrString(), "j")

	item.Junk = true
	assert.Contains(t, item.AttrString(), ">j<", "a marked item shows a j flag")

	data, err := yaml.Marshal(item)
	require.NoError(t, err)
	assert.Contains(t, string(data), "junk: true")
	var loaded Item
	require.NoError(t, yaml.Unmarshal(data, &loaded))
	assert.True(t, loaded.Junk, "the mark survives save and load")

	unmarked := New(testSwordID)
	data, err = yaml.Marshal(unmarked)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "junk", "legacy saves are unchanged")
}

func TestAutoJunkIsPlainJunkTypeOnly(t *testing.T) {
	spec := &ItemSpec{ItemId: 986010, Name: "rags", Type: Junk, Subtype: Mundane, Value: 1, Weight: 100}
	quest := &ItemSpec{ItemId: 986011, Name: "token", Type: Junk, Subtype: Mundane, Value: 1, Weight: 100, QuestToken: "9-1"}
	SetTestItemSpec(spec)
	SetTestItemSpec(quest)
	t.Cleanup(func() { RemoveTestItemSpec(986010); RemoveTestItemSpec(986011) })
	rollTestSpecs(t)

	rags := New(986010)
	assert.True(t, rags.IsAutoJunk())
	token := New(986011)
	assert.False(t, token.IsAutoJunk(), "a quest item is never junk")
	sword := New(testSwordID)
	assert.False(t, sword.IsAutoJunk())
}
