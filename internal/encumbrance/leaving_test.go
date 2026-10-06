package encumbrance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 43a: a cargo waterskin drunk dry leaves an empty one.
func TestConsumeUseLeavingReplacesTheLastUse(t *testing.T) {
	cargo := Cargo{LeaderUserID: 7, Stacks: []CargoStack{{ItemId: 5, Count: 1, Uses: 1}}}
	out, err := cargo.ConsumeUseLeaving(5, 5, 6)
	require.NoError(t, err)
	assert.Equal(t, []CargoStack{{ItemId: 6, Count: 1}}, out.Stacks)

	// Not the last use: the skin stays and nothing is left beside it.
	cargo = Cargo{LeaderUserID: 7, Stacks: []CargoStack{{ItemId: 5, Count: 1, Uses: 3}}}
	out, err = cargo.ConsumeUseLeaving(5, 5, 6)
	require.NoError(t, err)
	assert.Equal(t, []CargoStack{{ItemId: 5, Count: 1, Uses: 2}}, out.Stacks)

	// With nothing to leave, it is used up whole as before.
	cargo = Cargo{LeaderUserID: 7, Stacks: []CargoStack{{ItemId: 5, Count: 1, Uses: 1}}}
	out, err = cargo.ConsumeUse(5, 5)
	require.NoError(t, err)
	assert.Empty(t, out.Stacks)
}
