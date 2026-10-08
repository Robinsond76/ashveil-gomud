package company

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/strategy"
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
	hardTo(&tamsin.Character, 100)
	tamsin.Character.Health = 10
	tamsin.Character.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "arm", Points: 20}}
	garrick := b.companion(3)
	hardTo(&garrick.Character, 100)
	garrick.Character.Health = 75
	b.cmd("company", "tactics patch 70")
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

	assert.Contains(t, b.cmd("patch", ""), "No one is below the company's patch threshold (70%", "the alias, with nothing to do")
}

func TestCompanyPatchKeepsTheReserve(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 100, 60
	b.cmd("strategy", "oswin reserve 50")
	tamsin := b.companion(1)
	hardTo(&tamsin.Character, 1000)
	tamsin.Character.Health = 10
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
	assert.GreaterOrEqual(t, tamsin.Character.Health, 800, "patched to the default patch threshold of 80%")
}

// Phase 35d: the patch threshold is its own tactic, 80% until set; the
// in-battle healing threshold does not move it.
func TestPatchThresholdIsItsOwnTactic(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 500, 500
	tamsin := b.companion(1)
	hardTo(&tamsin.Character, 100)
	tamsin.Character.Health = 10
	b.cmd("company", "tactics healing 30")
	assert.Equal(t, strategy.DefaultPatch, strategy.TacticsFor(7).Patch, "patch stays at its default")

	out := b.cmd("company", "patch")
	assert.Contains(t, out, "Your company patches itself up.")
	limit := tamsin.Character.HealthLimit()
	assert.GreaterOrEqual(t, tamsin.Character.Health, (limit*80+99)/100, "healed to 80%, not to the healing threshold")

	assert.Contains(t, b.cmd("company", "tactics patch 60"), "patch everyone up to 60% of their wound limit")
	assert.Contains(t, b.cmd("company", "tactics patch 40"), "from 50 to 100")
	assert.Contains(t, b.cmd("company", "tactics patch many"), "from 50 to 100")
	assert.Equal(t, 60, strategy.TacticsFor(7).Patch)
	assert.Contains(t, b.cmd("company", "tactics"), "Patch:   after a battle your healers patch everyone up to 60%")
	assert.Contains(t, b.cmd("company", "tactics default"), "patching up to 80%")
	assert.Equal(t, 80, strategy.TacticsFor(7).Patch)
}

// Review regression: a leader downed in a won battle is patched back to
// their feet when the battle ends; a fallen companion is not (it needs a
// resurrection).
func TestBattleEndPatchesADownedLeader(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 200, 200
	tamsin := b.companion(1)
	tamsin.Character.Health = 0
	b.aria.Character.Health = -3
	events.AddToQueue(events.BattleEnded{UserId: b.aria.UserId, Outcome: "victory"})
	events.ProcessEvents()
	assert.GreaterOrEqual(t, b.aria.Character.Health, 1, "patched back to her feet")
	assert.Less(t, oswin.Character.Mana, 200)
	assert.Equal(t, 0, tamsin.Character.Health, "a fallen companion waits for a resurrection")
}

// Review regression: while a companion still has a foe, the battle-end
// patch waits, as `company patch` does.
func TestBattleEndPatchWaitsWhileTheCompanyFights(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 200, 200
	tamsin := b.companion(1)
	hardTo(&tamsin.Character, 100)
	tamsin.Character.Health = 10
	tamsin.Character.Aggro = &characters.Aggro{MobInstanceId: b.livingBandits()[0].InstanceId}
	events.AddToQueue(events.BattleEnded{UserId: b.aria.UserId, Outcome: "victory"})
	events.ProcessEvents()
	assert.Equal(t, 10, tamsin.Character.Health, "no patch while she still fights")
	assert.Equal(t, 200, oswin.Character.Mana)

	tamsin.Character.Aggro = nil
	events.AddToQueue(events.BattleEnded{UserId: b.aria.UserId, Outcome: "victory"})
	events.ProcessEvents()
	assert.Greater(t, tamsin.Character.Health, 10, "patched once the fighting stops")
}

// Merge review regression (the battle-end patch flake): an archer still
// taking aim at a bandit who fell in the battle's last round holds a stale
// aggro until the next round's combat pass; it does not stop the patch,
// from the battle's end or from `company patch`.
func TestPatchIgnoresAimAtAFallenFoe(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 200, 200
	tamsin := b.companion(1)
	hardTo(&tamsin.Character, 100)
	tamsin.Character.Health = 10
	fallen := b.livingBandits()[0]
	fallen.Character.Health = 0
	ysolde := b.companion(4)
	ysolde.Character.Aggro = &characters.Aggro{MobInstanceId: fallen.InstanceId, RoundsWaiting: 1}
	events.AddToQueue(events.BattleEnded{UserId: b.aria.UserId, Outcome: "victory"})
	events.ProcessEvents()
	assert.GreaterOrEqual(t, tamsin.Character.Health, 80, "patched though Ysolde still aims at the fallen bandit")

	tamsin.Character.Health = 10
	assert.Contains(t, b.cmd("company", "patch"), "Your company patches itself up.")
}

// Phase 35b review: a player's patch heals only their own company; an
// allied player's (33d) hurt companion keeps its health, and its healer
// its mana.
func TestCompanyPatchLeavesAnAllysCompanyAlone(t *testing.T) {
	b, _, allyTamsin := alliedBrawl(t)
	b.withArchetypes("")
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 200, 200
	hardTo(&allyTamsin.Character, 100)
	allyTamsin.Character.Health = 10
	tamsin := b.companion(1)
	hardTo(&tamsin.Character, 100)
	tamsin.Character.Health = 10

	assert.Contains(t, b.cmd("company", "patch"), "Your company patches itself up.")
	assert.Greater(t, tamsin.Character.Health, 10, "her own Tamsin is healed")
	assert.Equal(t, 10, allyTamsin.Character.Health, "the ally's companion is not")
}
