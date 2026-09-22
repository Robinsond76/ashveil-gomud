package hooks

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

// workedExample mirrors formationcombat's own worked example: A at
// (front, col 1), B at (back, col 1), C at (back, col 2).
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

func TestResolveAttackTargetLegalDirectHitNeedsNoRedirect(t *testing.T) {
	f := workedExample(t)
	final, ok := resolveAttackTarget(1, f, keyA, allAlive(keyA, keyB, keyC), formationcombat.ReachNone)
	require.True(t, ok)
	assert.Equal(t, keyA, final)
}

func TestResolveAttackTargetInterceptsBackRowAttackToFrontRow(t *testing.T) {
	f := workedExample(t)
	final, ok := resolveAttackTarget(1, f, keyB, allAlive(keyA, keyB, keyC), formationcombat.ReachNone)
	require.True(t, ok, "A is alive and blocks column 1, so the attack redirects to A rather than being blocked")
	assert.Equal(t, keyA, final)
}

func TestResolveAttackTargetSkipsWhenFrontIsDeadAndALivingMiddleStillBlocks(t *testing.T) {
	// Interception only ever redirects to the front row. If the front slot
	// is dead (not empty — still occupied by a corpse), no interception
	// applies, and a living middle occupant still blocks plain melee from
	// reaching the back: there is no legal path this round.
	var g company.Formation
	require.NoError(t, g.Place(keyA, 0, 0)) // front, dead
	require.NoError(t, g.Place(keyD, 1, 0)) // middle, alive: blocks the column
	require.NoError(t, g.Place(keyB, 2, 0)) // back: the original target

	alive := map[company.MemberKey]bool{keyD: true, keyB: true} // A is dead

	_, ok := resolveAttackTarget(0, g, keyB, alive, formationcombat.ReachNone)
	assert.False(t, ok)
}

func TestResolveAttackTargetSkipsWhenTargetGoneAndNoFormationEntry(t *testing.T) {
	f := workedExample(t)
	_, ok := resolveAttackTarget(1, f, company.MemberKey("ghost"), allAlive(keyA, keyB, keyC), formationcombat.ReachNone)
	assert.False(t, ok)
}

func TestResolveAttackTargetOutOfLateralRangeSkipsEvenWithInterception(t *testing.T) {
	var g company.Formation
	require.NoError(t, g.Place(keyA, 0, 0))
	_, ok := resolveAttackTarget(2, g, keyA, allAlive(keyA), formationcombat.ReachAny)
	assert.False(t, ok, "column 2 attacker is out of lateral range of column 0, even with ReachAny")
}

func TestEffectiveHPMatchesRankMobsFormula(t *testing.T) {
	// 200 HP, 0 defense: no mitigation, EHP == HP.
	assert.InDelta(t, 200.0, effectiveHP(200, 0), 0.001)
	// 100 HP, 100 defense (50% mitigation): EHP == 200.
	assert.InDelta(t, 200.0, effectiveHP(100, 100), 0.001)
	// Defense clamps at 95% mitigation even for very high defense values.
	assert.InDelta(t, 100.0/0.05, effectiveHP(100, 10000), 0.001)
}
