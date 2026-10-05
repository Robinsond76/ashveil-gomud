package items

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBulkAndShieldSizeValidate (Phase 35a2): armor without a bulk takes
// it from its weight, a shield without a size is a "shield", and unknown
// values or misplaced fields are refused.
func TestBulkAndShieldSizeValidate(t *testing.T) {
	for grams, want := range map[int]string{0: BulkLight, 2499: BulkLight, 2500: BulkMedium, 5999: BulkMedium, 6000: BulkHeavy} {
		spec := ItemSpec{Name: "vest", Type: Body, Subtype: Wearable, Weight: grams}
		require.NoError(t, spec.Validate())
		assert.Equal(t, want, spec.Bulk, "%d g", grams)
	}
	tagged := ItemSpec{Name: "cape", Type: Body, Subtype: Wearable, Weight: 9000, Bulk: " Light "}
	require.NoError(t, tagged.Validate())
	assert.Equal(t, BulkLight, tagged.Bulk, "an explicit bulk wins over weight")

	shield := ItemSpec{Name: "shield", Type: Offhand, Subtype: Wearable, DamageReduction: 5, Weight: 3000}
	require.NoError(t, shield.Validate())
	assert.Equal(t, ShieldNormal, shield.ShieldSize)
	assert.True(t, shield.IsShield())
	symbol := ItemSpec{Name: "symbol", Type: Offhand, Subtype: Wearable, Weight: 300}
	require.NoError(t, symbol.Validate())
	assert.False(t, symbol.IsShield(), "an off-hand with no armor is no shield")
	assert.Empty(t, symbol.ShieldSize)

	for name, bad := range map[string]ItemSpec{
		"unknown bulk":         {Name: "x", Type: Body, Subtype: Wearable, Bulk: "huge"},
		"bulk on a weapon":     {Name: "x", Type: Weapon, Subtype: Bludgeoning, Bulk: BulkHeavy, Hands: 1},
		"unknown shield size":  {Name: "x", Type: Offhand, Subtype: Wearable, DamageReduction: 3, ShieldSize: "wall"},
		"size on a non-shield": {Name: "x", Type: Offhand, Subtype: Wearable, ShieldSize: ShieldBuckler},
		"class on armor":       {Name: "x", Type: Body, Subtype: Wearable, WeaponClass: "mace"},
	} {
		assert.Error(t, bad.Validate(), name)
	}
}
