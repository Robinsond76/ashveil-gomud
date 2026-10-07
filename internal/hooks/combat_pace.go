package hooks

import (
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/combatpace"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
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
	// promptHeld are players shown their round-start prompt, whose live
	// prompt is drawn when their round ends.
	promptHeld = map[int]bool{}
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

// deliverItems sends a player's released entries in order: text lines, and
// each run of data entries as one batch before the line that follows it.
func deliverItems(user *users.UserRecord, items []combatpace.Release) {
	var batch []any
	flushBatch := func() {
		if len(batch) > 0 {
			sendCombatData(user.UserId, batch)
			batch = nil
		}
	}
	for _, it := range items {
		if it.IsData {
			batch = append(batch, it.Data)
			continue
		}
		flushBatch()
		deliver(user, it.Text)
	}
	flushBatch()
}

// paceOf is a player's chosen pace, or its default.
func paceOf(user *users.UserRecord) combatpace.Pace {
	return combatpace.For(user.GetConfigOption(combatpace.OptionKey), user.ScreenReader)
}

// PaceOf is a player's combat pace, for the web client's battle feed
// (Phase 40g2), which paces its animation by it.
func PaceOf(user *users.UserRecord) combatpace.Pace { return paceOf(user) }

// sendOrHold delivers a message's text to one recipient: held when a combat
// round caused it and the recipient paces combat, at once otherwise.
//
// One exception keeps the story in order: while the recipient has lines
// held, what the game does meanwhile (a mob that goes for someone after the
// round, a round tick's "you are bleeding out") waits behind them. What a
// player typed never waits: their own command's output, a tell to them,
// and anything said aloud (a say, an emote).
func sendOrHold(user *users.UserRecord, message events.Message) {
	if !holdBehindCombat(user, message.Text, message.IsCommunication) {
		deliver(user, message.Text)
	}
}

// holdBehindCombat holds text for a player, if pacing calls for it, and
// reports whether it did.
func holdBehindCombat(user *users.UserRecord, text string, spoken bool) bool {
	pacer := combatpace.Default()
	if round := events.Cause(); round != 0 {
		pace := paceOf(user)
		if pace == combatpace.Off {
			return false
		}
		deliverItems(user, pacer.Hold(user.UserId, round, events.Slot(), text, paceSpec(pace), paceNow()))
		return true
	}
	if spoken || events.Typed() {
		return false
	}
	return pacer.Follow(user.UserId, text)
}

// paceSpec is the timing a player's held lines follow: by action (one beat
// a turn) in a round the battle clock resolved (Phase 82c), by line within
// the round's window on the fixed cadence.
func paceSpec(pace combatpace.Pace) combatpace.Spec {
	if clock.active {
		return pace.Beats()
	}
	return pace.ForRound(configs.GetTimingConfig().CombatRoundDuration())
}

// CombatOnCadence is DoCombat's NewRound listener: combat resolves only on
// every CombatEveryRounds-th game round, with everything it causes tagged
// with the round. While a player fights, the battle clock (Phase 82c)
// resolves the rounds instead, on its own beat. Tests call DoCombat
// directly and are unaffected.
func CombatOnCadence(e events.Event) events.ListenerReturn {
	return combatOnCadence(e, DoCombat)
}

func combatOnCadence(e events.Event, doCombat events.Listener) events.ListenerReturn {
	evt, ok := e.(events.NewRound)
	if !ok || !configs.GetTimingConfig().CombatRoundDue(evt.RoundNumber) || playerFightLive() {
		return events.Continue
	}
	return resolveCombatRound(doCombat)
}

// resolveCombatRound runs one combat round, numbered by the combat round
// counter, with everything it causes tagged with that round.
func resolveCombatRound(doCombat events.Listener) events.ListenerReturn {
	startPacedRound()
	round := nextCombatRound()
	result := events.Continue
	events.WithCause(round, func() { result = doCombat(events.NewRound{RoundNumber: round}) })
	return result
}

// startPacedRound flushes the last round's leftovers, forgets its marks, and
// snapshots the prompt of every player who paces combat.
func startPacedRound() {
	pacer := combatpace.Default()
	released, drained := pacer.FlushAll()
	sendReleased(released, drained)

	snapshot := map[int]string{}
	var pacing []int
	for _, userId := range users.GetOnlineUserIds() {
		if user := users.GetByUserId(userId); user != nil && paceOf(user) != combatpace.Off && nearAFight(user) {
			snapshot[userId] = user.GetCommandPrompt()
			pacing = append(pacing, userId)
		}
	}
	pacer.StartRound(pacing...)
	roundPromptsMu.Lock()
	roundPrompts = snapshot
	promptHeld = map[int]bool{}
	roundPromptsMu.Unlock()
}

// nearAFight reports whether a player is fighting, or is in a room where a
// fight is going on: only they have a round opened (their prompt and web
// views held until its lines are out). Anyone else a round's text reaches
// is paced all the same, without the hold.
func nearAFight(user *users.UserRecord) bool {
	if user.Character.Aggro != nil {
		return true
	}
	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		return false
	}
	return len(room.GetPlayers(rooms.FindFighting)) > 0 || len(room.GetMobs(rooms.FindFighting)) > 0
}

// ReleasePacedCombat is a NewTurn listener: it sends every held line now due.
func ReleasePacedCombat(e events.Event) events.ListenerReturn {
	sendReleased(combatpace.Default().Due(paceNow()))
	return events.Continue
}

func sendReleased(released []combatpace.Release, drained []int) {
	for i := 0; i < len(released); {
		j := i + 1
		for j < len(released) && released[j].UserId == released[i].UserId {
			j++
		}
		if user := users.GetByUserId(released[i].UserId); user != nil {
			deliverItems(user, released[i:j])
		}
		i = j
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
	redraw := promptHeld[userId]
	delete(promptHeld, userId)
	roundPromptsMu.Unlock()
	if redraw {
		events.AddToQueue(events.RedrawPrompt{UserId: userId}, 100)
	}
	events.AddToQueue(events.CombatPaceDrained{UserId: userId})
}

// FlushPacedCombat sends a player's held lines at once.
func FlushPacedCombat(userId int) {
	lines, ended := combatpace.Default().Flush(userId)
	if !ended {
		return
	}
	if user := users.GetByUserId(userId); user != nil {
		deliverItems(user, lines)
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
	if ok {
		promptHeld[userId] = true
	}
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

// Phase 40e: structured combat data (the web client's Company.Battle.Event)
// rides the same queue as the narration. A producer queues events.CombatData
// as it emits the happening, so it is dispatched in order with the round's
// text: held with it for a player who paces combat, released with the next
// line that follows it (or when the last line goes out), flushed with it.
// What the data is, and how it is delivered, belongs to the module that
// queued it: hooks only orders it.

// combatDataSender delivers a batch of combat data to a player; the gmcp
// module sets it. Tests replace it.
var combatDataSender func(userId int, batch []any)

// SetCombatDataSender sets how released combat data reaches a player.
func SetCombatDataSender(fn func(userId int, batch []any)) { combatDataSender = fn }

func sendCombatData(userId int, batch []any) {
	if fn := combatDataSender; fn != nil {
		fn(userId, batch)
	}
}

// CombatData_Hold is events.CombatData's listener: it holds the data
// behind the round's text, or sends it at once for a player whose pace is
// off, for what they typed, and when nothing is held ahead of it.
func CombatData_Hold(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.CombatData)
	if !ok {
		return events.Continue
	}
	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}
	pacer := combatpace.Default()
	if round := events.Cause(); round != 0 {
		pace := paceOf(user)
		if pace == combatpace.Off {
			sendCombatData(user.UserId, []any{evt.Data})
			return events.Continue
		}
		deliverItems(user, pacer.HoldData(user.UserId, round, evt.Data, paceSpec(pace), paceNow()))
		return events.Continue
	}
	if events.Typed() || !pacer.FollowData(user.UserId, evt.Data) {
		sendCombatData(user.UserId, []any{evt.Data})
	}
	return events.Continue
}
