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

	text := captureText(t, func() { m.onPlayerSpawn(events.PlayerSpawn{UserId: 51}) })
	assert.Equal(t, 20+3+1+10+4, u.Character.TrainingPoints, "1+2 for peep, 1 for portal, 1+2+3+4 for tame, 4 for protection's retired level 4")
	assert.Equal(t, map[string]int{"track": 1, "protection": 3}, u.Character.Skills, "kept skills stay; protection drops to its cap; a level-0 entry goes silently")
	assert.Equal(t, 1, saves, "the refund is saved")
	assert.Contains(t, text, "No longer part of the world: peep, portal, tame, protection level 4.")
	assert.Contains(t, text, "18 training points")
	assert.NotContains(t, text, "scribe", "a level-0 entry is not refunded or named")

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
		for _, id := range []string{"changeform", "peep", "portal", "scribe", "tame"} {
			assert.False(t, a.ListsSkill(id), "%s lists %s", a.ID, id)
		}
	}
}
