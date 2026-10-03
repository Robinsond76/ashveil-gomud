package combat

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMeterOpeningSequencesAndCaps(t *testing.T) {
	for _, tc := range []struct {
		rate float64
		want []int
	}{
		{0.6, []int{1, 0, 1, 0, 1, 1}},
		{1, []int{1, 1, 1, 1, 1, 1}},
		{1.5, []int{1, 1, 2, 1, 2, 1}},
	} {
		var m Meter
		for _, want := range tc.want {
			assert.Equal(t, want, m.Fill(tc.rate, 2))
		}
	}
	var m Meter
	assert.Equal(t, 1, m.Fill(1, 2))
	assert.Equal(t, 2, m.Fill(10, 2))
	assert.Equal(t, 99.0, m.Points)
	assert.Equal(t, 1, m.Fill(0.6, 2), "whole turns were discarded, not banked")
}

func TestTempoOwnSpeedAndClamp(t *testing.T) {
	c := characters.Character{}
	for _, tc := range []struct {
		speed int
		want  float64
	}{{-100, 0.6}, {10, 1}, {30, 1.5}, {1000, 1.5}} {
		c.Stats.Speed.ValueAdj = tc.speed
		assert.InDelta(t, tc.want, Tempo(&c), 1e-9)
	}
	c.Stats.Speed.ValueAdj = 10
	target := characters.Character{}
	target.Stats.Speed.ValueAdj = 1000
	assert.Equal(t, 1, combatAttackCount(c, target), "one unarmed turn regardless of target Speed")
}

func TestTempoBurdenAndAttackModifiers(t *testing.T) {
	burdenSpecs(t)
	c := characters.Character{}
	c.Stats.Speed.ValueAdj = 10
	assert.InDelta(t, 1, Tempo(&c), 1e-9)
	heavy(&c)
	assert.InDelta(t, 0.65, Tempo(&c), 1e-9, "full burden reduces tempo by 35 percent")
	const id = 99425
	items.SetTestItemSpec(&items.ItemSpec{ItemId: id, Name: "test tempo ring", Type: items.Ring, StatMods: statmods.StatMods{"attacks": 2}})
	t.Cleanup(func() { items.RemoveTestItemSpec(id) })
	c.Equipment.Ring = items.New(id)
	assert.InDelta(t, 1.2*0.65, Tempo(&c), 1e-9, "attacks bonus applies before burden")
	c.Equipment.Body = items.Item{}
	assert.InDelta(t, 1.2, Tempo(&c), 1e-9)
}

func TestMeterFractionalRateOverManyRounds(t *testing.T) {
	var m Meter
	turns := 0
	for i := 0; i < 10001; i++ {
		turns += m.Fill(0.6, 2)
	}
	assert.Equal(t, 6001, turns)
	assert.InDelta(t, 0, m.Points, 1e-6)
}
