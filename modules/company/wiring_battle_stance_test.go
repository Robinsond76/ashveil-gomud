package company

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/stance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 69 wiring: weapon stances through the real `stance` command, the
// combat round (DoCombat's tempo fill) and the company's Combat data, in the
// brawl world.

const (
	stanceGlaiveID = 10151 // militia glaive: two-handed
	stanceTowerID  = 20048 // tower shield
)

// A stance set with the command reaches the fighter's battle state in the
// round, is announced once, and moves its tempo; an enemy never has one.
func TestAStanceIsAppliedInTheRoundAndAnnouncedOnce(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, _, _, _, _ := b.shapeBandits()
	tamsin := b.companion(1)
	tamsin.Character.Equipment.Weapon = items.New(stanceGlaiveID)
	tamsin.Character.Equipment.Offhand = items.Item{}
	b.aria.Character.Equipment.Offhand = items.New(stanceTowerID)

	assert.Contains(t, b.cmd("stance", "tamsin heavy"), "Tamsin Reed is now in the Heavy blows stance")
	assert.Contains(t, b.cmd("stance", "me wall"), "You are now in the Shield wall stance")
	assert.Equal(t, stance.Heavy, stance.For(7, "companion:1"), "the engine reads the store through internal/stance")
	assert.Nil(t, tamsin.Character.RT, "nothing is applied before a battle")
	arias := combat.Tempo(b.aria.Character)

	b.cmd("attack", fmt.Sprintf("#%d", captain))
	b.toughen()
	out := b.fight()
	require.NotNil(t, tamsin.Character.RT)
	assert.Equal(t, stance.Heavy, tamsin.Character.Stance(), "Tamsin fights in the stance")
	assert.Equal(t, stance.Wall, b.aria.Character.Stance())
	assert.Contains(t, out, "Tamsin Reed takes the Heavy blows stance: blows land 30% harder, but 15 points less likely to hit.")
	assert.Contains(t, out, "You take the Shield wall stance")
	assert.InDelta(t, 0.70, combat.Tempo(b.aria.Character)/arias, 0.01, "the wall costs 30% of her turns")

	out = b.fight()
	assert.NotContains(t, out, "takes the Heavy blows stance", "said once")
	for _, id := range b.bandits["bandit captain"] {
		if m := mobs.GetInstance(id); m != nil && m.Character.RT != nil {
			assert.Equal(t, stance.None, m.Character.RT.Stance, "an enemy has no stance")
		}
	}
}

// A stance whose weapon is not in hand is quiet: no line, no change.
func TestAStanceWithoutItsWeaponIsQuietInBattle(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, _, _, _, _ := b.shapeBandits()
	tamsin := b.companion(1)
	tamsin.Character.Equipment.Weapon = items.Item{}
	require.Contains(t, b.cmd("stance", "tamsin quick"), "It does nothing until Tamsin Reed holds a bow.")
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	b.toughen()
	out := b.fight()
	assert.Equal(t, stance.None, tamsin.Character.Stance())
	assert.NotContains(t, out, "Quick draw")
}

// The command refuses to change a stance in a battle; the stance in force stays.
func TestAStanceCannotBeChangedInABattle(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, _, _, _, _ := b.shapeBandits()
	b.companion(1).Character.Equipment.Weapon = items.New(stanceGlaiveID)
	b.cmd("stance", "tamsin heavy")
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	b.toughen()
	b.fight()
	assert.Contains(t, b.cmd("stance", "tamsin off"), "battle")
	assert.Equal(t, stance.Heavy, stance.For(7, "companion:1"))
}

// Phase 82d review: reading a member's tempo for the panels, which happens
// on every Company snapshot, applies the chosen stance to the number but
// never announces it or touches the member's battle state; the battle's
// first round still says it once.
func TestReadingTempoNeverAnnouncesTheStance(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, _, _, _, _ := b.shapeBandits()
	b.aria.Character.Equipment.Offhand = items.New(stanceTowerID)
	t.Cleanup(hooks.UseTempoForTest(nil)) // the real calculation
	before := combat.Tempo(b.aria.Character)
	assert.Contains(t, b.cmd("stance", "me wall"), "You are now in the Shield wall stance")

	heard := b.ariaHears()
	for i := 0; i < 3; i++ {
		got, ok := hooks.MemberTempo(7, company.LeaderMemberKey)
		require.True(t, ok)
		assert.InDelta(t, 0.70, got/before, 0.02, "the number counts the stance she will fight in")
	}
	events.ProcessEvents()
	assert.Empty(t, *heard, "reading the tempo says nothing")
	assert.Nil(t, b.aria.Character.RT, "and applies nothing to her before the battle")

	b.cmd("attack", fmt.Sprintf("#%d", captain))
	b.toughen()
	out := b.fight()
	assert.Equal(t, 1, strings.Count(out, "You take the Shield wall stance"), "the battle's first round says it once")
}
