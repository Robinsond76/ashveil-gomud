package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// Phase 32d: companions regain mana out of combat while their leader is
// online, and not in combat.
func TestRegenCompanionMana(t *testing.T) {
	const room = 990301
	leader := users.NewUserRecord(93010, 1)
	users.SetTestUser(leader)
	t.Cleanup(func() { users.RemoveTestUser(93010) })

	idle := engagementMob(t, 9301, 10, room)
	fighting := engagementMob(t, 9302, 10, room)
	fighting.Character.Aggro = &characters.Aggro{MobInstanceId: 1}
	offline := engagementMob(t, 9303, 10, room)
	stranger := engagementMob(t, 9304, 10, room)
	for _, m := range []*characters.Character{&idle.Character, &fighting.Character, &offline.Character, &stranger.Character} {
		m.ManaMax.Value, m.Mana = 20, 5
	}
	leaderOf := func(id int) (int, company.MemberKey, bool) {
		switch id {
		case 9301, 9302:
			return 93010, company.CompanionMemberKey(id), true
		case 9303:
			return 93099, company.CompanionMemberKey(id), true // leader not online
		}
		return 0, "", false
	}
	regenCompanionMana(leaderOf)
	assert.Equal(t, 5+idle.Character.ManaPerRound(), idle.Character.Mana, "idle companion regains")
	assert.Equal(t, 5, fighting.Character.Mana, "not in combat")
	assert.Equal(t, 5, offline.Character.Mana, "not while the leader is away")
	assert.Equal(t, 5, stranger.Character.Mana, "not a mob outside a company")

	idle.Character.Mana = 20
	regenCompanionMana(leaderOf)
	assert.Equal(t, 20, idle.Character.Mana, "capped at its maximum")
}
