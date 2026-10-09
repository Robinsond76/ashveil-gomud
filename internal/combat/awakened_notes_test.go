package combat

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const awakenedBladeID = 99731

func awakenedBlade(t *testing.T) {
	t.Helper()
	items.SetTestItemSpec(&items.ItemSpec{ItemId: awakenedBladeID, Name: "Test Waker", Type: items.Weapon, Subtype: items.Slashing, Hands: 1, Tier: 6,
		Damage: items.Damage{DiceRoll: "1d1", Attacks: 1, DiceCount: 1, SideCount: 1},
		Relic: &items.RelicSpec{Signature: "Waking", Effects: map[string]int{classes.Wounded: 10}, ILvl: 30, Mob: 1, Chance: 5,
			Awakenings: []items.AwakeningSpec{
				{Name: "Flat Edge", Kind: items.AwakenSlay, Races: []string{"ogre"}, Count: 1, Target: "ogres", Effects: map[string]int{classes.Damage: 1}},
				{Name: "Finishing Edge", Kind: items.AwakenLair, Mob: 1, Target: "x", Effects: map[string]int{classes.Wounded: 5}},
				{Name: "Quiet Hide", Kind: items.AwakenPlace, Zone: "Z", Target: "Z", Effects: map[string]int{classes.Armor: 2}},
			}}})
	t.Cleanup(func() { items.RemoveTestItemSpec(awakenedBladeID) })
}

func TestAwakenedPowersRaiseBlowsAndAreNamedOnTheStrike(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 0, 0, 0)
	awakenedBlade(t)

	hit := func(foeHealth int, wake ...int) (AttackResult, int) {
		bearer := armed(awakenedBladeID)
		for _, idx := range wake {
			bearer.Equipment.Weapon.AdvanceAwakening(idx, 5)
		}
		foe := armed(0)
		foe.HealthMax.Value, foe.Health = 100, foeHealth
		r := strikeAt(bearer, foe)
		require.Len(t, r.Strikes, 1)
		require.True(t, r.Strikes[0].Hit)
		return r, r.Strikes[0].Raw
	}
	noteOf := func(r AttackResult) string { return strings.Join(r.Strikes[0].Notes, "|") }

	r, plain := hit(100)
	assert.NotContains(t, noteOf(r), "woke", "a sleeping relic names nothing")

	// +1 damage on every landed blow, said on the line and in the number.
	r, withDamage := hit(100, 0)
	assert.Equal(t, plain+1, withDamage, "the awakened power really raised the blow")
	assert.Contains(t, noteOf(r), "Test Waker woke Flat Edge: +1 damage on every landed blow")
	assert.Contains(t, strings.Join(r.Strikes[0].Explain(), "\n"), "woke Flat Edge")

	// The wounded-foe bonus is named only when the foe is wounded.
	r, _ = hit(100, 1)
	assert.NotContains(t, noteOf(r), "Finishing Edge", "a healthy foe: the power did not act")
	r, wounded := hit(40, 1)
	assert.Contains(t, noteOf(r), "woke Finishing Edge: blows deal 5% more to a foe at or below half health")
	_, woundedPlain := hit(40)
	assert.GreaterOrEqual(t, wounded, woundedPlain, "never less")

	// An awakened power with no blow to name stays off the line.
	r, _ = hit(100, 2)
	assert.NotContains(t, noteOf(r), "Quiet Hide")
}
