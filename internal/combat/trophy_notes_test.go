package combat

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	trophyNoteHeartID = 99741
	trophyNoteHideID  = 99742
)

// Phase 71: a trophy enchant that raised a landed blow is named on the
// strike's explained line, only when its condition held, and the blow
// really is stronger.
func TestATrophyEnchantRaisesTheBlowAndIsNamedOnTheStrike(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 0, 0, 0)
	items.SetTestItemSpec(&items.ItemSpec{ItemId: trophyNoteHeartID, Name: "brute's heart", Type: items.Commodity,
		Trophy: &items.TrophySpec{Part: items.TrophyHeart, Races: []string{"ogre"}, Chance: 20, Effects: map[string]int{classes.Damage: 1}}})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: trophyNoteHideID, Name: "beast hide", Type: items.Commodity,
		Trophy: &items.TrophySpec{Part: items.TrophyHide, Races: []string{"canine"}, Chance: 20, Effects: map[string]int{classes.Armor: 2}}})
	t.Cleanup(func() { items.RemoveTestItemSpec(trophyNoteHeartID); items.RemoveTestItemSpec(trophyNoteHideID) })

	hit := func(trophy int) (AttackResult, int) {
		bearer := armed(edgeSwordID)
		if trophy != 0 {
			require.NoError(t, bearer.Equipment.Weapon.EnchantWithTrophy(trophy))
		}
		foe := armed(0)
		foe.HealthMax.Value, foe.Health = 100, 100
		r := strikeAt(bearer, foe)
		require.Len(t, r.Strikes, 1)
		require.True(t, r.Strikes[0].Hit)
		return r, r.Strikes[0].Raw
	}
	noteOf := func(r AttackResult) string { return strings.Join(r.Strikes[0].Notes, "|") }

	r, plain := hit(0)
	assert.NotContains(t, noteOf(r), "Enchanted")

	r, withHeart := hit(trophyNoteHeartID)
	assert.Equal(t, plain+1, withHeart, "the enchant really raised the blow")
	assert.Contains(t, noteOf(r), "Enchanted with brute's heart: +1 damage on every landed blow")
	assert.Contains(t, strings.Join(r.Strikes[0].Explain(), "\n"), "Enchanted with brute's heart")

	// A defence enchant acts on the numbers it changes, not on the blow's line.
	r, withHide := hit(trophyNoteHideID)
	assert.Equal(t, plain, withHide)
	assert.NotContains(t, noteOf(r), "Enchanted")
}
