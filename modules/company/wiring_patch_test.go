package company

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 35b wiring: after a battle, and on `company patch`, the company's
// healers heal everyone to the healing threshold with Minor Heal, each
// stopping at its mana reserve.

func TestCompanyPatchHealsToTheThreshold(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 200, 200
	tamsin := b.companion(1)
	tamsin.Character.HealthMax.Value, tamsin.Character.Health = 100, 10
	tamsin.Character.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "arm", Points: 20}}
	garrick := b.companion(3)
	garrick.Character.HealthMax.Value, garrick.Character.Health = 100, 75
	b.cmd("company", "tactics healing 70")
	turn, round := util.GetTurnCount(), util.GetRoundCount()

	out := b.cmd("company", "patch")
	assert.Contains(t, out, "Your company patches itself up.")
	assert.Contains(t, out, "Brother Oswin lays glowing hands on Tamsin Reed.")
	limit := tamsin.Character.HealthLimit()
	assert.GreaterOrEqual(t, tamsin.Character.Health, (limit*70+99)/100, "healed to 70% of her wound limit")
	assert.Less(t, tamsin.Character.Health, (limit*70+99)/100+25, "and no further than one heal past it")
	assert.Len(t, tamsin.Character.Wounds, 1, "patching tends no wound")
	assert.Equal(t, 75, garrick.Character.Health, "above the threshold, left alone")
	assert.Less(t, oswin.Character.Mana, 200)
	assert.Equal(t, turn, util.GetTurnCount(), "never advances the clock")
	assert.Equal(t, round, util.GetRoundCount())

	assert.Contains(t, b.cmd("patch", ""), "No one is below the company's healing threshold (70%", "the alias, with nothing to do")
}

func TestCompanyPatchKeepsTheReserve(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 100, 60
	b.cmd("strategy", "oswin reserve 50")
	tamsin := b.companion(1)
	tamsin.Character.HealthMax.Value, tamsin.Character.Health = 1000, 10
	b.cmd("company", "patch")
	assert.GreaterOrEqual(t, oswin.Character.Mana, 50, "he keeps half his mana")
	assert.Less(t, oswin.Character.Mana, 53, "and spends down to it")
	assert.Contains(t, b.cmd("company", "patch"), "Your healers are down to their mana reserves.")
}

func TestCompanyPatchIsRefusedInABattle(t *testing.T) {
	b := guardBrawl(t)
	_, inBattle := battle.Current(7)
	require.True(t, inBattle)
	tamsin := b.companion(1)
	tamsin.Character.Health = 1
	assert.Contains(t, b.cmd("company", "patch"), "Not in the middle of a battle.")
	assert.Equal(t, 1, tamsin.Character.Health)
}

// The real battle end patches the company: the battle pass ends the
// battle, queues BattleEnded, and the company module heals.
func TestBattleEndPatchesTheCompany(t *testing.T) {
	b := guardBrawl(t)
	forceBlows(t, true)
	noCounters(t)
	oswin := b.companion(2)
	tamsin := b.companion(1)
	var out []string
	for i := 0; i < 200; i++ {
		_, inBattle := battle.Current(7)
		if !inBattle {
			break
		}
		b.toughen()
		oswin.Character.ManaMax.Value, oswin.Character.Mana = 1000, 1000
		tamsin.Character.Health = 300 // below half of 1000
		for _, m := range b.livingBandits() {
			m.Character.Health = min(m.Character.Health, 1)
		}
		out = append(out, b.fight())
	}
	_, inBattle := battle.Current(7)
	require.False(t, inBattle, "the battle ended")
	events.ProcessEvents()
	text := strings.Join(out, "\n")
	assert.Contains(t, text, "Your company patches itself up.")
	assert.GreaterOrEqual(t, tamsin.Character.Health, 500, "healed to the default threshold of half")
}
