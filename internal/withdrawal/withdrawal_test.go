package withdrawal

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMobilityBurdenAndLastingWounds(t *testing.T) {
	c := &characters.Character{}
	c.HealthMax.Value = 100
	c.Stats.Speed.ValueAdj = 20
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.AgilityBaseKg = 1
	gameplay.Combat.AgilityStrengthKg = 0
	gameplay.Combat.AgilityFreeLoad = 0
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 999999, Name: "heavy load", Weight: 1000})
	t.Cleanup(func() { items.RemoveTestItemSpec(999999) })
	before := Mobility(c)
	c.Items = []items.Item{items.New(999999)}
	require.Positive(t, c.Burden())
	assert.Less(t, Mobility(c), before)
	loaded := Mobility(c)
	c.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Points: 20}}
	assert.Less(t, Mobility(c), loaded)
	c.Wounds[0].Light = true
	assert.Equal(t, loaded, Mobility(c), "temporary wounds don't count as lasting injury")
	assert.Greater(t, Chance(before, 20, 0), Chance(loaded, 20, 0))
	assert.Equal(t, min(95, Chance(loaded, 20, 0)+15), Chance(loaded, 20, 15))
	assert.Equal(t, 100, Chance(loaded, 0, 0))
}
