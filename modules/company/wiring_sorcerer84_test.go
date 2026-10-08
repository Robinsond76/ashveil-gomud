package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 84 wiring: through the real strategy pass, a Sorcerer showers a
// crowd with sparks and looses its Lance once few enough foes stand.
// thinBandits removes bandits (never the captain the fight starts on) until
// foes remain: a Sorcerer looses its Lance only at a few foes.
func thinBandits(b *brawl, foes int) {
	b.t.Helper()
	for name, ids := range b.bandits {
		if name == "bandit captain" {
			continue
		}
		for _, id := range ids {
			if len(b.livingBandits()) <= foes {
				break
			}
			b.road.RemoveMob(id)
			mobs.DestroyInstance(id)
		}
	}
	require.Len(b.t, b.livingBandits(), foes)
}

func firstSorcererCast(t *testing.T, class string, level, foes int) string {
	t.Helper()
	b := eliteCaster(t, class, level, "arcanelance", "sparks", "mm")
	thinBandits(b, foes)
	b.startWitchFight()
	agg := b.aria.Character.Aggro
	require.NotNil(t, agg)
	require.Equal(t, characters.SpellCast, agg.Type)
	return agg.SpellInfo.SpellId
}

func TestSorcererSparksACrowdAndLancesAFew(t *testing.T) {
	t.Run("crowd", func(t *testing.T) {
		assert.Equal(t, "sparks", firstSorcererCast(t, "sorcerer", 10, 4), "a crowd gets the Shower of Sparks")
	})
	t.Run("few", func(t *testing.T) {
		assert.Equal(t, "arcanelance", firstSorcererCast(t, "sorcerer", 10, 3), "a few get the Lance")
	})
	t.Run("high sorcerer crowd", func(t *testing.T) {
		assert.Equal(t, "arcanelance", firstSorcererCast(t, "high-sorcerer", 30, 5), "a High Sorcerer keeps the Lance against a crowd")
	})
}
