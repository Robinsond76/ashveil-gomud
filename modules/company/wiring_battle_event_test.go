package company

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/combatpace"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/modules/gmcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 40e wiring: Company.Battle.Event through the real round (DoCombat
// on the brawl world), the real pacer and the real delivery listeners, as
// the web client receives it.

type eventPayload struct {
	Fight      uint64           `json:"fight"`
	Round      uint64           `json:"round"`
	FightRound uint64           `json:"fight_round"`
	Events     []map[string]any `json:"events"`
	raw        string
	at         time.Duration
}

var eventRefPattern = regexp.MustCompile(`^(me|\?|leader|companion:\d+|m:\d+|u:\d+)$`)

func TestBattleEventsThroughTheRealRound(t *testing.T) {
	b := newBrawl(t)
	t.Cleanup(gmcp.SubscribeBattleEventsForTest(combatstream.Default()))
	gmcp.WebClientForTest(users.GetConnectionId(7))

	start := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	now := start
	t.Cleanup(combatpace.UseForTest(combatpace.New()))
	t.Cleanup(hooks.SetPaceClockForTest(func() time.Time { return now }))
	type step struct {
		at      time.Duration
		text    string
		payload *eventPayload
	}
	var steps []step
	t.Cleanup(hooks.SetWriteTextForTest(func(userId int, text string) {
		if userId == 7 {
			steps = append(steps, step{at: now.Sub(start), text: companyTagPattern.ReplaceAllString(text, "")})
		}
	}))
	freshEvents(t)
	gid := events.RegisterListener(gmcp.GMCPOut{}, func(e events.Event) events.ListenerReturn {
		out, ok := e.(gmcp.GMCPOut)
		if !ok || out.UserId != 7 || out.Module != "Company.Battle.Event" {
			return events.Continue
		}
		raw, err := json.Marshal(out.Payload)
		require.NoError(t, err)
		p := &eventPayload{raw: string(raw), at: now.Sub(start)}
		require.NoError(t, json.Unmarshal(raw, p))
		steps = append(steps, step{at: p.at, payload: p})
		return events.Cancel // no connection to deliver to
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(gmcp.GMCPOut{}, gid) })
	for _, reg := range []struct {
		evt events.Event
		fn  events.Listener
	}{
		{events.NewRound{}, hooks.CombatOnCadence},
		{events.Message{}, hooks.Message_SendMessage},
		{events.CombatData{}, hooks.CombatData_Hold},
		{events.NewTurn{}, hooks.ReleasePacedCombat},
		{events.NewRound{}, hooks.IdleMobs},
	} {
		freshEvents(t)
		id := events.RegisterListener(reg.evt, reg.fn)
		evt := reg.evt
		t.Cleanup(func() { events.UnregisterListener(evt, id) })
	}

	for _, mv := range []string{"move #1 1 1", "move #3 1 2", "move me 1 3", "move #2 2 2", "move #4 3 2"} {
		require.Contains(t, b.cmd("formation", mv), "Placed")
	}
	b.aimAt("bandit captain")
	b.companion(1).Character.Health = 1
	form, ok := enemyparty.CompanyFormation(7)
	require.True(t, ok)
	cells := map[string]bool{}
	for _, row := range form {
		for _, key := range row {
			if key != "" {
				cells[string(key)] = true
			}
		}
	}
	steps = nil
	rounds := combatRoundsToPlay(t, b, &now)

	var payloads []*eventPayload
	for _, s := range steps {
		if s.payload != nil {
			payloads = append(payloads, s.payload)
		}
	}
	require.NotEmpty(t, payloads, "the fight sent events")
	kinds := map[string]int{}
	var fight uint64
	for _, p := range payloads {
		if fight == 0 {
			fight = p.Fight
		}
		assert.Equal(t, fight, p.Fight)
		assert.NotContains(t, p.raw, "health")
		for _, ev := range p.Events {
			kinds[ev["kind"].(string)]++
			for _, key := range []string{"src", "tgt", "prev"} {
				if ref, ok := ev[key].(string); ok {
					assert.Regexp(t, eventRefPattern, ref, "%s of %v", key, ev)
				}
			}
		}
	}
	for _, kind := range []string{"fight-start", "attack", "death", "fight-end"} {
		assert.Positive(t, kinds[kind], "a %s event was sent: %v", kind, kinds)
	}
	assert.Equal(t, 1, kinds["fight-end"])

	// The fight-start roster names the company by the keys of their cells
	// in Company.Battle.positions, the leader included.
	first := payloads[0].Events[0]
	require.Equal(t, "fight-start", first["kind"])
	roster, _ := first["company"].([]any)
	require.NotEmpty(t, roster)
	assert.Contains(t, roster, "leader")
	for _, ref := range roster {
		assert.True(t, cells[ref.(string)], "%v has a cell in Company.Battle", ref)
	}
	last := payloads[len(payloads)-1]
	assert.Equal(t, "fight-end", last.Events[len(last.Events)-1]["kind"])
	assert.Equal(t, "victory", last.Events[len(last.Events)-1]["outcome"])

	// fight_round counts the fight's own rounds from 1 (the battle screen's
	// title), while round is the server's counter.
	assert.Equal(t, uint64(1), payloads[0].FightRound)
	for _, p := range payloads {
		assert.Equal(t, p.Round-payloads[0].Round+1, p.FightRound, "round %d", p.Round)
	}
	assert.Greater(t, payloads[0].Round, uint64(1), "the server's counter is not the fight's")

	// Sequence numbers only rise, across messages.
	var prev float64
	for _, p := range payloads {
		for _, ev := range p.Events {
			seq := ev["seq"].(float64)
			assert.Greater(t, seq, prev)
			prev = seq
		}
	}

	// Every attack is released no earlier than the first line of its round
	// and no later than the round's last line; in a busy round the events
	// are spread over several moments rather than dumped at once.
	byRound := map[int][]step{}
	for _, s := range steps {
		byRound[int(s.at/(8*time.Second))] = append(byRound[int(s.at/(8*time.Second))], s)
	}
	spread := false
	for r, list := range byRound {
		var firstText, lastText time.Duration = -1, -1
		moments := map[time.Duration]bool{}
		for _, s := range list {
			if s.payload == nil {
				if firstText < 0 {
					firstText = s.at
				}
				lastText = s.at
			} else {
				moments[s.at] = true
			}
		}
		if firstText < 0 {
			continue
		}
		for _, s := range list {
			if s.payload != nil && !strings.Contains(s.payload.raw, `"fight-start"`) {
				assert.GreaterOrEqual(t, s.at, firstText, "round %d: data before its narration", r)
				assert.LessOrEqual(t, s.at, lastText, "round %d: data after its narration", r)
			}
		}
		if len(moments) >= 3 {
			spread = true
		}
	}
	assert.True(t, spread, "a busy round's events were spread with its lines")
	assert.Positive(t, rounds)
}

// combatRoundsToPlay runs the real round loop to the end of the fight and
// lets the last lines out, returning how many game rounds it ran.
func combatRoundsToPlay(t *testing.T, b *brawl, now *time.Time) int {
	t.Helper()
	var round uint64 = 1
	n := 0
	for i := 0; i < 400 && len(b.livingBandits()) > 0; i++ {
		b.aria.Character.HealthMax.Value = 1000
		b.aria.Character.Health = 1000
		round++
		n++
		events.AddToQueue(events.NewRound{RoundNumber: round})
		events.ProcessEvents()
		for turn := 0; turn < 80; turn++ {
			*now = now.Add(50 * time.Millisecond)
			events.AddToQueue(events.NewTurn{})
			events.ProcessEvents()
		}
	}
	require.Empty(t, b.livingBandits(), "the bandits fall")
	for turn := 0; turn < 200; turn++ {
		*now = now.Add(50 * time.Millisecond)
		events.AddToQueue(events.NewTurn{})
		events.ProcessEvents()
	}
	return n
}

// TestBattleEventsGoOutAtOnceWithPacingOff: with pace off, the events are
// sent as they happen, in the round's own turn, and the narration is the
// same lines it always was.
func TestBattleEventsGoOutAtOnceWithPacingOff(t *testing.T) {
	b := newBrawl(t)
	t.Cleanup(gmcp.SubscribeBattleEventsForTest(combatstream.Default()))
	gmcp.WebClientForTest(users.GetConnectionId(7))
	t.Cleanup(combatpace.UseForTest(combatpace.New()))
	b.aria.SetConfigOption(combatpace.OptionKey, string(combatpace.Off))

	var got int
	freshEvents(t)
	gid := events.RegisterListener(gmcp.GMCPOut{}, func(e events.Event) events.ListenerReturn {
		if out, ok := e.(gmcp.GMCPOut); ok && out.UserId == 7 && out.Module == "Company.Battle.Event" {
			got++
			return events.Cancel
		}
		return events.Continue
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(gmcp.GMCPOut{}, gid) })
	for _, reg := range []struct {
		evt events.Event
		fn  events.Listener
	}{
		{events.NewRound{}, hooks.CombatOnCadence},
		{events.Message{}, hooks.Message_SendMessage},
		{events.CombatData{}, hooks.CombatData_Hold},
	} {
		freshEvents(t)
		id := events.RegisterListener(reg.evt, reg.fn)
		evt := reg.evt
		t.Cleanup(func() { events.UnregisterListener(evt, id) })
	}

	b.toughen()
	b.aimAt("bandit captain")
	events.AddToQueue(events.NewRound{RoundNumber: 2})
	events.ProcessEvents() // no NewTurn: nothing is paced
	assert.Positive(t, got, "events went out in the round itself")
	assert.False(t, combatpace.Default().Busy(7))
}
