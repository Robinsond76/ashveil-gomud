package mobs

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShippedPoisonSusceptibility (Phase 43b): weapon poison has nothing to
// take hold of in the dead, constructs and practice dummies, and little in
// plants; ordinary creatures are normal. Marked per template, not inferred
// from names.
func TestShippedPoisonSusceptibility(t *testing.T) {
	loadShippedPronounData(t)
	for _, tc := range []struct {
		id   MobId
		want string
		what string
	}{
		{15, items.PoisonImmune, "skeleton"},
		{14, items.PoisonImmune, "lich"},
		{34, items.PoisonResistant, "ent"},
		{85, items.PoisonNormal, "forest ogre"},
	} {
		spec := GetMobSpec(tc.id)
		require.NotNil(t, spec, tc.what)
		assert.Equal(t, tc.want, spec.PoisonSusceptibility, tc.what)
	}
}
