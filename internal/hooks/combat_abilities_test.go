package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// TestReadiedStrikeIsSetBack (33e review): a readied opening strike or
// aimed shot whose swing never came (its foe fell first) goes back to a
// plain attack after the round: a melee weapon's default attack, a
// shooting weapon's shot.
func TestReadiedStrikeIsSetBack(t *testing.T) {
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	const slingID = 99441
	items.SetTestItemSpec(&items.ItemSpec{ItemId: slingID, Name: "test sling", Type: items.Weapon, Subtype: items.Shooting, Hands: 2})
	t.Cleanup(func() { items.RemoveTestItemSpec(slingID) })

	rogue := users.NewUserRecord(8801, 1)
	users.SetTestUser(rogue)
	rogue.Character.SetAggro(0, 5, characters.BackStab, 0)
	ranger := users.NewUserRecord(8802, 1)
	users.SetTestUser(ranger)
	ranger.Character.Equipment.Weapon = items.New(slingID)
	ranger.Character.SetAggro(0, 5, characters.BackStab, 0)
	t.Cleanup(ResetAbilitiesForTest)

	abilityStrikes[caster{userId: 8801}] = true
	abilityStrikes[caster{userId: 8802}] = true
	abilityStrikes[caster{userId: 8803}] = true // gone: nothing to do
	endAbilityStrikes()
	assert.Equal(t, characters.DefaultAttack, rogue.Character.Aggro.Type)
	assert.Equal(t, characters.Shooting, ranger.Character.Aggro.Type)
	assert.Equal(t, 5, ranger.Character.Aggro.MobInstanceId, "the same foe")
	assert.Empty(t, abilityStrikes)
}
