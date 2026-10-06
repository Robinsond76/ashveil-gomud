package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
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
