package summons

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/stretchr/testify/assert"
)

func TestInfoGrowsWithTheRoute(t *testing.T) {
	hierarch := classes.EffectsFor("hierarch", 60, nil)
	info, ok := Info(Angel, hierarch, false)
	assert.True(t, ok)
	assert.Equal(t, 100, info.HPPct)
	assert.Equal(t, 10, info.Sides, "Sword of the Host")
	assert.Equal(t, 50, info.Smite)
	assert.Equal(t, 3, info.Guards)
	assert.Equal(t, 2, info.MercyEvery)
	assert.True(t, info.MercyFull)
	assert.True(t, info.MercyTwo)
	assert.True(t, info.Cleanse)

	early, _ := Info(Angel, classes.EffectsFor("hierarch", 30, nil), true)
	assert.Equal(t, 125, early.HPPct, "a holy symbol adds a quarter")
	assert.Equal(t, 8, early.Sides)
	assert.Equal(t, 1, early.Guards)
	assert.Equal(t, 3, early.MercyEvery)

	demon, _ := Info(Demon, classes.EffectsFor("demonologist", 30, nil), false)
	assert.Equal(t, 80, demon.HPPct, "a Demon has 80% of a warrior's health")
	assert.Equal(t, 6, demon.Sides)
	assert.Equal(t, 1, demon.Dread)
	assert.False(t, demon.Mastered)

	_, ok = Info("imp", hierarch, false)
	assert.False(t, ok)
}
