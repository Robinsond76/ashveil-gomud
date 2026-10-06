package creatures

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFamiliesAreKeptByKind(t *testing.T) {
	hound, ok := ForArchetype(" Hound ")
	assert.True(t, ok)
	assert.True(t, hound.NeedsUpkeep())
	assert.True(t, hound.MendsOnItsOwn())
	assert.False(t, hound.Bound())
	assert.False(t, hound.Repaired())

	golem, ok := ForArchetype(StoneGolem)
	assert.True(t, ok)
	assert.False(t, golem.NeedsUpkeep())
	assert.False(t, golem.MendsOnItsOwn())
	assert.True(t, golem.Bound())
	assert.True(t, golem.Repaired())

	for _, id := range []string{"", "warrior", "wizard"} {
		assert.False(t, Is(id), id)
		assert.Equal(t, Kind(""), KindOf(id))
	}
	assert.Len(t, All(), 2)
}

func TestRepairedHealthMendsHalfWithinTheLimit(t *testing.T) {
	assert.Equal(t, 60, RepairedHealth(10, 100, 100), "half of 100")
	assert.Equal(t, 100, RepairedHealth(80, 100, 100), "never past the limit")
	assert.Equal(t, 90, RepairedHealth(90, 90, 100), "never lowers")
	assert.Equal(t, 7, RepairedHealth(7, 0, 100), "no limit known")
	assert.Equal(t, 2, RepairedHealth(1, 10, 1), "at least one point")
}

func TestCanWearKeepsCreatureGearToCreatures(t *testing.T) {
	for _, tc := range []struct {
		who   string
		cut   []string
		allow bool
	}{
		{"warrior", nil, true},
		{"", nil, true},
		{"warrior", []string{"hound"}, false},
		{"hound", nil, false},
		{"hound", []string{"hound"}, true},
		{"hound", []string{"Hound", "wolf"}, true},
		{"stone-golem", []string{"hound"}, false},
		{"stone-golem", nil, false},
	} {
		ok, why := CanWear(tc.who, tc.cut)
		assert.Equal(t, tc.allow, ok, "%q wearing %v", tc.who, tc.cut)
		assert.Equal(t, !tc.allow, why != "", "a refusal gives its reason")
	}
}
