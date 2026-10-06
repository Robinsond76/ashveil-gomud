package classes

import (
	"fmt"
	"sort"
)

// Phase 36d: gear effects. A legendary's signature and a set's bonuses are
// class effects that gear grants while worn, so they ride the same keys the
// combat, spell and company code already read. Only the keys listed here
// may be granted by gear (a test holds every shipped relic and set to the
// list), each with the cap one item or bonus may carry and the sentence a
// player reads. The sentence is built from the effect, so the text can never
// claim more than the number does.

// GearEffect describes one effect gear may grant.
type GearEffect struct {
	Key  string
	Max  int                // the most one signature or set bonus may give
	Text func(v int) string // what it does, for the value
}

var gearEffects = []GearEffect{
	{Attack, 12, func(v int) string { return fmt.Sprintf("+%d Attack", v) }},
	{Evasion, 12, func(v int) string { return fmt.Sprintf("+%d Evasion", v) }},
	{Damage, 4, func(v int) string { return fmt.Sprintf("+%d damage on every landed blow", v) }},
	{Armor, 10, func(v int) string { return fmt.Sprintf("+%d%% damage reduction on top of worn armor", v) }},
	{Block, 10, func(v int) string { return fmt.Sprintf("+%d%% chance to block", v) }},
	{Parry, 10, func(v int) string { return fmt.Sprintf("+%d%% chance to parry", v) }},
	{HealthPct, 10, func(v int) string { return fmt.Sprintf("+%d%% maximum health", v) }},
	{HealPct, 20, func(v int) string { return fmt.Sprintf("+%d%% to heals it casts", v) }},
	{SpellPct, 20, func(v int) string { return fmt.Sprintf("+%d%% spell damage", v) }},
	{Wounded, 30, func(v int) string { return fmt.Sprintf("blows deal %d%% more to a foe at or below half health", v) }},
	{Smite, 30, func(v int) string { return fmt.Sprintf("blows deal %d%% more to undead and demons", v) }},
	{RangedPct, 25, func(v int) string { return fmt.Sprintf("%d%% more damage with a shooting weapon", v) }},
	{SecondWind, 20, func(v int) string {
		return fmt.Sprintf("once a battle, below a quarter of its health at the start of its turn, heals %d%% of maximum health", v)
	}},
	{Bargain, 1, func(int) string { return "once a battle a felling blow leaves it at 1 health" }},
	{DivineShield, 1, func(int) string { return "once a battle the next blow against it is ignored" }},
	{AuraResolv, 12, func(v int) string { return fmt.Sprintf("allies in its row take %d%% less damage", v) }},
	{AuraEvade, 10, func(v int) string { return fmt.Sprintf("allies in its row gain %d Evasion", v) }},
	{AuraCompan, 6, func(v int) string { return fmt.Sprintf("the whole company takes %d%% less damage", v) }},
}

// GearEffectFor returns the gear effect for a key.
func GearEffectFor(key string) (GearEffect, bool) {
	for _, e := range gearEffects {
		if e.Key == key {
			return e, true
		}
	}
	return GearEffect{}, false
}

// GearEffectKeys lists the keys gear may grant, in a stable order.
func GearEffectKeys() []string {
	out := make([]string, 0, len(gearEffects))
	for _, e := range gearEffects {
		out = append(out, e.Key)
	}
	return out
}

// ValidateGearEffects checks a set of gear effects: known keys, values from
// 1 up to the key's cap.
func ValidateGearEffects(fx map[string]int) error {
	for k, v := range fx {
		e, ok := GearEffectFor(k)
		if !ok {
			return fmt.Errorf("%q is not an effect gear may grant", k)
		}
		if v < 1 || v > e.Max {
			return fmt.Errorf("effect %q value %d is outside 1 to %d", k, v, e.Max)
		}
	}
	return nil
}

// DescribeGearEffects lists what a set of gear effects does, one sentence
// each, in the stable key order.
func DescribeGearEffects(fx map[string]int) []string {
	keys := make([]string, 0, len(fx))
	for k := range fx {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []string
	for _, k := range keys {
		if e, ok := GearEffectFor(k); ok {
			out = append(out, e.Text(fx[k]))
		}
	}
	return out
}

// WithGear returns the character's effects with gear's added on top. Values
// add (a talent does the same); the base is never changed, because it is a
// cached map.
func WithGear(base Effects, gear map[string]int) Effects {
	if len(gear) == 0 {
		return base
	}
	out := base.Clone()
	if out == nil {
		out = Effects{}
	}
	for k, v := range gear {
		out[k] += v
	}
	return out
}
