package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

const (
	loadMailID  = 99501 // 9 kg of mail
	loadSwordID = 99502 // 1.5 kg
	loadPackID  = 99503 // a 2 kg pack that adds 20 kg of cargo room
	loadStoneID = 99504 // 4 kg
)

// agilityConfig pins the agility keys at the design's proposals.
func agilityConfig(t *testing.T) {
	t.Helper()
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.AgilityBaseKg = 15
	gameplay.Combat.AgilityStrengthKg = 0.5
	gameplay.Combat.AgilityFreeLoad = 0.35
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
}

func loadSpecs(t *testing.T) {
	t.Helper()
	items.SetTestItemSpec(&items.ItemSpec{ItemId: loadMailID, Name: "test mail", Type: items.Body, Weight: 9000})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: loadSwordID, Name: "test sword", Type: items.Weapon, Weight: 1500})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: loadPackID, Name: "test pack", Type: items.Object, Weight: 2000, CarryBonus: 20000})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: loadStoneID, Name: "test stone", Type: items.Object, Weight: 4000})
	t.Cleanup(func() {
		for _, id := range []int{loadMailID, loadSwordID, loadPackID, loadStoneID} {
			items.RemoveTestItemSpec(id)
		}
	})
}

func loadFighter(strength int) *Character {
	c := New()
	c.Stats.Strength.ValueAdj = strength
	return c
}

// PersonalGrams is what is worn plus what is carried; a pack counts its
// own weight, never the cargo room it adds.
func TestPersonalGrams(t *testing.T) {
	loadSpecs(t)
	c := loadFighter(10)
	assert.Equal(t, 0, c.PersonalGrams(), "nothing worn or carried")
	c.Equipment.Body = items.New(loadMailID)
	c.Equipment.Weapon = items.New(loadSwordID)
	assert.Equal(t, 10500, c.PersonalGrams(), "worn")
	c.Items = append(c.Items, items.New(loadPackID), items.New(loadStoneID))
	assert.Equal(t, 16500, c.PersonalGrams(), "worn and carried; the pack's bonus is cargo room, not weight")
}

// Capacity is 15 kg + 0.5 kg a point of Strength; a pack adds nothing.
func TestAgilityCapacity(t *testing.T) {
	agilityConfig(t)
	loadSpecs(t)
	assert.Equal(t, 20000, loadFighter(10).AgilityCapacityGrams())
	assert.Equal(t, 40000, loadFighter(50).AgilityCapacityGrams())
	assert.Equal(t, 15000, loadFighter(-5).AgilityCapacityGrams(), "never below the base")
	packed := loadFighter(10)
	packed.Items = append(packed.Items, items.New(loadPackID))
	assert.Equal(t, 20000, packed.AgilityCapacityGrams(), "a pack's carry bonus is company cargo")
}

func TestBurdenFor(t *testing.T) {
	agilityConfig(t)
	cases := []struct {
		name      string
		load, cap int
		want      float64
		wantWord  string
	}{
		{"nothing", 0, 20000, 0, BurdenNone},
		{"inside the free share", 7000, 20000, 0, BurdenNone},
		{"just past it", 8300, 20000, 0.1, BurdenLight},
		{"half way", 13500, 20000, 0.5, BurdenMedium},
		{"near full", 18700, 20000, 0.9, BurdenHeavy},
		{"full", 20000, 20000, 1, BurdenHeavy},
		{"beyond capacity is held at 1", 60000, 20000, 1, BurdenHeavy},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := BurdenFor(tc.load, tc.cap)
			assert.InDelta(t, tc.want, b, 0.0001)
			assert.Equal(t, tc.wantWord, BurdenWord(b))
		})
	}
}

func TestBurdenWordBands(t *testing.T) {
	assert.Equal(t, BurdenNone, BurdenWord(0))
	assert.Equal(t, BurdenLight, BurdenWord(0.01))
	assert.Equal(t, BurdenLight, BurdenWord(0.33))
	assert.Equal(t, BurdenMedium, BurdenWord(0.34))
	assert.Equal(t, BurdenMedium, BurdenWord(0.66))
	assert.Equal(t, BurdenHeavy, BurdenWord(0.67))
	assert.Equal(t, BurdenHeavy, BurdenWord(1))
}

// Strength carries the same kit more lightly: a fraction of capacity, not
// raw kilograms.
func TestStrengthEasesBurden(t *testing.T) {
	agilityConfig(t)
	loadSpecs(t)
	kit := func(c *Character) *Character {
		c.Equipment.Body = items.New(loadMailID)
		c.Equipment.Weapon = items.New(loadSwordID)
		c.Items = append(c.Items, items.New(loadStoneID))
		return c
	}
	weak, strong := kit(loadFighter(10)), kit(loadFighter(50))
	assert.InDelta(t, (14.5/20-0.35)/0.65, weak.Burden(), 0.0001)
	assert.Equal(t, BurdenMedium, weak.BurdenWord())
	assert.InDelta(t, (14.5/40-0.35)/0.65, strong.Burden(), 0.0001)
	assert.Equal(t, BurdenLight, strong.BurdenWord())
}

// Review finding: a test config that skips validation (a free share of 1
// or more) gives no NaN; the free share is treated as none.
func TestBurdenForUnvalidatedFreeShare(t *testing.T) {
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.AgilityFreeLoad = 1
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	assert.Equal(t, 1.0, BurdenFor(20000, 20000))
	assert.InDelta(t, 0.5, BurdenFor(10000, 20000), 0.0001)
}
