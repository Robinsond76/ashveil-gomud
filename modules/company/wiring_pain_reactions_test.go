package company

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPainReactionsThroughCombatRound drives the real strategy, attack,
// DoCombat, and message dispatch path. The player is both a witness to enemy
// pain and the victim of enemy critical strikes.
func TestPainReactionsThroughCombatRound(t *testing.T) {
	b := newBrawl(t)
	b.cmd("company", "dismiss all")
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.CritChanceMin, gameplay.Combat.CritChanceMax = 100, 100
	gameplay.Combat.ToHitMin, gameplay.Combat.ToHitMax = 100, 100
	gameplay.Combat.DodgeChanceMin, gameplay.Combat.DodgeChanceMax = 0, 0
	gameplay.Combat.ParryChanceMin, gameplay.Combat.ParryChanceMax = 0, 0
	gameplay.Combat.BlockChanceMin, gameplay.Combat.BlockChanceMax = 0, 0
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	b.aria.Character.Health, b.aria.Character.HealthMax.Value = 1000, 1000
	for _, ids := range b.bandits {
		for _, id := range ids {
			mob := mobs.GetInstance(id)
			mob.Character.Health, mob.Character.HealthMax.Value = 1000, 1000
		}
	}
	require.Contains(t, b.cmd("attack", "bandit cutthroats"), "You go for")
	// A backstab is a real critical strike on the first landed blow.
	b.aria.Character.SetAggro(0, b.bandits["bandit cutthroat"][0], characters.BackStab)
	transcript := b.fight()
	for i := 0; i < 4 && !strings.Contains(transcript, "Pain flashes through you"); i++ {
		transcript += "\n" + b.fight()
	}
	assert.Contains(t, transcript, "Pain flashes through you", "player victim hears the second-person reaction")
	assert.Contains(t, transcript, "staggers and draws a ragged breath", "player sees an NPC's third-person reaction")
	var lines []string
	for _, line := range strings.Split(transcript, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	for i, line := range lines {
		if strings.Contains(line, "staggers and draws a ragged breath") {
			require.Positive(t, i)
			assert.Contains(t, lines[i-1], "critical hit", "the witness line immediately follows the critical hit")
		}
	}
}

func TestLiveMobPainOverrideWinsOverRace(t *testing.T) {
	b := newBrawl(t)
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.CritChanceMin, gameplay.Combat.CritChanceMax = 100, 100
	gameplay.Combat.ToHitMin, gameplay.Combat.ToHitMax = 100, 100
	gameplay.Combat.DodgeChanceMin, gameplay.Combat.DodgeChanceMax = 0, 0
	gameplay.Combat.ParryChanceMin, gameplay.Combat.ParryChanceMax = 0, 0
	gameplay.Combat.BlockChanceMin, gameplay.Combat.BlockChanceMax = 0, 0
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	victim := mobs.GetInstance(b.bandits["bandit cutthroat"][0])
	victim.Character.Health = 1000
	victim.PainReactions = []races.PainReaction{{ToVictim: "You clutch the torn cloak.", ToRoom: "{name} clutches the torn cloak."}}
	b.aria.Character.SetAggro(0, victim.InstanceId, characters.DefaultAttack)
	result := combat.AttackPlayerVsMob(b.aria, victim)
	require.Positive(t, result.DamageToTarget)
	require.Len(t, result.MessagesToSource, 2)
	assert.Contains(t, result.MessagesToSource[1], "clutches the torn cloak")
	assert.Equal(t, "You clutch the torn cloak.", result.MessagesToTarget[1])
}
