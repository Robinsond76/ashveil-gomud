package death

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

var table = []Scenario{
	{ID: "rescue", Kind: Rescued, Weight: 2},
	{ID: "capture", Kind: Captured, Weight: 3, Foes: []string{"human", "Goblin"}, Room: 9},
	{ID: "wild", Kind: LeftForDead, Foes: []string{"beast"}, Zones: []string{"Old Kings Road"}},
}

func TestScenarioFitsByFoeAndZone(t *testing.T) {
	assert.True(t, table[0].Fits("Anywhere", "", nil), "no foes named fits an unknown killer")
	assert.False(t, table[1].Fits("Anywhere", "", nil), "a named foe needs a known killer")
	assert.True(t, table[1].Fits("Anywhere", "goblin", nil), "race names compare without case")
	assert.True(t, table[1].Fits("Anywhere", "dog", []string{"human"}), "a mob group counts as its kind")
	assert.False(t, table[2].Fits("Frostfang", "beast", nil), "wrong zone")
	assert.True(t, table[2].Fits("old kings road", "beast", nil))
}

func TestPickIsWeightedAndFallsBackWhenNoneFit(t *testing.T) {
	// Weights: rescue 2, capture 3 -> rolls 0-1 rescue, 2-4 capture.
	at := func(n int) func(int) int { return func(int) int { return n } }
	for roll, want := range map[int]string{0: "rescue", 1: "rescue", 2: "capture", 4: "capture"} {
		got, ok := Pick(table, "Frostfang", "human", nil, at(roll))
		assert.True(t, ok)
		assert.Equal(t, want, got.ID, "roll %d", roll)
	}
	got, ok := Pick(table, "Frostfang", "", nil, at(1))
	assert.True(t, ok)
	assert.Equal(t, "rescue", got.ID)
	_, ok = Pick([]Scenario{table[1]}, "Frostfang", "beast", nil, at(0))
	assert.False(t, ok, "nothing fits: the church stands")
	_, ok = Pick(nil, "x", "", nil, at(0))
	assert.False(t, ok)
}

func TestScenarioValid(t *testing.T) {
	assert.NoError(t, table[0].Valid())
	assert.Error(t, Scenario{Kind: Rescued}.Valid(), "no id")
	assert.Error(t, Scenario{ID: "x", Kind: "dragon"}.Valid(), "unknown kind")
	assert.Error(t, Scenario{ID: "x", Kind: Captured}.Valid(), "a capture needs a room")
	assert.Error(t, Scenario{ID: "x", Kind: Robbed, GoldLossPct: 101}.Valid())
}

func TestGoldShareNeverExceedsTheGold(t *testing.T) {
	assert.Equal(t, 50, GoldShare(200, 25))
	assert.Equal(t, 0, GoldShare(3, 25))
	assert.Equal(t, 7, GoldShare(7, 500))
	assert.Equal(t, 0, GoldShare(-5, 50))
}

func TestRobSplitsThePackWithoutDuplicatingOrLosingAnything(t *testing.T) {
	var pack []items.Item
	for i := 0; i < 8; i++ {
		pack = append(pack, items.Item{ItemId: 38, Uses: i})
	}
	first := func(int) int { return 0 }
	any := func(int) bool { return true }
	kept, taken := Rob(pack, 25, 0, first, any)
	assert.Len(t, taken, 2, "a quarter of eight")
	assert.Len(t, kept, 6)
	assert.Len(t, kept, len(pack)-len(taken))

	_, taken = Rob(pack, 100, 3, first, any)
	assert.Len(t, taken, 3, "capped by the limit")
	_, taken = Rob(pack, 0, 0, first, any)
	assert.Empty(t, taken, "no share, nothing taken")
	kept, taken = Rob(nil, 50, 5, first, any)
	assert.Empty(t, kept)
	assert.Empty(t, taken)
}

func TestRobbableLeavesCampGearAlone(t *testing.T) {
	for id := 45; id <= 50; id++ {
		assert.False(t, Robbable(id), "camp gear %d", id)
	}
	assert.False(t, Robbable(-1), "an unknown item is never taken")
}

func TestCaptorGroupIsPerCompanyAndRoom(t *testing.T) {
	assert.NotEqual(t, CaptorGroup(91001, 1), CaptorGroup(91001, 2))
	assert.NotEqual(t, CaptorGroup(91001, 1), CaptorGroup(91002, 1))
}
