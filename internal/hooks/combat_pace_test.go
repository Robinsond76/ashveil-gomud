package hooks

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatpace"
	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// paceRig is one online player whose output and prompts are captured, a
// fresh pacer, and a clock the test moves.
type paceRig struct {
	t       *testing.T
	user    *users.UserRecord
	now     time.Time
	got     []string
	prompts []string
}

func newPaceRig(t *testing.T) *paceRig {
	t.Helper()
	r := &paceRig{t: t, now: time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)}
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	r.user = users.NewUserRecord(41, 1)
	r.user.Character.Name = "Aria"
	users.SetTestUser(r.user)

	t.Cleanup(combatpace.UseForTest(combatpace.New()))
	t.Cleanup(SetPaceClockForTest(func() time.Time { return r.now }))
	t.Cleanup(SetWriteTextForTest(func(userId int, text string) {
		if userId == r.user.UserId {
			r.got = append(r.got, strings.TrimSpace(text))
		}
	}))
	t.Cleanup(SetWritePromptForTest(func(userId int, prompt string) {
		r.prompts = append(r.prompts, prompt)
	}))
	for _, reg := range []struct {
		evt events.Event
		fn  events.Listener
	}{
		{events.Message{}, Message_SendMessage},
		{events.NewTurn{}, ReleasePacedCombat},
		{events.RedrawPrompt{}, RedrawPrompt_SendRedraw},
		{events.RoomChange{}, FlushPacedOnRoomChange},
		{events.PlayerDespawn{}, FlushPacedOnDespawn},
	} {
		freshEvents(t)
		id := events.RegisterListener(reg.evt, reg.fn)
		evt := reg.evt
		t.Cleanup(func() { events.UnregisterListener(evt, id) })
	}
	events.ProcessEvents()
	return r
}

// round runs a stand-in combat round n through the shared round path of
// the cadence and the battle clock: it sends the lines, as DoCombat's sends
// would.
func (r *paceRig) round(n uint64, lines ...string) {
	r.t.Helper()
	combatRoundCounter.Store(n - 1)
	resolveCombatRound(func(events.Event) events.ListenerReturn {
		for _, l := range lines {
			r.user.SendText(l)
		}
		return events.Continue
	})
	events.ProcessEvents()
}

// advance moves the clock in 50ms turns, releasing due lines.
func (r *paceRig) advance(d time.Duration) {
	for end := r.now.Add(d); r.now.Before(end); {
		r.now = r.now.Add(50 * time.Millisecond)
		events.AddToQueue(events.NewTurn{})
		events.ProcessEvents()
	}
}

func TestCombatOnCadenceRunsOnlyOnDueRounds(t *testing.T) {
	ResetBattleClockForTest()
	t.Cleanup(ResetBattleClockForTest)
	var ran []uint64
	var causes []uint64
	for n := uint64(1); n <= 6; n++ {
		combatOnCadence(events.NewRound{RoundNumber: n}, func(e events.Event) events.ListenerReturn {
			ran = append(ran, e.(events.NewRound).RoundNumber)
			causes = append(causes, events.Cause())
			return events.Continue
		})
	}
	// Combat runs on game rounds 2, 4 and 6, numbered by the combat round
	// counter (Phase 82c), not the game round.
	if len(ran) != 3 || ran[0] != 1 || ran[1] != 2 || ran[2] != 3 {
		t.Fatalf("combat ran as rounds %v, want [1 2 3]", ran)
	}
	for i, c := range causes {
		if c != ran[i] {
			t.Fatalf("combat on round %d ran with cause %d", ran[i], c)
		}
	}
	if events.Cause() != 0 {
		t.Fatal("cause leaked past the combat round")
	}
}

func TestPacedLinesReleaseOverTheRound(t *testing.T) {
	r := newPaceRig(t)
	r.round(2, "one", "two", "three")
	// The first line is due at once, on the next turn.
	if len(r.got) != 0 {
		t.Fatalf("lines sent before any turn: %v", r.got)
	}
	r.advance(50 * time.Millisecond)
	if strings.Join(r.got, "|") != "one" {
		t.Fatalf("after one turn got %v", r.got)
	}
	// What a player typed (a tell to Aria) goes out at once mid-round.
	events.AddTyped(events.Message{UserId: r.user.UserId, Text: "Brin tells you, hello\n"})
	events.ProcessEvents()
	if strings.Join(r.got, "|") != "one|Brin tells you, hello" {
		t.Fatalf("chat was delayed: %v", r.got)
	}
	r.advance(800 * time.Millisecond)
	if strings.Join(r.got, "|") != "one|Brin tells you, hello|two" {
		t.Fatalf("after 0.85s got %v", r.got)
	}
	r.advance(800 * time.Millisecond)
	if strings.Join(r.got, "|") != "one|Brin tells you, hello|two|three" {
		t.Fatalf("after 1.65s got %v", r.got)
	}
}

func TestLeftoversFlushBeforeTheNextRound(t *testing.T) {
	r := newPaceRig(t)
	r.round(2, "a1", "a2", "a3")
	r.advance(50 * time.Millisecond)
	// A game round that is not a combat round flushes nothing.
	combatOnCadence(events.NewRound{RoundNumber: 3}, func(events.Event) events.ListenerReturn { return events.Continue })
	events.ProcessEvents()
	if strings.Join(r.got, "|") != "a1" {
		t.Fatalf("a game round flushed combat lines: %v", r.got)
	}
	r.round(3, "b1", "b2")
	if strings.Join(r.got, "|") != "a1|a2|a3" {
		t.Fatalf("next combat round did not flush the last one first: %v", r.got)
	}
	r.advance(50 * time.Millisecond)
	if strings.Join(r.got, "|") != "a1|a2|a3|b1" {
		t.Fatalf("got %v", r.got)
	}
}

func TestPaceOffAndScreenReaderSendAtOnce(t *testing.T) {
	r := newPaceRig(t)
	r.user.SetConfigOption(combatpace.OptionKey, "off")
	r.round(2, "x1", "x2")
	if strings.Join(r.got, "|") != "x1|x2" {
		t.Fatalf("pace off held lines: %v", r.got)
	}

	r.got = nil
	r.user.SetConfigOption(combatpace.OptionKey, nil)
	r.user.ScreenReader = true
	r.round(4, "y1", "y2")
	if strings.Join(r.got, "|") != "y1|y2" {
		t.Fatalf("screen reader default held lines: %v", r.got)
	}
}

func TestPromptWaitsForTheLines(t *testing.T) {
	r := newPaceRig(t)
	r.user.SetConfigOption(`prompt-compiled`, `HP {hp}`)
	r.user.SetConfigOption(`fprompt-compiled`, `HP {hp}`)
	r.user.Character.Health = 20
	r.user.Character.SetAggro(0, 1, characters.DefaultAttack) // in a fight
	start := r.user.GetCommandPrompt()
	combatRoundCounter.Store(1)
	resolveCombatRound(func(events.Event) events.ListenerReturn {
		r.user.Character.Health = 7 // the round's damage lands at once
		r.user.SendText("You are hit hard.")
		r.user.SendText("You stagger.")
		return events.Continue
	})
	events.ProcessEvents()
	if r.user.GetCommandPrompt() == start {
		t.Fatal("test prompt does not show health")
	}
	r.prompts = nil
	r.advance(50 * time.Millisecond)
	if len(r.prompts) == 0 || r.prompts[len(r.prompts)-1] != start {
		t.Fatalf("prompt during held lines = %q, want the round-start %q", r.prompts, start)
	}
	r.advance(1 * time.Second)
	if last := r.prompts[len(r.prompts)-1]; last != r.user.GetCommandPrompt() {
		t.Fatalf("prompt after the lines drained = %q, want live %q", last, r.user.GetCommandPrompt())
	}
}

func TestHeldLinesFlushOnMoveAndDespawn(t *testing.T) {
	r := newPaceRig(t)
	r.round(2, "m1", "m2", "m3")
	// A move the round caused (a flight) keeps its place.
	events.WithCause(2, func() { events.AddToQueue(events.RoomChange{UserId: r.user.UserId, FromRoomId: 1, ToRoomId: 2}) })
	events.ProcessEvents()
	if len(r.got) != 0 {
		t.Fatalf("a flight flushed the round: %v", r.got)
	}
	// Walking away sends everything held first.
	events.AddToQueue(events.RoomChange{UserId: r.user.UserId, FromRoomId: 2, ToRoomId: 3})
	events.ProcessEvents()
	if strings.Join(r.got, "|") != "m1|m2|m3" {
		t.Fatalf("moving did not flush: %v", r.got)
	}

	r.got = nil
	r.round(4, "d1", "d2")
	events.AddToQueue(events.PlayerDespawn{UserId: r.user.UserId})
	events.ProcessEvents()
	if strings.Join(r.got, "|") != "d1|d2" {
		t.Fatalf("leaving did not flush: %v", r.got)
	}
}

func TestCopyoverFlushesHeldLines(t *testing.T) {
	r := newPaceRig(t)
	r.round(2, "c1", "c2")
	copyover.ResetRegistry()
	t.Cleanup(copyover.ResetRegistry)
	copyover.Register(PaceCopyoverContributor())
	var buf strings.Builder
	if err := copyover.Save(&buf); err != nil {
		t.Fatal(err)
	}
	if strings.Join(r.got, "|") != "c1|c2" {
		t.Fatalf("copyover did not flush: %v", r.got)
	}
	if combatpace.Default().Busy(r.user.UserId) {
		t.Fatal("lines still held after copyover")
	}
}

func TestRoomGoingsOnWaitBehindHeldLines(t *testing.T) {
	r := newPaceRig(t)
	r.round(2, "h1", "h2")
	r.advance(50 * time.Millisecond) // h1 out, h2 held

	// Something else happens in the room after the round: it waits.
	sendOrHold(r.user, events.Message{RoomId: 9, Text: "The captain goes for Ysolde.\n"})
	// Said aloud, or said to Aria herself: at once.
	sendOrHold(r.user, events.Message{RoomId: 9, Text: "Brin says, run!\n", IsCommunication: true})
	events.AddTyped(events.Message{UserId: r.user.UserId, Text: "You are carrying nothing.\n"})
	events.ProcessEvents()
	if strings.Join(r.got, "|") != "h1|Brin says, run!|You are carrying nothing." {
		t.Fatalf("got %v", r.got)
	}
	r.advance(2 * time.Second)
	if strings.Join(r.got, "|") != "h1|Brin says, run!|You are carrying nothing.|h2|The captain goes for Ysolde." {
		t.Fatalf("room line did not wait behind the round: %v", r.got)
	}

	// With nothing held, room goings-on go out at once.
	r.got = nil
	sendOrHold(r.user, events.Message{RoomId: 9, Text: "A rat scurries past.\n"})
	if strings.Join(r.got, "|") != "A rat scurries past." {
		t.Fatalf("got %v", r.got)
	}
}

func TestChangingPaceSendsHeldLines(t *testing.T) {
	r := newPaceRig(t)
	freshEvents(t)
	id := events.RegisterListener(events.UserSettingChanged{}, FlushPacedOnPaceChange)
	t.Cleanup(func() { events.UnregisterListener(events.UserSettingChanged{}, id) })
	r.round(2, "p1", "p2")
	events.AddToQueue(events.UserSettingChanged{UserId: r.user.UserId, Name: "tinymap"})
	events.ProcessEvents()
	if len(r.got) != 0 {
		t.Fatalf("another setting flushed: %v", r.got)
	}
	r.user.SetConfigOption(combatpace.OptionKey, "off")
	events.AddToQueue(events.UserSettingChanged{UserId: r.user.UserId, Name: combatpace.OptionKey})
	events.ProcessEvents()
	if strings.Join(r.got, "|") != "p1|p2" {
		t.Fatalf("changing pace did not send held lines: %v", r.got)
	}
}

func TestIdlePlayerIsNotRedrawnEachRound(t *testing.T) {
	r := newPaceRig(t)
	var drained []int
	freshEvents(t)
	id := events.RegisterListener(events.CombatPaceDrained{}, func(e events.Event) events.ListenerReturn {
		drained = append(drained, e.(events.CombatPaceDrained).UserId)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.CombatPaceDrained{}, id) })
	r.prompts = nil
	r.user.Character.SetAggro(0, 1, characters.DefaultAttack) // in a fight
	r.round(2)                                                // a combat round that sends Aria nothing
	r.advance(100 * time.Millisecond)
	if len(r.prompts) != 0 {
		t.Fatalf("an idle player's prompt was redrawn: %q", r.prompts)
	}
	if len(drained) != 1 || drained[0] != r.user.UserId {
		t.Fatalf("drained %v, want the idle player's round to end once", drained)
	}
	if combatpace.Default().Busy(r.user.UserId) {
		t.Fatal("idle player still busy after the first turn")
	}
}

func TestUntypedNoticesWaitButTypedOutputDoesNot(t *testing.T) {
	r := newPaceRig(t)
	r.round(2, "k1", "k2")
	r.advance(50 * time.Millisecond) // k1 out, k2 held
	// A round tick's notice to Aria ("you are bleeding out") waits.
	r.user.SendText("You are bleeding out!")
	events.ProcessEvents()
	if strings.Join(r.got, "|") != "k1" {
		t.Fatalf("an untyped notice jumped ahead: %v", r.got)
	}
	r.advance(2 * time.Second)
	if strings.Join(r.got, "|") != "k1|k2|You are bleeding out!" {
		t.Fatalf("got %v", r.got)
	}
}

func TestNearAFight(t *testing.T) {
	r := newPaceRig(t)
	if nearAFight(r.user) {
		t.Fatal("a player alone in no room is not near a fight")
	}
	r.user.Character.SetAggro(0, 1, characters.DefaultAttack)
	if !nearAFight(r.user) {
		t.Fatal("a fighting player is near a fight")
	}
}
