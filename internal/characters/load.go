package characters

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// Personal load and agility (Phase 30g3). A character's own worn and
// carried weight against a Strength-based capacity sets how burdened they
// are in a fight. Company cargo, packs' carrying bonus, and mounts are not
// part of it and add nothing to the capacity (owner decision 2). Nothing
// here is saved: it is read from the live items whenever it is needed.

// Burden words, lightest first. Players see only these, never a ratio.
const (
	BurdenNone    = `unburdened`
	BurdenLight   = `lightly burdened`
	BurdenMedium  = `burdened`
	BurdenHeavy   = `heavily burdened`
	burdenLightTo = 1.0 / 3
	burdenMedTo   = 2.0 / 3
)

// PersonalGrams weighs everything the character wears and carries.
func (c *Character) PersonalGrams() int {
	total := 0
	for i := range c.Items {
		total += c.Items[i].Weight()
	}
	for _, itm := range c.Equipment.GetAllItems() {
		total += itm.Weight()
	}
	return total
}

// AgilityCapacityGrams is how much the character can wear and carry at
// full burden: AgilityBaseKg + AgilityStrengthKg per point of adjusted
// Strength. It is never below AgilityBaseKg.
func (c *Character) AgilityCapacityGrams() int {
	cfg := configs.GetCombatConfig()
	kg := float64(cfg.AgilityBaseKg) + float64(cfg.AgilityStrengthKg)*float64(max(0, c.Stats.Strength.ValueAdj))
	return max(1, int(math.Round(kg*1000)))
}

// Burden is how burdened the character is, 0 to 1: nothing within the
// free share (AgilityFreeLoad) of their capacity, rising evenly to 1 at
// full capacity, and held at 1 beyond it.
func (c *Character) Burden() float64 {
	return BurdenFor(c.PersonalGrams(), c.AgilityCapacityGrams())
}

// BurdenFor is the burden of a load against a capacity, both in grams.
func BurdenFor(loadGrams, capacityGrams int) float64 {
	if loadGrams <= 0 || capacityGrams <= 0 {
		return 0
	}
	free := float64(configs.GetCombatConfig().AgilityFreeLoad)
	b := (float64(loadGrams)/float64(capacityGrams) - free) / (1 - free)
	return max(0, min(1, b))
}

// BurdenWord names a burden for players.
func BurdenWord(b float64) string {
	switch {
	case b <= 0:
		return BurdenNone
	case b < burdenLightTo:
		return BurdenLight
	case b < burdenMedTo:
		return BurdenMedium
	}
	return BurdenHeavy
}

// BurdenWord names how burdened the character is.
func (c *Character) BurdenWord() string {
	return BurdenWord(c.Burden())
}
