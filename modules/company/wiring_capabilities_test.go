package company

import (
	"encoding/json"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/modules/gmcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCapabilityFeedTracksRealStrategyRanksAndSpells(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("warrior")
	gmcp.AcceptGMCPForTest(b.aria.ConnectionId())
	b.aria.Character.SetSkill("brawling", 1)
	var received struct {
		Automatic []struct {
			Name, Skill, Reason string
			Enabled             bool
		}
	}
	count := 0
	freshEvents(t)
	listener := events.RegisterListener(gmcp.GMCPOut{}, func(e events.Event) events.ListenerReturn {
		out := e.(gmcp.GMCPOut)
		if out.Module == "Char.Capabilities" {
			assert.Equal(t, 7, out.UserId)
			require.NoError(t, json.Unmarshal(out.Payload.([]byte), &received))
			count++
		}
		return events.Cancel
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(gmcp.GMCPOut{}, listener) })
	refresh := func() { companyview.RefreshUser(7); events.ProcessEvents() }
	refresh()
	require.Len(t, received.Automatic, 1)
	assert.Equal(t, "Tackle", received.Automatic[0].Name)
	assert.True(t, received.Automatic[0].Enabled)
	before := count
	refresh()
	assert.Equal(t, before, count)
	b.cmd("strategy", "you abilities off")
	refresh()
	assert.False(t, received.Automatic[0].Enabled)
	assert.Contains(t, received.Automatic[0].Reason, "disabled")
	b.aria.Character.SetSkill("brawling", 0)
	refresh()
	assert.Empty(t, received.Automatic)
	b.withArchetypes("wizard")
	b.aria.Character.SetSkill("cast", 1)
	b.aria.Character.SpellBook = map[string]int{"mm": 1, "tend": 1}
	b.aria.Character.ManaMax.Value, b.aria.Character.Mana = 100, 100
	b.cmd("strategy", "you caster")
	refresh()
	require.Len(t, received.Automatic, 1, "tend is not configured for automatic battle use")
	assert.Equal(t, "cast", received.Automatic[0].Skill)
	assert.True(t, received.Automatic[0].Enabled, "NoAbilities governs class abilities, not spells")
	b.cmd("strategy", "you reserve 90")
	b.aria.Character.Mana = 90
	refresh()
	assert.Equal(t, "Mana reserve prevents casting", received.Automatic[0].Reason)
	b.cmd("strategy", "you fighter")
	refresh()
	assert.Equal(t, "Requires caster strategy role", received.Automatic[0].Reason)
	before = count
	events.AddToQueue(gmcp.GMCPCompanyRequest{UserId: 7})
	events.ProcessEvents()
	assert.Greater(t, count, before)
}
