package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39i: a Tempest Lord's Storm wall chains Lightning through the aim's
// row, nearest foe first, and no other row.
func TestRowFoesAreTheAimsRowNearestFirst(t *testing.T) {
	var f company.Formation
	place := func(id, row, col int) {
		require.NoError(t, f.Place(mobparty.MemberKeyFor(id), row, col))
	}
	place(1, 0, 0)
	place(2, 0, 2)
	place(3, 0, 1)
	place(4, 1, 1)
	g := enemyparty.Group{Party: mobparty.Party{Members: []int{1, 2, 3, 4}, Formation: f}}
	assert.Equal(t, []int{3, 2}, rowFoes(g, 1, []int{1, 2, 3, 4}), "the other two of row 0, the nearer first")
	assert.Equal(t, []int{3, 1}, rowFoes(g, 2, []int{1, 3, 4}), "only those listed, nearest first")
	assert.Empty(t, rowFoes(g, 4, []int{1, 2, 3, 4}), "alone in its row")
	assert.Empty(t, rowFoes(g, 9, []int{1, 2}), "an unplaced aim has no row")
}

// Review fix: a Tempest Lord whose aim stands alone in its row still chains
// Lightning to a second foe, as its Full fork rank promises.
func TestStormWallStillChainsWhenTheAimIsAloneInItsRow(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(mobparty.MemberKeyFor(1), 0, 0))
	require.NoError(t, f.Place(mobparty.MemberKeyFor(2), 0, 2))
	require.NoError(t, f.Place(mobparty.MemberKeyFor(4), 1, 1))
	g := enemyparty.Group{Party: mobparty.Party{Members: []int{1, 2, 4}, Formation: f}}
	assert.Equal(t, []int{2}, stormWall(g, 1, []int{1, 2, 4}), "the row")
	assert.Equal(t, []int{1}, stormWall(g, 4, []int{1, 2, 4}), "alone in its row: a second foe all the same")
	assert.Empty(t, stormWall(g, 4, []int{4}), "no other foe")
}
