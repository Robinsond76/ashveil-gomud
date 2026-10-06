package company

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var damageSuffix = regexp.MustCompile(`\(((?:glancing|telling), )?(critical hit, )?(\d+) damage(, \d+ absorbed)?(, [a-z ]+)*\)`)

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
	opening := b.cmd("attack", "bandit cutthroats")
	assert.Contains(t, opening, "You go for the band of bandit cutthroats.")
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
			key := line[m[6]:m[7]] // N
			if m[4] >= 0 {
				key = "critical hit, " + key
			}
			if m[2] >= 0 { // Phase 35d: a glancing or telling blow is named first
				key = line[m[2]:m[3]] + key
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
			if e.Quality != "" { // Phase 35d: the line names a glancing or telling blow
				key = e.Quality + ", " + key
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
				// A crit can also be glancing or telling (Phase 35d), which
				// the suffix names first: "(glancing, critical hit, 3 damage)".
				assert.Regexp(t, `\((glancing, |telling, )?critical hit, `, strings.Join(lines, "\n"))
			}
		})
		if crit {
			return
		}
	}
	t.Fatal("no critical hit landed in six fights")
}

// TestTwoPlayersOneGroupOneOpener (review fix): when a second player joins
// the fight against a group Aria's company is already fighting, the room
// hears no second opener, and one closing line when the group falls.
func TestTwoPlayersOneGroupOneOpener(t *testing.T) {
	b := newBrawl(t)
	room := b.ariaHears()
	b.cmd("attack", "bandit cutthroats")
	b.toughen()
	b.fight()

	brom := users.NewUserRecord(8, 2)
	brom.Username = "brom"
	brom.Password = "$2a$test"
	brom.Character.Name = "Brom"
	brom.Character.RaceId = 1
	brom.Character.Level = 3
	brom.Character.RoomId = b.road.RoomId
	brom.Character.Validate()
	users.SetTestUser(brom)
	b.road.AddPlayer(brom.UserId)
	t.Cleanup(func() { b.road.RemovePlayer(8) })
	*b.messages = nil
	_, err := usercommands.TryCommand("attack", "bandit cutthroats", 8, events.CmdSkipScripts)
	require.NoError(t, err)
	events.ProcessEvents()
	require.NotNil(t, brom.Character.Aggro, "Brom joins by the group's name: %q", *b.messages)

	shared := false
	for i := 0; i < 200 && len(b.livingBandits()) > 0; i++ {
		b.toughen()
		brom.Character.HealthMax.Value = 1000
		brom.Character.Health = 1000
		b.fight()
		aria, okA := battle.Current(7)
		bromB, okB := battle.Current(8)
		if okA && okB && aria.PartyID == bromB.PartyID {
			shared = true
		}
	}
	require.True(t, shared, "both players fought the one group")
	b.fight()

	lines := strings.Split(strings.Join(*room, ""), "\n")
	opener, closing := poolPattern(banditOpeners), poolPattern(banditClosings)
	openers, closings := 0, 0
	for _, line := range lines {
		if opener.MatchString(line) {
			openers++
		}
		if closing.MatchString(line) {
			closings++
		}
	}
	assert.Equal(t, 1, openers, "one opener for the one fight")
	assert.Equal(t, 1, closings, "one closing line when the group falls")
}
