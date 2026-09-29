package company

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/combatpace"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pacedLine is a line Aria was sent, and when.
type pacedLine struct {
	text   string
	at     time.Duration // since the first combat round began
	marked bool          // marked dramatic (a pain or death line)
}

// TestPacedCombatThroughTheRealRound (Phase 29f) fights the brawl to its end
// through the registered entry points: CombatOnCadence on every game round,
// Message_SendMessage delivering, and ReleasePacedCombat on every 50ms turn
// of a fake clock. Aria is sent exactly the lines she would be told unpaced,
// in the same order (so a companion's death notice follows its death line,
// and the closing line and battle summary come last), spread over each
// combat round and never past its end.
func TestPacedCombatThroughTheRealRound(t *testing.T) {
	b := newBrawl(t)
	heard := b.ariaHears() // the order she is told, before any pacing

	start := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	now := start
	t.Cleanup(combatpace.UseForTest(combatpace.New()))
	t.Cleanup(hooks.SetPaceClockForTest(func() time.Time { return now }))
	var sent []pacedLine
	t.Cleanup(hooks.SetWriteTextForTest(func(userId int, text string) {
		if userId == 7 {
			sent = append(sent, pacedLine{text: companyTagPattern.ReplaceAllString(text, ""), at: now.Sub(start), marked: combatpace.Default().Marked(text)})
		}
	}))
	for _, reg := range []struct {
		evt events.Event
		fn  events.Listener
	}{
		{events.NewRound{}, hooks.CombatOnCadence},
		{events.Message{}, hooks.Message_SendMessage},
		{events.NewTurn{}, hooks.ReleasePacedCombat},
		{events.NewRound{}, hooks.IdleMobs},
	} {
		id := events.RegisterListener(reg.evt, reg.fn)
		evt := reg.evt
		t.Cleanup(func() { events.UnregisterListener(evt, id) })
	}

	// A frail companion, so one falls and its death notice is paced too.
	for _, mv := range []string{"move #1 1 1", "move #3 1 2", "move me 1 3", "move #2 2 2", "move #4 3 2"} {
		require.Contains(t, b.cmd("formation", mv), "Placed")
	}
	b.aimAt("bandit captain")
	b.companion(1).Character.Health = 1
	*heard = nil
	sent = nil

	// Game rounds every 4s, turns every 50ms; combat on every second round.
	var round uint64 = 1
	for i := 0; i < 400 && len(b.livingBandits()) > 0; i++ {
		b.aria.Character.HealthMax.Value = 1000
		b.aria.Character.Health = 1000
		round++
		events.AddToQueue(events.NewRound{RoundNumber: round})
		events.ProcessEvents()
		for turn := 0; turn < 80; turn++ {
			now = now.Add(50 * time.Millisecond)
			events.AddToQueue(events.NewTurn{})
			events.ProcessEvents()
		}
	}
	require.Empty(t, b.livingBandits(), "the bandits fall")
	// Let the last round's lines finish.
	for turn := 0; turn < 200; turn++ {
		now = now.Add(50 * time.Millisecond)
		events.AddToQueue(events.NewTurn{})
		events.ProcessEvents()
	}

	var got []string
	for _, l := range sent {
		got = append(got, l.text)
	}
	require.NotEmpty(t, *heard)
	assert.Equal(t, *heard, got, "paced lines are exactly the unpaced ones, in order")

	// The fall of a companion: its notice comes after its death line (the
	// round's other lines may come between, exactly as unpaced).
	fell := -1
	for i, l := range got {
		if strings.Contains(l, "has fallen.") {
			fell = i
			break
		}
	}
	require.Greater(t, fell, 0, "a companion fell in the fight")
	name := strings.TrimSpace(strings.SplitN(got[fell], " has fallen", 2)[0])
	deathLine := false
	for j := fell - 1; j >= 0; j-- {
		deathLine = deathLine || strings.Contains(got[j], name)
	}
	assert.True(t, deathLine, "the death notice follows its death line: %q", got[max(0, fell-4):fell+1])
	// The closing line, then the battle summary.
	summary := -1
	for i, l := range got {
		if strings.Contains(l, "is over ──") {
			summary = i
		}
	}
	require.Greater(t, summary, 0, "the battle summary was sent")
	assert.Contains(t, got[summary-1], "The last bandit", "the closing line comes before the summary")

	// The dramatic beat: every death line, as the player receives it, is
	// marked for the longer gap; a critical hit's own line is not.
	deaths := 0
	for _, l := range sent {
		switch {
		case strings.Contains(l.text, "does not rise") || strings.Contains(l.text, "lies still") || strings.Contains(l.text, "and is still"):
			deaths++
			assert.True(t, l.marked, "death line not marked: %q", l.text)
		case strings.Contains(l.text, "(critical hit"):
			assert.False(t, l.marked, "critical hit line marked: %q", l.text)
		}
	}
	assert.Positive(t, deaths, "death lines were seen")

	// Timing: combat rounds start every 8s (rounds 2, 4, ... begin at
	// 0s, 8s, ...). No line goes out more than 6s into its round, and a
	// round's lines are spread out rather than sent at once.
	perRound := map[int][]time.Duration{}
	for _, l := range sent {
		r := int(l.at / (8 * time.Second))
		offset := l.at - time.Duration(r)*8*time.Second
		perRound[r] = append(perRound[r], offset)
	}
	spread := false
	for r, offs := range perRound {
		for _, off := range offs {
			assert.LessOrEqual(t, off, 6*time.Second+50*time.Millisecond, "round %d line at +%v overruns the window", r, off)
		}
		if len(offs) > 2 && offs[len(offs)-1]-offs[0] >= time.Second {
			spread = true
		}
	}
	assert.True(t, spread, "some round's lines were spread over time")
}

// TestCombatCadenceLeavesOtherRoundsAlone (Phase 29f): with combat on every
// second round, every other round listener (here the company's drift and
// chemistry clock, and a listener registered after combat) still runs every
// round, and the world's round count is never touched by combat.
func TestCombatCadenceLeavesOtherRoundsAlone(t *testing.T) {
	b := newBrawl(t)
	t.Cleanup(combatpace.UseForTest(combatpace.New()))
	var fought []uint64
	unsub := combatstream.Default().Subscribe(func(e combatstream.Event) {
		if len(fought) == 0 || fought[len(fought)-1] != e.Round {
			fought = append(fought, e.Round)
		}
	})
	t.Cleanup(unsub)
	combat := events.RegisterListener(events.NewRound{}, hooks.CombatOnCadence)
	t.Cleanup(func() { events.UnregisterListener(events.NewRound{}, combat) })
	after := 0
	counter := events.RegisterListener(events.NewRound{}, func(events.Event) events.ListenerReturn {
		after++
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.NewRound{}, counter) })

	b.toughen()
	b.aimAt("bandit captain")
	_, every := module.alignmentConfig()
	require.Greater(t, every, 6)
	module.registry.DriftIn = every // no drift tick fires in six rounds
	driftBefore := module.registry.DriftIn
	roundsBefore := util.GetRoundCount()

	for round := uint64(101); round <= 106; round++ {
		b.toughen()
		events.AddToQueue(events.NewRound{RoundNumber: round})
		events.ProcessEvents()
	}

	assert.Equal(t, 6, after, "a listener after combat runs every round")
	assert.Equal(t, driftBefore-6, module.registry.DriftIn, "the drift clock counts every game round")
	assert.Equal(t, roundsBefore, util.GetRoundCount(), "combat never moves the round count")
	assert.Equal(t, []uint64{102, 104, 106}, fought, "combat resolves only on every second round")
}
