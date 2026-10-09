package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// healcompany through the real command entry point: it heals the leader and
// every companion, raises the dead whole (no level lost), and never moves
// the clock.
func TestHealCompanyRestoresAndRaises(t *testing.T) {
	b := newBrawl(t)
	b.aria.Role = users.RoleAdmin
	tamsin := b.companion(1)
	hardTo(&tamsin.Character, 100)
	tamsin.Character.Health, tamsin.Character.Mana = 3, 0
	tamsin.Character.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "arm", Points: 20}}
	b.aria.Character.Health, b.aria.Character.Mana = 1, 0
	b.aria.Character.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "leg", Points: 10}}

	// Companion #2 falls for good.
	fallenMob := b.companion(2)
	level := fallenMob.Character.Level
	id := fallenMob.InstanceId
	module.onMobDeath(events.MobDeath{InstanceId: id, Level: level})
	record, _ := module.registry.Get(7)
	require.True(t, record.Companions[1].Dead())
	module.runtime.Detach(7, id)
	module.clearInstance(7, 2)

	turn, round := util.GetTurnCount(), util.GetRoundCount()
	out := b.cmd("healcompany", "")
	assert.Contains(t, out, "Your company is whole again: you, ")
	assert.Contains(t, out, " are at full health and mana")
	assert.Contains(t, out, "stands again")
	assert.Equal(t, b.aria.Character.HealthMax.Value, b.aria.Character.Health)
	assert.Equal(t, b.aria.Character.ManaMax.Value, b.aria.Character.Mana)
	assert.Empty(t, b.aria.Character.Wounds)
	assert.Empty(t, tamsin.Character.Wounds)
	assert.Equal(t, tamsin.Character.HealthMax.Value, tamsin.Character.Health)
	assert.Equal(t, tamsin.Character.ManaMax.Value, tamsin.Character.Mana)

	record, _ = module.registry.Get(7)
	assert.False(t, record.Companions[1].Dead(), "raised")
	raised := b.companion(2)
	assert.Equal(t, level, raised.Character.Level, "no level lost")
	assert.Equal(t, raised.Character.HealthMax.Value, raised.Character.Health, "raised at full health")
	assert.Equal(t, turn, util.GetTurnCount(), "never advances the clock")
	assert.Equal(t, round, util.GetRoundCount())
}

// A player without the admin role gets no hint the command exists, and
// nothing changes; and a company in a battle is left alone.
func TestHealCompanyIsAdminOnlyAndNotInBattle(t *testing.T) {
	b := newBrawl(t)
	b.aria.Character.Health = 1
	handled, _ := usercommands.TryCommand("healcompany", "", b.aria.UserId, events.CmdSkipScripts)
	events.ProcessEvents()
	assert.False(t, handled)
	assert.Equal(t, 1, b.aria.Character.Health, "a player is not healed")

	b.aria.Role = users.RoleAdmin
	b.aimAt("bandit")
	assert.Contains(t, b.cmd("healcompany", ""), "in a battle")
	assert.Equal(t, 1, b.aria.Character.Health, "nothing restored mid-battle")
}
