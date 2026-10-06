package company

import (
	"fmt"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 35d wiring: the company grows smarter with its leader's level. A
// default focus and a default guard apply until the player sets their own.

// aimsAtLevel has Tamsin (set to go for the strongest foe) aim in a fresh
// battle against the shaped bandits, with the leader at level.
func aimsAtLevel(t *testing.T, level int, setup func(b *brawl)) (aim, bruiser, weakest int) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	_, bruiser, slinger, cutA, _ := b.shapeBandits()
	b.cmd("strategy", "tamsin strongest")
	b.aria.Character.Level = level
	if setup != nil {
		setup(b)
	}
	b.cmd("attack", fmt.Sprintf("#%d", slinger))
	return aimOf(&b.companion(1).Character), bruiser, cutA
}

// The company's default focus follows the leader's level, through a real
// attack: members aim by their own strategies below level 10, by the weakest
// foe from 10, and a set focus (none included) always wins.
func TestCompanyDefaultsGrowWithTheLeadersLevel(t *testing.T) {
	aim, bruiser, _ := aimsAtLevel(t, 5, nil)
	assert.Equal(t, bruiser, aim, "level 5: no default focus, Tamsin goes for her strongest")

	aim, _, weakest := aimsAtLevel(t, 10, nil)
	assert.Equal(t, weakest, aim, "level 10: Tamsin follows the default focus, the weakest foe")

	aim, bruiser, _ = aimsAtLevel(t, 10, func(b *brawl) {
		require.Contains(t, b.cmd("company", "tactics focus none"), "focus is now none")
	})
	assert.Equal(t, bruiser, aim, "a focus of none keeps Tamsin's own rule at level 10")

	aim, bruiser, _ = aimsAtLevel(t, 10, func(b *brawl) { b.saveTactics(strategy.Tactics{Focus: strategy.Strongest}) })
	assert.Equal(t, bruiser, aim, "a set focus wins over the default")
}

// The level ladder's rungs and the tactics command that shows and clears them.
func TestCompanyLadderRungsAndCommand(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	for level, want := range map[int]strategy.Rule{1: strategy.NoFocus, 10: strategy.Weakest, 24: strategy.Weakest, 25: strategy.Casters} {
		b.aria.Character.Level = level
		focus, _ := strategy.FocusFor(7, level)
		assert.Equal(t, want, focus, "level %d", level)
		got, ok := enemyparty.Focus(7)
		assert.Equal(t, want != strategy.NoFocus, ok, "level %d", level)
		if ok {
			assert.Equal(t, want, got, "level %d", level)
		}
	}

	b.aria.Character.Level = 12
	out := b.cmd("company", "tactics")
	assert.Contains(t, out, "Focus:   weakest")
	assert.Contains(t, out, "[default at your level]")

	b.saveTactics(strategy.Tactics{Focus: strategy.Strongest})
	require.Contains(t, b.cmd("company", "tactics focus default"), "back to the default for your level: weakest")
	focus, _ := enemyparty.Focus(7)
	assert.Equal(t, strategy.Weakest, focus)
	b.aria.Character.Level = 30
	require.Contains(t, b.cmd("company", "tactics default"), "focus casters")
}

// The first companion warrior with no role or ward of its own guards the
// company's healer; a set role or ward is never overridden.
func TestWarriorGuardsTheHealerByDefault(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	tamsin, oswin, garrick := domain.CompanionMemberKey(1), domain.CompanionMemberKey(2), domain.CompanionMemberKey(3)

	s := enemyparty.MemberStrategy(7, tamsin)
	assert.Equal(t, strategy.Guardian, s.Role, "Tamsin, the first warrior, guards")
	assert.Equal(t, string(oswin), s.Ward, "the healer")
	assert.Equal(t, strategy.Fighter, enemyparty.MemberStrategy(7, garrick).Role, "only the first warrior")
	assert.Equal(t, strategy.Healer, enemyparty.MemberStrategy(7, oswin).Role)
	assert.Equal(t, strategy.Fighter, enemyparty.MemberStrategy(7, domain.LeaderMemberKey).Role)
	assert.Contains(t, b.cmd("company", "tactics"), "[default: guards Brother Oswin]")

	// A role the player set is theirs: Tamsin as a fighter, and no other
	// warrior takes her place.
	b.cmd("strategy", "tamsin fighter")
	assert.Equal(t, strategy.Fighter, enemyparty.MemberStrategy(7, tamsin).Role)
	assert.Equal(t, strategy.Fighter, enemyparty.MemberStrategy(7, garrick).Role, "Garrick stays a fighter")
	b.cmd("strategy", "tamsin default")
	assert.Equal(t, strategy.Guardian, enemyparty.MemberStrategy(7, tamsin).Role, "back to the default guard")

	// A ward the player set stays.
	b.cmd("strategy", "garrick guard ysolde")
	g := enemyparty.MemberStrategy(7, garrick)
	assert.Equal(t, strategy.Guardian, g.Role)
	assert.Equal(t, string(domain.CompanionMemberKey(4)), g.Ward)
}

// With no healer in the company no one guards by default.
func TestNoDefaultGuardWithoutAHealer(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypesFor("", map[int]string{1: "warrior", 2: "ranger", 3: "warrior", 4: "ranger"})
	assert.Equal(t, strategy.Fighter, enemyparty.MemberStrategy(7, domain.CompanionMemberKey(1)).Role)
}

// Phase 35e wiring: from level 5 the company's default goes for an enemy
// healer first, through a real attack and in its battle text; a set focus
// (none included) wins, and below level 5 nothing changes.
func healerAims(t *testing.T, level int, setup func(b *brawl)) (aim, healer, bruiser int, out string) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	_, bruiser, slinger, _, _ := b.shapeBandits()
	// The slinger is the group's healer.
	mobs.GetInstance(slinger).Role = "healer"
	b.cmd("strategy", "tamsin strongest")
	b.aria.Character.Level = level
	if setup != nil {
		setup(b)
	}
	b.cmd("attack", fmt.Sprintf("#%d", bruiser))
	return aimOf(&b.companion(1).Character), slinger, bruiser, b.cmd("company", "tactics")
}

func TestCompanyDefaultGoesForTheEnemyHealer(t *testing.T) {
	aim, healer, bruiser, out := healerAims(t, 5, nil)
	assert.Equal(t, healer, aim, "level 5: Tamsin, set to the strongest, goes for the healer by default")
	assert.Contains(t, out, "whenever the enemy has a healer within reach")

	aim, healer, _, _ = healerAims(t, 12, nil)
	assert.Equal(t, healer, aim, "level 12: the healers default beats the weakest default")

	aim, _, bruiser, _ = healerAims(t, 4, nil)
	assert.Equal(t, bruiser, aim, "level 4: no healers default")

	aim, _, bruiser, _ = healerAims(t, 12, func(b *brawl) {
		require.Contains(t, b.cmd("company", "tactics focus none"), "focus is now none")
	})
	assert.Equal(t, bruiser, aim, "a focus of none wins over the healers default")

	aim, _, bruiser, out = healerAims(t, 12, func(b *brawl) { b.saveTactics(strategy.Tactics{Focus: strategy.Strongest}) })
	assert.Equal(t, bruiser, aim, "a set focus wins over the healers default")
	assert.NotContains(t, out, "whenever the enemy has a healer within reach")

	aim, healer, _, _ = healerAims(t, 3, func(b *brawl) { b.saveTactics(strategy.Tactics{Focus: strategy.Healers}) })
	assert.Equal(t, healer, aim, "an explicit healers focus works at any level")
}

func TestCompanyHealersDefaultWithoutAHealer(t *testing.T) {
	aim, bruiser, _ := aimsAtLevel(t, 5, nil)
	assert.Equal(t, bruiser, aim, "no enemy healer: level 5 keeps Tamsin's own rule")
}

func TestHealersDefaultInTheBattleSurfaces(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.aria.Character.Level = 6
	assert.True(t, enemyparty.HealersDefault(7))
	assert.Contains(t, b.cmd("company", "tactics"), "[default: whenever the enemy has a healer within reach, everyone goes for it first]")
	assert.Contains(t, b.cmd("company", "tactics focus default"), "Whenever the enemy has a healer within reach")
	b.cmd("company", "tactics focus healers")
	assert.False(t, enemyparty.HealersDefault(7), "a set focus ends the default")
	assert.Contains(t, b.cmd("company", "tactics"), "Focus:   healers (everyone goes for their healers first")
}

// The leader who rejoins a battle by the healers default is told why.
func TestHealersDefaultSaysWhyTheLeaderTurns(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	_, bruiser, slinger, _, _ := b.shapeBandits()
	mobs.GetInstance(slinger).Role = "healer"
	b.aria.Character.Level = 6
	b.cmd("attack", fmt.Sprintf("#%d", bruiser))
	b.toughen()
	b.aria.Character.EndAggro()
	got := b.fight()
	assert.Contains(t, got, "Your company marks the bandit slinger as a healer and goes for it first.")
	assert.NotContains(t, got, "You turn toward")
}
