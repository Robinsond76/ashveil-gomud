package company

import (
	"encoding/json"
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/users"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/modules/gmcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 66 wiring: the bestiary fills from real kills (mobcommands.Suicide
// through the real round), reads through the `bestiary` and `consider`
// commands, and reaches the web client as Char.Bestiary and as the Battle
// view's known habits.

func (b *brawl) fightAll(maxRounds int) string {
	b.t.Helper()
	var seen strings.Builder
	for i := 0; i < maxRounds && len(b.livingBandits()) > 0; i++ {
		b.toughen()
		seen.WriteString(b.fight())
		seen.WriteString("\n")
	}
	require.Empty(b.t, b.livingBandits(), "the fight ran to the end")
	return seen.String()
}

func TestBestiaryFillsFromRealKills(t *testing.T) {
	b := newBrawl(t)
	b.aria.Character.KD.Kills = nil
	b.aria.Character.KD.TotalKills = 0

	assert.Contains(t, b.cmd("bestiary", ""), "Your bestiary is empty")

	b.aimAt("bandit captain")
	seen := b.fightAll(80)
	assert.Contains(t, seen, "Bestiary: bandit captain, its lore is now in your bestiary", "the first kill says what it taught")
	assert.Equal(t, 1, strings.Count(seen, "bandit captain, its lore is now in your bestiary"), "said once")

	list := b.cmd("bestiary", "")
	assert.Contains(t, list, "Your bestiary: ")
	assert.Contains(t, list, "bandit captain")
	assert.Contains(t, list, "bandit cutthroat")
	assert.NotContains(t, list, "hill ogre", "nothing of a kind never beaten")

	entry := b.cmd("bestiary", "bandit captain")
	assert.Contains(t, entry, "Known: lore (1 kill); 2 more kills to learn its defences.")
	assert.Contains(t, entry, "Lore")
	assert.NotContains(t, entry, "Defences")
	assert.NotContains(t, entry, "Habits and weaknesses")

	// The same kills teach more as they pile up (the tally is the
	// character's own, so it also survives a restart).
	b.aria.Character.KD.Kills[9104] = 3
	assert.Contains(t, b.cmd("bestiary", "bandit captain"), "Defences")
	assert.NotContains(t, b.cmd("bestiary", "bandit captain"), "Habits and weaknesses")
	b.aria.Character.KD.Kills[9104] = 6
	assert.Contains(t, b.cmd("bestiary", "bandit captain"), "Habits and weaknesses")
	assert.Contains(t, b.cmd("bestiary", "bandit captain"), "everything is known")
	assert.Contains(t, b.cmd("bestiary", "hill ogre"), `Nothing called "hill ogre" is in your bestiary`)
	assert.Contains(t, b.cmd("bestiary", "brawl"), "bandit captain", "a zone name filters")
}

func TestBestiaryReachesTheWebClientAndTheBattleView(t *testing.T) {
	b, ogre := ogreBrawl(t)
	gmcp.AcceptGMCPForTest(users.GetConnectionId(7)) // a web client
	events.AddToQueue(events.PlayerSpawn{UserId: 7})
	events.ProcessEvents()
	freshEvents(t)
	views := map[int][]map[string]any{}
	var bestiary []map[string]any
	id := events.RegisterListener(gmcp.GMCPOut{}, func(e events.Event) events.ListenerReturn {
		out, ok := e.(gmcp.GMCPOut)
		if !ok || out.UserId != 7 {
			return events.Continue
		}
		var body map[string]any
		switch out.Module {
		case "Company.Battle":
			if raw, ok := out.Payload.([]byte); ok {
				_ = json.Unmarshal(raw, &body)
			}
			views[out.UserId] = append(views[out.UserId], body)
		case "Char.Bestiary":
			raw, _ := json.Marshal(out.Payload)
			_ = json.Unmarshal(raw, &body)
			bestiary = append(bestiary, body)
		}
		return events.Cancel // no connection to deliver to
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(gmcp.GMCPOut{}, id) })

	// Nothing known of the ogre: the view names nothing, and consider
	// says it is new.
	b.aria.Character.KD.Kills = nil
	b.toughen()
	b.fight()
	b.refresh(7)
	require.NotEmpty(t, viewEnemies(lastView(views, 7)), "the battle is on the view")
	for _, e := range viewEnemies(lastView(views, 7)) {
		assert.Nil(t, e["known"], "nothing known of a kind never beaten")
	}
	assert.Contains(t, b.cmd("consider", "hill ogre"), "hill ogre")
	assert.Contains(t, b.cmd("consider", "hill ogre"), "New to you: ")

	// Six kills: its habits are known, in the battle view and on consider.
	b.aria.Character.KD.Kills = map[int]int{9109: 6}
	b.refresh(7)
	foe := viewEnemies(lastView(views, 7))[fmt.Sprintf("m:%d", ogre.InstanceId)]
	require.NotNil(t, foe, "%v", viewEnemies(lastView(views, 7)))
	assert.Contains(t, foe["known"], "winds up crushing blow")
	assert.Contains(t, b.cmd("consider", "hill ogre"), "Known: hill ogre (habits: winds up crushing blow)")

	// Char.Bestiary is sent when asked, then refreshed as battles end.
	events.AddToQueue(gmcp.GMCPCharUpdate{UserId: 7, Identifier: "Char.Bestiary"})
	events.ProcessEvents()
	require.Len(t, bestiary, 1)
	entries := bestiary[0]["entries"].([]any)
	require.Len(t, entries, 1)
	entry := entries[0].(map[string]any)
	assert.Equal(t, "hill ogre", entry["name"])
	assert.EqualValues(t, 3, entry["tier"])
	assert.NotEmpty(t, entry["habits"])
	events.AddToQueue(events.BattleEnded{UserId: 7, Outcome: "won"})
	events.ProcessEvents()
	assert.Len(t, bestiary, 2, "a battle ending refreshes a client that asked")
}
