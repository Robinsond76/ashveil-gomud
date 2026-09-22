package formationcombat_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	keyA = company.MemberKey("a")
	keyB = company.MemberKey("b")
	keyC = company.MemberKey("c")
	keyD = company.MemberKey("d")
)

// workedExample builds the spec's own worked example: A at (front, col 1),
// B at (back, col 1), C at (back, col 2).
func workedExample(t *testing.T) company.Formation {
	t.Helper()
	var f company.Formation
	require.NoError(t, f.Place(keyA, 0, 1))
	require.NoError(t, f.Place(keyB, 2, 1))
	require.NoError(t, f.Place(keyC, 2, 2))
	return f
}

func allAlive(keys ...company.MemberKey) map[company.MemberKey]bool {
	m := make(map[company.MemberKey]bool, len(keys))
	for _, k := range keys {
		m[k] = true
	}
	return m
}

func TestFrontmostOccupantSkipsDeadAndEmptyCells(t *testing.T) {
	f := workedExample(t)

	row, ok := formationcombat.FrontmostOccupant(f, 1, allAlive(keyA, keyB))
	require.True(t, ok)
	assert.Equal(t, 0, row, "A (front) blocks column 1")

	row, ok = formationcombat.FrontmostOccupant(f, 1, allAlive(keyB)) // A dead
	require.True(t, ok)
	assert.Equal(t, 2, row, "B becomes frontmost once A is gone")

	_, ok = formationcombat.FrontmostOccupant(f, 0, allAlive(keyA, keyB, keyC))
	assert.False(t, ok, "column 0 is empty in the worked example")
}

func TestInLateralRangeEdgeColumns(t *testing.T) {
	cases := []struct {
		attackerCol, defenderCol int
		want                     bool
	}{
		{0, 0, true}, {0, 1, true}, {0, 2, false},
		{1, 0, true}, {1, 1, true}, {1, 2, true},
		{2, 0, false}, {2, 1, true}, {2, 2, true},
	}
	for _, c := range cases {
		got := formationcombat.InLateralRange(c.attackerCol, c.defenderCol)
		assert.Equal(t, c.want, got, "attackerCol=%d defenderCol=%d", c.attackerCol, c.defenderCol)
	}
}

func TestInReachDepthNoneOnlyFrontmost(t *testing.T) {
	assert.True(t, formationcombat.InReachDepth(0, 0, formationcombat.ReachNone))
	assert.False(t, formationcombat.InReachDepth(0, 1, formationcombat.ReachNone))
	assert.False(t, formationcombat.InReachDepth(0, 2, formationcombat.ReachNone))
}

func TestInReachDepthExtendedFrontOrMiddleNotBack(t *testing.T) {
	assert.True(t, formationcombat.InReachDepth(0, 0, formationcombat.ReachExtended))
	assert.True(t, formationcombat.InReachDepth(0, 1, formationcombat.ReachExtended))
	assert.False(t, formationcombat.InReachDepth(0, 2, formationcombat.ReachExtended))
}

func TestInReachDepthAnyIgnoresDepth(t *testing.T) {
	assert.True(t, formationcombat.InReachDepth(0, 0, formationcombat.ReachAny))
	assert.True(t, formationcombat.InReachDepth(0, 1, formationcombat.ReachAny))
	assert.True(t, formationcombat.InReachDepth(0, 2, formationcombat.ReachAny))
}

func TestLegalWorkedExamplePlainMelee(t *testing.T) {
	f := workedExample(t)
	alive := allAlive(keyA, keyB, keyC)

	// Attacker in column 1: legal against A and C, illegal against B
	// (blocked by A in the same column).
	assert.True(t, formationcombat.Legal(1, f, keyA, alive, formationcombat.ReachNone))
	assert.True(t, formationcombat.Legal(1, f, keyC, alive, formationcombat.ReachNone))
	assert.False(t, formationcombat.Legal(1, f, keyB, alive, formationcombat.ReachNone))
}

func TestLegalSelfHealsWhenBlockerDies(t *testing.T) {
	f := workedExample(t)
	aliveWithA := allAlive(keyA, keyB, keyC)
	aliveWithoutA := allAlive(keyB, keyC) // A died; formation itself is unchanged

	assert.False(t, formationcombat.Legal(1, f, keyB, aliveWithA, formationcombat.ReachNone),
		"B is blocked while A is alive")
	assert.True(t, formationcombat.Legal(1, f, keyB, aliveWithoutA, formationcombat.ReachNone),
		"B becomes legal automatically once A dies, with no formation change")
}

func TestLegalReachExtendedHitsMiddleNotBack(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(keyA, 0, 0)) // front
	require.NoError(t, f.Place(keyD, 1, 0)) // middle
	require.NoError(t, f.Place(keyB, 2, 0)) // back
	alive := allAlive(keyA, keyB, keyD)

	assert.True(t, formationcombat.Legal(0, f, keyA, alive, formationcombat.ReachExtended))
	assert.True(t, formationcombat.Legal(0, f, keyD, alive, formationcombat.ReachExtended), "extended reach hits the middle occupant")
	assert.False(t, formationcombat.Legal(0, f, keyB, alive, formationcombat.ReachExtended), "still blocked from the back slot")
}

func TestLegalRangedIgnoresColumnDepth(t *testing.T) {
	var f company.Formation
	require.NoError(t, f.Place(keyA, 0, 0)) // front
	require.NoError(t, f.Place(keyD, 1, 0)) // middle
	require.NoError(t, f.Place(keyB, 2, 0)) // back
	alive := allAlive(keyA, keyB, keyD)

	assert.True(t, formationcombat.Legal(0, f, keyA, alive, formationcombat.ReachAny))
	assert.True(t, formationcombat.Legal(0, f, keyD, alive, formationcombat.ReachAny))
	assert.True(t, formationcombat.Legal(0, f, keyB, alive, formationcombat.ReachAny), "ranged ignores blocking entirely")
}

func TestLegalOutOfLateralRangeIsIllegal(t *testing.T) {
	var g company.Formation
	require.NoError(t, g.Place(keyA, 0, 0))
	assert.False(t, formationcombat.Legal(2, g, keyA, allAlive(keyA), formationcombat.ReachAny),
		"column 2 attacker is out of lateral range of column 0")
}

func TestLegalDeadOrMissingTargetIsIllegal(t *testing.T) {
	f := workedExample(t)
	assert.False(t, formationcombat.Legal(1, f, keyA, allAlive(keyB, keyC), formationcombat.ReachAny), "A is dead")
	assert.False(t, formationcombat.Legal(1, f, company.MemberKey("ghost"), allAlive(keyA, keyB, keyC), formationcombat.ReachAny), "not in the formation at all")
}

func TestInterceptFrontRowRedirectsToSameColumnFrontRowMember(t *testing.T) {
	f := workedExample(t)
	alive := allAlive(keyA, keyB, keyC)

	interceptor, ok := formationcombat.InterceptFrontRow(f, keyB, alive)
	require.True(t, ok)
	assert.Equal(t, keyA, interceptor)
}

func TestInterceptFrontRowNoInterceptionWhenTargetIsAlreadyFrontRow(t *testing.T) {
	f := workedExample(t)
	alive := allAlive(keyA, keyB, keyC)

	_, ok := formationcombat.InterceptFrontRow(f, keyA, alive)
	assert.False(t, ok, "attacking the front row directly needs no redirect")
}

func TestInterceptFrontRowOriginalTargetStandsWithNoLivingFrontRow(t *testing.T) {
	f := workedExample(t)
	alive := allAlive(keyB, keyC) // A dead: column 1's front slot is empty

	_, ok := formationcombat.InterceptFrontRow(f, keyB, alive)
	assert.False(t, ok, "no living front-row member in that column: attack proceeds against B")
}

func TestLegalTargetsListsExactlyTheLegalMembers(t *testing.T) {
	f := workedExample(t)
	alive := allAlive(keyA, keyB, keyC)

	targets := formationcombat.LegalTargets(1, f, alive, formationcombat.ReachNone)

	assert.ElementsMatch(t, []company.MemberKey{keyA, keyC}, targets)
}
