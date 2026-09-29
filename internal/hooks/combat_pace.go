package hooks

import (
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/combatpace"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Ashveil Phase 29f: paced combat output.
//
// Combat resolves every CombatEveryRounds game rounds (CombatOnCadence).
// Everything a combat round causes is tagged with it (events.WithCause), and
// Message_SendMessage holds a tagged line for each recipient whose pace
// isn't off. ReleasePacedCombat, on every turn, sends the lines now due.
// Held lines are flushed, never dropped: at the next combat round, when the
// player moves, leaves, or turns pacing off, and at copyover.
//
// While a player has held lines, their prompt shows its state from the
// round's start, so it never runs ahead of the narration; the web client's
// views wait for CombatPaceDrained in the same way.

var (
	// paceNow is the pacing clock; tests replace it.
	paceNow = time.Now
	// writeText writes text to a player's connection; tests replace it.
	writeText = writeToConnection

	roundPromptsMu sync.Mutex
	roundPrompts   = map[int]string{}
)

// SetPaceClockForTest makes now the pacing clock and returns a restore func.
func SetPaceClockForTest(now func() time.Time) (restore func()) {
	prev := paceNow
	paceNow = now
	return func() { paceNow = prev }
}

// SetWriteTextForTest captures what players are sent (the text before ANSI
// parsing) and returns a restore func.
func SetWriteTextForTest(fn func(userId int, text string)) (restore func()) {
	prev := writeText
	writeText = func(user *users.UserRecord, text string) { fn(user.UserId, text) }
	return func() { writeText = prev }
}

func writeToConnection(user *users.UserRecord, text string) {
	textOut := templates.AnsiParse(text)
	if user.ScreenReader {
		textOut = util.StripCharsForScreenReaders(textOut)
	}
	connections.SendTo([]byte(term.AnsiMoveCursorColumn.String()+term.AnsiEraseLine.String()+textOut), user.ConnectionId())
}

// deliver sends text to a player now and redraws their prompt.
func deliver(user *users.UserRecord, text string) {
	writeText(user, text)
	events.AddToQueue(events.RedrawPrompt{UserId: user.UserId}, 100)
}

// paceOf is a player's chosen pace, or its default.
func paceOf(user *users.UserRecord) combatpace.Pace {
	return combatpace.For(user.GetConfigOption(combatpace.OptionKey), user.ScreenReader)
}

// sendOrHold delivers a message's text to one recipient: held when a combat
// round caused it and the recipient paces combat, at once otherwise.
//
// One exception keeps the story in order: while the recipient has lines
// held, other goings-on in their room (a mob that goes for someone after
// the round, a player walking in) wait behind them. What is said to them
// directly (their own command's output, a tell) and what is said aloud (a
// say, an emote) never waits.
func sendOrHold(user *users.UserRecord, message events.Message) {
	text := message.Text
	pace := paceOf(user)
	if round := events.Cause(); round != 0 && pace != combatpace.Off {
		spec := pace.ForRound(configs.GetTimingConfig().CombatRoundDuration())
		for _, older := range combatpace.Default().Hold(user.UserId, round, text, spec, paceNow()) {
			deliver(user, older)
		}
		return
	}
	direct := message.UserId == user.UserId
	if !direct && !message.IsCommunication && combatpace.Default().Follow(user.UserId, text) {
		return
	}
	deliver(user, text)
}

// CombatOnCadence is DoCombat's NewRound listener: combat resolves only on
// every CombatEveryRounds-th game round, with everything it causes tagged
// with the round. Tests call DoCombat directly and are unaffected.
func CombatOnCadence(e events.Event) events.ListenerReturn {
	return combatOnCadence(e, DoCombat)
}

func combatOnCadence(e events.Event, doCombat events.Listener) events.ListenerReturn {
	evt, ok := e.(events.NewRound)
	if !ok || !configs.GetTimingConfig().CombatRoundDue(evt.RoundNumber) {
		return events.Continue
	}
	startPacedRound()
	result := events.Continue
	events.WithCause(evt.RoundNumber, func() { result = doCombat(e) })
	return result
}

// startPacedRound flushes the last round's leftovers, forgets its marks, and
// snapshots the prompt of every player who paces combat.
func startPacedRound() {
	pacer := combatpace.Default()
	released, drained := pacer.FlushAll()
	sendReleased(released, drained)
	pacer.StartRound()

	snapshot := map[int]string{}
	for _, userId := range users.GetOnlineUserIds() {
		if user := users.GetByUserId(userId); user != nil && paceOf(user) != combatpace.Off {
			snapshot[userId] = user.GetCommandPrompt()
		}
	}
	roundPromptsMu.Lock()
	roundPrompts = snapshot
	roundPromptsMu.Unlock()
}

// ReleasePacedCombat is a NewTurn listener: it sends every held line now due.
func ReleasePacedCombat(e events.Event) events.ListenerReturn {
	sendReleased(combatpace.Default().Due(paceNow()))
	return events.Continue
}

func sendReleased(released []combatpace.Release, drained []int) {
	for _, r := range released {
		if user := users.GetByUserId(r.UserId); user != nil {
			deliver(user, r.Text)
		}
	}
	for _, userId := range drained {
		finishDrain(userId)
	}
}

// finishDrain lets what was held back with a player's lines catch up: the
// live prompt, and (on CombatPaceDrained) the web client's views.
func finishDrain(userId int) {
	roundPromptsMu.Lock()
	delete(roundPrompts, userId)
	roundPromptsMu.Unlock()
	events.AddToQueue(events.RedrawPrompt{UserId: userId}, 100)
	events.AddToQueue(events.CombatPaceDrained{UserId: userId})
}

// FlushPacedCombat sends a player's held lines at once.
func FlushPacedCombat(userId int) {
	lines := combatpace.Default().Flush(userId)
	if len(lines) == 0 {
		return
	}
	if user := users.GetByUserId(userId); user != nil {
		for _, text := range lines {
			deliver(user, text)
		}
	}
	finishDrain(userId)
}

// heldPrompt is the prompt to show a player with held lines: the one from
// the round's start. ok is false when the live prompt applies.
func heldPrompt(userId int) (string, bool) {
	if !combatpace.Default().Busy(userId) {
		return "", false
	}
	roundPromptsMu.Lock()
	defer roundPromptsMu.Unlock()
	p, ok := roundPrompts[userId]
	return p, ok
}

// FlushPacedOnRoomChange sends a player's held lines when they leave the
// room by their own doing, before the new room's text. A move the combat
// round caused (a flight) keeps its place in the round's lines.
func FlushPacedOnRoomChange(e events.Event) events.ListenerReturn {
	if evt, ok := e.(events.RoomChange); ok && evt.UserId > 0 && events.Cause() == 0 {
		FlushPacedCombat(evt.UserId)
	}
	return events.Continue
}

// FlushPacedOnDespawn sends a leaving player's held lines.
func FlushPacedOnDespawn(e events.Event) events.ListenerReturn {
	if evt, ok := e.(events.PlayerDespawn); ok {
		FlushPacedCombat(evt.UserId)
	}
	return events.Continue
}

// PaceCopyoverContributor flushes every held line as a copyover saves:
// nothing held is carried over, and nothing is lost.
func PaceCopyoverContributor() copyover.Contributor {
	const name = "combatpace"
	return copyover.FuncContributor(name,
		func(enc *copyover.Encoder) error {
			sendReleased(combatpace.Default().FlushAll())
			return enc.WriteSection(name, struct{}{})
		},
		func(dec *copyover.Decoder) error { return nil },
	)
}

// FlushPacedOnPaceChange sends a player's held lines when they change their
// pace (`set combatpace`): the new pace applies from the next round.
func FlushPacedOnPaceChange(e events.Event) events.ListenerReturn {
	if evt, ok := e.(events.UserSettingChanged); ok && evt.Name == combatpace.OptionKey {
		FlushPacedCombat(evt.UserId)
	}
	return events.Continue
}
