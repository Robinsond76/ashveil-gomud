package items

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// makeItem builds a test Item with an inline Spec so no file loading is needed.
func makeItem(id int, name string) Item {
	return Item{
		ItemId: id,
		UUID:   uuid.New(UUIDItem),
		Spec:   &ItemSpec{ItemId: id, Name: name, NameSimple: name},
	}
}

func TestItemSpecParsesNutritionAndHydration(t *testing.T) {
	var spec ItemSpec
	require.NoError(t, yaml.Unmarshal([]byte("itemid: 30004\nname: cheese sandwich\ntype: food\nsubtype: edible\nnutrition: 35\nhydration: 5\n"), &spec))
	assert.Equal(t, 35, spec.Nutrition)
	assert.Equal(t, 5, spec.Hydration)
}

func TestItemSpecSurvivalMetadataDefaultsToZero(t *testing.T) {
	var spec ItemSpec
	require.NoError(t, yaml.Unmarshal([]byte("itemid: 1\nname: note\ntype: junk\nsubtype: mundane\n"), &spec))
	assert.Zero(t, spec.Nutrition)
	assert.Zero(t, spec.Hydration)
}

func TestItemSpecParsesHydrationOnlyDrink(t *testing.T) {
	var spec ItemSpec
	require.NoError(t, yaml.Unmarshal([]byte("itemid: 30015\nname: waterskin\ntype: drink\nsubtype: drinkable\nhydration: 40\n"), &spec))
	assert.Equal(t, 40, spec.Hydration)
	assert.Zero(t, spec.Nutrition)
}

func TestItem_NameMatch(t *testing.T) {
	itm := makeItem(1, "Golden Sword")

	tests := []struct {
		input         string
		allowContains bool
		wantPartial   bool
		wantFull      bool
	}{
		{"golden sword", false, true, true},
		{"golden", false, true, false},
		{"sword", false, false, false},
		{"sword", true, true, false},
		{"golden sword", true, true, true},
		{"xyz", false, false, false},
		{"xyz", true, false, false},
		{"GOLDEN", false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			partial, full := itm.NameMatch(tt.input, tt.allowContains)
			assert.Equal(t, tt.wantPartial, partial, "partial")
			assert.Equal(t, tt.wantFull, full, "full")
		})
	}
}

func TestItem_NameMatch_EmptyItem(t *testing.T) {
	itm := Item{ItemId: 0}
	partial, full := itm.NameMatch("anything", true)
	assert.False(t, partial)
	assert.False(t, full)
}

func TestFindMatchIn_ExactMatch(t *testing.T) {
	sword := makeItem(1, "sword")
	shield := makeItem(2, "shield")

	partial, full := FindMatchIn("sword", sword, shield)
	assert.Equal(t, Item{}, partial)
	assert.Equal(t, sword.ItemId, full.ItemId)
}

func TestFindMatchIn_PartialMatch(t *testing.T) {
	longsword := makeItem(1, "longsword")
	shield := makeItem(2, "shield")

	partial, full := FindMatchIn("long", longsword, shield)
	assert.Equal(t, longsword.ItemId, partial.ItemId)
	assert.Equal(t, Item{}, full)
}

func TestFindMatchIn_NoMatch(t *testing.T) {
	sword := makeItem(1, "sword")

	partial, full := FindMatchIn("axe", sword)
	assert.Equal(t, Item{}, partial)
	assert.Equal(t, Item{}, full)
}

func TestFindMatchIn_ContainsFallback(t *testing.T) {
	// "sword" is a prefix of the word "sword" in the two-word name "long sword",
	// so the contains pass finds it when the prefix pass on the full name does not.
	longsword := makeItem(1, "long sword")

	partial, full := FindMatchIn("sword", longsword)
	assert.Equal(t, longsword.ItemId, partial.ItemId)
	assert.Equal(t, Item{}, full)
}

func TestFindMatchIn_NthMatch(t *testing.T) {
	sword1 := makeItem(1, "sword")
	sword2 := makeItem(2, "sword")
	sword3 := makeItem(3, "sword")

	_, full := FindMatchIn("sword#2", sword1, sword2, sword3)
	assert.Equal(t, sword2.ItemId, full.ItemId)

	_, full = FindMatchIn("sword#3", sword1, sword2, sword3)
	assert.Equal(t, sword3.ItemId, full.ItemId)
}

func TestFindMatchIn_ShorthandById(t *testing.T) {
	sword := makeItem(10, "sword")

	_, full := FindMatchIn("!10", sword)
	assert.Equal(t, sword.ItemId, full.ItemId)
}

func TestFindMatchIn_ShorthandByUUID(t *testing.T) {
	sword := makeItem(10, "sword")
	shield := makeItem(20, "shield")

	uuidStr := sword.UUID.String()
	_, full := FindMatchIn("!10:"+uuidStr, sword, shield)
	assert.Equal(t, sword.ItemId, full.ItemId)
	assert.Equal(t, sword.UUID, full.UUID)
}

func TestFindMatchIn_ShorthandByUUID_WrongUUID(t *testing.T) {
	sword := makeItem(10, "sword")
	other := makeItem(10, "sword")

	// other has a different UUID; searching for sword's UUID should not return other
	uuidStr := sword.UUID.String()
	partial, full := FindMatchIn("!10:"+uuidStr, other)
	assert.Equal(t, Item{}, partial)
	assert.Equal(t, Item{}, full)
}

func TestFindMatchIn_EmptyList(t *testing.T) {
	partial, full := FindMatchIn("sword")
	assert.Equal(t, Item{}, partial)
	assert.Equal(t, Item{}, full)
}

func TestItem_HasAdjective(t *testing.T) {
	itm := makeItem(1, "sword")

	assert.False(t, itm.HasAdjective("exploding"))
	itm.SetAdjective("exploding", true)
	assert.True(t, itm.HasAdjective("exploding"))
}

func TestItem_SetAdjective_Remove(t *testing.T) {
	itm := makeItem(1, "sword")
	itm.SetAdjective("exploding", true)
	itm.SetAdjective("exploding", false)
	assert.False(t, itm.HasAdjective("exploding"))
}

func TestItem_SetAdjective_NoDuplicate(t *testing.T) {
	itm := makeItem(1, "sword")
	itm.SetAdjective("exploding", true)
	itm.SetAdjective("exploding", true)
	assert.Equal(t, 1, len(itm.Adjectives))
}

func TestItem_IsDisabled(t *testing.T) {
	assert.True(t, ItemDisabledSlot.IsDisabled())
	assert.False(t, makeItem(1, "sword").IsDisabled())
}

func TestItem_IsCursed(t *testing.T) {
	itm := makeItem(1, "cursed blade")
	itm.Spec.Cursed = true

	assert.True(t, itm.IsCursed())
	itm.Uncurse()
	assert.False(t, itm.IsCursed())
}

func TestItem_Equals(t *testing.T) {
	a := makeItem(1, "sword")
	b := makeItem(1, "sword")

	assert.False(t, a.Equals(b), "different UUIDs should not be equal")
	assert.True(t, a.Equals(a), "same item should equal itself")
}

func TestItem_TempData(t *testing.T) {
	itm := makeItem(1, "sword")

	assert.Nil(t, itm.GetTempData("key"))
	itm.SetTempData("key", "value")
	assert.Equal(t, "value", itm.GetTempData("key"))
	itm.SetTempData("key", nil)
	assert.Nil(t, itm.GetTempData("key"))
}
