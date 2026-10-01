package combatstream

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skirmish runs a small fight through a stream and returns its summary.
func skirmish(t *testing.T) *Summary {
	t.Helper()
	s := New()
	id := s.Open(1, 100, "bandits#0", aria, []Ref{garrick, tamsin}, []Ref{captain, slinger})
	emit := func(e Event) { s.Emit(e) }

	emit(Event{Kind: Attack, Source: aria, Target: captain, Damage: 4, Outcome: OutcomeHit})
	emit(Event{Kind: Attack, Source: garrick, Target: captain, Damage: 9, Crit: true, Outcome: OutcomeCrit})
	emit(Event{Kind: Attack, Source: tamsin, Target: slinger, Damage: 0, Outcome: OutcomeMiss})
	emit(Event{Kind: SpellHit, Source: tamsin, Target: slinger, Damage: 3, SpellId: "spark"})
	emit(Event{Kind: Attack, Source: captain, Target: tamsin, Damage: 6, Outcome: OutcomeHit})
	emit(Event{Kind: Attack, Source: slinger, Target: aria, Damage: 2, Outcome: OutcomeHit})
	emit(Event{Kind: Heal, Source: tamsin, Target: aria, Amount: 2})
	emit(Event{Kind: Heal, Source: slinger, Target: slinger, Amount: 5}) // enemy healing is not counted
	emit(Event{Kind: StatusApplied, Source: garrick, Target: captain, Status: "dazed"})
	emit(Event{Kind: StatusApplied, Source: captain, Target: tamsin, Status: "dazed"})
	emit(Event{Kind: StatusApplied, Source: tamsin, Target: slinger, Status: "slowed"})
	emit(Event{Kind: Death, Target: captain, Outcome: OutcomeSlain})
	emit(Event{Kind: Flee, Source: slinger})

	sum, ok := s.EndFight(id, 9, OutcomeVictory, Final{Company: []MemberHealth{
		{Ref: aria, Health: 12, Max: 14},
		{Ref: garrick, Health: 15, Max: 15},
		{Ref: tamsin, Fallen: true},
	}})
	require.True(t, ok)
	return sum
}

func TestSummaryFold(t *testing.T) {
	sum := skirmish(t)
	assert.Equal(t, 16, sum.CompanyDamage, "4 + 9 + 3 (spell)")
	assert.Equal(t, 8, sum.EnemyDamage)
	assert.Equal(t, 2, sum.Healing)
	assert.Equal(t, []Amount{{Who: garrick, Value: 9}, {Who: aria, Value: 4}, {Who: tamsin, Value: 3}}, sum.MostDamage)
	require.NotNil(t, sum.HighestHit)
	assert.Equal(t, Hit{Source: garrick, Target: captain, Damage: 9, Crit: true}, *sum.HighestHit)
	assert.Equal(t, []Count{{Name: "dazed", Count: 2}, {Name: "slowed", Count: 1}}, sum.Effects)
	assert.Equal(t, []Amount{{Who: garrick, Value: 1}}, sum.Kills)
	assert.Equal(t, []EnemyEnding{{Ref: captain, Ending: EndingSlain}, {Ref: slinger, Ending: EndingFled}}, sum.Enemies)
	assert.Equal(t, OutcomeVictory, sum.Outcome)
	assert.Equal(t, 7, sum.LeaderUserId)
}

func TestSummaryStillStandingAndInterruptsAndGuards(t *testing.T) {
	s := New()
	id := s.Open(1, 100, "bandits#0", aria, []Ref{tamsin}, []Ref{captain})
	s.Emit(Event{Kind: Interrupt, Source: aria, Target: captain, Outcome: OutcomeSucceeded, Status: "Crushing Blow"})
	s.Emit(Event{Kind: Interrupt, Source: tamsin, Target: captain, Outcome: OutcomeFailed})
	s.Emit(Event{Kind: Interrupt, Source: captain, Target: tamsin, Outcome: OutcomeSucceeded})
	// Review (30d1b): a company blow a company chant withstood is no
	// failed interrupt, nor is an enemy's blow a company chant withstood.
	s.Emit(Event{Kind: Interrupt, Source: aria, Target: tamsin, Outcome: OutcomeFailed})
	s.Emit(Event{Kind: Interrupt, Source: captain, Target: tamsin, Outcome: OutcomeFailed})
	s.Emit(Event{Kind: GuardUsed, Source: tamsin})
	s.Emit(Event{Kind: Heal, Source: tamsin, Target: tamsin, Amount: 3, HeldBack: 1})
	sum, _ := s.EndFight(id, 3, OutcomeBrokenOff, Final{})
	assert.Equal(t, []EnemyEnding{{Ref: captain, Ending: EndingStanding}}, sum.Enemies)
	assert.Equal(t, []string{"Crushing Blow"}, sum.InterruptsDealt)
	assert.Equal(t, 1, sum.InterruptsFailed)
	assert.Equal(t, 1, sum.InterruptsTaken)
	assert.Equal(t, []Amount{{Who: tamsin, Value: 1}}, sum.Guards)
	assert.Equal(t, 1, sum.HeldBack)

	lines := Render(*sum, 7)
	assert.Equal(t, "── The fight breaks off ──", lines[0])
	assert.Contains(t, lines, "Healing        Company 3 (1 held back by a wound)")
	assert.Contains(t, lines, "Interrupts     dealt 1 (Crushing Blow) · failed 1 · taken 1")
	assert.Contains(t, lines, "Guards         Tamsin Reed 1")
	assert.Contains(t, lines, "Enemies        bandit captain still standing")
}

// With no failed interrupts the line leaves "failed" out.
func TestSummaryInterruptsWithoutFailures(t *testing.T) {
	s := New()
	id := s.Open(1, 100, "bandits#0", aria, []Ref{tamsin}, []Ref{captain})
	s.Emit(Event{Kind: Interrupt, Source: aria, Target: captain, Outcome: OutcomeSucceeded, Status: "Withering Hex"})
	s.Emit(Event{Kind: Interrupt, Source: tamsin, Target: captain, Outcome: OutcomeSucceeded, Status: "Withering Hex"})
	s.Emit(Event{Kind: Interrupt, Source: captain, Target: tamsin, Outcome: OutcomeSucceeded, Status: "Minor Heal"})
	sum, _ := s.EndFight(id, 3, OutcomeBrokenOff, Final{})
	assert.Contains(t, Render(*sum, 7), "Interrupts     dealt 2 (Withering Hex, Withering Hex) · taken 1")
}

func TestRender(t *testing.T) {
	lines := Render(*skirmish(t), 7)
	assert.Equal(t, []string{
		"── The fighting is over ──",
		"Damage dealt   Company 16 · Enemies 8",
		"Healing        Company 2",
		"Most damage    Garrick Vane 9 · You 4 · Tamsin Reed 3",
		"Highest hit    Garrick Vane 9 on bandit captain (critical)",
		"Effects        dazed 2 · slowed 1",
		"Kills          Garrick Vane 1",
		"Enemies        bandit captain slain · bandit slinger fled",
		"Company        You 12/14 · Garrick Vane 15/15 · Tamsin Reed fallen",
	}, lines)
}

// Phase 30g2: every defended strike counts to the defender's side, a
// round that also hit included; the line shows only what isn't zero.
func TestSummaryCountsDefenses(t *testing.T) {
	s := New()
	id := s.Open(1, 100, "bandits#0", aria, []Ref{tamsin}, []Ref{captain})
	s.Emit(Event{Kind: Attack, Source: captain, Target: tamsin, Outcome: OutcomeMiss, Defenses: []string{"blocked"}})
	s.Emit(Event{Kind: Attack, Source: captain, Target: tamsin, Outcome: OutcomeHit, Damage: 3, Defenses: []string{"blocked"}})
	s.Emit(Event{Kind: Attack, Source: captain, Target: aria, Outcome: OutcomeMiss, Defenses: []string{"parried"}})
	s.Emit(Event{Kind: Attack, Source: aria, Target: captain, Outcome: OutcomeMiss, Defenses: []string{"dodged", "dodged"}})
	s.Emit(Event{Kind: Attack, Source: aria, Target: captain, Outcome: OutcomeMiss})
	sum, _ := s.EndFight(id, 3, OutcomeBrokenOff, Final{})
	assert.Equal(t, DefenseCounts{Blocked: 2, Parried: 1}, sum.CompanyDefenses)
	assert.Equal(t, DefenseCounts{Dodged: 2}, sum.EnemyDefenses)
	assert.Equal(t, 3, sum.EnemyDamage, "a round with a defended strike still counts its damage")
	assert.Contains(t, Render(*sum, 7), "Defenses       Company 2 blocked, 1 parried · Enemies 2 dodged")
}

func TestRenderLeavesOutEmptyLines(t *testing.T) {
	s := New()
	id := s.Open(1, 100, "bandits#0", aria, nil, []Ref{captain})
	s.Emit(Event{Kind: Attack, Source: captain, Target: aria, Outcome: OutcomeMiss})
	sum, _ := s.EndFight(id, 2, OutcomeDefeat, Final{Company: []MemberHealth{{Ref: aria, Health: -2, Max: 14}}})
	lines := Render(*sum, 99)
	assert.Equal(t, []string{
		"── The company is beaten ──",
		"Damage dealt   Company 0 · Enemies 0",
		"Enemies        bandit captain still standing",
		"Company        Aria -2/14",
	}, lines, "another viewer reads the leader's name")
}

// TestSlainLeaderIsFallenAndEnemyEndings (review L3, L4): a leader slain in
// the fight reads as fallen though respawned at full health; an enemy
// that walked out "left"; a practice foe is "beaten" and no kill.
func TestSlainLeaderIsFallenAndEnemyEndings(t *testing.T) {
	s := New()
	straw := Ref{MobInstanceId: 51, Name: "straw footman"}
	id := s.Open(1, 100, "straw#0", aria, []Ref{garrick}, []Ref{captain, slinger, straw})
	s.Emit(Event{Kind: Attack, Source: captain, Target: aria, Damage: 30})
	s.Emit(Event{Kind: Death, Target: aria, Outcome: OutcomeSlain})
	s.Emit(Event{Kind: Attack, Source: garrick, Target: straw, Damage: 3})
	d, _ := s.Emit(Event{Kind: Death, Target: straw, Outcome: OutcomeBeaten})
	assert.Equal(t, garrick, d.Source)
	_, ok := s.Emit(Event{Kind: Death, Target: straw, Outcome: OutcomeSlain})
	assert.False(t, ok, "a beaten foe doesn't die again")

	sum, _ := s.EndFight(id, 5, OutcomeVictory, Final{
		Company: []MemberHealth{{Ref: aria, Health: 14, Max: 14}, {Ref: garrick, Health: 9, Max: 15}},
		Gone:    []Ref{slinger},
	})
	assert.True(t, sum.Company[0].Fallen)
	assert.False(t, sum.Company[1].Fallen)
	assert.Equal(t, []EnemyEnding{
		{Ref: captain, Ending: EndingStanding},
		{Ref: slinger, Ending: EndingLeft},
		{Ref: straw, Ending: EndingBeaten},
	}, sum.Enemies)
	assert.Empty(t, sum.Kills, "beating a practice foe is no kill")
	assert.Contains(t, Render(*sum, 7), "Company        You fallen · Garrick Vane 9/15")
}

func TestNamedFightHeadings(t *testing.T) {
	cases := map[string]string{
		OutcomeVictory:   "── The fight with a band of ruffians is over ──",
		OutcomeDefeat:    "── The company is beaten by a band of ruffians ──",
		OutcomeBrokenOff: "── The fight with a band of ruffians breaks off ──",
	}
	for outcome, want := range cases {
		s := New()
		id := s.Open(1, 100, "ruffians#0", aria, nil, []Ref{captain})
		s.Name(id, "a band of ruffians")
		info, ok := s.Fight(id)
		require.True(t, ok)
		assert.Equal(t, "a band of ruffians", info.GroupName)
		sum, ok := s.EndFight(id, 3, outcome, Final{})
		require.True(t, ok)
		assert.Equal(t, "a band of ruffians", sum.GroupName)
		assert.Equal(t, want, Render(*sum, 7)[0], outcome)
	}
}

func TestUnnamedFightHeadingsAreUnchanged(t *testing.T) {
	assert.Equal(t, "── The company is beaten ──", Render(Summary{Outcome: OutcomeDefeat}, 0)[0])
	assert.Equal(t, "── The fight breaks off ──", Render(Summary{Outcome: OutcomeBrokenOff}, 0)[0])
	assert.Equal(t, "── The fighting is over ──", Render(Summary{Outcome: OutcomeVictory}, 0)[0])
}

// Phase 30a: a status's damage counts to the side that suffers it's foes;
// a lost action counts to nobody.
func TestSummaryCountsStatusDamage(t *testing.T) {
	s := New()
	id := s.Open(1, 100, "bandits#0", aria, []Ref{tamsin}, []Ref{captain})
	s.Emit(Event{Kind: StatusTick, Target: captain, Damage: 3, Status: "bleeding"})
	s.Emit(Event{Kind: StatusTick, Target: tamsin, Damage: 2, Status: "burning"})
	s.Emit(Event{Kind: StatusTick, Target: captain, Status: "staggered", Outcome: OutcomeLostAction})
	sum, ok := s.EndFight(id, 3, OutcomeVictory, Final{})
	require.True(t, ok)
	assert.Equal(t, 3, sum.CompanyDamage, "the captain's bleeding counts to the company")
	assert.Equal(t, 2, sum.EnemyDamage, "Tamsin's burning counts to the enemies")
	assert.Empty(t, sum.MostDamage, "no attacker is credited")
}
