package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
)

type eliteStore struct{ state classes.State }

func (e eliteStore) PlayerClass(int) classes.State { return e.state }

// Phase 38c1: an elite rank applies from its level for a companion (class
// on the mob) and for a player (class in the registry), is lost with a level
// to death and returns when it is regained; a talent the level has not
// earned is off.
func TestEliteRanksFollowTheLevelForPlayersAndCompanions(t *testing.T) {
	companion := &Character{}
	companion.SetClassState("warlord", nil)
	player := &Character{}
	player.SetUserId(9)
	classes.SetProvider(eliteStore{classes.State{Class: "warlord"}})
	t.Cleanup(func() { classes.SetProvider(nil) })

	for _, who := range []struct {
		name string
		c    *Character
	}{{"companion", companion}, {"player", player}} {
		for _, tc := range []struct {
			level int
			key   string
			want  int
		}{{29, classes.MarkRuin, 0}, {30, classes.MarkRuin, 5}, {34, classes.BattleCry, 0}, {35, classes.BattleCry, 3}, {49, classes.MarkRuin, 5}, {50, classes.MarkRuin, 10}, {59, classes.WarCommand, 0}, {60, classes.WarCommand, 25}} {
			who.c.Level = tc.level
			assert.Equal(t, tc.want, who.c.ClassEffects().Int(tc.key), "%s %s at level %d", who.name, tc.key, tc.level)
		}
		who.c.Level = 60
		assert.Equal(t, 25, who.c.ClassEffects().Int(classes.WarCommand))
		who.c.Level = 55 // a level lost to death
		assert.Zero(t, who.c.ClassEffects().Int(classes.WarCommand), who.name+": the capstone is off below its level")
		who.c.Level = 60 // regained
		assert.Equal(t, 25, who.c.ClassEffects().Int(classes.WarCommand), who.name+": and back")
	}

	// An advanced Mercenary at 45 has none of the elite ranks.
	companion.SetClassState("mercenary", nil)
	companion.Level = 45
	assert.Zero(t, companion.ClassEffects().Int(classes.MarkRuin))
	assert.Equal(t, 2, companion.ClassEffects().Int(classes.Attack), "its advanced ranks stay")
}

// The elite talent Iron Hide adds the armor GetDefense reads (it adds the
// Armor effect to worn armor).
func TestIronHideAddsArmor(t *testing.T) {
	c := &Character{}
	c.SetClassState("warlord", []string{"toughness", "toughness", "keen-edge", "iron-hide"})
	c.Level = 35
	assert.Equal(t, 5, c.ClassEffects().Int(classes.Armor))
	c.Level = 34 // the fourth talent is not earned yet
	assert.Zero(t, c.ClassEffects().Int(classes.Armor))
}

// Phase 38c3: the elite wards. A ward absorbs up to its cap from each of its
// blows and breaks when its blows run out; an Archon's Reflection returns
// half of what the ward took, once a battle; a Ward of Life and a Lich's
// Bargain each leave a felled character at 1 health, once.
func TestWardBreaksAfterItsBlowsAndReflectsOnce(t *testing.T) {
	archon := &Character{}
	by := archon.RTState()
	holder := &Character{}
	holder.RTState().Ward, holder.RT.WardCap, holder.RT.WardReflectBy = 2, 10, by

	left, absorbed := holder.AbsorbWard(25)
	assert.Equal(t, []int{15, 10}, []int{left, absorbed})
	broke, back := holder.WardAbsorbed(absorbed)
	assert.False(t, broke, "one blow left")
	assert.Equal(t, 5, back, "half of the 10 taken")
	assert.True(t, by.ReflectUsed)

	left, absorbed = holder.AbsorbWard(4)
	assert.Equal(t, []int{0, 4}, []int{left, absorbed})
	broke, back = holder.WardAbsorbed(absorbed)
	assert.True(t, broke, "the second blow breaks it")
	assert.Zero(t, back, "Reflection is once a battle")

	broke, back = holder.WardAbsorbed(0)
	assert.False(t, broke)
	assert.Zero(t, back)
}

func TestWardOfLifeAndBargainLeaveOneHealthOnce(t *testing.T) {
	// A Wise One's Ward of Life: only a warded ally, only once.
	wise := &Character{}
	by := wise.RTState()
	ally := &Character{}
	ally.RTState().WardLifeBy = by
	dmg, saved := ally.GuardFall(50, 10, false)
	assert.Equal(t, []any{50, ""}, []any{dmg, saved}, "no ward on it when the blow came")
	dmg, saved = ally.GuardFall(9, 10, true)
	assert.Equal(t, []any{9, ""}, []any{dmg, saved}, "a blow that doesn't fell is left alone")
	dmg, saved = ally.GuardFall(50, 10, true)
	assert.Equal(t, []any{9, "ward of life"}, []any{dmg, saved}, "10 health left: 9 damage leaves 1")
	assert.True(t, by.LifeUsed)
	dmg, saved = ally.GuardFall(50, 10, true)
	assert.Equal(t, []any{50, ""}, []any{dmg, saved}, "once a battle")

	// A Necromancer's Lich's Bargain, warded or not.
	necro := &Character{}
	necro.SetClassState("necromancer", nil)
	necro.Level = 60
	dmg, saved = necro.GuardFall(50, 10, false)
	assert.Equal(t, []any{9, "bargain"}, []any{dmg, saved})
	dmg, saved = necro.GuardFall(50, 10, false)
	assert.Equal(t, []any{50, ""}, []any{dmg, saved}, "once a battle")
	necro.Level = 55
	necro.RT.BargainUsed = false
	dmg, saved = necro.GuardFall(50, 10, false)
	assert.Equal(t, []any{50, ""}, []any{dmg, saved}, "the capstone is off below its level")
}

// A raise costs a tenth of the caster's mana, and a hex costs less with
// Cheaper hexes.
func TestEliteSpellCosts(t *testing.T) {
	necro := &Character{}
	necro.SetClassState("necromancer", nil)
	necro.Level = 30
	necro.ManaMax.Value = 200
	assert.Equal(t, 20, necro.SpellCost(&spells.SpellData{SpellId: "raisefallen", Cost: 10}))

	mother := &Character{}
	mother.SetClassState("coven-mother", nil)
	mother.Level = 39
	assert.Equal(t, 9, mother.SpellCost(&spells.SpellData{SpellId: "slumber", Cost: 10}), "the Coven Sage's 10%")
	mother.Level = 40
	assert.Equal(t, 7, mother.SpellCost(&spells.SpellData{SpellId: "slumber", Cost: 10}), "Cheaper hexes: 30% in all")
	assert.Equal(t, 9, mother.SpellCost(&spells.SpellData{SpellId: "mm", Cost: 10}), "only hexes get the 20%")
}
