package items

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPackSpecRequiresCapacityAndNoCombatModifiers(t *testing.T) {
	valid := ItemSpec{ItemId: 989786, Name: "knapsack", Type: Pack, Subtype: Wearable, Weight: 400, CarryBonus: 10000}
	assert.NoError(t, valid.Validate())
	for _, mutate := range []func(*ItemSpec){
		func(s *ItemSpec) { s.CarryBonus = 0 },
		func(s *ItemSpec) { s.CarryBonus = -1 },
		func(s *ItemSpec) { s.Subtype = Mundane },
		func(s *ItemSpec) { s.DamageReduction = 1 },
		func(s *ItemSpec) { s.StatMods = map[string]int{"strength": 1} },
		func(s *ItemSpec) { s.WornBuffIds = []int{1} },
	} {
		spec := valid
		mutate(&spec)
		assert.Error(t, spec.Validate())
	}
}
