package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 62: `why` reads a leader's roll log back in plain words, and its
// help page and the pages that point at it render.

func TestWhyExplainsTheLatestRounds(t *testing.T) {
	useWorld(t, "default")
	user := users.NewUserRecord(7, 71)
	user.Character.Name = "Aria"
	log := combatstream.DefaultRollLog()
	log.Clear(user.UserId)
	t.Cleanup(func() { log.Clear(user.UserId) })

	out := func(arg string) string {
		text := heard(t, func() {
			_, err := Why(arg, user, nil, 0)
			require.NoError(t, err)
		})
		return tagPattern.ReplaceAllString(text, "")
	}

	assert.Contains(t, out(""), "There is no battle line to explain yet")

	aria := combatstream.Ref{UserId: user.UserId, Name: "Aria"}
	foe := combatstream.Ref{MobInstanceId: 21, Name: "the cutthroat captain"}
	log.Add(user.UserId, combatstream.Roll{Round: 1, Source: aria, Target: foe, Outcome: combatstream.OutcomeHit, Damage: 5, Ours: true,
		Strikes: []combatstream.Strike{{Chance: 60, Base: 60, Roll: 11, Hit: true, Raw: 6, Armor: 2, ArmorTook: 1, Reduced: 1, Damage: 5}}})
	log.Add(user.UserId, combatstream.Roll{Round: 2, Source: foe, Target: aria, Outcome: combatstream.OutcomeMiss,
		Strikes: []combatstream.Strike{{Chance: 38, Base: 38, Roll: 80}}})

	text := out("")
	assert.Contains(t, text, "The last 2 rounds, newest first")
	assert.Contains(t, text, "1. Round 2: the cutthroat captain -> You, missed.")
	assert.Contains(t, text, "To hit: 38 in 100, rolled 81 (it missed).")
	assert.Contains(t, text, "2. Round 1: You -> the cutthroat captain, hit for 5.")
	assert.Contains(t, text, "Damage: 6 before armor, armor 2 took 1, 5 got through.")

	list := out("list")
	assert.Contains(t, list, " 1. Round 2:")
	assert.Contains(t, list, " 2. Round 1:")
	assert.NotContains(t, list, "To hit", "the list is headings only")

	one := out("2")
	assert.Contains(t, one, "Round 1: You -> the cutthroat captain, hit for 5.")
	assert.NotContains(t, one, "Round 2")

	assert.Contains(t, out("9"), "Only the last 2 rounds are kept")
	assert.Contains(t, out("nonsense"), "Type why for the latest rounds")
}

func TestBattlelogHelpRenders(t *testing.T) {
	useWorld(t, "default")
	text, err := GetHelpContents("battlelog")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{"Help for battlelog", "why list", "why [number]", "To hit", "Last rounds"} {
		assert.Contains(t, plain, want)
	}
	for _, hub := range []string{"combat", "battle-summary", "narration", "webclient"} {
		page, err := GetHelpContents(hub)
		require.NoError(t, err, hub)
		assert.Contains(t, page, "battlelog", "help %s points at battlelog", hub)
	}
	summary, err := GetHelpContents("battle-summary")
	require.NoError(t, err)
	for _, want := range []string{"Damage taken", "Never landed", "Moves", "Sigil"} {
		assert.Contains(t, summary, want, "the summary page documents its "+want+" line")
	}
}
