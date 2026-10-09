package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	trophyBladeID = 99631
	trophyHeartID = 99632
)

// Phase 71: a trophy enchant on an ordinary worn piece changes its wearer's
// effects at once, through the gear cache, with no relic worn at all, and
// goes when the piece comes off.
func TestAnEnchantedPlainPieceChangesTheWearersEffects(t *testing.T) {
	items.SetTestItemSpec(&items.ItemSpec{ItemId: trophyBladeID, Name: "plain blade", Type: items.Weapon, Subtype: items.Slashing, Hands: 1, Tier: 1,
		Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 6}})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: trophyHeartID, Name: "test heart", Type: items.Commodity,
		Trophy: &items.TrophySpec{Part: items.TrophyHeart, Races: []string{"ogre"}, Chance: 20, Effects: map[string]int{classes.Damage: 1}}})
	t.Cleanup(func() { items.RemoveTestItemSpec(trophyBladeID); items.RemoveTestItemSpec(trophyHeartID) })

	c := New()
	c.Level = 10
	wear(t, c, trophyBladeID)
	assert.Zero(t, c.ClassEffects().Int(classes.Damage))
	fx, sets := c.WornGear()
	assert.Empty(t, fx)
	assert.Empty(t, sets)

	// Enchanted while worn (what `imbue` does): the cache sees it at once.
	require.NoError(t, c.Equipment.Weapon.EnchantWithTrophy(trophyHeartID))
	assert.Equal(t, 1, c.ClassEffects().Int(classes.Damage))
	fx, _ = c.WornGear()
	assert.Equal(t, map[string]int{classes.Damage: 1}, fx)

	worn := c.Equipment.Weapon
	c.Equipment.Weapon = items.Item{}
	assert.Zero(t, c.ClassEffects().Int(classes.Damage), "off the body, no effect")
	c.Equipment.Weapon = worn
	assert.Equal(t, 1, c.ClassEffects().Int(classes.Damage), "and back on, with its enchant")
}

// A piece enchanted in the pack carries the enchant when it is worn.
func TestAnItemEnchantedInThePackCarriesItWhenWorn(t *testing.T) {
	items.SetTestItemSpec(&items.ItemSpec{ItemId: trophyBladeID, Name: "plain blade", Type: items.Weapon, Subtype: items.Slashing, Hands: 1, Tier: 1,
		Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 6}})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: trophyHeartID, Name: "test heart", Type: items.Commodity,
		Trophy: &items.TrophySpec{Part: items.TrophyHeart, Races: []string{"ogre"}, Chance: 20, Effects: map[string]int{classes.Damage: 1}}})
	t.Cleanup(func() { items.RemoveTestItemSpec(trophyBladeID); items.RemoveTestItemSpec(trophyHeartID) })

	c := New()
	c.Level = 10
	blade := items.New(trophyBladeID)
	require.NoError(t, blade.EnchantWithTrophy(trophyHeartID))
	c.StoreItem(blade)
	assert.Zero(t, c.ClassEffects().Int(classes.Damage), "carried, not worn")
	wear2(t, c, blade)
	assert.Equal(t, 1, c.ClassEffects().Int(classes.Damage))
}

func wear2(t *testing.T, c *Character, it items.Item) {
	t.Helper()
	loadShippedRaces(t)
	_, worn, reason := c.Wear(it)
	require.True(t, worn, reason)
}
