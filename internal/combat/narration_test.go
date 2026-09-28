package combat

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDamageSuffix(t *testing.T) {
	assert.Equal(t, " (5 damage)", damageSuffix(5, false, 0))
	assert.Equal(t, " (critical hit, 9 damage)", damageSuffix(9, true, 0))
	assert.Equal(t, " (5 damage, 2 blocked)", damageSuffix(5, false, 2))
	assert.Equal(t, " (critical hit, 9 damage, 1 blocked)", damageSuffix(9, true, 1))
}

// TestRoundLinesCarryTheirMechanics (Phase 29c): through the real attack
// calculation, every hit line ends in its damage (a crit's names the crit),
// the defender's names what was blocked, misses carry no numbers, and
// nothing is starred.
func TestRoundLinesCarryTheirMechanics(t *testing.T) {
	edgeSpecs(t)
	target := edgeFighter(90231)
	target.Name = "bandit captain"

	crits, hits, misses := 0, 0, 0
	for i := 0; i < 400; i++ {
		src := edgeFighter(90231)
		src.Name = "Aria"
		src.Equipment.Weapon = sharpenedItem(edgeSwordID, 0, 0)
		res := calculateCombat(*src, *target, User, Mob, 0, 0)
		all := append(append(append([]string{}, res.MessagesToSource...), res.MessagesToTarget...), res.MessagesToSourceRoom...)
		for _, line := range all {
			assert.NotContains(t, line, "***")
		}
		if !res.Hit {
			misses++
			continue
		}
		hits++
		if res.Crit {
			crits++
		}
		want := damageSuffix(res.DamageToTarget, res.Crit, 0)
		require.NotEmpty(t, res.MessagesToSource)
		assert.True(t, strings.HasSuffix(res.MessagesToSource[len(res.MessagesToSource)-1], want), "attacker line %q ends %q", res.MessagesToSource, want)
		assert.True(t, strings.HasSuffix(res.MessagesToSourceRoom[len(res.MessagesToSourceRoom)-1], want), "room line %q ends %q", res.MessagesToSourceRoom, want)
		wantDef := damageSuffix(res.DamageToTarget, res.Crit, res.DamageToTargetReduction)
		assert.True(t, strings.HasSuffix(res.MessagesToTarget[len(res.MessagesToTarget)-1], wantDef), "defender line %q ends %q", res.MessagesToTarget, wantDef)
	}
	require.Positive(t, hits, fmt.Sprintf("hits %d misses %d", hits, misses))
	require.Positive(t, crits, "some rounds must crit")
}
