package company

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestRanksCostMatchesAPlayersTraining(t *testing.T) {
	assert.Equal(t, 1, RanksCost(0, 1))
	assert.Equal(t, 3, RanksCost(0, 2))
	assert.Equal(t, 10, RanksCost(0, 4))
	assert.Equal(t, 7, RanksCost(2, 4))
	assert.Zero(t, RanksCost(3, 3))
	assert.Zero(t, RanksCost(3, 1), "never negative")
}

// TestTrainingPointsAreDerived is the plan's points table: a level-6
// companion (6 earned at the shipped rate) that trains Cooking 1 and 2 has
// 3 left; granted ranks cost nothing; a lost level leaves a shortfall.
func TestTrainingPointsAreDerived(t *testing.T) {
	c := Companion{ID: 1}
	assert.Equal(t, 6, c.TrainingPoints(6))
	c.Skills = map[string]int{"cooking": 2}
	assert.Equal(t, 3, c.TrainingPoints(6))
	assert.Equal(t, -1, c.TrainingPoints(2), "a level lost to death owes points; the rank stays")

	recruit := Companion{ID: 2, Skills: map[string]int{"cooking": 2}, GrantedSkills: map[string]int{"cooking": 2}}
	assert.Equal(t, 1, recruit.TrainingPoints(1), "granted ranks cost nothing")
	recruit.Skills["cooking"] = 3
	assert.Equal(t, 3, recruit.SpentTrainingPoints(), "only the ranks past the grant are paid for")
}

func TestSetSkillRankAndGrantSkills(t *testing.T) {
	r := NewRegistry()
	r.Put(Record{LeaderUserID: 7, Companions: []Companion{{ID: 1, MobTemplateID: 61}, {ID: 2, MobTemplateID: 62}}})

	require.NoError(t, r.GrantSkills(7, 2, map[string]int{"Cooking": 1, "": 3, "scribe": 0}))
	require.NoError(t, r.SetSkillRank(7, 1, "Cooking", 2))
	assert.ErrorIs(t, r.SetSkillRank(7, 9, "cooking", 1), ErrUnknownMember)
	assert.ErrorIs(t, r.GrantSkills(8, 1, nil), ErrUnknownMember)

	record, _ := r.Get(7)
	assert.Equal(t, map[string]int{"cooking": 2}, record.Companions[0].Skills)
	assert.Nil(t, record.Companions[0].GrantedSkills)
	assert.Equal(t, map[string]int{"cooking": 1}, record.Companions[1].Skills)
	assert.Equal(t, map[string]int{"cooking": 1}, record.Companions[1].GrantedSkills)

	require.NoError(t, r.SetSkillRank(7, 2, "cooking", 0))
	record, _ = r.Get(7)
	assert.Equal(t, 1, record.Companions[1].SkillRank("cooking"), "never below the rank it arrived with")

	// Get hands out copies: changing one never reaches the registry.
	record.Companions[0].Skills["cooking"] = 4
	record.Companions[0].Identity().Skills["cooking"] = 4
	again, _ := r.Get(7)
	assert.Equal(t, 2, again.Companions[0].SkillRank("cooking"))
	assert.Equal(t, map[string]int{"cooking": 2}, again.Companions[0].Identity().Skills)
}

func TestTrainedRanksRoundTripAndOldSavesLoad(t *testing.T) {
	r := NewRegistry()
	r.Put(Record{LeaderUserID: 7, Companions: []Companion{{ID: 1, MobTemplateID: 61,
		Skills: map[string]int{"cooking": 3}, GrantedSkills: map[string]int{"cooking": 1}}}})
	data, err := yaml.Marshal(r)
	require.NoError(t, err)
	assert.Contains(t, string(data), "granted_skills")
	var back Registry
	require.NoError(t, yaml.Unmarshal(data, &back))
	record, _ := back.Get(7)
	assert.Equal(t, map[string]int{"cooking": 3}, record.Companions[0].Skills)
	assert.Equal(t, map[string]int{"cooking": 1}, record.Companions[0].GrantedSkills)

	// A record from before 35c has neither field: no ranks, nothing spent.
	var old Registry
	require.NoError(t, yaml.Unmarshal([]byte("companies:\n  7:\n    leader_user_id: 7\n    companions:\n      - id: 1\n        mob_template_id: 61\n"), &old))
	record, _ = old.Get(7)
	require.Len(t, record.Companions, 1)
	assert.Nil(t, record.Companions[0].Skills)
	assert.Equal(t, 4, record.Companions[0].TrainingPoints(4))
}

// scriptedRand returns its values in order, then zeros.
type scriptedRand struct{ values []int }

func (s *scriptedRand) Intn(n int) int {
	if len(s.values) == 0 {
		return 0
	}
	v := s.values[0]
	s.values = s.values[1:]
	return v % n
}

func skilledRules() RosterRules {
	rules := testRosterRules()
	rules.Archetypes = []RosterArchetype{{Archetype: "wizard", MobTemplateID: 82, Weight: 1}}
	rules.Traits = nil
	rules.BynamePercent = 0
	rules.SkilledPercent, rules.SkillRank2Percent, rules.SkillPricePercent = 20, 20, 15
	return rules
}

func TestCandidatesMayArriveWithAnOptionalSkill(t *testing.T) {
	rules := skilledRules()
	learnable := func(arch string) []string {
		assert.Equal(t, "wizard", arch)
		return []string{"cooking", "scribe"}
	}
	ctx := RosterContext{LeaderLevel: 2, Learnable: learnable}
	// Rolls: archetype, name, byname, level, alignment, then skilled?,
	// which skill, rank 2?
	plain, _ := generateCandidate(rules, ctx, nil, &scriptedRand{values: []int{0, 0, 99, 1, 0, 20}})
	assert.Empty(t, plain.Skill, "20%: a roll of 20 misses")
	base := plain.Price

	one, _ := generateCandidate(rules, ctx, nil, &scriptedRand{values: []int{0, 0, 99, 1, 0, 19, 1, 20}})
	assert.Equal(t, "scribe", one.Skill)
	assert.Equal(t, 1, one.SkillRank)
	assert.Equal(t, base*115/100, one.Price, "15% more per rank")
	assert.Equal(t, map[string]int{"scribe": 1}, one.Skills())

	two, _ := generateCandidate(rules, ctx, nil, &scriptedRand{values: []int{0, 0, 99, 1, 0, 0, 0, 19}})
	assert.Equal(t, "cooking", two.Skill)
	assert.Equal(t, 2, two.SkillRank)
	assert.Equal(t, base*130/100, two.Price)

	ctx.Learnable = func(string) []string { return nil }
	none, _ := generateCandidate(rules, ctx, nil, &scriptedRand{values: []int{0, 0, 99, 1, 0, 0}})
	assert.Empty(t, none.Skill, "an archetype with nothing to learn arrives untrained")
	assert.Equal(t, base, none.Price)
	assert.Nil(t, none.Skills())
}

// The skill roll only happens when skills are on, so rosters seeded before
// 35c (and their tests) draw exactly as before.
func TestSkillRollsLeaveOldRostersUnchanged(t *testing.T) {
	rules := testRosterRules()
	ctx := RosterContext{Now: 10000, LeaderLevel: 3}
	before, _ := RefreshRoster(Roster{RoomID: 1}, rules, ctx, rand.New(rand.NewSource(5)))
	rules.SkilledPercent = 20
	after, _ := RefreshRoster(Roster{RoomID: 1}, rules, ctx, rand.New(rand.NewSource(5)))
	assert.Equal(t, before, after, "no Learnable: no extra rolls")

	// With skills on, about a fifth of many candidates arrive trained.
	ctx.Learnable = func(string) []string { return []string{"cooking"} }
	rng := rand.New(rand.NewSource(9))
	skilled, rank2, total := 0, 0, 0
	for i := 0; i < 2000; i++ {
		c, ok := generateCandidate(rules, ctx, nil, rng)
		require.True(t, ok)
		total++
		if c.Skill != "" {
			skilled++
			if c.SkillRank == 2 {
				rank2++
			}
		}
	}
	assert.InDelta(t, 0.20, float64(skilled)/float64(total), 0.04)
	assert.InDelta(t, 0.0, float64(rank2), 1, "SkillRank2Percent is 0 in these rules")
}
