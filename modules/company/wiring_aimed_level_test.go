package company

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// aimedShotDamage runs rounds until Ysolde, a ranger at level, takes an
// aimed shot, and returns its damage.
func aimedShotDamage(t *testing.T, level int) int {
	t.Helper()
	b, stream := abilityBrawl(t, map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"})
	b.cmd("strategy", "tamsin abilities off")
	b.cmd("strategy", "garrick abilities off")
	alwaysLand(t)
	b.companion(4).Character.Level = level
	b.start()
	for i := 0; i < 4; i++ {
		b.toughen()
		b.hardenBandits()
		n := len(*stream)
		b.fight()
		round := since(*stream, n)
		if len(abilityEvents(round, "Ysolde")) > 0 {
			shots := swingsBy(round, "Ysolde")
			require.NotEmpty(t, shots)
			return shots[0].Damage
		}
	}
	t.Fatal("no aimed shot")
	return 0
}

// Phase 35b: an Aimed Shot adds 2 + level/3 damage to its blow.
func TestAimedShotGrowsWithLevel(t *testing.T) {
	// Each brawl in its own subtest, so the first one's listeners are gone
	// before the second begins.
	shot := func(level int) (damage int) {
		t.Run(fmt.Sprint(level), func(t *testing.T) { damage = aimedShotDamage(t, level) })
		return damage
	}
	low, high := shot(1), shot(60)
	t.Logf("level 1: %d, level 60: %d", low, high)
	// The bonus joins the blow before armor, so the foe's armor trims a
	// little of the difference.
	assert.GreaterOrEqual(t, high-low, (strategy.AimedShotBonus(60)-strategy.AimedShotBonus(1))*3/4)
}
