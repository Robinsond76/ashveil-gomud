package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttackEntryPointsUseEnemyLabels(t *testing.T) {
	edgeSpecs(t)
	battle.Reset()
	t.Cleanup(battle.Reset)

	user := users.NewUserRecord(9921, 9921)
	user.Character = edgeFighter(90231)
	user.Character.Name = "Aria"
	user.Character.Equipment.Weapon = sharpenedItem(edgeSwordID, 1, 10)
	target := &mobs.Mob{InstanceId: 9922, Character: *edgeFighter(90231)}
	target.Character.Name = "bandit cutthroat"
	battle.Begin(user.UserId, 90231, 1, "bandits", []int{target.InstanceId, 9923})
	battle.AssignEnemyNames(user.UserId, []battle.EnemyName{{InstanceId: target.InstanceId, BaseName: target.Character.Name}, {InstanceId: 9923, BaseName: target.Character.Name}})
	beforeName, beforeHealth := target.Character.Name, target.Character.Health
	playerResult := AttackPlayerVsMob(user, target)
	require.NotEmpty(t, playerResult.MessagesToSource)
	assert.True(t, hasMessage(playerResult.MessagesToSource, "first cutthroat"))
	assert.Equal(t, beforeName, target.Character.Name)
	assert.Equal(t, beforeHealth-playerResult.DamageToTarget, target.Character.Health)
	assert.Equal(t, playerResult.DamageToTarget, target.Character.PlayerDamage[user.UserId])
	assert.Equal(t, 10-playerResult.EdgeSpent[items.Weapon], user.Character.Equipment.Weapon.SharpStrikes)

	attacker := &mobs.Mob{InstanceId: 9924, Character: *edgeFighter(90231)}
	attacker.Character.Name = "bandit cutthroat"
	attacker.Character.Equipment.Weapon = sharpenedItem(edgeSwordID, 1, 10)
	battle.Grow(user.UserId, "bandits", []int{attacker.InstanceId})
	battle.AssignEnemyNames(user.UserId, []battle.EnemyName{{InstanceId: attacker.InstanceId, BaseName: attacker.Character.Name}})
	mobResult := AttackMobVsPlayer(attacker, user)
	require.NotEmpty(t, mobResult.MessagesToSourceRoom)
	assert.True(t, hasMessage(mobResult.MessagesToSourceRoom, "third cutthroat"))
	assert.Equal(t, "bandit cutthroat", attacker.Character.Name)
	assert.Equal(t, 10-mobResult.EdgeSpent[items.Weapon], attacker.Character.Equipment.Weapon.SharpStrikes)

	ally := &mobs.Mob{InstanceId: 9925, Character: *edgeFighter(90231)}
	ally.Character.Name = "bandit cutthroat"
	ally.Character.Equipment.Weapon = sharpenedItem(edgeSwordID, 1, 10)
	attacker.Character.Charm(user.UserId, characters.CharmPermanent, characters.CharmExpiredRevert)
	battle.Grow(user.UserId, "bandits", []int{ally.InstanceId})
	battle.AssignEnemyNames(user.UserId, []battle.EnemyName{{InstanceId: ally.InstanceId, BaseName: ally.Character.Name}})
	mobVsMob := AttackMobVsMob(attacker, ally)
	require.NotEmpty(t, mobVsMob.MessagesToSourceRoom)
	assert.True(t, hasMessage(mobVsMob.MessagesToSourceRoom, "third cutthroat"))
	assert.True(t, hasMessage(mobVsMob.MessagesToSourceRoom, "fourth cutthroat"))
	assert.Equal(t, "bandit cutthroat", ally.Character.Name)
	assert.Equal(t, mobVsMob.DamageToTarget, ally.Character.PlayerDamage[user.UserId])
	assert.Equal(t, 10, ally.Character.Equipment.Weapon.SharpStrikes, "defender edge is untouched")

	untracked := &mobs.Mob{InstanceId: 9999, Character: *edgeFighter(90231)}
	untracked.Character.Name = "untracked foe"
	assert.Equal(t, "untracked foe", mobCombatCharacter(untracked).Name)

	pvpTarget := users.NewUserRecord(9926, 9926)
	pvpTarget.Character = edgeFighter(90231)
	pvpTarget.Character.Name = "Brom"
	pvp := AttackPlayerVsPlayer(user, pvpTarget)
	assert.NotEmpty(t, pvp.MessagesToSource)
	assert.True(t, hasMessage(pvp.MessagesToSource, "Brom"))
}
