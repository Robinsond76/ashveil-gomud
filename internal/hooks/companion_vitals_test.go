package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
)

// Phase 32d/33h2: companions regain health out of combat while their
// leader is online, and not in combat; health stops at the wound limit.
// Phase 35b: they regain no mana, and health only trickles to half of max.
func TestRegenCompanionVitals(t *testing.T) {
	const room = 990301
	leader := users.NewUserRecord(93010, 1)
	users.SetTestUser(leader)
	t.Cleanup(func() { users.RemoveTestUser(93010) })

	idle := engagementMob(t, 9301, 4, room)
	fighting := engagementMob(t, 9302, 4, room)
	fighting.Character.Aggro = &characters.Aggro{MobInstanceId: 1}
	offline := engagementMob(t, 9303, 4, room)
	stranger := engagementMob(t, 9304, 4, room)
	wounded := engagementMob(t, 9305, 4, room)
	wounded.Character.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "arm", Points: 15}}
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
	assert.Equal(t, 5, idle.Character.Mana, "Phase 35b: no passive mana")
	assert.Equal(t, 4+idle.Character.HealthPerRound(), idle.Character.Health, "health trickles back")
	assert.Equal(t, 4, fighting.Character.Health, "not in combat")
	assert.Equal(t, 4, offline.Character.Health, "not while the leader is away")
	assert.Equal(t, 4, stranger.Character.Health, "not a mob outside a company")
	assert.Equal(t, 5, wounded.Character.Health, "a wound limit of 5 stops it below half")
	assert.Equal(t, 5, wounded.Character.Mana)
	assert.Equal(t, 0, downed.Character.Health, "the downed aren't raised")

	for range 50 {
		regenCompanionVitals(leaderOf)
	}
	assert.Equal(t, 10, idle.Character.Health, "Phase 35b: the trickle stops at half of max")
	assert.Equal(t, 5, idle.Character.Mana, "and mana never comes back")

	idle.Character.Health = 15
	regenCompanionVitals(leaderOf)
	assert.Equal(t, 15, idle.Character.Health, "above half, nothing")
}
