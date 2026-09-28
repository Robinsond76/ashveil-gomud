package company

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var damageSuffix = regexp.MustCompile(`\((critical hit, )?(\d+) damage(, \d+ blocked)?\)`)

// poolPattern matches any line of a pool of "%s" lines.
func poolPattern(pool []string) *regexp.Regexp {
	var alts []string
	for _, line := range pool {
		alts = append(alts, strings.ReplaceAll(regexp.QuoteMeta(line), "%s", ".*"))
	}
	return regexp.MustCompile(`^(` + strings.Join(alts, "|") + `)$`)
}

// Shipped narration lines, as the transcript shows them (tags stripped).
var (
	banditOpeners = []string{
		`The bandits spread out, blades low, and close in.`,
		`A sharp whistle, and the bandits come on.`,
		`The bandits trade a look, and draw.`,
	}
	banditClosings = []string{
		`The last bandit goes down, and no one is left to run.`,
		`The last bandit falls, and it is quiet again.`,
	}
)

// narratedFight runs the full 5v5 brawl to its end and returns its lines
// and the stream's events.
func narratedFight(t *testing.T) ([]string, []combatstream.Event) {
	b := newBrawl(t)
	events := b.listen()
	opening := b.cmd("attack", "bandit captain")
	assert.NotContains(t, opening, "prepares to fight")
	lines := strings.Split(opening+"\n"+b.fightItOut(200), "\n")
	return lines, *events
}

// TestNarrationThroughTheRealRound (Phase 29c): over the full 5v5 through
// hooks.DoCombat with the shipped config, every hit carries its damage at
// the end of a line, nothing is starred, shouted, or tagged, one opener
// starts each fight and one closing line ends each won fight after the
// last death line, and a retarget reads "turns toward".
func TestNarrationThroughTheRealRound(t *testing.T) {
	lines, stream := narratedFight(t)
	transcript := strings.Join(lines, "\n")

	assert.NotContains(t, transcript, "***")
	assert.NotContains(t, transcript, "prepares to fight")
	assert.NotContains(t, transcript, "♥")
	assert.NotContains(t, transcript, " turns on ")
	assert.NotContains(t, transcript, "has died")
	assert.Contains(t, transcript, " turns toward ", "someone takes a new target")

	// Each hit the stream reports is narrated with its damage, and every
	// damage suffix closes its line.
	narrated := map[string]int{}
	for _, line := range lines {
		for _, m := range damageSuffix.FindAllStringSubmatchIndex(line, -1) {
			assert.Equal(t, len(strings.TrimRight(line, " ")), m[1], "the suffix ends its line: %q", line)
			key := line[m[4]:m[5]] // N
			if m[2] >= 0 {
				key = "critical hit, " + key
			}
			narrated[key]++
		}
	}
	hits := map[string]int{}
	starts, victories := 0, 0
	for _, e := range stream {
		switch {
		case e.Kind == combatstream.Attack && e.Damage > 0:
			key := fmt.Sprint(e.Damage)
			if e.Crit {
				key = "critical hit, " + key
			}
			hits[key]++
		case e.Kind == combatstream.FightStart:
			starts++
		case e.Kind == combatstream.FightEnd && e.Outcome == combatstream.OutcomeVictory:
			victories++
		}
	}
	require.NotEmpty(t, hits)
	for key, n := range hits {
		assert.GreaterOrEqual(t, narrated[key], n, "every hit of (%s damage) is narrated", key)
	}

	// One opener per fight, one closing per won fight, after the deaths.
	opener, closing, death := poolPattern(banditOpeners), poolPattern(banditClosings), poolPattern(combat.DeathLines)
	var openers, closings, lastDeath, firstClosing = 0, 0, -1, -1
	for i, line := range lines {
		switch {
		case opener.MatchString(line):
			openers++
		case closing.MatchString(line):
			closings++
			if firstClosing < 0 {
				firstClosing = i
			}
		case death.MatchString(line):
			lastDeath = i
		}
	}
	require.Positive(t, starts)
	assert.Equal(t, starts, openers, "one opener per fight")
	require.Positive(t, victories)
	assert.Equal(t, victories, closings, "one closing line per won fight")
	assert.Positive(t, lastDeath, "the bandits' deaths are narrated")
	if victories == 1 {
		assert.Greater(t, firstClosing, lastDeath, "the closing line follows the last death line:\n%s", transcript)
	}
}

// TestCriticalHitNarration: a critical hit in the real round reads as one.
// Crits are chance, so a few fights are run until one lands.
func TestCriticalHitNarration(t *testing.T) {
	for try := 0; try < 6; try++ {
		var crit bool
		t.Run(fmt.Sprint("fight ", try), func(t *testing.T) {
			lines, stream := narratedFight(t)
			crit = slices.ContainsFunc(stream, func(e combatstream.Event) bool {
				return e.Kind == combatstream.Attack && e.Crit && e.Damage > 0
			})
			if crit {
				assert.Contains(t, strings.Join(lines, "\n"), "(critical hit, ")
			}
		})
		if crit {
			return
		}
	}
	t.Fatal("no critical hit landed in six fights")
}
