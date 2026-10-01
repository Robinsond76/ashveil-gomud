package combat

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDamageSuffix(t *testing.T) {
	assert.Equal(t, " (5 damage)", damageSuffix(5, false, 0))
	assert.Equal(t, " (critical hit, 9 damage)", damageSuffix(9, true, 0))
	assert.Equal(t, " (5 damage, 2 absorbed)", damageSuffix(5, false, 2))
	assert.Equal(t, " (critical hit, 9 damage, 1 absorbed)", damageSuffix(9, true, 1))
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
		// Phase 30a: a critical hit also names the status it leaves.
		words := status.Words(res.BuffTarget)
		want := damageSuffix(res.DamageToTarget, res.Crit, 0, words...)
		require.NotEmpty(t, res.MessagesToSource)
		assert.True(t, strings.HasSuffix(res.MessagesToSource[0], want), "attacker hit line %q ends %q", res.MessagesToSource, want)
		assert.True(t, strings.HasSuffix(res.MessagesToSourceRoom[0], want), "room hit line %q ends %q", res.MessagesToSourceRoom, want)
		wantDef := damageSuffix(res.DamageToTarget, res.Crit, res.DamageToTargetReduction, words...)
		assert.True(t, strings.HasSuffix(res.MessagesToTarget[0], wantDef), "defender hit line %q ends %q", res.MessagesToTarget, wantDef)
	}
	require.Positive(t, hits, fmt.Sprintf("hits %d misses %d", hits, misses))
	require.Positive(t, crits, "some rounds must crit")
}

// TestPlayerNamesTakeNoArticle (review fix): a player's name, whatever its
// case, is never given "the"; a mob's lowercase one is.
func TestPlayerNamesTakeNoArticle(t *testing.T) {
	edgeSpecs(t)
	for i := 0; i < 200; i++ {
		src := edgeFighter(90231)
		src.Name = "bob"
		src.Equipment.Weapon = sharpenedItem(edgeSwordID, 0, 0)
		target := edgeFighter(90231)
		target.Name = "bandit captain"
		res := calculateCombat(*src, *target, User, Mob, 0, 0)
		for _, line := range res.MessagesToSourceRoom {
			assert.NotContains(t, line, "the bob", line)
			assert.NotContains(t, line, "The bob", line)
		}
		res = calculateCombat(*target, *src, Mob, User, 0, 0)
		for _, line := range append(append([]string{}, res.MessagesToSource...), res.MessagesToSourceRoom...) {
			assert.NotContains(t, line, "the bob", line)
			assert.NotContains(t, line, "The bob", line)
		}
	}
}

// poolRegexp matches a rendered line of pool (tokens as .*).
func poolRegexp(pool items.MessageOptions) *regexp.Regexp {
	var alts []string
	for _, line := range pool {
		visible := ansiTags.ReplaceAllString(string(line), "")
		alts = append(alts, tokenPattern.ReplaceAllString(regexp.QuoteMeta(visible), ".*"))
	}
	return regexp.MustCompile(`(?i)^(` + strings.Join(alts, "|") + `)$`)
}

var (
	tokenPattern = regexp.MustCompile(`\\\{[a-z]+\\\}`)
	ansiTags     = regexp.MustCompile(`<[^>]*>`)
)

// TestAbsorbedCritReadsAsAMiss (review fix): a critical hit that armor
// takes entirely reads as a miss (the 0% pool, no brackets), as any fully
// blocked blow does, not as a bloody critical with no damage.
func TestAbsorbedCritReadsAsAMiss(t *testing.T) {
	edgeSpecs(t)
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 99239, Name: "test plate", Type: items.Body, DamageReduction: 100000})
	t.Cleanup(func() { items.RemoveTestItemSpec(99239) })
	miss := poolRegexp(items.GetAttackMessage(items.Slashing, 0, false).Together.ToAttacker)

	seen := 0
	for i := 0; i < 4000 && seen < 3; i++ {
		src := edgeFighter(90231)
		src.Name = "Aria"
		src.Equipment.Weapon = items.New(edgeSwordID)
		target := edgeFighter(90231)
		target.Name = "bandit captain"
		target.Equipment.Body = items.New(99239)
		res := calculateCombat(*src, *target, User, Mob, 0, 0)
		if !res.Crit || res.DamageToTarget > 0 {
			continue
		}
		seen++
		require.NotEmpty(t, res.MessagesToSource)
		line := ansiTags.ReplaceAllString(res.MessagesToSource[0], "")
		assert.NotContains(t, line, "(", line)
		assert.Regexp(t, miss, ansiTags.ReplaceAllString(res.MessagesToSource[0], ""), "a miss line")
	}
	require.Positive(t, seen, "an absorbed critical hit happened")
}

// TestCritTextComesFromTheCriticalPool (review gap): a critical hit reads
// from the critical pool; a non-critical one, even a sharpened top roll,
// never does.
func TestCritTextComesFromTheCriticalPool(t *testing.T) {
	edgeSpecs(t)
	critical := poolRegexp(items.GetAttackMessage(items.Slashing, 100, true).Together.ToAttacker)
	suffix := regexp.MustCompile(` \([^)]*\)$`)
	crits, plain := 0, 0
	for i := 0; i < 3000 && (crits < 3 || plain < 50); i++ {
		src := edgeFighter(90231)
		src.Name = "Aria"
		src.Equipment.Weapon = sharpenedItem(edgeSwordID, 1, 5)
		target := edgeFighter(90231)
		target.Name = "bandit captain"
		res := calculateCombat(*src, *target, User, Mob, 0, 0)
		if !res.Hit || res.DamageToTarget <= 0 || len(res.MessagesToSource) == 0 {
			continue
		}
		line := suffix.ReplaceAllString(ansiTags.ReplaceAllString(res.MessagesToSource[0], ""), "")
		if res.Crit {
			crits++
			assert.Regexp(t, critical, line)
		} else {
			plain++
			assert.NotRegexp(t, critical, line)
		}
	}
	require.Positive(t, crits)
	require.Positive(t, plain)
}
