package combat

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 87: a critical hit is never called glancing in its line, though
// its damage still carries the quality's cut.
func TestACriticalHitIsNeverCalledGlancing(t *testing.T) {
	assert.Equal(t, " (critical hit, 3 damage)", damageSuffixQuality(3, true, 0, QualityGlancing))
	assert.Equal(t, " (glancing, 3 damage)", damageSuffixQuality(3, false, 0, QualityGlancing))
	assert.Equal(t, " (telling, critical hit, 14 damage)", damageSuffixQuality(14, true, 0, QualityTelling))
	assert.Equal(t, " (critical hit, 3 damage, bleeding)", damageSuffixQuality(3, true, 0, QualityGlancing, "bleeding"))
}

func TestWithoutBleeding(t *testing.T) {
	in := []int{status.Bleeding, status.Staggered}
	assert.Equal(t, in, withoutBleeding(in, false))
	assert.Equal(t, []int{status.Staggered}, withoutBleeding(in, true))
	assert.Equal(t, in, []int{status.Bleeding, status.Staggered}, "the caller's slice is untouched")
	assert.Empty(t, withoutBleeding([]int{status.Bleeding}, true))
}

// Phase 87: through a real pass, a skeleton takes critical blows as bone and
// dust: no blood, neck, skin or throat in the line, and no bleeding status.
// A living target, struck the same way, gets the flesh lines and bleeds.
func TestBloodlessFoesNeitherBleedNorGetFleshLines(t *testing.T) {
	edgeSpecs(t)
	cfg := configs.GetGamePlayConfig()
	cfg.Combat.ToHitMin, cfg.Combat.ToHitMax = 100, 100
	cfg.Combat.CritChanceMin, cfg.Combat.CritChanceMax = 100, 100
	cfg.Combat.DodgeChanceMin, cfg.Combat.DodgeChanceMax = 0, 0
	cfg.Combat.ParryChanceMin, cfg.Combat.ParryChanceMax = 0, 0
	cfg.Combat.BlockChanceMin, cfg.Combat.BlockChanceMax = 0, 0
	t.Cleanup(configs.SetTestGamePlayConfig(cfg))
	flesh := []string{"collarbone", "blood", "neck", "skin", "throat", "flesh", "bleeding"}

	run := func(raceId int) (fleshLines, bleeds, hits int) {
		for i := 0; i < 300; i++ {
			src, target := edgeFighter(90231), edgeFighter(90231)
			src.Equipment.Weapon = items.New(edgeSwordID)
			target.RaceId = raceId
			res := calculateCombat(*src, *target, User, Mob, 0, 0)
			if !res.Hit {
				continue
			}
			hits++
			for _, line := range append(append([]string{}, res.MessagesToSource...), res.MessagesToSourceRoom...) {
				for _, w := range flesh {
					if strings.Contains(strings.ToLower(line), w) {
						fleshLines++
						break
					}
				}
			}
			for _, id := range res.BuffTarget {
				if id == status.Bleeding {
					bleeds++
				}
			}
		}
		return
	}

	livingFlesh, livingBleeds, livingHits := run(1)
	require.Greater(t, livingHits, 100)
	require.Greater(t, livingFlesh, 0, "the sanity check: a person gets the flesh lines")
	require.Greater(t, livingBleeds, 0, "and bleeds from a slashing crit")

	deadFlesh, deadBleeds, deadHits := run(6) // undead
	require.Greater(t, deadHits, 100)
	assert.Zero(t, deadFlesh, "no collarbone, blood, neck, skin, throat or bleeding on a skeleton")
	assert.Zero(t, deadBleeds, "a skeleton does not bleed")
}

func TestBloodlessRaces(t *testing.T) {
	edgeSpecs(t)
	c := edgeFighter(90231)
	for raceId, want := range map[int]bool{1: false, 11: false, 6: true, 16: true, 25: true, 0: true} {
		c.RaceId = raceId
		assert.Equal(t, want, IsBloodless(c), "race %d", raceId)
	}
	assert.False(t, IsBloodless(nil))
	_, ok := items.GetBloodlessAttackMessage(50, true)
	assert.True(t, ok, "the shipped world has bloodless hit lines")
}
