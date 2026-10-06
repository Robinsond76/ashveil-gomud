package classes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 36d: the effects gear may grant.
func TestGearEffectsAreValidatedAndDescribed(t *testing.T) {
	seen := map[string]bool{}
	for _, e := range gearEffects {
		assert.False(t, seen[e.Key], "%s listed twice", e.Key)
		seen[e.Key] = true
		assert.GreaterOrEqual(t, e.Max, 1, e.Key)
		assert.NotEmpty(t, e.Text(1), e.Key)
		assert.NotEmpty(t, e.Text(e.Max), e.Key)
	}
	assert.NoError(t, ValidateGearEffects(map[string]int{Wounded: 20, Bargain: 1}))
	assert.Error(t, ValidateGearEffects(map[string]int{"warlock": 5}), "an unknown key")
	assert.Error(t, ValidateGearEffects(map[string]int{Wounded: 0}), "zero grants nothing")
	assert.Error(t, ValidateGearEffects(map[string]int{Wounded: 31}), "over the cap")
	assert.Error(t, ValidateGearEffects(map[string]int{Bargain: 2}), "a once-a-battle flag is 1")
}

func TestGearEffectTextCarriesTheNumber(t *testing.T) {
	got := DescribeGearEffects(map[string]int{Wounded: 25, Armor: 4})
	require.Len(t, got, 2)
	assert.Contains(t, got[0], "4%", "armor sorts before wounded")
	assert.Contains(t, got[1], "25%")
}

func TestWithGearAddsToAClassWithoutChangingIt(t *testing.T) {
	base := Effects{Wounded: 10, Armor: 2}
	merged := WithGear(base, map[string]int{Wounded: 20, Evasion: 4})
	assert.Equal(t, 30, merged.Int(Wounded), "gear adds to what the class has")
	assert.Equal(t, 4, merged.Int(Evasion))
	assert.Equal(t, 2, merged.Int(Armor))
	assert.Equal(t, 10, base.Int(Wounded), "the cached class effects are never changed")
	assert.Zero(t, base.Int(Evasion))

	assert.Equal(t, 20, WithGear(nil, map[string]int{Wounded: 20}).Int(Wounded), "a character with no class still gets gear effects")
	assert.Equal(t, base, WithGear(base, nil))
}
