package users

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clericGear is warriorHP with a cleric's 35a2 gear rules.
type clericGear struct{ warriorHP }

func (clericGear) PlayerArchetype(int) (string, bool) { return "cleric", true }
func (clericGear) CombatProfile(id string) (archetypes.Profile, bool) {
	return archetypes.Profile{Name: "Cleric", AttackRate: 0.7, EvasionRate: 0.75, HPStart: 2, ArmorTraining: items.BulkLight,
		ShieldSizes: []string{"none"}, WeaponClasses: []string{"staff", "rod", "mace"}}, id == "cleric"
}

// TestSettleClassGearAcrossCopyover (Phase 35a2): a cleric restored by
// copyover holding a shield has it put away once, saved, and told; a
// second pass or a reload changes and says nothing. (Weapons take the same
// path; characters' TestClassGearRules covers a club, and the company
// wiring tests the real equip.)
func TestSettleClassGearAcrossCopyover(t *testing.T) {
	replayDataDir(t)
	const shieldID = 99701
	shield := &items.ItemSpec{ItemId: shieldID, Name: "test shield", Type: items.Offhand, Subtype: items.Wearable, DamageReduction: 5, Weight: 3000}
	require.NoError(t, shield.Validate())
	items.SetTestItemSpec(shield)
	t.Cleanup(func() { items.RemoveTestItemSpec(shieldID) })

	archetypes.SetProvider(nil)
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	u := veteran()
	u.Character.Equipment.Offhand = items.New(shieldID)
	require.NoError(t, SaveUser(*u))
	restored, err := loadUserById(7, true)
	require.NoError(t, err)
	assert.Equal(t, shieldID, restored.Character.Equipment.Offhand.ItemId, "copyover's early restore can't see the class yet")
	SetTestUser(restored)
	t.Cleanup(func() { RemoveTestUser(7) })

	archetypes.SetProvider(clericGear{})
	ValidateActiveCharacters()
	assert.Zero(t, restored.Character.Equipment.Offhand.ItemId)
	carried := map[int]bool{}
	for _, itm := range restored.Character.Items {
		carried[itm.ItemId] = true
	}
	assert.True(t, carried[shieldID], "put with carried items")

	reloaded, err := loadUserById(7, true)
	require.NoError(t, err)
	assert.Zero(t, reloaded.Character.Equipment.Offhand.ItemId, "the move was saved")
	assert.Empty(t, SettleClassGear(reloaded, nil), "a second pass says nothing")
	assert.Empty(t, SettleClassGear(restored, nil))

	fresh := veteran()
	fresh.Character.Equipment.Offhand = items.New(shieldID)
	note := SettleClassGear(fresh, func(*UserRecord) error { return nil })
	assert.Contains(t, note, "Clerics don't carry shields.")
	assert.Contains(t, note, "help shields")
}
