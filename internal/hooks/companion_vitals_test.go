package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
)

// Phase 32d/33h2: companions regain mana and health out of combat while
// their leader is online, and not in combat; health stops at the wound
// limit.
func TestRegenCompanionVitals(t *testing.T) {
	const room = 990301
	leader := users.NewUserRecord(93010, 1)
	users.SetTestUser(leader)
	t.Cleanup(func() { users.RemoveTestUser(93010) })

	idle := engagementMob(t, 9301, 10, room)
	fighting := engagementMob(t, 9302, 10, room)
	fighting.Character.Aggro = &characters.Aggro{MobInstanceId: 1}
	offline := engagementMob(t, 9303, 10, room)
	stranger := engagementMob(t, 9304, 10, room)
	wounded := engagementMob(t, 9305, 10, room)
	wounded.Character.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "arm", Points: 10}}
	downed := engagementMob(t, 9306, 0, room)
	for _, m := range []*characters.Character{&idle.Character, &fighting.Character, &offline.Character, &stranger.Character, &wounded.Character, &downed.Character} {
		m.ManaMax.Value, m.Mana = 20, 5
		m.HealthMax.Value = 20
	}
	leaderOf := func(id int) (int, company.MemberKey, bool) {
		switch id {
		case 9301, 9302, 9305, 9306:
			return 93010, company.CompanionMemberKey(id), true
		case 9303:
			return 93099, company.CompanionMemberKey(id), true // leader not online
		}
		return 0, "", false
	}
	regenCompanionVitals(leaderOf)
	assert.Equal(t, 5+idle.Character.ManaPerRound(), idle.Character.Mana, "idle companion regains mana")
	assert.Equal(t, 10+idle.Character.HealthPerRound(), idle.Character.Health, "and health")
	assert.Equal(t, 5, fighting.Character.Mana, "not in combat")
	assert.Equal(t, 10, fighting.Character.Health, "not in combat")
	assert.Equal(t, 5, offline.Character.Mana, "not while the leader is away")
	assert.Equal(t, 10, offline.Character.Health, "not while the leader is away")
	assert.Equal(t, 5, stranger.Character.Mana, "not a mob outside a company")
	assert.Equal(t, 10, stranger.Character.Health, "not a mob outside a company")
	assert.Equal(t, 10, wounded.Character.Health, "already at its wound limit")
	assert.Equal(t, 5+wounded.Character.ManaPerRound(), wounded.Character.Mana, "a wound doesn't stop its mana")
	assert.Equal(t, 0, downed.Character.Health, "the downed aren't raised")

	idle.Character.Mana, idle.Character.Health = 20, 20
	regenCompanionVitals(leaderOf)
	assert.Equal(t, 20, idle.Character.Mana, "capped at its maximum")
	assert.Equal(t, 20, idle.Character.Health, "capped at its maximum")
}
