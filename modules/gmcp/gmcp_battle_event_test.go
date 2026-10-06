package gmcp

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/combatpace"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// eventRig is the Phase 40e wiring in miniature: a stream the module
// listens to, the real hooks listeners ordering the data with the text, a
// pacer on a fake clock, and a capture of what the player is sent.
type eventRig struct {
	t      *testing.T
	stream *combatstream.Stream
	now    time.Time
	got    []rigEntry
	user   *users.UserRecord
}

type rigEntry struct {
	at      time.Time
	text    string
	payload *battleEventPayload
}

func newEventRig(t *testing.T) *eventRig {
	r := &eventRig{t: t, stream: combatstream.New(), now: time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)}
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	r.user = users.NewUserRecord(7, 71)
	r.user.Character.Name = "Aria"
	users.SetTestUser(r.user)
	AcceptGMCPForTest(r.user.ConnectionId())
	settings, _ := gmcpModule.cache.Get(r.user.ConnectionId())
	settings.Client.Name = `WebClient` // the web client opts in by being the web client
	gmcpModule.cache.Add(r.user.ConnectionId(), settings)

	t.Cleanup(combatstream.UseForTest(r.stream))
	t.Cleanup(SubscribeBattleEventsForTest(r.stream))
	t.Cleanup(combatpace.UseForTest(combatpace.New()))
	t.Cleanup(hooks.SetPaceClockForTest(func() time.Time { return r.now }))
	t.Cleanup(hooks.SetWriteTextForTest(func(userId int, text string) {
		if userId == 7 {
			r.got = append(r.got, rigEntry{at: r.now, text: strings.TrimSpace(text)})
		}
	}))
	for _, reg := range []struct {
		evt events.Event
		fn  events.Listener
	}{
		{events.Message{}, hooks.Message_SendMessage},
		{events.CombatData{}, hooks.CombatData_Hold},
		{events.NewTurn{}, hooks.ReleasePacedCombat},
	} {
		freshEvents(t)
		id := events.RegisterListener(reg.evt, reg.fn)
		evt := reg.evt
		t.Cleanup(func() { events.UnregisterListener(evt, id) })
	}
	// The module's own GMCPOut listener has no connection to write to; take
	// the payloads here, ahead of it.
	id := events.RegisterListener(GMCPOut{}, func(e events.Event) events.ListenerReturn {
		if out := e.(GMCPOut); out.UserId == 7 && out.Module == battleEventModule {
			p := out.Payload.(battleEventPayload)
			r.got = append(r.got, rigEntry{at: r.now, payload: &p})
			return events.Cancel
		}
		return events.Continue
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(GMCPOut{}, id) })
	events.ProcessEvents()
	return r
}

func (r *eventRig) payloads() []battleEventPayload {
	var out []battleEventPayload
	for _, e := range r.got {
		if e.payload != nil {
			out = append(out, *e.payload)
		}
	}
	return out
}

func (r *eventRig) kinds() []string {
	var out []string
	for _, p := range r.payloads() {
		for _, e := range p.Events {
			out = append(out, e.Kind+":"+e.Src+">"+e.Tgt)
		}
	}
	return out
}

func (r *eventRig) turns(n int) {
	for i := 0; i < n; i++ {
		r.now = r.now.Add(50 * time.Millisecond)
		events.AddToQueue(events.NewTurn{})
		events.ProcessEvents()
	}
}

var (
	rigLeader    = combatstream.Ref{UserId: 7, Name: "Aria", LeaderUserId: 7, MemberKey: "leader"}
	rigCompanion = combatstream.Ref{MobInstanceId: 501, Name: "Bran", LeaderUserId: 7, MemberKey: "companion:1"}
)

func rigEnemy(id int) combatstream.Ref {
	return combatstream.Ref{MobInstanceId: id, Name: "a bandit"}
}

// TestBattleEventsReachTheLeaderWithSharedIDs: with pacing off, a fight's
// events arrive at once with the IDs Company and Company.Battle use, and
// carry no health.
func TestBattleEventsReachTheLeaderWithSharedIDs(t *testing.T) {
	r := newEventRig(t)
	r.user.SetConfigOption(combatpace.OptionKey, string(combatpace.Off))
	t.Cleanup(spells.UseSpellsForTest(&spells.SpellData{SpellId: "mend", Name: "Mend Wounds"}))

	id := r.stream.Open(341, 100, "party-a", rigLeader, []combatstream.Ref{rigCompanion}, []combatstream.Ref{rigEnemy(88), rigEnemy(89)})
	events.WithCause(341, func() {
		r.stream.Emit(combatstream.Event{Round: 341, Kind: combatstream.Attack, FightID: id, Source: rigLeader, Target: rigEnemy(88),
			Outcome: combatstream.OutcomeCrit, Damage: 6, Crit: true, WeaponType: "slashing", Quality: "telling"})
		r.stream.Emit(combatstream.Event{Round: 341, Kind: combatstream.StatusApplied, FightID: id, Source: rigLeader, Target: rigEnemy(88), Status: "Bleeding"})
		r.stream.Emit(combatstream.Event{Round: 341, Kind: combatstream.Attack, FightID: id, Source: rigEnemy(89), Target: rigCompanion, Outcome: combatstream.OutcomeMiss, Defenses: []string{"dodge"}})
		r.stream.Emit(combatstream.Event{Round: 341, Kind: combatstream.Heal, FightID: id, Source: rigCompanion, Target: rigLeader, SpellId: "mend", Amount: 5, HeldBack: 2})
		r.stream.Emit(combatstream.Event{Round: 341, Kind: combatstream.Death, FightID: id, Target: rigEnemy(88), Outcome: combatstream.OutcomeSlain})
	})
	events.ProcessEvents()
	r.stream.EndFight(id, 342, combatstream.OutcomeVictory, combatstream.Final{})
	events.ProcessEvents()

	assert.Equal(t, []string{
		"fight-start:>",
		"attack:leader>m:88",
		"status-applied:leader>m:88",
		"attack:m:89>companion:1",
		"heal:companion:1>leader",
		"death:leader>m:88",
		"fight-end:>",
	}, r.kinds())

	ps := r.payloads()
	require.NotEmpty(t, ps)
	start := ps[0].Events[0]
	assert.Equal(t, []string{"leader", "companion:1"}, start.Company)
	assert.Equal(t, []string{"m:88", "m:89"}, start.Enemies)

	var attack, heal battleEvent
	for _, p := range ps {
		assert.Equal(t, id, p.Fight)
		// The fight opened at the server's round 341: its first is 1.
		assert.Equal(t, p.Round-340, p.FightRound)
		for _, e := range p.Events {
			if e.Kind == "attack" && e.Tgt == "m:88" {
				attack = e
			}
			if e.Kind == "heal" {
				heal = e
			}
			if e.Kind == "fight-end" {
				assert.Equal(t, combatstream.OutcomeVictory, e.Outcome)
			}
		}
	}
	assert.True(t, attack.Crit)
	assert.Equal(t, 6, attack.Damage)
	assert.Equal(t, "slashing", attack.Weapon)
	assert.Equal(t, "telling", attack.Quality)
	assert.Equal(t, "mend", heal.Spell)
	assert.Equal(t, "Mend Wounds", heal.SpellName, "the spell's display name rides with its id")

	raw, err := json.Marshal(ps)
	require.NoError(t, err)
	for _, banned := range []string{"health", "hp", "max"} {
		assert.NotContains(t, string(raw), banned, "no health in the feed")
	}
}

// TestBattleEventsStayWithTheirOwnFight: another company's fight in the
// same room sends this player nothing, and a client that did not ask for
// the module is sent nothing.
func TestBattleEventsStayWithTheirOwnFight(t *testing.T) {
	r := newEventRig(t)
	r.user.SetConfigOption(combatpace.OptionKey, string(combatpace.Off))
	other := users.NewUserRecord(8, 81)
	other.Character.Name = "Brannoc"
	users.SetTestUser(other)

	otherRef := combatstream.Ref{UserId: 8, Name: "Brannoc", LeaderUserId: 8, MemberKey: "leader"}
	oid := r.stream.Open(341, 100, "party-b", otherRef, nil, []combatstream.Ref{rigEnemy(90)})
	r.stream.Emit(combatstream.Event{Round: 341, Kind: combatstream.Attack, FightID: oid, Source: otherRef, Target: rigEnemy(90), Outcome: combatstream.OutcomeHit, Damage: 3})
	r.stream.EndFight(oid, 342, combatstream.OutcomeVictory, combatstream.Final{})
	events.ProcessEvents()
	assert.Empty(t, r.payloads(), "another company's fight is not this player's")

	// A telnet client that has not accepted GMCP: nothing.
	RefuseGMCPForTest(r.user.ConnectionId())
	id := r.stream.Open(343, 100, "party-a", rigLeader, nil, []combatstream.Ref{rigEnemy(88)})
	r.stream.Emit(combatstream.Event{Round: 343, Kind: combatstream.Attack, FightID: id, Source: rigLeader, Target: rigEnemy(88), Outcome: combatstream.OutcomeHit, Damage: 2})
	events.ProcessEvents()
	assert.Empty(t, r.payloads(), "a client without GMCP gets nothing")

	// Accepting GMCP is not enough for a client that isn't the web client:
	// it must ask for the module by name.
	settings, _ := gmcpModule.cache.Get(r.user.ConnectionId())
	settings.GMCPAccepted = true
	settings.Client.Name = "Mudlet"
	settings.EnabledModules = map[string]int{"Company": 1}
	gmcpModule.cache.Add(r.user.ConnectionId(), settings)
	r.stream.Emit(combatstream.Event{Round: 343, Kind: combatstream.Attack, FightID: id, Source: rigLeader, Target: rigEnemy(88), Outcome: combatstream.OutcomeHit, Damage: 2})
	events.ProcessEvents()
	assert.Empty(t, r.payloads(), "Company support alone does not opt in")

	settings.EnabledModules[battleEventModule] = 1
	gmcpModule.cache.Add(r.user.ConnectionId(), settings)
	r.stream.Emit(combatstream.Event{Round: 343, Kind: combatstream.Attack, FightID: id, Source: rigLeader, Target: rigEnemy(88), Outcome: combatstream.OutcomeHit, Damage: 2})
	events.ProcessEvents()
	assert.Len(t, r.payloads(), 1, "asking for the module opts in")
}

// TestBattleEventsMaskEnemiesTheLeaderCannotSee: an enemy last seen hidden
// (as in the dark) is "?", and a status on it is not sent.
func TestBattleEventsMaskEnemiesTheLeaderCannotSee(t *testing.T) {
	r := newEventRig(t)
	r.user.SetConfigOption(combatpace.OptionKey, string(combatpace.Off))
	battleSeen.note(7, battleSeenKey{}, 88, true)
	t.Cleanup(func() { battleSeen.forget(7) })

	id := r.stream.Open(341, 100, "party-a", rigLeader, nil, []combatstream.Ref{rigEnemy(88), rigEnemy(89)})
	r.stream.Emit(combatstream.Event{Round: 341, Kind: combatstream.Attack, FightID: id, Source: rigLeader, Target: rigEnemy(88), Outcome: combatstream.OutcomeHit, Damage: 4})
	r.stream.Emit(combatstream.Event{Round: 341, Kind: combatstream.StatusApplied, FightID: id, Source: rigLeader, Target: rigEnemy(88), Status: "Bleeding"})
	r.stream.Emit(combatstream.Event{Round: 341, Kind: combatstream.StatusApplied, FightID: id, Source: rigLeader, Target: rigEnemy(89), Status: "Bleeding"})
	events.ProcessEvents()

	assert.Equal(t, []string{
		"fight-start:>",
		"attack:leader>?",
		"status-applied:leader>m:89",
	}, r.kinds())
	assert.Equal(t, []string{"m:89"}, r.payloads()[0].Events[0].Enemies, "an unseen enemy is not listed")
}

// TestBattleEventsAreReleasedInStepWithTheNarration: with pace normal, each
// event goes out with the text line that follows it, never before the line
// ahead of it, and a trailing event goes out with the round's last line.
func TestBattleEventsAreReleasedInStepWithTheNarration(t *testing.T) {
	r := newEventRig(t)
	id := r.stream.Open(1, 100, "party-a", rigLeader, nil, []combatstream.Ref{rigEnemy(88)})
	events.ProcessEvents() // the fight-start, before the round
	r.got = nil

	events.WithCause(2, func() {
		r.user.SendText("Aria swings.")
		r.stream.Emit(combatstream.Event{Round: 2, Kind: combatstream.Attack, FightID: id, Source: rigLeader, Target: rigEnemy(88), Outcome: combatstream.OutcomeHit, Damage: 3})
		r.user.SendText("The bandit is cut. (3)")
		r.stream.Emit(combatstream.Event{Round: 2, Kind: combatstream.Attack, FightID: id, Source: rigEnemy(88), Target: rigLeader, Outcome: combatstream.OutcomeMiss})
		r.user.SendText("The bandit misses.")
		r.stream.Emit(combatstream.Event{Round: 2, Kind: combatstream.Death, FightID: id, Target: rigEnemy(88), Outcome: combatstream.OutcomeSlain})
	})
	events.ProcessEvents()
	require.Empty(t, r.got, "pacing holds the text and the data alike")
	assert.True(t, combatpace.Default().Busy(7))

	r.turns(200)

	// Flatten to a sequence of text and data, with release times.
	type step struct {
		label string
		at    time.Duration
	}
	var steps []step
	start := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	for _, e := range r.got {
		if e.payload != nil {
			for _, ev := range e.payload.Events {
				steps = append(steps, step{ev.Kind + ":" + ev.Src + ">" + ev.Tgt, e.at.Sub(start)})
			}
		} else {
			steps = append(steps, step{e.text, e.at.Sub(start)})
		}
	}
	at := map[string]time.Duration{}
	for _, s := range steps {
		at[s.label] = s.at
	}
	require.Len(t, steps, 6, "%v", steps)
	// "Aria swings." has nothing before it; the first attack is held for
	// the line that follows it.
	assert.Equal(t, at["The bandit is cut. (3)"], at["attack:leader>m:88"], "an event goes out with the line that follows it")
	assert.Greater(t, at["The bandit is cut. (3)"], at["Aria swings."])
	assert.Equal(t, at["The bandit misses."], at["attack:m:88>leader"])
	// Nothing follows the death: it goes out with the last line.
	assert.Equal(t, at["The bandit misses."], at["death:leader>m:88"])
	assert.Greater(t, at["The bandit misses."], at["The bandit is cut. (3)"])
}

// TestBattleEventsFlushWithTheText: a move (or quit) mid-round sends the
// held events along with the held text, and nothing is lost.
func TestBattleEventsFlushWithTheText(t *testing.T) {
	r := newEventRig(t)
	id := r.stream.Open(1, 100, "party-a", rigLeader, nil, []combatstream.Ref{rigEnemy(88)})
	events.ProcessEvents()
	r.got = nil

	events.WithCause(2, func() {
		r.user.SendText("Aria swings.")
		r.stream.Emit(combatstream.Event{Round: 2, Kind: combatstream.Attack, FightID: id, Source: rigLeader, Target: rigEnemy(88), Outcome: combatstream.OutcomeHit, Damage: 3})
		r.user.SendText("The bandit is cut. (3)")
	})
	events.ProcessEvents()
	require.Empty(t, r.got)

	hooks.FlushPacedCombat(7)
	events.ProcessEvents()
	assert.Equal(t, []string{"attack:leader>m:88"}, r.kinds())
	texts := 0
	for _, e := range r.got {
		if e.payload == nil {
			texts++
		}
	}
	assert.Equal(t, 2, texts)
	assert.False(t, combatpace.Default().Busy(7))
}

// TestBattleEventsNameTheLeaderByTheirCellKey: a company's leader is
// "leader", the key of their cell in Company.Battle.positions, so the
// battle screen needs no mapping; a player who leads no company is "me".
func TestBattleEventsNameTheLeaderByTheirCellKey(t *testing.T) {
	v := battleViewer{userId: 7}
	assert.Equal(t, "leader", v.refID(rigLeader))
	assert.Equal(t, "me", v.refID(combatstream.Ref{UserId: 7, Name: "Aria"}))
	assert.Equal(t, "u:8", v.refID(combatstream.Ref{UserId: 8, LeaderUserId: 8, MemberKey: "leader"}))
	assert.Equal(t, "m:9", v.refID(combatstream.Ref{MobInstanceId: 9, LeaderUserId: 8, MemberKey: "companion:1"}))
}
