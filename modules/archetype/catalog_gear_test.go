package archetype

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 36b: the shipped catalog's weapon classes and shield sizes meet the
// shipped class rules. Clerics fight with maces and staffs only, every
// other base class may wield any weapon, rangers carry bucklers only and
// clerics and wizards no shield.
func TestCatalogGearMeetsClassRules(t *testing.T) {
	m, _ := testModule(t)

	wield := func(class string, id int) (bool, string) {
		spec := items.GetItemSpec(id)
		require.NotNil(t, spec, "catalog item %d", id)
		profile, ok := m.CombatProfile(class)
		require.True(t, ok, class)
		return profile.CanWield(archetypes.Gear{
			Shield: spec.IsShield(), ShieldSize: spec.ShieldSize,
			Weapon: spec.Type == items.Weapon, WeaponClass: spec.WeaponClass,
		})
	}

	// sword 10101, axe 10111, mace 10121, spear 10131, war spear 10141,
	// glaive 10151, staff 10162, bow 10171, crossbow 10181.
	for _, id := range []int{10101, 10111, 10131, 10141, 10151, 10171, 10181} {
		ok, why := wield("cleric", id)
		assert.False(t, ok, "a cleric may not wield %s", items.GetItemSpec(id).Name)
		assert.Contains(t, why, "fight with")
	}
	for _, id := range []int{10121, 10162} {
		ok, why := wield("cleric", id)
		assert.True(t, ok, "%s: %s", items.GetItemSpec(id).Name, why)
	}
	for _, id := range []int{10101, 10111, 10121, 10131, 10141, 10151, 10162, 10171, 10181} {
		ok, why := wield("warrior", id)
		assert.True(t, ok, "%s: %s", items.GetItemSpec(id).Name, why)
	}

	// Shields: 20300 steel buckler, 20304 kite shield.
	ok, _ := wield("warrior", 20304)
	assert.True(t, ok, "a warrior may carry any shield")
	ok, _ = wield("ranger", 20300)
	assert.True(t, ok, "a ranger carries bucklers")
	ok, why := wield("ranger", 20304)
	assert.False(t, ok)
	assert.Contains(t, why, "buckler")
	ok, _ = wield("cleric", 20300)
	assert.False(t, ok, "a cleric carries no shield")
}
