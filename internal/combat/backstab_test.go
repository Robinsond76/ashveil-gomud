package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBackstabCritsOnlyALandedBlow (Phase 33e regression): a backstab
// (the automatic Opening Strike and Aimed Shot use it) makes the first blow
// that lands a critical hit, and a round in which no blow landed reports
// no crit at all (it used to report one), without the old shouted prefix.
func TestBackstabCritsOnlyALandedBlow(t *testing.T) {
	edgeSpecs(t)
	landed, missed := 0, 0
	for i := 0; i < 400 && (landed < 3 || missed < 3); i++ {
		src := edgeFighter(90231)
		src.Equipment.Weapon = items.New(edgeDaggerID)
		src.SetAggro(0, 1, characters.BackStab, 0)
		target := edgeFighter(90231)
		res := calculateCombat(*src, *target, User, Mob, 0, 0)
		for _, line := range res.MessagesToSource {
			assert.NotContains(t, line, "BACKSTAB")
		}
		if res.Hit {
			landed++
			assert.True(t, res.Crit, "the first landed blow is a crit")
		} else {
			missed++
			assert.False(t, res.Crit, "no blow landed, so no crit")
		}
	}
	require.Positive(t, landed)
	require.Positive(t, missed)
}
