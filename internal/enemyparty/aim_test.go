package enemyparty

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 32d: a group's foes as an attacker sees them, and aims by rule.
// With no company formation the attacker isn't placed, so it reaches
// everyone (as at the attack gates).
func TestAimByRule(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	leader := users.NewUserRecord(92010, 1)
	leader.Character.Health, leader.Character.HealthMax.Value = 4, 20 // 20%
	users.SetTestUser(leader)
	t.Cleanup(func() { users.RemoveTestUser(92010) })

	boss := testMob(t, 9201, 40)  // toughest: the leader, front row
	grunt := testMob(t, 9202, 20) // front row
	rat := testMob(t, 9203, 6)    // front row, the weakest
	rat.Character.Health = 5
	grunt.Character.Health = 8 // 40%: the most wounded by fraction
	boss.SpawnGroup, grunt.SpawnGroup, rat.SpawnGroup = "g", "g", "g"
	grunt.Character.Aggro = &characters.Aggro{UserId: 92010}
	room := testRoom(t, 990201, 9201, 9202, 9203)
	g, ok := GroupOf(room, 9201)
	require.True(t, ok)

	foes := Foes(g, Attacker{LeaderId: 92010, Key: company.LeaderMemberKey})
	require.Len(t, foes, 3)
	byId := map[int]strategy.Foe{}
	for _, f := range foes {
		byId[f.ID] = f
	}
	assert.True(t, byId[9201].Leader, "the toughest leads")
	assert.False(t, byId[9202].Leader)
	assert.Equal(t, 20, byId[9202].StrikesPct, "the grunt strikes the player, at 20%")
	assert.Equal(t, -1, byId[9203].StrikesPct)

	aim := func(rule strategy.Rule) int {
		id, ok := Aim(g, Attacker{LeaderId: 92010, Key: company.LeaderMemberKey, Rule: rule})
		require.True(t, ok)
		return id
	}
	assert.Equal(t, 9203, aim(strategy.Weakest))
	assert.Equal(t, 9201, aim(strategy.Strongest))
	assert.Equal(t, 9202, aim(strategy.Wounded))
	assert.Equal(t, 9201, aim(strategy.Leader))
	assert.Equal(t, 9202, aim(strategy.Defend))

	// A fallen member is not a foe; when the leader falls the next toughest leads.
	boss.Character.Health = 0
	foes = Foes(g, Attacker{LeaderId: 92010, Key: company.LeaderMemberKey})
	require.Len(t, foes, 2)
	assert.True(t, foes[0].Leader && foes[0].ID == 9202)

	_, ok = RuleChoice(g, Attacker{LeaderId: 92010, Key: company.LeaderMemberKey, Rule: strategy.Assist, AssistId: 9999}, 0)
	assert.False(t, ok, "assist with no target of the player's has no choice of its own")
	id, ok := RuleChoice(g, Attacker{LeaderId: 92010, Key: company.LeaderMemberKey, Rule: strategy.Assist, AssistId: 9203}, 0)
	assert.True(t, ok)
	assert.Equal(t, 9203, id)
}

// 32d review: defend keeps a foe that ties with its choice, and a foe
// chanting a harmful spell at one of us counts as striking them.
func TestDefendKeepsATieAndCountsSpells(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	leader := users.NewUserRecord(92020, 1)
	leader.Character.Health, leader.Character.HealthMax.Value = 5, 20
	users.SetTestUser(leader)
	t.Cleanup(func() { users.RemoveTestUser(92020) })
	a, b := testMob(t, 9211, 20), testMob(t, 9212, 20)
	a.SpawnGroup, b.SpawnGroup = "d", "d"
	a.Character.Aggro = &characters.Aggro{UserId: 92020}
	b.Character.Aggro = &characters.Aggro{UserId: 92020}
	room := testRoom(t, 990211, 9211, 9212)
	g, ok := GroupOf(room, 9211)
	require.True(t, ok)
	att := Attacker{LeaderId: 92020, Key: company.LeaderMemberKey, Rule: strategy.Defend}
	first, _ := RuleChoice(g, att, 0)
	other := 9211
	if first == 9211 {
		other = 9212
	}
	id, ok := RuleChoice(g, att, other)
	require.True(t, ok)
	assert.Equal(t, other, id, "both strike the player: the current foe is kept")

	assert.Equal(t, -1, StrikesPct(&characters.Aggro{Type: characters.SpellCast, SpellInfo: characters.SpellAggroInfo{SpellId: "nosuch", TargetUserIds: []int{92020}}}, 92020), "not a harmful spell")
}
