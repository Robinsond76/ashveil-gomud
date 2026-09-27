package company

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const summaryHeading = "── The fighting is over ──"

// listen records every event on the brawl's stream.
func (b *brawl) listen() *[]combatstream.Event {
	var got []combatstream.Event
	b.t.Cleanup(combatstream.Default().Subscribe(func(e combatstream.Event) { got = append(got, e) }))
	return &got
}

// fightItOut runs rounds until no bandit stands. Only Aria is kept alive
// (her maximum is raised each round, since health changes clamp to it, but
// her health is not restored), so companions can fall and every blow the
// stream reports is a real one. If all of them fall, she attacks again
// herself whenever she's idle. It returns everything the rounds sent.
func (b *brawl) fightItOut(maxRounds int) string {
	b.t.Helper()
	b.aria.Character.HealthMax.Value = 1000
	b.aria.Character.Health = 1000
	var seen []string
	for i := 0; i < maxRounds && len(b.livingBandits()) > 0; i++ {
		b.aria.Character.HealthMax.Value = 1000
		if living := b.livingBandits(); b.aria.Character.Aggro == nil && len(b.companyInstances()) == 0 {
			// Her company has fallen: alone, she fights on as a player
			// would (a solo player isn't kept engaged by the 29a upkeep).
			b.cmd("attack", living[0].Character.Name)
		}
		seen = append(seen, b.fight())
	}
	require.Empty(b.t, b.livingBandits(), "the bandits fall")
	// One more round: nothing further is reported for the ended fight.
	b.aria.Character.HealthMax.Value = 1000
	seen = append(seen, b.fight())
	return strings.Join(seen, "\n")
}

// sides classifies a ref as company (Aria or a companion) or bandit.
type sides struct {
	company map[string]bool
	bandits map[string]bool
}

func (b *brawl) sides() sides {
	s := sides{company: map[string]bool{"u:7": true}, bandits: map[string]bool{}}
	for instance := range b.companyInstances() {
		s.company[combatstream.Ref{MobInstanceId: instance}.Key()] = true
	}
	for _, ids := range b.bandits {
		for _, id := range ids {
			s.bandits[combatstream.Ref{MobInstanceId: id}.Key()] = true
		}
	}
	return s
}

func (s sides) across(e combatstream.Event) bool {
	src, tgt := e.Source.Key(), e.Target.Key()
	return (s.company[src] && s.bandits[tgt]) || (s.bandits[src] && s.company[tgt])
}

// TestCombatEventStreamThroughTheRealRound drives the 29a 5v5 through the
// real round (shipped config, real commands, DoCombat, idle mobs, queued
// mob commands) and checks the stream: one fight from its start to its
// end, every blow between the sides in it, each bandit's death once, and a
// summary whose totals are the sum of the events. The leader is sent the
// summary once, at the end, and the clock never moves.
func TestCombatEventStreamThroughTheRealRound(t *testing.T) {
	b := newBrawl(t)
	got := b.listen()
	turn, round := util.GetTurnCount(), util.GetRoundCount()
	for _, mv := range []string{"move #1 1 1", "move #3 1 2", "move me 1 3", "move #2 2 2", "move #4 3 2"} {
		require.Contains(t, b.cmd("formation", mv), "Placed")
	}
	s := b.sides() // before anyone falls

	b.cmd("attack", "bandit captain")
	seen := b.fightItOut(200)

	var starts, ends []combatstream.Event
	for _, e := range *got {
		switch e.Kind {
		case combatstream.FightStart:
			starts = append(starts, e)
		case combatstream.FightEnd:
			ends = append(ends, e)
		}
	}
	require.Len(t, starts, 1, "one fight")
	require.Len(t, ends, 1, "which ends once")
	fight := starts[0].FightID
	require.NotZero(t, fight)
	assert.Equal(t, fight, ends[0].FightID)
	assert.Equal(t, combatstream.OutcomeVictory, ends[0].Outcome)
	assert.Equal(t, starts[0].PartyID, ends[0].PartyID)
	assert.NotEmpty(t, starts[0].PartyID)
	assert.Equal(t, 7, starts[0].Source.UserId, "the leader's fight")

	// Order and ids: nothing of the fight before its start or after its end;
	// every blow between the two sides carries the fight's id; sequence
	// numbers rise.
	var companyDamage, enemyDamage, attacks int
	deaths := map[string]int{}
	kills := map[string]int{}
	slainAt := map[string]uint64{}
	var lastSeq uint64
	for _, e := range *got {
		assert.Greater(t, e.Seq, lastSeq)
		lastSeq = e.Seq
		if e.FightID == fight {
			assert.GreaterOrEqual(t, e.Seq, starts[0].Seq, "%s before the fight started", e.Kind)
			assert.LessOrEqual(t, e.Seq, ends[0].Seq, "%s after the fight ended", e.Kind)
		}
		switch e.Kind {
		case combatstream.Attack, combatstream.SpellHit:
			if !s.across(e) {
				continue
			}
			attacks++
			assert.Equal(t, fight, e.FightID, "a blow between the sides is in the fight")
			assert.NotZero(t, e.Round)
			if s.company[e.Source.Key()] {
				companyDamage += e.Damage
			} else {
				enemyDamage += e.Damage
			}
			// Nobody strikes after falling (29b fixed interceptors that did).
			_, fell := slainAt[e.Source.Key()]
			assert.False(t, fell, "%s strikes after falling", e.Source.Name)
		case combatstream.Death:
			assert.Equal(t, fight, e.FightID)
			if e.Outcome == combatstream.OutcomeSlain {
				deaths[e.Target.Key()]++
				slainAt[e.Target.Key()] = e.Round
				if s.bandits[e.Target.Key()] && s.company[e.Source.Key()] {
					kills[e.Source.Key()]++
				}
			}
		}
	}
	require.Positive(t, attacks)
	for key := range s.bandits {
		assert.Equal(t, 1, deaths[key], "bandit %s dies once", key)
	}

	sum := ends[0].Summary
	require.NotNil(t, sum)
	assert.Equal(t, companyDamage, sum.CompanyDamage, "the summary's company damage is the sum of the company's blows")
	assert.Equal(t, enemyDamage, sum.EnemyDamage, "and the enemies' of theirs")
	summed := 0
	for _, a := range sum.MostDamage {
		summed += a.Value
	}
	assert.Equal(t, companyDamage, summed, "most damage splits the company's damage")
	killed := 0
	for _, a := range sum.Kills {
		assert.Equal(t, kills[a.Who.Key()], a.Value, "kills by %s", a.Who.Name)
		killed += a.Value
	}
	assert.Equal(t, len(s.bandits), killed, "every bandit was killed by someone in the company")
	require.Len(t, sum.Enemies, len(s.bandits))
	for _, e := range sum.Enemies {
		assert.Equal(t, combatstream.EndingSlain, e.Ending, e.Ref.Name)
	}
	require.NotEmpty(t, sum.Company)
	assert.Equal(t, 7, sum.Company[0].Ref.UserId)
	assert.Equal(t, b.aria.Character.Health, sum.Company[0].Health, "the company's health is read from the world")

	// The leader is sent the summary once, at the end.
	assert.Equal(t, 1, strings.Count(seen, summaryHeading))
	assert.Contains(t, seen, "Damage dealt   Company ")
	for _, line := range combatstream.Render(*sum, 7) {
		assert.Contains(t, seen, line)
	}

	assert.Equal(t, turn, util.GetTurnCount(), "the clock never moves")
	assert.Equal(t, round, util.GetRoundCount())
}

// TestBattleSummaryCanBeTurnedOff: `set battlesummary` turns the summary
// off; the fight is still reported on the stream.
func TestBattleSummaryCanBeTurnedOff(t *testing.T) {
	b := newBrawl(t)
	got := b.listen()
	assert.Contains(t, b.cmd("set", ""), "battlesummary:")
	assert.Contains(t, b.cmd("set", "battlesummary"), "Battle summary toggled OFF")

	b.cmd("attack", "bandit cutthroat")
	seen := b.fightItOut(200)
	assert.NotContains(t, seen, summaryHeading)
	ended := 0
	for _, e := range *got {
		if e.Kind == combatstream.FightEnd {
			ended++
			assert.NotNil(t, e.Summary)
		}
	}
	assert.Equal(t, 1, ended)

	assert.Contains(t, b.cmd("set", "battlesummary"), "Battle summary toggled ON")
}

// TestSoloFightIsNotAFight: with no companion present (the 29a upkeep's
// rule), blows are reported with fight id 0 and no summary is made.
func TestSoloFightIsNotAFight(t *testing.T) {
	b := newBrawl(t)
	got := b.listen()
	b.cmd("company", "dismiss all")
	require.Empty(t, b.companyInstances())

	b.cmd("attack", "bandit cutthroat")
	var seen []string
	for i := 0; i < 5; i++ {
		b.aria.Character.HealthMax.Value = 1000
		b.aria.Character.Health = 1000
		seen = append(seen, b.fight())
	}
	attacks := 0
	for _, e := range *got {
		assert.NotEqual(t, combatstream.FightStart, e.Kind)
		assert.NotEqual(t, combatstream.FightEnd, e.Kind)
		assert.Zero(t, e.FightID)
		if e.Kind == combatstream.Attack && e.Source.UserId == 7 {
			attacks++
		}
	}
	assert.Positive(t, attacks, "Aria's blows are still reported")
	assert.NotContains(t, strings.Join(seen, "\n"), summaryHeading)
}

// TestFightBreaksOffWhenTheLeaderLeaves: a company that walks out of a
// fight ends it as broken off, with the summary so far.
func TestFightBreaksOffWhenTheLeaderLeaves(t *testing.T) {
	b := newBrawl(t)
	got := b.listen()
	b.cmd("attack", "bandit cutthroat")
	b.aria.Character.HealthMax.Value = 1000
	b.aria.Character.Health = 1000
	b.fight()
	require.Len(t, combatstream.Default().OpenFights(), 1)

	// Everyone stops: Aria and her company stand down, and the bandits
	// lose interest (as when the company walks off).
	b.aria.Character.Aggro = nil
	for instance := range b.companyInstances() {
		if mob := mobs.GetInstance(instance); mob != nil {
			mob.Character.Aggro = nil
		}
	}
	for _, mob := range b.livingBandits() {
		mob.Character.Aggro = nil
	}
	b.road.RemovePlayer(7)
	b.aria.Character.RoomId = 1
	seen := b.fight()

	assert.Empty(t, combatstream.Default().OpenFights())
	last := (*got)[len(*got)-1]
	require.Equal(t, combatstream.FightEnd, last.Kind)
	assert.Equal(t, combatstream.OutcomeBrokenOff, last.Outcome)
	assert.Contains(t, seen, "── The fight breaks off ──")
}
