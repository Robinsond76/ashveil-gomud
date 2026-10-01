package company

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// summaryHeading ends a victory's heading, named or not: "── The fighting
// is over ──", "── The fight with a band of bandit cutthroats is over ──".
const summaryHeading = " is over ──"

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
		// Her company may fall: alone, she turns on the next bandit by
		// herself (Phase 32c).
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

	b.aimAt("bandit captain")
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
	assert.Equal(t, "a band of bandit cutthroats", sum.GroupName, "the fight is named after its group (32c)")
	assert.Contains(t, seen, "── The fight with a band of bandit cutthroats is over ──")
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

	b.aimAt("bandit cutthroat")
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

// TestFleeBreaksTheFightOff: a real `flee` through the round is reported
// in the fight, and the fight ends as broken off once the company is gone
// from the room, with the summary so far.
func TestFleeBreaksTheFightOff(t *testing.T) {
	b := newBrawl(t)
	got := b.listen()
	b.aimAt("bandit cutthroat")
	var seen []string
	fled := false
	for i := 0; i < 40 && !fled; i++ {
		b.aria.Character.HealthMax.Value = 1000
		b.aria.Character.Health = 1000
		if b.aria.Character.RoomId == b.road.RoomId && (b.aria.Character.Aggro == nil || b.aria.Character.Aggro.Type != characters.Retreat) {
			b.cmd("flee", "")
		}
		seen = append(seen, b.fight())
		fled = b.aria.Character.RoomId != b.road.RoomId
	}
	require.True(t, fled, "Aria gets away")
	for i := 0; i < 3 && len(combatstream.Default().OpenFights()) > 0; i++ {
		seen = append(seen, b.fight())
	}
	assert.Empty(t, combatstream.Default().OpenFights())

	var flee, end *combatstream.Event
	for i := range *got {
		switch e := &(*got)[i]; e.Kind {
		case combatstream.Flee:
			flee = e
		case combatstream.FightEnd:
			end = e
		}
	}
	require.NotNil(t, flee)
	require.NotNil(t, end)
	assert.Equal(t, 7, flee.Source.UserId)
	assert.Equal(t, end.FightID, flee.FightID, "the flee is in the fight")
	assert.Equal(t, combatstream.OutcomeBrokenOff, end.Outcome)
	assert.Contains(t, strings.Join(seen, "\n"), "── The fight with a band of bandit cutthroats breaks off ──")
}

// TestInterceptedBlowFellsTheLeaderThatRound (the 29b death fix): the
// leader shielding a companion behind her is struck through interception
// alone. A blow that drops her is reported, and she is dropped, in that
// same round; before 29b nothing named her in the round's affected list.
func TestInterceptedBlowFellsTheLeaderThatRound(t *testing.T) {
	b := newBrawl(t)
	got := b.listen()
	// One enemy, too tough to die: the captain.
	for name, ids := range b.bandits {
		if name == "bandit captain" {
			continue
		}
		for _, id := range ids {
			b.road.RemoveMob(id)
			mobs.DestroyInstance(id)
		}
		delete(b.bandits, name)
	}
	captain := mobs.GetInstance(b.bandits["bandit captain"][0])
	require.NotNil(t, captain)
	// Aria in front of Tamsin, in the middle column the lone captain faces.
	for _, mv := range []string{"move me 1 2", "move #1 2 2"} {
		require.Contains(t, b.cmd("formation", mv), "Placed")
	}
	tamsin := b.companion(1)
	for i := 0; i < 60; i++ {
		captain.Character.HealthMax.Value = 1000
		captain.Character.Health = 1000
		captain.Character.SetAggro(0, tamsin.InstanceId, characters.DefaultAttack)
		b.aria.Character.Health = 1
		b.aria.Character.Aggro = nil
		mark := len(*got)
		b.fight()
		var hit, fell *combatstream.Event
		for j := mark; j < len(*got); j++ {
			e := &(*got)[j]
			if e.Kind == combatstream.Attack && e.Source.MobInstanceId == captain.InstanceId && e.Target.UserId == 7 && e.Damage > 0 {
				hit = e
			}
			if e.Kind == combatstream.Death && e.Target.UserId == 7 {
				fell = e
			}
		}
		if hit == nil {
			assert.Nil(t, fell, "round %d: no blow, no fall", b.round)
			continue
		}
		require.NotNil(t, fell, "the intercepted blow drops Aria in its round")
		assert.Equal(t, hit.Round, fell.Round)
		assert.Equal(t, captain.InstanceId, fell.Source.MobInstanceId, "credited to the captain")
		return
	}
	t.Fatal("the captain never landed an intercepted blow")
}

// TestSpellEventsThroughTheRealRound: a cast's waiting round, its end, and
// what it did are reported through the round, with the shipped Minor Heal
// and Magic Missile: a heal on Aria, and spell damage on a bandit in the
// fight, counted in the summary's company damage.
func TestSpellEventsThroughTheRealRound(t *testing.T) {
	b := newBrawl(t)
	got := b.listen()
	b.aimAt("bandit cutthroat")
	b.aria.Character.HealthMax.Value = 1000
	b.aria.Character.Health = 1000
	b.fight() // the fight is open

	cast := func(spellId string, users, mobIds []int) *combatstream.Event {
		for i := 0; i < 60; i++ {
			b.aria.Character.HealthMax.Value = 1000
			b.aria.Character.Health = 500
			b.aria.Character.Aggro = &characters.Aggro{Type: characters.SpellCast, RoundsWaiting: 1,
				SpellInfo: characters.SpellAggroInfo{SpellId: spellId, TargetUserIds: users, TargetMobInstanceIds: mobIds}}
			mark := len(*got)
			b.fight() // the waiting round
			b.fight() // the cast
			var progress, complete *combatstream.Event
			for j := mark; j < len(*got); j++ {
				e := &(*got)[j]
				if e.Source.UserId != 7 || e.SpellId != spellId {
					continue
				}
				switch e.Kind {
				case combatstream.CastProgress:
					progress = e
				case combatstream.CastComplete:
					complete = e
				}
			}
			require.NotNil(t, progress, "the waiting round is reported")
			require.NotNil(t, complete, "the cast's end is reported")
			if complete.Outcome == combatstream.OutcomeCast {
				return complete
			}
			// Phase 30d1: a bandit's blow may break the chant; try again.
			assert.Contains(t, []string{combatstream.OutcomeFizzled, combatstream.OutcomeInterrupted}, complete.Outcome)
		}
		t.Fatalf("%s never went off", spellId)
		return nil
	}

	complete := cast("heal", []int{7}, nil)
	var heal *combatstream.Event
	for i := range *got {
		if e := &(*got)[i]; e.Kind == combatstream.Heal && e.Seq > complete.Seq && e.Target.UserId == 7 {
			heal = e
			break
		}
	}
	require.NotNil(t, heal, "the heal is reported")
	assert.GreaterOrEqual(t, heal.Amount, 2, "Minor Heal is 2d3")
	assert.LessOrEqual(t, heal.Amount, 6)
	assert.NotZero(t, heal.FightID, "healing the leader mid-fight is in the fight")

	living := b.livingBandits()
	require.NotEmpty(t, living)
	target := living[0]
	target.Character.HealthMax.Value = 1000
	target.Character.Health = 1000
	complete = cast("mm", nil, []int{target.InstanceId})
	var hit *combatstream.Event
	for i := range *got {
		if e := &(*got)[i]; e.Kind == combatstream.SpellHit && e.Seq > complete.Seq && e.Target.MobInstanceId == target.InstanceId {
			hit = e
			break
		}
	}
	require.NotNil(t, hit, "the spell's damage is reported")
	assert.Positive(t, hit.Damage)
	assert.NotZero(t, hit.FightID)
	target.Character.HealthMax.Value = 10 // back to a bandit's toughness
	target.Character.Health = 5

	// The fight's summary counts the spell with the blows.
	seen := b.fightItOut(200)
	var end *combatstream.Event
	companyDamage := 0
	for i := range *got {
		e := &(*got)[i]
		if (e.Kind == combatstream.Attack || e.Kind == combatstream.SpellHit) && e.FightID == hit.FightID && (e.Source.UserId == 7 || b.companyInstances()[e.Source.MobInstanceId] || e.Source.LeaderUserId == 7) {
			companyDamage += e.Damage
		}
		if e.Kind == combatstream.FightEnd {
			end = e
		}
	}
	require.NotNil(t, end)
	assert.Equal(t, companyDamage, end.Summary.CompanyDamage)
	assert.Contains(t, seen, "Healing        Company ")
}
