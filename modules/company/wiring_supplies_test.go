package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 43a: the shipped camp supplies and waterskins, through the real
// item and buff files.
func TestShippedCampSuppliesAndBuffsLoad(t *testing.T) {
	b := newBrawl(t)
	copyShipped(t, configs.GetFilePathsConfig().DataFiles.String(), "buffs", "buffs-flags")
	buffs.LoadFlagDataFiles()
	buffs.LoadDataFiles()

	for _, s := range camping.Supplies {
		spec := items.GetItemSpec(s.ItemID)
		require.NotNil(t, spec, s.Key)
		assert.Equal(t, s.Name, spec.Name)
		assert.Equal(t, 1, spec.Uses, "one dose each")
		assert.Positive(t, spec.Value, "%s has a shop value", s.Key)
		assert.Positive(t, spec.Weight, "%s has a small weight", s.Key)
		assert.NotEqual(t, items.Drinkable, spec.Subtype, "%s is prepared at camp, not quaffed", s.Key)
		assert.Empty(t, spec.BuffIds)
	}

	// Each Fortified size raises the health limit by its bonus, and its
	// expiry never leaves current health above the ordinary limit.
	for _, tier := range camping.BrothBuffs {
		c := b.aria.Character
		c.Validate()
		base := c.HealthMax.Value
		c.Health = base
		require.NoError(t, c.AddBuff(tier.BuffID, false), tier.BuffID)
		c.Validate()
		assert.Equal(t, base+tier.Bonus, c.HealthMax.Value, "buff %d", tier.BuffID)
		c.Health = c.HealthMax.Value // healed up to the boosted limit
		c.RemoveBuff(tier.BuffID)
		c.Validate()
		assert.Equal(t, base, c.HealthMax.Value)
		assert.LessOrEqual(t, c.Health, c.HealthMax.Value, "expiry clamps health and cannot kill")
		assert.Positive(t, c.Health)
	}

	for _, id := range []int{camping.WarmingBuffID, camping.CoolingBuffID} {
		spec := buffs.GetBuffSpec(id)
		require.NotNil(t, spec, id)
		assert.Positive(t, spec.RoundInterval, "a fifteen minute buff")
	}
	assert.NotNil(t, buffs.GetFlagSpec("warming-draught"))
	assert.NotNil(t, buffs.GetFlagSpec("cooling-salve"))
}

func TestShippedWaterskinLeavesAnEmptyOneThatRefills(t *testing.T) {
	b := newBrawl(t)
	full := items.GetItemSpec(30015)
	require.NotNil(t, full)
	empty := items.GetItemSpec(full.EmptyItemId)
	require.NotNil(t, empty, "the waterskin names its empty form")
	assert.Equal(t, full.ItemId, empty.FilledItemId)
	assert.Zero(t, empty.Hydration)
	assert.NotEqual(t, items.Drinkable, empty.Subtype, "an empty skin is not a drink")
	uses, ok := usercommands.RefillableUses(*empty)
	assert.True(t, ok)
	assert.Equal(t, full.Uses, uses)

	c := b.aria.Character
	skin := items.New(30015)
	skin.Uses = 1
	c.Items = append(c.Items, skin)
	c.UseItem(skin)
	var found bool
	for _, itm := range c.Items {
		if itm.ItemId == empty.ItemId {
			found = true
			assert.Zero(t, itm.Uses)
		}
	}
	assert.True(t, found, "the last sip leaves the empty skin in the pack")
	for _, itm := range c.Items {
		assert.NotEqual(t, 30015, itm.ItemId)
	}
}
