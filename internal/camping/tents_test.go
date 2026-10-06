package camping

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPickTentUsesTheChoiceThenTheFirstCarried(t *testing.T) {
	kind, ok := PickTent([]TentKind{TentFur, TentLarge}, "")
	assert.True(t, ok)
	assert.Equal(t, TentFur, kind, "no choice: the first carried, canvas first")

	kind, _ = PickTent([]TentKind{TentCanvas, TentFur, TentLarge}, TentLarge)
	assert.Equal(t, TentLarge, kind)

	kind, _ = PickTent([]TentKind{TentCanvas}, TentLarge)
	assert.Equal(t, TentCanvas, kind, "a chosen tent no longer carried falls back")

	_, ok = PickTent(nil, TentFur)
	assert.False(t, ok, "no tent carried, nothing pitched")
}

func TestTentOfTreatsAnEmptyKindAsCanvas(t *testing.T) {
	assert.Equal(t, TentCanvas, TentOf("").Kind, "a camp saved before tent kinds")
	assert.Equal(t, TentCanvas, TentOf("bogus").Kind)
}

func TestParseTentTakesKindsShortWordsAndNames(t *testing.T) {
	for word, want := range map[string]TentKind{
		"fur": TentFur, "Fur-Lined Tent": TentFur, "camouflaged": TentCamouflaged,
		"large": TentLarge, "large pavilion tent": TentLarge, "canvas": TentCanvas, "oiled canvas tent": TentCanvas,
	} {
		got, ok := ParseTent(word)
		assert.True(t, ok, word)
		assert.Equal(t, want, got, word)
	}
	_, ok := ParseTent("palace")
	assert.False(t, ok)
}

func TestScaleChanceRoundsAndCapsAtOneHundred(t *testing.T) {
	assert.Equal(t, 15, ScaleChance(30, 50))
	assert.Equal(t, 45, ScaleChance(30, 150))
	assert.Equal(t, 100, ScaleChance(80, 150), "never above certain")
	assert.Equal(t, 30, ScaleChance(30, 100))
	assert.Equal(t, 30, ScaleChance(30, 0), "an unset multiplier changes nothing")
	assert.Equal(t, 0, ScaleChance(0, 150))
}

func TestEveryTentTradesSomethingAndHasItsOwnItem(t *testing.T) {
	seen := map[int]bool{}
	for _, tent := range Tents {
		assert.False(t, seen[tent.ItemID], "item %d is one tent's", tent.ItemID)
		seen[tent.ItemID] = true
		assert.NotEmpty(t, tent.Effect, tent.Kind)
		assert.NotEmpty(t, tent.Name)
	}
	assert.Equal(t, 100, TentOf(TentCanvas).RaidPct, "canvas is the plain shelter")
	assert.True(t, TentOf(TentFur).FullShelter)
	assert.Less(t, TentOf(TentCamouflaged).RaidPct, 100)
	assert.Less(t, TentOf(TentCamouflaged).RestedPct, 100, "camouflage costs rest")
	assert.Greater(t, TentOf(TentLarge).RaidPct, 100)
	assert.True(t, TentOf(TentLarge).WellRested)
}
