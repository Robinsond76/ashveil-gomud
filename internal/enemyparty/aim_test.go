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

	_, ok = RuleChoice(g, Attacker{LeaderId: 92010, Key: company.LeaderMemberKey, Rule: strategy.Assist, AssistId: 9999})
	assert.False(t, ok, "assist with no target of the player's has no choice of its own")
	id, ok := RuleChoice(g, Attacker{LeaderId: 92010, Key: company.LeaderMemberKey, Rule: strategy.Assist, AssistId: 9203})
	assert.True(t, ok)
	assert.Equal(t, 9203, id)
}
