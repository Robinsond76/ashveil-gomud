package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/stretchr/testify/assert"
)

// Phase 39b: the Samurai's battle state, through the hooks that keep it.

// samuraiMob is a samurai companion (a mob with the runtime archetype) of a
// level and class.
func samuraiMob(t *testing.T, id, level int, class string) *statusHolder {
	t.Helper()
	m := engagementMob(t, id, 100, 1)
	m.Character.Level = level
	m.Character.HealthMax.Value = 100
	m.Character.HPArchetype = "samurai"
	m.Character.SetClassState(class, nil)
	m.Character.RTState()
	h := mobHolder(m)
	return &h
}

func TestIaijutsuStartsTheActionMeterHalfATurnUp(t *testing.T) {
	ResetTempoForTest()
	t.Cleanup(ResetTempoForTest)
	s := samuraiMob(t, 8601, 1, "")
	plain := engagementMob(t, 8602, 100, 1)
	for _, c := range []*characters.Character{&plain.Character, s.char} {
		c.Stats.Speed.ValueAdj = 30
		c.SetAggro(1, 0, characters.DefaultAttack, 0)
	}
	fillTempo(caster{mobId: 8601}, s.char)
	fillTempo(caster{mobId: 8602}, &plain.Character)
	assert.Equal(t, 1, tempoTurns[caster{mobId: 8601}], "the opening is still one turn")
	assert.InDelta(t, 50, tempoMeters[caster{mobId: 8601}].meter.Points, 1e-9, "with half a turn banked")
	assert.Zero(t, tempoMeters[caster{mobId: 8602}].meter.Points, "everyone else starts even")
}

func TestZanshinGivesHalfATurnWhenItFellsAFoeOncePerRound(t *testing.T) {
	ResetTempoForTest()
	t.Cleanup(ResetTempoForTest)
	s := samuraiMob(t, 8611, 8, "")
	s.char.Stats.Speed.ValueAdj = 30
	s.char.SetAggro(1, 0, characters.DefaultAttack, 0)
	who := caster{mobId: 8611}
	fillTempo(who, s.char)
	clear(tempoTurns)
	fillTempo(who, s.char)
	base := tempoMeters[who].meter.Points

	foe := engagementMob(t, 8612, 0, 1)
	felled := mobHolder(foe)
	hit := combat.AttackResult{Hit: true, DamageToTarget: 9}
	combatRound.Store(40)
	samuraiBlow(*s, felled, hit)
	assert.InDelta(t, min(99, base+50), tempoMeters[who].meter.Points, 1e-9, "half a turn")
	assert.Equal(t, uint64(40), s.char.RT.ZanshinRound)

	before := tempoMeters[who].meter.Points
	samuraiBlow(*s, felled, hit)
	assert.Equal(t, before, tempoMeters[who].meter.Points, "once a round")
	combatRound.Store(41)
	samuraiBlow(*s, felled, hit)
	assert.Equal(t, min(99, before+50), tempoMeters[who].meter.Points, "and again the next")

	// A foe left standing, a miss, and a Samurai without Zanshin give nothing.
	tempoMeters[who].meter.Points = 0
	combatRound.Store(42)
	standing := mobHolder(engagementMob(t, 8613, 20, 1))
	samuraiBlow(*s, standing, hit)
	samuraiBlow(*s, felled, combat.AttackResult{DamageToTarget: 0})
	young := samuraiMob(t, 8614, 7, "")
	young.char.SetAggro(1, 0, characters.DefaultAttack, 0)
	fillTempo(caster{mobId: 8614}, young.char)
	samuraiBlow(*young, felled, hit)
	assert.Zero(t, tempoMeters[who].meter.Points)
	assert.InDelta(t, 50, tempoMeters[caster{mobId: 8614}].meter.Points, 1e-9, "only its opening bank, no Zanshin before level 8")
}

func TestFocusCountsQuietRoundsAndAHitResetsIt(t *testing.T) {
	s := samuraiMob(t, 8621, 3, "")
	side := []actor{{who: caster{mobId: 8621}, char: s.char}}
	round := func() { samuraiRound(side) }

	round()
	assert.Zero(t, s.char.RT.Quiet, "the first round has nothing behind it")
	round()
	round()
	assert.Equal(t, 2, s.char.RT.Quiet)
	assert.Equal(t, 6, s.char.ClassCrit())

	foe := mobHolder(engagementMob(t, 8622, 50, 1))
	samuraiBlow(foe, *s, combat.AttackResult{Hit: true, DamageToTarget: 3})
	assert.True(t, s.char.RT.Struck)
	round()
	assert.Zero(t, s.char.RT.Quiet, "a blow that landed resets it")
	assert.False(t, s.char.RT.Struck)
	samuraiBlow(foe, *s, combat.AttackResult{Hit: true, DamageToTarget: 0})
	samuraiBlow(foe, *s, combat.AttackResult{})
	round()
	assert.Equal(t, 1, s.char.RT.Quiet, "a blow armor took whole, or a miss, doesn't")
}

func TestVengeanceCountsTheSideThatHasFallen(t *testing.T) {
	ronin := samuraiMob(t, 8631, 10, "ronin")
	a := func(id int, c *characters.Character) actor { return actor{who: caster{mobId: id}, char: c} }
	ally1 := engagementMob(t, 8632, 50, 1)
	ally2 := engagementMob(t, 8633, 50, 1)
	full := []actor{a(8631, ronin.char), a(8632, &ally1.Character), a(8633, &ally2.Character)}

	samuraiRound(full)
	assert.Zero(t, ronin.char.Aura.Fallen)
	assert.Equal(t, 3, ronin.char.RT.SidePeak)
	samuraiRound(full[:2])
	assert.Equal(t, 1, ronin.char.Aura.Fallen, "one has fallen")
	samuraiRound(full[:1])
	assert.Equal(t, 2, ronin.char.Aura.Fallen)
	assert.Equal(t, 3, ronin.char.RT.SidePeak, "the peak holds")

	kensai := samuraiMob(t, 8634, 10, "kensai")
	samuraiRound(full)
	samuraiRound(full[:1])
	assert.Zero(t, kensai.char.Aura.Fallen, "only a Ronin keeps count")
}

func TestBodyguardUsesAreCountedPerBattle(t *testing.T) {
	h := samuraiMob(t, 8641, 10, "hatamoto")
	assert.Equal(t, 2, bodyguardLeft(h.char))
	h.char.RT.Bodyguards = 1
	assert.Equal(t, 1, bodyguardLeft(h.char))
	h.char.Level = 20
	assert.Equal(t, 2, bodyguardLeft(h.char), "Loyal blade: three a battle")
	h.char.EndFightRT()
	assert.Equal(t, 3, bodyguardLeft(h.char), "a new battle renews them")
	assert.Zero(t, bodyguardLeft(samuraiMob(t, 8642, 10, "kensai").char))
}
