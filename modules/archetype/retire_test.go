package archetype

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// TestRetiredSkillsRefundedOnceAtSpawn (33f1): a player still holding a
// retired skill gets back the training points it cost when they enter the
// world, the skill leaves their sheet, the user is saved, and a second
// spawn changes nothing.
func TestRetiredSkillsRefundedOnceAtSpawn(t *testing.T) {
	m := registered(t)
	saves := 0
	m.saveUser = func(*users.UserRecord) error { saves++; return nil }
	u := trainee(t, 51, 96008)
	u.Character.Skills = map[string]int{"peep": 2, "portal": 1, "tame": 4, "scribe": 0, "track": 1, "protection": 4}
	u.Character.ScribeReset = true // 36a: scribe is a live skill again, reset once (see TestOldScribeRankIsRefundedOnce)

	text := captureText(t, func() { m.onPlayerSpawn(events.PlayerSpawn{UserId: 51}) })
	assert.Equal(t, 20+3+1+10+4, u.Character.TrainingPoints, "1+2 for peep, 1 for portal, 1+2+3+4 for tame, 4 for protection's retired level 4")
	assert.Equal(t, map[string]int{"scribe": 0, "track": 1, "protection": 3}, u.Character.Skills, "kept skills stay; protection drops to its cap")
	assert.Equal(t, 1, saves, "the refund is saved")
	assert.Contains(t, text, "No longer part of the world: peep, portal, tame, protection level 4.")
	assert.Contains(t, text, "18 training points")

	text = captureText(t, func() { m.onPlayerSpawn(events.PlayerSpawn{UserId: 51}) })
	assert.Equal(t, 38, u.Character.TrainingPoints, "never refunded twice")
	assert.Equal(t, 1, saves)
	assert.NotContains(t, text, "No longer part of the world")
}

// TestShippedArchetypesClaimNoRetiredSkill (33f1): no archetype trains or
// grants a retired skill.
func TestShippedArchetypesClaimNoRetiredSkill(t *testing.T) {
	m, _ := testModule(t)
	for _, a := range m.table.List() {
		for _, id := range []string{"changeform", "peep", "portal", "tame"} {
			assert.False(t, a.ListsSkill(id), "%s lists %s", a.ID, id)
		}
	}
}

// TestSearchAndTradingAreRefunded (33f2): the skills 33f2 retires join the
// one-time refund.
func TestSearchAndTradingAreRefunded(t *testing.T) {
	m := registered(t)
	m.saveUser = func(*users.UserRecord) error { return nil }
	u := trainee(t, 52, 96008)
	u.Character.Skills = map[string]int{"search": 3, "trading": 2, "track": 1}
	text := captureText(t, func() { m.onPlayerSpawn(events.PlayerSpawn{UserId: 52}) })
	assert.Equal(t, 20+6+3, u.Character.TrainingPoints)
	assert.Equal(t, map[string]int{"track": 1}, u.Character.Skills)
	assert.Contains(t, text, "search, trading")
}

// TestOldScribeRankIsRefundedOnce (36a): scribe was retired in 33f1 and
// returns as a new caster skill. A character still holding the old rank
// gets its points back and loses the rank, once; ranks trained afterwards
// are kept.
func TestOldScribeRankIsRefundedOnce(t *testing.T) {
	m := registered(t)
	m.saveUser = func(*users.UserRecord) error { return nil }
	u := trainee(t, 53, 96008)
	u.Character.Skills = map[string]int{"scribe": 3, "track": 1}

	text := captureText(t, func() { m.onPlayerSpawn(events.PlayerSpawn{UserId: 53}) })
	assert.Equal(t, 20+6, u.Character.TrainingPoints, "1+2+3 for the old rank")
	assert.Equal(t, map[string]int{"track": 1}, u.Character.Skills)
	assert.True(t, u.Character.ScribeReset)
	assert.Contains(t, text, "scribe")

	u.Character.Skills["scribe"] = 2 // trained under the new skill
	text = captureText(t, func() { m.onPlayerSpawn(events.PlayerSpawn{UserId: 53}) })
	assert.Equal(t, 2, u.Character.Skills["scribe"], "a new rank is never taken back")
	assert.Equal(t, 26, u.Character.TrainingPoints)
	assert.NotContains(t, text, "No longer part of the world")
}
