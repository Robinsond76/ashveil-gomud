package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// Phase 87: an Aimed Shot is not lost with its target. A kill earlier in
// the round turns the ranger onto another foe, and the shot goes with it.
func TestRetargetKeepsAReadiedStrike(t *testing.T) {
	c := &characters.Character{}
	c.SetAggro(0, 5, characters.BackStab, 0)
	c.Aggro.StrikeBonus = 6

	retargetKeepingStrike(c, 0, 9)
	assert.Equal(t, 9, c.Aggro.MobInstanceId)
	assert.Equal(t, characters.BackStab, c.Aggro.Type, "the aimed shot is still readied")
	assert.Equal(t, 6, c.Aggro.StrikeBonus, "and keeps its bonus")

	plain := &characters.Character{}
	plain.SetAggro(0, 5, characters.DefaultAttack, 0)
	retargetKeepingStrike(plain, 0, 9)
	assert.Equal(t, characters.DefaultAttack, plain.Aggro.Type)
	assert.Equal(t, 9, plain.Aggro.MobInstanceId)
	assert.Zero(t, plain.Aggro.StrikeBonus)
}

// Phase 87: an Aimed Shot that is never loosed (no foe left in reach, or the
// turn was lost) costs no wait, and the fighter is back to a plain attack.
func TestUnloosedAimedShotCostsNoWait(t *testing.T) {
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	t.Cleanup(ResetAbilitiesForTest)
	ranger := users.NewUserRecord(8811, 1)
	users.SetTestUser(ranger)
	ranger.Character.Health = 20
	ranger.Character.SetAggro(0, 5, characters.BackStab, 0)
	ranger.Character.Aggro.StrikeBonus = 4

	who := caster{userId: 8811}
	key := abilityKey{who: who, id: strategy.AimedShot}
	abilityReady[key] = abilityRounds + 3
	abilityStrikes[who] = true
	abilityKind[who] = strategy.AimedShot

	endAbilityStrike(who)
	_, waiting := abilityReady[key]
	assert.False(t, waiting, "the shot never flew, so the wait is refunded")
	assert.Equal(t, characters.DefaultAttack, ranger.Character.Aggro.Type)
	assert.Zero(t, ranger.Character.Aggro.StrikeBonus)

	// A shot that was loosed (the swing consumed the BackStab) keeps its wait.
	ranger.Character.SetAggro(0, 5, characters.DefaultAttack, 0)
	abilityReady[key] = abilityRounds + 3
	abilityStrikes[who] = true
	abilityKind[who] = strategy.AimedShot
	endAbilityStrike(who)
	_, waiting = abilityReady[key]
	assert.True(t, waiting, "a loosed shot still waits")
}
