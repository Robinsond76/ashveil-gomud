package loot

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

type fixedRoll int

func (f fixedRoll) Intn(n int) int { return int(f) % n }

func TestInstrumentRollFollowsEachChance(t *testing.T) {
	for _, spec := range []*items.ItemSpec{
		{ItemId: 990301, Name: "test horn", Type: items.Object, Instrument: "winds", InstrumentTier: 4, InstrumentMob: 990300, InstrumentChance: 25},
		{ItemId: 990302, Name: "other drum", Type: items.Object, Instrument: "drums", InstrumentTier: 4, InstrumentMob: 990399, InstrumentChance: 100},
		{ItemId: 990303, Name: "bought flute", Type: items.Object, Instrument: "winds", InstrumentTier: 2},
	} {
		items.SetTestItemSpec(spec)
		id := spec.ItemId
		t.Cleanup(func() { items.RemoveTestItemSpec(id) })
	}
	assert.Len(t, InstrumentDropsOf(990300), 1, "only this boss's masterwork")
	assert.Empty(t, InstrumentDropsOf(1), "other mobs drop none")
	hit := InstrumentRoll(990300, fixedRoll(24))
	if assert.Len(t, hit, 1) {
		assert.Equal(t, 990301, hit[0].ItemId)
	}
	assert.Empty(t, InstrumentRoll(990300, fixedRoll(25)), "a 25% chance misses at 25")
}
