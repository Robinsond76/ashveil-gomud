package hooks

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatpace"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
)

// TestNarrationPoolsVoice (Phase 29c): every opener and closing is in the
// narration voice, and every keyed pool has a partner.
func TestNarrationPoolsVoice(t *testing.T) {
	for _, pools := range []map[string][]string{fightOpeners, fightClosings} {
		assert.NotEmpty(t, pools[""], "a generic pool")
		for key, pool := range pools {
			assert.NotEmpty(t, pool, key)
			for _, line := range pool {
				assert.Empty(t, util.NarrationVoiceProblem(line), line)
			}
		}
	}
	for key := range fightOpeners {
		assert.Contains(t, fightClosings, key, "a group with an opener has a closing")
	}
}

func TestNarrationPoolByGroup(t *testing.T) {
	assert.Equal(t, fightOpeners["bandits"], narrationPool(fightOpeners, []string{"bandits"}))
	assert.Equal(t, fightClosings["bandits"], narrationPool(fightClosings, []string{"wolves", "bandits"}), "the first group with a pool")
	assert.Equal(t, fightOpeners[""], narrationPool(fightOpeners, []string{"wolves"}), "an unknown group falls back")
	assert.Equal(t, fightOpeners[""], narrationPool(fightOpeners, nil))
	assert.Contains(t, fightClosings["practice-squad"], fightClosing([]string{"practice-squad"}))
	assert.Contains(t, fightOpeners["slum-ruffians"], fightOpener([]string{"slum-ruffians"}))
	assert.Contains(t, fightOpeners[""], fightOpener([]string{""}))
}

// TestShieldBreakLines (review fix): a breaking shield is told in the
// narration voice, no *** and no !, with the owner's article.
func TestShieldBreakLines(t *testing.T) {
	owner := shieldBreaksOwnerLine("wooden shield")
	room := shieldBreaksRoomLine("wooden shield", mobTag("bandit bruiser"))
	player := shieldBreaksRoomLine("wooden shield", userTag("bob"))
	for _, line := range []string{owner, room, player} {
		assert.Empty(t, util.NarrationVoiceProblem(line), line)
		assert.NotContains(t, line, "***")
	}
	assert.Contains(t, room, `The <ansi fg="mobname">bandit bruiser</ansi>'s <ansi fg="item">wooden shield</ansi> cracks apart`)
	assert.Contains(t, player, `<ansi fg="username">Bob</ansi>'s`, "a player's name takes no article")
}

// TestDeathNoticeOnce (review fix): a mob still at 0 health when a later
// round reports it again (its queued suicide not yet run) gets no second
// death line.
func TestDeathNoticeOnce(t *testing.T) {
	room := &rooms.Room{RoomId: 990301}
	rooms.SetTestRoom(room)
	t.Cleanup(func() { rooms.RemoveTestRoom(room.RoomId) })
	m := engagementMob(t, 8301, 0, room.RoomId)
	m.Character.Name = "bandit captain"

	var deaths int
	freshEvents(t)
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if msg := e.(events.Message); msg.RoomId == room.RoomId && strings.Contains(msg.Text, "bandit captain") {
			deaths++
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	events.ProcessEvents()
	deaths = 0

	pacer := combatpace.New()
	t.Cleanup(combatpace.UseForTest(pacer))
	var line string
	freshEvents(t)
	lineId := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if msg := e.(events.Message); msg.RoomId == room.RoomId {
			line = msg.Text
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, lineId) })

	handleAffected(nil, []int{8301})
	handleAffected(nil, []int{8301}) // the next round, suicide still queued
	events.ProcessEvents()
	assert.Equal(t, 1, deaths, "one death line")
	// Phase 29f: a death line waits the longer gap.
	assert.True(t, pacer.Marked(line), "death line marked dramatic: %q", line)
}
