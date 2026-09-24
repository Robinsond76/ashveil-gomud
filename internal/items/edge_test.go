package items

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestSharpenNeverStacksOrResets(t *testing.T) {
	var blade Item
	require.True(t, blade.Sharpen(1, 20))
	assert.True(t, blade.Sharpened())
	assert.Equal(t, 1, blade.SharpBonus)
	assert.Equal(t, 20, blade.SharpStrikes)

	blade.SpendEdge(5)
	assert.False(t, blade.Sharpen(2, 20), "an edge is never stacked or reset")
	assert.Equal(t, 1, blade.SharpBonus)
	assert.Equal(t, 15, blade.SharpStrikes)

	var dull Item
	assert.False(t, dull.Sharpen(0, 20))
	assert.False(t, dull.Sharpen(1, 0))
	assert.False(t, dull.Sharpened())
}

func TestSpendEdgeEndsAtZero(t *testing.T) {
	var blade Item
	blade.Sharpen(2, 3)
	blade.SpendEdge(2)
	assert.Equal(t, 1, blade.SharpStrikes)
	assert.Equal(t, 2, blade.SharpBonus)
	blade.SpendEdge(5)
	assert.Zero(t, blade.SharpStrikes)
	assert.Zero(t, blade.SharpBonus, "the edge ends at zero strikes")
	assert.False(t, blade.Sharpened())
	blade.SpendEdge(0)
	blade.SpendEdge(-3)
	assert.False(t, blade.Sharpened())
}

func TestEdgeSurvivesYAMLRoundTrip(t *testing.T) {
	blade := Item{ItemId: 10002}
	blade.Sharpen(1, 17)
	out, err := yaml.Marshal(blade)
	require.NoError(t, err)
	assert.Contains(t, string(out), "sharpbonus: 1")
	assert.Contains(t, string(out), "sharpstrikes: 17")

	var loaded Item
	require.NoError(t, yaml.Unmarshal(out, &loaded))
	assert.Equal(t, 1, loaded.SharpBonus)
	assert.Equal(t, 17, loaded.SharpStrikes)

	plain, err := yaml.Marshal(Item{ItemId: 10002})
	require.NoError(t, err)
	assert.NotContains(t, string(plain), "sharp", "no edge, no fields")
}

func TestEdgeShownInNameComplexAndDescription(t *testing.T) {
	blade := Item{ItemId: 10002, Spec: &ItemSpec{ItemId: 10002, Name: "sword", Type: Weapon, Subtype: Slashing, Hands: 1}}
	assert.Empty(t, blade.EdgeLabel())
	assert.NotContains(t, blade.NameComplex(), "sharp")
	assert.NotContains(t, blade.GetLongDescription(), "honed")

	blade.Sharpen(1, 12)
	assert.Contains(t, blade.EdgeLabel(), "sharp: 12")
	assert.Contains(t, blade.NameComplex(), "sharp: 12")
	assert.NotContains(t, blade.DisplayName(), "sharp", "combat messages keep the plain name")
	assert.Contains(t, blade.GetLongDescription(), "Its edge is honed: +1 damage for its next 12 strikes.")
}
