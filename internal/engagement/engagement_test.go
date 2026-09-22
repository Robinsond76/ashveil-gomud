package engagement_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/engagement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func alwaysLegal(attacker, defender engagement.Combatant) bool {
	return true
}

func TestAssignTargetWeakestPicksLowestHP(t *testing.T) {
	attacker := engagement.Combatant{ID: 100, HP: 20}
	candidates := []engagement.Combatant{
		{ID: 1, HP: 30},
		{ID: 2, HP: 5},
		{ID: 3, HP: 15},
	}

	targetID, ok := engagement.AssignTarget(attacker, candidates, engagement.Weakest, alwaysLegal)

	require.True(t, ok)
	assert.Equal(t, 2, targetID)
}

func TestAssignTargetStrongestPicksHighestHP(t *testing.T) {
	attacker := engagement.Combatant{ID: 100, HP: 20}
	candidates := []engagement.Combatant{
		{ID: 1, HP: 30},
		{ID: 2, HP: 5},
		{ID: 3, HP: 15},
	}

	targetID, ok := engagement.AssignTarget(attacker, candidates, engagement.Strongest, alwaysLegal)

	require.True(t, ok)
	assert.Equal(t, 1, targetID)
}

func TestAssignTargetRandomPicksAmongLegalCandidates(t *testing.T) {
	attacker := engagement.Combatant{ID: 100, HP: 20}
	candidates := []engagement.Combatant{
		{ID: 1, HP: 30},
		{ID: 2, HP: 5},
	}

	legalIDs := map[int]bool{1: true, 2: true}

	for i := 0; i < 20; i++ {
		targetID, ok := engagement.AssignTarget(attacker, candidates, engagement.Random, alwaysLegal)
		require.True(t, ok)
		assert.True(t, legalIDs[targetID], "target %d must be one of the candidates", targetID)
	}
}

func TestAssignTargetSkipsDeadCandidates(t *testing.T) {
	attacker := engagement.Combatant{ID: 100, HP: 20}
	candidates := []engagement.Combatant{
		{ID: 1, HP: 0},  // dead, previously the target
		{ID: 2, HP: 12}, // alive, should be picked as the only survivor
	}

	targetID, ok := engagement.AssignTarget(attacker, candidates, engagement.Weakest, alwaysLegal)

	require.True(t, ok)
	assert.Equal(t, 2, targetID)
}

func TestAssignTargetFiltersByLegalFunc(t *testing.T) {
	attacker := engagement.Combatant{ID: 100, HP: 20, Col: 1}
	candidates := []engagement.Combatant{
		{ID: 1, HP: 10, Col: 0}, // in lateral range
		{ID: 2, HP: 5, Col: 2},  // in lateral range, weaker, but marked illegal below
	}

	// Only candidate 1 is "legal" in this stub, even though candidate 2 is
	// weaker — proves AssignTarget respects the injected predicate over
	// pure HP ordering.
	onlyFirstLegal := func(attacker, defender engagement.Combatant) bool {
		return defender.ID == 1
	}

	targetID, ok := engagement.AssignTarget(attacker, candidates, engagement.Weakest, onlyFirstLegal)

	require.True(t, ok)
	assert.Equal(t, 1, targetID)
}

func TestAssignTargetNoLegalTargetReturnsFalseWithoutPanicking(t *testing.T) {
	attacker := engagement.Combatant{ID: 100, HP: 20}
	candidates := []engagement.Combatant{
		{ID: 1, HP: 10},
		{ID: 2, HP: 5},
	}

	neverLegal := func(attacker, defender engagement.Combatant) bool {
		return false
	}

	targetID, ok := engagement.AssignTarget(attacker, candidates, engagement.Weakest, neverLegal)

	assert.False(t, ok)
	assert.Equal(t, 0, targetID)
}

func TestAssignTargetEmptyCandidatesReturnsFalse(t *testing.T) {
	attacker := engagement.Combatant{ID: 100, HP: 20}

	targetID, ok := engagement.AssignTarget(attacker, nil, engagement.Weakest, alwaysLegal)

	assert.False(t, ok)
	assert.Equal(t, 0, targetID)
}

func TestPartyAliveTrueWhenAnyMemberHasPositiveHP(t *testing.T) {
	assert.True(t, engagement.PartyAlive([]engagement.Combatant{
		{ID: 1, HP: 0},
		{ID: 2, HP: 3},
	}))
}

func TestPartyAliveFalseWhenAllMembersDead(t *testing.T) {
	assert.False(t, engagement.PartyAlive([]engagement.Combatant{
		{ID: 1, HP: 0},
		{ID: 2, HP: 0},
	}))
}

func TestPartyAliveFalseWhenNoMembers(t *testing.T) {
	assert.False(t, engagement.PartyAlive(nil))
}
