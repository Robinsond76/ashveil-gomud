package death

import (
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/death"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 77: an Iron character's defeat goes through the real `suicide`
// command and always takes the church route, costing two levels.
func TestIronDefeatAlwaysCostsTwoLevelsAtTheChurch(t *testing.T) {
	env := newDefeatEnv(t, 86)
	forceScenario(t, "wayfarer-rescue") // a rescue is rolled; Iron never claims it
	env.foe(86)
	c := env.user.Character
	c.SetIron(true)
	assert.Equal(t, 5, c.Level)

	out := env.run("suicide", "")
	assert.NotContains(t, out, "carried you to", "no rescue for Iron")
	assert.Contains(t, out, "you lose 2 levels (now level 3)")
	assert.Equal(t, 3, c.Level)
	assert.Equal(t, 5, c.PeakLevel, "the peak is kept, so levelling back grants nothing twice")
	assert.Equal(t, 2007, c.RoomId, "the church of the last city")
	assert.Nil(t, c.GetMiscData(domain.ScenarioKey), "no scenario was claimed")
	assert.False(t, module.Pending(env.user.UserId))
	assert.True(t, c.IsIron(), "defeat never takes the Iron")
}

func TestStandardDefeatStillTakesTheScenarioAndNoLevel(t *testing.T) {
	env := newDefeatEnv(t, 86)
	forceScenario(t, "wayfarer-rescue")
	env.foe(86)
	c := env.user.Character
	out := env.run("suicide", "")
	assert.Contains(t, out, "carried you to")
	assert.Equal(t, 5, c.Level)
}

func TestIronLevelLossStopsAtLevelOne(t *testing.T) {
	env := newDefeatEnv(t, 9101)
	env.foe(9101)
	c := env.user.Character
	c.SetIron(true)
	c.Level = 2
	c.Experience = c.XPTL(1)
	env.run("suicide", "")
	require.Equal(t, 1, c.Level, "two levels from level 2 stop at 1")
	assert.GreaterOrEqual(t, c.PeakLevel, 2)
}
