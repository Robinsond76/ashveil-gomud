package company

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatpace"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
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

// clockRoundStarts records when each combat round of the battle clock
// resolved (since start), by subscribing to the fight stream.
func clockRoundStarts(t *testing.T, now *time.Time, start time.Time) *[]time.Duration {
	t.Helper()
	starts := &[]time.Duration{}
	var last uint64
	unsub := combatstream.Default().Subscribe(func(e combatstream.Event) {
		if e.Round != last {
			last = e.Round
			*starts = append(*starts, now.Sub(start))
		}
	})
	t.Cleanup(unsub)
	return starts
}

// TestPacedCombatThroughTheRealRound (Phase 29f, on the battle clock since
// 82c) fights the brawl to its end through the registered entry points:
// BattleClock and ReleasePacedCombat on every 50ms turn of a fake clock,
// Message_SendMessage delivering. Aria is sent exactly the lines she would
// be told unpaced, in the same order (so a companion's death notice follows
// its death line, and the closing line and battle summary come last), one
// action per beat, and the next round waits for the last one's lines.
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
		{events.CombatReport{}, hooks.CombatReport_Mark},
		{events.NewTurn{}, hooks.BattleClock},
		{events.NewTurn{}, hooks.ReleasePacedCombat},
		{events.NewRound{}, hooks.IdleMobs},
	} {
		freshEvents(t)
		id := events.RegisterListener(reg.evt, reg.fn)
		evt := reg.evt
		t.Cleanup(func() { events.UnregisterListener(evt, id) })
	}
	starts := clockRoundStarts(t, &now, start)

	// A frail companion, so one falls and its death notice is paced too.
	for _, mv := range []string{"move #1 1 1", "move #3 1 2", "move me 1 3", "move #2 2 2", "move #4 3 2"} {
		require.Contains(t, b.cmd("formation", mv), "Placed")
	}
	b.aimAt("bandit captain")
	b.companion(1).Character.Health = 1
	*heard = nil
	sent = nil

	combatRoundsToPlay(t, b, &now)

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

	// Timing (Phase 82c): each round's lines come one beat a turn (1s at
	// normal, 0.25s between a turn's own lines), a round is never squeezed,
	// and the next round resolves only after the last line plus the tail
	// (0.6s) and at least 3s after the round before it.
	require.GreaterOrEqual(t, len(*starts), 2, "the fight took more than one round")
	perRound := make([][]time.Duration, len(*starts))
	for i, l := range sent {
		if i > summary {
			// The summary goes out whole behind its heading, not a beat a line.
			assert.Equal(t, sent[summary].at, l.at, "summary line %q came apart from its heading", l.text)
			continue
		}
		r := 0
		for r+1 < len(*starts) && l.at >= (*starts)[r+1] {
			r++
		}
		perRound[r] = append(perRound[r], l.at)
	}
	beats := 0
	for r, offs := range perRound {
		for i := 1; i < len(offs); i++ {
			gap := offs[i] - offs[i-1]
			// A line due as the round resolves goes out on the next
			// 50ms turn, so a follow-up's 0.25s can read as 0.2s.
			assert.GreaterOrEqual(t, gap, 200*time.Millisecond, "round %d: lines %d and %d came %v apart", r, i-1, i, gap)
			if gap >= time.Second {
				beats++
			}
		}
		if r+1 < len(*starts) {
			assert.GreaterOrEqual(t, (*starts)[r+1]-(*starts)[r], 3*time.Second, "round %d resolved sooner than the minimum round", r+1)
			if len(offs) > 0 {
				assert.GreaterOrEqual(t, (*starts)[r+1]-offs[len(offs)-1], 600*time.Millisecond, "round %d resolved before the last line's tail", r+1)
			}
		}
	}
	assert.Greater(t, beats, 3, "turns came a beat apart")
	last := sent[len(sent)-1].at
	t.Logf("FIGHTLEN | normal pace | %d rounds | %d lines | %.1fs | %.1fs a round", len(*starts), len(sent), last.Seconds(), last.Seconds()/float64(len(*starts)))
}

// TestCombatCadenceLeavesOtherRoundsAlone (Phase 29f; 82c): every round
// listener (here the company's drift and chemistry clock, and a listener
// registered after combat) runs every game round, the world's round count
// is never touched by combat, and a player's fight resolves on the battle
// clock's turns, not on the game rounds, numbered by the combat round
// counter.
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
	freshEvents(t)
	combat := events.RegisterListener(events.NewRound{}, hooks.CombatOnCadence)
	t.Cleanup(func() { events.UnregisterListener(events.NewRound{}, combat) })
	clock := events.RegisterListener(events.NewTurn{}, hooks.BattleClock)
	t.Cleanup(func() { events.UnregisterListener(events.NewTurn{}, clock) })
	after := 0
	freshEvents(t)
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
	assert.Empty(t, fought, "a player's fight is not resolved on the game rounds")

	// The clock resolves it on a turn, numbered from 1.
	events.AddToQueue(events.NewTurn{})
	events.ProcessEvents()
	assert.Equal(t, []uint64{1}, fought, "the first turn resolves the first round")
	assert.Equal(t, roundsBefore, util.GetRoundCount())
}

// TestPacedPlayerDeathStaysInOrder (Phase 29f review finding 2): a slain
// player's death is part of the round that killed them. DoCombat issues
// "suicide" for them (user.Command, inside the round); its "has DIED!"
// broadcast, penalty lines, and move to the land of the dead are held and
// come after the round's lines, paced, instead of cutting the round short.
func TestPacedPlayerDeathStaysInOrder(t *testing.T) {
	b := newBrawl(t)
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
		{events.Broadcast{}, hooks.Broadcast_SendToAll},
		{events.RoomChange{}, hooks.FlushPacedOnRoomChange},
		{events.NewTurn{}, hooks.BattleClock},
		{events.NewTurn{}, hooks.ReleasePacedCombat},
		// The world loop runs a player's queued commands.
		{events.Input{}, func(e events.Event) events.ListenerReturn {
			if in, ok := e.(events.Input); ok && in.UserId > 0 && in.MobInstanceId == 0 {
				c, rest, _ := strings.Cut(in.InputText, " ")
				_, _ = usercommands.TryCommand(strings.ToLower(c), rest, in.UserId, 0)
			}
			return events.Continue
		}},
	} {
		freshEvents(t)
		id := events.RegisterListener(reg.evt, reg.fn)
		evt := reg.evt
		t.Cleanup(func() { events.UnregisterListener(evt, id) })
	}

	b.toughen()
	b.aimAt("bandit captain")
	sent = nil

	// A real combat round (the clock's first, on the first turn), its
	// lines held...
	events.AddToQueue(events.NewTurn{})
	events.ProcessEvents()
	require.True(t, combatpace.Default().Busy(7))
	// ...in which Aria is slain: DoCombat's handleAffected issues her
	// "suicide" within the round.
	events.WithCause(1, func() {
		b.aria.Character.Health = -10
		b.aria.Command(`suicide`)
	})
	events.ProcessEvents()
	assert.Empty(t, sent, "nothing, the death included, goes out before the round's first turn")

	for turn := 0; turn < 900; turn++ { // 45s at the slower normal pace: a round's lines one beat a turn, and the death after them
		now = now.Add(50 * time.Millisecond)
		events.AddToQueue(events.NewTurn{})
		events.ProcessEvents()
	}
	require.NotEmpty(t, sent)
	died := -1
	for i, l := range sent {
		if strings.Contains(l.text, "DIED!") {
			died = i
		}
	}
	require.Greater(t, died, 2, "the death announcement comes after the round's lines: %v", sent)
	// Paced like the rest: at least a follow-up's beat after the line
	// before it.
	assert.GreaterOrEqual(t, sent[died].at-sent[died-1].at, 250*time.Millisecond,
		"the death is paced like the round's other lines: %v after the line before it", sent[died].at-sent[died-1].at)
	for _, l := range sent[:died] {
		assert.NotContains(t, l.text, "You lose", "no penalty line before the announcement")
	}
}

// TestWalkingAwayFlushesHeldLines (Phase 29f): mid-round, Aria's own move
// out of the room (a real `go` command, typed) sends the round's held lines
// at once, before the new room's text; her command's output isn't held.
func TestWalkingAwayFlushesHeldLines(t *testing.T) {
	b := newBrawl(t)
	start := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	now := start
	t.Cleanup(combatpace.UseForTest(combatpace.New()))
	t.Cleanup(hooks.SetPaceClockForTest(func() time.Time { return now }))
	var sent []string
	t.Cleanup(hooks.SetWriteTextForTest(func(userId int, text string) {
		if userId == 7 {
			sent = append(sent, companyTagPattern.ReplaceAllString(text, ""))
		}
	}))
	for _, reg := range []struct {
		evt events.Event
		fn  events.Listener
	}{
		{events.NewRound{}, hooks.CombatOnCadence},
		{events.Message{}, hooks.Message_SendMessage},
		{events.RoomChange{}, hooks.FlushPacedOnRoomChange},
		{events.NewTurn{}, hooks.BattleClock},
		{events.NewTurn{}, hooks.ReleasePacedCombat},
		{events.Input{}, func(e events.Event) events.ListenerReturn {
			if in, ok := e.(events.Input); ok && in.UserId > 0 && in.MobInstanceId == 0 {
				c, rest, _ := strings.Cut(in.InputText, " ")
				_, _ = usercommands.TryCommand(strings.ToLower(c), rest, in.UserId, 0)
			}
			return events.Continue
		}},
	} {
		freshEvents(t)
		id := events.RegisterListener(reg.evt, reg.fn)
		evt := reg.evt
		t.Cleanup(func() { events.UnregisterListener(evt, id) })
	}

	b.toughen()
	b.aimAt("bandit captain")
	sent = nil
	events.AddToQueue(events.NewTurn{}) // the clock resolves the round; its lines are held
	events.ProcessEvents()
	now = now.Add(50 * time.Millisecond)
	events.AddToQueue(events.NewTurn{})
	events.ProcessEvents()
	require.Len(t, sent, 1, "one line out; the rest of the round is held")
	require.True(t, combatpace.Default().Busy(7))

	// Mid-battle she can't walk off, and is told so at once: her own
	// command's output is never held.
	events.AddTyped(events.Input{UserId: 7, InputText: "east"})
	events.ProcessEvents()
	require.Len(t, sent, 2)
	assert.Contains(t, sent[1], "Use retreat or flee to leave it")
	require.True(t, combatpace.Default().Busy(7), "the round is still held")

	// Her battle over (as for a bystander, or once it ends), she walks east.
	b.aria.Character.EndAggro()
	battle.Reset()
	b.aria.Character.ActionPoints = 100
	events.AddTyped(events.Input{UserId: 7, InputText: "east"})
	events.ProcessEvents()

	verge := -1
	for i, l := range sent {
		if strings.Contains(l, "Verge") || strings.Contains(l, "verge") {
			verge = i
			break
		}
	}
	require.Greater(t, verge, 3, "the new room was shown after the held lines: %q", sent)
	assert.False(t, combatpace.Default().Busy(7), "nothing is held once she has gone")
	for _, l := range sent[2:verge] {
		assert.NotContains(t, l, "Verge", "held combat lines come before the new room")
	}
}

// TestPaceOffStillKeepsTheMinimumRound (Phase 82c): a player whose pace is
// off reads each round at once, but the fight still runs on the battle
// clock, a round at least every MinRoundMs; and a clock that starts over
// (a copyover, a restart) resolves its first round on the next turn.
func TestPaceOffStillKeepsTheMinimumRound(t *testing.T) {
	b := newBrawl(t)
	b.aria.SetConfigOption(combatpace.OptionKey, string(combatpace.Off))
	start := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	now := start
	t.Cleanup(combatpace.UseForTest(combatpace.New()))
	t.Cleanup(hooks.SetPaceClockForTest(func() time.Time { return now }))
	var sentAt []time.Duration
	t.Cleanup(hooks.SetWriteTextForTest(func(userId int, text string) {
		if userId == 7 {
			sentAt = append(sentAt, now.Sub(start))
		}
	}))
	for _, reg := range []struct {
		evt events.Event
		fn  events.Listener
	}{
		{events.Message{}, hooks.Message_SendMessage},
		{events.NewTurn{}, hooks.BattleClock},
		{events.NewTurn{}, hooks.ReleasePacedCombat},
	} {
		freshEvents(t)
		id := events.RegisterListener(reg.evt, reg.fn)
		evt := reg.evt
		t.Cleanup(func() { events.UnregisterListener(evt, id) })
	}
	starts := clockRoundStarts(t, &now, start)
	b.toughen()
	b.aimAt("bandit captain")

	for turn := 0; turn < 200 && len(*starts) < 3; turn++ { // 10s
		b.toughen()
		events.AddToQueue(events.NewTurn{})
		events.ProcessEvents()
		now = now.Add(50 * time.Millisecond)
	}
	require.GreaterOrEqual(t, len(*starts), 3, "rounds resolved on the clock")
	assert.Equal(t, time.Duration(0), (*starts)[0], "the first round resolves on the first turn")
	for i := 1; i < len(*starts); i++ {
		gap := (*starts)[i] - (*starts)[i-1]
		assert.GreaterOrEqual(t, gap, 3*time.Second, "round %d came %v after the one before", i+1, gap)
		assert.Less(t, gap, 3*time.Second+200*time.Millisecond, "with nothing held, the round waits only the minimum")
	}
	// Her lines came as each round resolved, none held.
	assert.False(t, combatpace.Default().Busy(7))
	for _, at := range sentAt {
		assert.Contains(t, *starts, at, "a line went out at a round's own moment")
	}

	// The clock starts over (a copyover): the next round is due at once.
	before := len(*starts)
	hooks.ResetBattleClockForTest()
	events.AddToQueue(events.NewTurn{})
	events.ProcessEvents()
	assert.Equal(t, before+1, len(*starts), "a fresh clock resolves on its first turn")
}

// TestAnEmptyRoundIsNotWaitedOut (Phase 82d review): when every fighter is
// slower than tempo 1, the round after the opening turn has no turns; the
// clock does not hold the fight for the minimum round then but resolves the
// next round on the next turn, while rounds with turns keep the minimum.
func TestAnEmptyRoundIsNotWaitedOut(t *testing.T) {
	b := newBrawl(t)
	t.Cleanup(hooks.UseTempoForTest(func(*characters.Character) float64 { return 0.9 }))
	b.aria.SetConfigOption(combatpace.OptionKey, string(combatpace.Off))
	start := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	now := start
	t.Cleanup(combatpace.UseForTest(combatpace.New()))
	t.Cleanup(hooks.SetPaceClockForTest(func() time.Time { return now }))
	var rounds []time.Duration // when Aria was told each "Round N"
	t.Cleanup(hooks.SetWriteTextForTest(func(userId int, text string) {
		if userId == 7 && strings.Contains(text, "Round ") {
			rounds = append(rounds, now.Sub(start))
		}
	}))
	for _, reg := range []struct {
		evt events.Event
		fn  events.Listener
	}{
		{events.Message{}, hooks.Message_SendMessage},
		{events.NewTurn{}, hooks.BattleClock},
		{events.NewTurn{}, hooks.ReleasePacedCombat},
	} {
		freshEvents(t)
		id := events.RegisterListener(reg.evt, reg.fn)
		evt := reg.evt
		t.Cleanup(func() { events.UnregisterListener(evt, id) })
	}
	b.toughen()
	b.aimAt("bandit captain")

	for turn := 0; turn < 200 && len(rounds) < 4; turn++ { // 10s
		b.toughen()
		events.AddToQueue(events.NewTurn{})
		events.ProcessEvents()
		now = now.Add(50 * time.Millisecond)
	}
	require.GreaterOrEqual(t, len(rounds), 4, "rounds resolved on the clock: %v", rounds)
	gaps := make([]time.Duration, 0, 3)
	for i := 1; i < 4; i++ {
		gaps = append(gaps, rounds[i]-rounds[i-1])
	}
	assert.GreaterOrEqual(t, gaps[0], 3*time.Second, "the opening round, with everyone's turn, keeps the minimum: %v", rounds)
	assert.LessOrEqual(t, gaps[1], 100*time.Millisecond, "the empty second round is followed at once: %v", rounds)
	assert.GreaterOrEqual(t, gaps[2], 3*time.Second, "the third round, with turns again, keeps the minimum: %v", rounds)
}
