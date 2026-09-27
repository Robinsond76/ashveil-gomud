package combatstream

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	aria    = Ref{UserId: 7, Name: "Aria", LeaderUserId: 7, MemberKey: "leader"}
	garrick = Ref{MobInstanceId: 11, MobId: 63, Name: "Garrick Vane", LeaderUserId: 7, MemberKey: "companion:3"}
	tamsin  = Ref{MobInstanceId: 12, MobId: 61, Name: "Tamsin Reed", LeaderUserId: 7, MemberKey: "companion:1"}
	captain = Ref{MobInstanceId: 21, MobId: 9104, Name: "bandit captain"}
	slinger = Ref{MobInstanceId: 22, MobId: 9103, Name: "bandit slinger"}
	fence   = Ref{MobInstanceId: 30, MobId: 9105, Name: "bandit fence"}
)

func record(s *Stream) *[]Event {
	var got []Event
	s.Subscribe(func(e Event) { got = append(got, e) })
	return &got
}

func TestRefKey(t *testing.T) {
	assert.Equal(t, "u:7", aria.Key())
	assert.Equal(t, "m:11", garrick.Key())
	assert.Equal(t, "", Ref{}.Key())
	assert.True(t, Ref{}.Zero())
}

func TestEmitStampsInOrderAndCallsSinks(t *testing.T) {
	s := New()
	got := record(s)
	a, ok := s.Emit(Event{Kind: Attack, Source: aria, Target: captain})
	require.True(t, ok)
	b, _ := s.Emit(Event{Kind: Attack, Source: captain, Target: aria})
	assert.Equal(t, uint64(1), a.Seq)
	assert.Equal(t, uint64(2), b.Seq)
	require.Len(t, *got, 2)
	assert.Equal(t, a, (*got)[0])
	assert.Zero(t, a.FightID, "no fight is open")
}

func TestUnsubscribeStopsSink(t *testing.T) {
	s := New()
	n := 0
	stop := s.Subscribe(func(Event) { n++ })
	s.Emit(Event{Kind: Flee, Source: aria})
	stop()
	s.Emit(Event{Kind: Flee, Source: aria})
	assert.Equal(t, 1, n)
}

func TestSinkMayEmitWithoutDeadlock(t *testing.T) {
	s := New()
	var kinds []Kind
	s.Subscribe(func(e Event) {
		kinds = append(kinds, e.Kind)
		if e.Kind == Attack {
			s.Emit(Event{Kind: StatusApplied, Source: aria, Status: "dazed"})
			_ = s.OpenFights()
		}
	})
	s.Emit(Event{Kind: Attack, Source: aria, Target: captain})
	assert.Equal(t, []Kind{Attack, StatusApplied}, kinds)
}

func TestEngageOpensOnceAndMerges(t *testing.T) {
	s := New()
	got := record(s)
	id := s.Engage(5, 100, "bandits#0", aria, []Ref{garrick}, []Ref{captain})
	require.NotZero(t, id)
	require.Len(t, *got, 1)
	assert.Equal(t, FightStart, (*got)[0].Kind)
	assert.Equal(t, id, (*got)[0].FightID)
	assert.Equal(t, "bandits#0", (*got)[0].PartyID)
	assert.Equal(t, uint64(5), (*got)[0].Round)

	// Same party next round, with a new member on each side: same fight.
	again := s.Engage(6, 100, "bandits#0", aria, []Ref{garrick, tamsin}, []Ref{captain, slinger})
	assert.Equal(t, id, again)
	assert.Len(t, *got, 1, "no second fight-start")

	// The party id changed but a known enemy is in it: still the same fight.
	third := s.Engage(7, 100, "bandits#1", aria, []Ref{garrick}, []Ref{slinger})
	assert.Equal(t, id, third)

	fights := s.OpenFights()
	require.Len(t, fights, 1)
	assert.Equal(t, "bandits#1", fights[0].PartyID)
	assert.Equal(t, []Ref{aria, garrick, tamsin}, fights[0].Company)
	assert.Equal(t, []Ref{captain, slinger}, fights[0].Enemies)
	assert.Equal(t, uint64(5), fights[0].StartRound)

	// Another room: another fight.
	other := s.Engage(7, 200, "bandits#0", aria, nil, []Ref{fence})
	assert.NotEqual(t, id, other)
}

func TestEmitPlacesEventsInTheirFight(t *testing.T) {
	s := New()
	id := s.Engage(1, 100, "bandits#0", aria, []Ref{garrick}, []Ref{captain})

	e, _ := s.Emit(Event{Kind: Attack, Source: garrick, Target: captain, Damage: 3, Outcome: OutcomeHit})
	assert.Equal(t, id, e.FightID)
	assert.Equal(t, "bandits#0", e.PartyID)
	assert.Equal(t, 100, e.RoomId)

	e, _ = s.Emit(Event{Kind: Attack, Source: captain, Target: aria})
	assert.Equal(t, id, e.FightID, "found from the enemy side")

	e, _ = s.Emit(Event{Kind: Attack, Source: garrick, Target: fence})
	assert.Zero(t, e.FightID, "an outsider is not in the fight")

	e, _ = s.Emit(Event{Kind: Flee, Source: aria})
	assert.Equal(t, id, e.FightID, "a one-actor event")

	e, _ = s.Emit(Event{Kind: Heal, Source: garrick, Target: aria, Amount: 2})
	assert.Equal(t, id, e.FightID, "same-side events count")
}

func TestDeathCreditAndDedupe(t *testing.T) {
	s := New()
	s.Engage(1, 100, "bandits#0", aria, []Ref{garrick}, []Ref{captain})
	s.Emit(Event{Kind: Attack, Source: aria, Target: captain, Damage: 2})
	s.Emit(Event{Kind: Attack, Source: garrick, Target: captain, Damage: 5})
	s.Emit(Event{Kind: Attack, Source: aria, Target: captain, Damage: 0, Outcome: OutcomeMiss})

	d, ok := s.Emit(Event{Kind: Death, Target: captain, Outcome: OutcomeSlain})
	require.True(t, ok)
	assert.Equal(t, garrick, d.Source, "credited to the last actor who damaged it")

	_, ok = s.Emit(Event{Kind: Death, Target: captain, Outcome: OutcomeSlain})
	assert.False(t, ok, "a second death is dropped")

	// A downed leader who later dies: both are reported, once each.
	_, ok = s.Emit(Event{Kind: Death, Target: aria, Outcome: OutcomeIncapacitated})
	assert.True(t, ok)
	_, ok = s.Emit(Event{Kind: Death, Target: aria, Outcome: OutcomeIncapacitated})
	assert.False(t, ok)
	_, ok = s.Emit(Event{Kind: Death, Target: aria, Outcome: OutcomeSlain})
	assert.True(t, ok)
}

func TestEndFightEmitsSummaryAndForgets(t *testing.T) {
	s := New()
	got := record(s)
	id := s.Engage(1, 100, "bandits#0", aria, []Ref{garrick}, []Ref{captain})
	s.Emit(Event{Kind: Attack, Source: garrick, Target: captain, Damage: 5})
	sum, ok := s.EndFight(id, 4, OutcomeVictory, []MemberHealth{{Ref: aria, Health: 10, Max: 14}})
	require.True(t, ok)
	require.NotNil(t, sum)
	assert.Equal(t, 5, sum.CompanyDamage)
	assert.Equal(t, uint64(1), sum.StartRound)
	assert.Equal(t, uint64(4), sum.EndRound)

	last := (*got)[len(*got)-1]
	assert.Equal(t, FightEnd, last.Kind)
	assert.Equal(t, id, last.FightID)
	assert.Equal(t, OutcomeVictory, last.Outcome)
	assert.Equal(t, sum, last.Summary)
	assert.Equal(t, aria, last.Source)

	assert.Empty(t, s.OpenFights())
	_, ok = s.EndFight(id, 5, OutcomeVictory, nil)
	assert.False(t, ok)
	e, _ := s.Emit(Event{Kind: Attack, Source: garrick, Target: captain})
	assert.Zero(t, e.FightID)
}

func TestUseForTestRestores(t *testing.T) {
	before := Default()
	mine := New()
	restore := UseForTest(mine)
	assert.Same(t, mine, Default())
	restore()
	assert.Same(t, before, Default())
}
