package combat

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// StatEdge is the attacker's edge over the defender in one stat (30g6
// amendment): the difference over the configured StatEdgeSpan, held to
// −1..1. Stats grow in small steps (30g4), so a ratio of the two stats
// would swing a chance far too much for one point; a span keeps each point
// a fixed share of the way to a bound.
func StatEdge(atkStat, defStat int) float64 {
	span := float64(configs.GetCombatConfig().StatEdgeSpan)
	if span <= 0 || math.IsNaN(span) || math.IsInf(span, 0) {
		span = 10
	}
	return max(-1, min(1, (float64(atkStat)-float64(defStat))/span))
}

// statAdvantage is StatEdge, never below 0: one-sided chances start at
// their minimum and only grow with an advantage.
func statAdvantage(atkStat, defStat int) float64 {
	return max(0, StatEdge(atkStat, defStat))
}

// edgeChance moves an even value toward the upper bound when edge is
// positive and toward the lower bound when it is negative.
func edgeChance(even, lo, hi float64, edge float64) float64 {
	even = max(lo, min(hi, even))
	if edge >= 0 {
		return even + edge*(hi-even)
	}
	return even + edge*(even-lo)
}

// floorChance rounds a chance down, forgiving float error just below a
// whole number (0.7 × 50 is 34.99… in binary).
func floorChance(v float64) int {
	return int(math.Floor(v + 1e-9))
}

// advantageChance grows from lo to hi with an advantage in 0..1.
func advantageChance(lo, hi, advantage float64) float64 {
	return lo + advantage*(hi-lo)
}

// resolveAttackWeapons returns the candidate weapon list for a character,
// applying the same selection logic used by calculateCombat.
// It does not trim for dual-wield skill - callers handle that themselves.
func resolveAttackWeapons(char characters.Character) []items.Item {
	attackWeapons, _ := resolveAttackWeaponSlots(char)
	return attackWeapons
}

// resolveAttackWeaponSlots is resolveAttackWeapons plus, for each weapon,
// the equipment slot it came from (items.Weapon, items.Offhand, or ""
// for the unarmed placeholder), so a Phase 23b edge spent in a round can
// be charged to the right real weapon afterwards.
func resolveAttackWeaponSlots(char characters.Character) ([]items.Item, []items.ItemType) {
	attackWeapons := []items.Item{}
	slots := []items.ItemType{}
	if char.Equipment.Weapon.ItemId > 0 {
		attackWeapons = append(attackWeapons, char.Equipment.Weapon)
		slots = append(slots, items.Weapon)
	}
	if char.Equipment.Offhand.ItemId > 0 && char.Equipment.Offhand.GetSpec().Type == items.Weapon {
		attackWeapons = append(attackWeapons, char.Equipment.Offhand)
		slots = append(slots, items.Offhand)
	}
	if len(attackWeapons) == 0 {
		attackWeapons = append(attackWeapons, items.Item{ItemId: 0})
		slots = append(slots, ``)
	}
	return attackWeapons, slots
}

// spendEdges charges the strikes an attack round spent to the attacker's
// real weapons (calculateCombat works on a copy of the character).
func spendEdges(char *characters.Character, spent map[items.ItemType]int) {
	for slot, n := range spent {
		if itm := char.Equipment.Get(slot); itm != nil {
			itm.SpendEdge(n)
		}
	}
}

// damageBonus is the flat bonus a blow adds: the minimum, plus Strength ×
// DamagePerStrength, plus DamageEdgeMax scaled by the Strength advantage,
// rounded down and held to DamageBonusMin–DamageBonusMax.
func damageBonus(atkStr, defStr int) int {
	cfg := configs.GetCombatConfig()
	lo, hi := int(cfg.DamageBonusMin), int(cfg.DamageBonusMax)
	strength := math.Max(float64(atkStr), 0) * float64(cfg.DamagePerStrength)
	bonus := math.Floor(float64(lo) + strength + statAdvantage(atkStr, defStr)*float64(cfg.DamageEdgeMax) + 1e-9)
	if bonus >= float64(hi) {
		return hi
	}
	if bonus <= float64(lo) {
		return lo
	}
	return int(bonus)
}

// hitChance returns a hit probability in [ToHitMin, ToHitMax]: ToHitEven
// at equal Speed, moved toward a bound by the Speed edge.
func hitChance(atkSpd, defSpd int) int {
	cfg := configs.GetCombatConfig()
	chance := edgeChance(float64(cfg.ToHitEven), float64(cfg.ToHitMin), float64(cfg.ToHitMax), StatEdge(atkSpd, defSpd))
	return clampToHit(floorChance(chance))
}

// Hits returns whether an attack connects, incorporating an optional modifier.
// equal stats will result in 0% of max
func Hits(atkSpd, defSpd, hitModifier int) bool {
	hit, _ := hitRoll(atkSpd, defSpd, hitModifier, 0)
	return hit
}

// clampToHit bounds a hit chance to [ToHitMin, ToHitMax].
func clampToHit(toHit int) int {
	cfg := configs.GetCombatConfig()
	minHit := int(cfg.ToHitMin)
	maxHit := int(cfg.ToHitMax)
	if toHit < minHit {
		toHit = minHit
	}
	if toHit > maxHit {
		toHit = maxHit
	}
	return toHit
}

// hitRoll is Hits with a separate bonus (Phase 24 company chemistry) added
// to the modifier before the chance is bounded. byBonus reports a hit that
// only the bonus made: the roll fell between the chance without it and the
// chance with it. It rolls once, as Hits always has.
func hitRoll(atkSpd, defSpd, hitModifier, bonus int) (hit, byBonus bool) {
	base := hitChance(atkSpd, defSpd) + hitModifier
	without := clampToHit(base)
	toHit := clampToHit(base + bonus)

	roll := util.Rand(100)
	util.LogRoll(`Hits`, roll, toHit)
	hit = roll < toHit
	return hit, hit && roll >= without
}

// combatAttackCount resolves one complete weapon turn. Frequency is owned
// by the action meter, never by the current target or weapon category.
func combatAttackCount(sourceChar characters.Character, targetChar characters.Character) int {
	return 1
}

// critChance returns the integer crit probability in [CritChanceMin,
// CritChanceMax]: CritChanceEven at equal Smarts, moved by the Smarts edge.
// Buff flags are applied after.
func critChance(atkSmarts, defSmarts int, hasAccuracy, targetHasBlink bool) int {
	cfg := configs.GetCombatConfig()
	minChance := int(cfg.CritChanceMin)
	actual := floorChance(edgeChance(float64(cfg.CritChanceEven), float64(minChance), float64(cfg.CritChanceMax), StatEdge(atkSmarts, defSmarts)))
	if hasAccuracy {
		actual *= 2
	}
	if targetHasBlink {
		actual /= 2
	}
	if actual < minChance {
		actual = minChance
	}
	if actual > 100 {
		actual = 100
	}
	return actual
}

// Crits rolls whether an attack is a critical hit.
func Crits(sourceChar characters.Character, targetChar characters.Character) bool {
	chance := critChance(
		sourceChar.Stats.Smarts.ValueAdj,
		targetChar.Stats.Smarts.ValueAdj,
		sourceChar.HasBuffFlag("accuracy"),
		targetChar.HasBuffFlag("blink"),
	)
	// Phase 30a: an exposed target is easier to catch open.
	if targetChar.HasBuffFlag(status.FlagExposed) {
		chance = min(chance+status.ExposedCritBonus, 100)
	}
	critRoll := util.Rand(100)
	util.LogRoll(`Crits`, critRoll, chance)
	return critRoll < chance
}

// critMultiplier returns the damage multiplier for a critical hit in
// [CritMultMin, CritMultMax]: the minimum, grown by the Perception advantage.
func critMultiplier(atkPerc, defPerc int) float64 {
	cfg := configs.GetCombatConfig()
	return advantageChance(float64(cfg.CritMultMin), float64(cfg.CritMultMax), statAdvantage(atkPerc, defPerc))
}

// critDamageBonus returns the extra damage added to a hit that is a critical,
// scaled by the attacker's crit multiplier relative to the defender.
func critDamageBonus(dCount, dSides, dBonus, atkPerc, defPerc int) int {
	base := dCount*dSides + dBonus
	if base < 0 {
		base = 0
	}
	mult := critMultiplier(atkPerc, defPerc)
	return int(math.Floor(float64(base) * (mult - 1.0)))
}

// dodgeChance returns the probability in [DodgeChanceMin, DodgeChanceMax] that
// the defender dodges an incoming hit: the minimum, grown by the defender's
// Perception advantage over the attacker.
func dodgeChance(defPerc, atkPerc int) int {
	cfg := configs.GetCombatConfig()
	return floorChance(advantageChance(float64(cfg.DodgeChanceMin), float64(cfg.DodgeChanceMax), statAdvantage(defPerc, atkPerc)))
}

// burdenDodgeLoss is the share of dodge a fully burdened character loses
// (Phase 30g3, decision 3).
const burdenDodgeLoss = 0.6

// burdenedDodge is a dodge chance after the defender's burden (Phase
// 30g3): dodge × (1 − 0.6 × burden), rounded. A fully burdened fighter
// keeps 40% of their dodge; parry and block are not touched.
func burdenedDodge(dodge int, burden float64) int {
	if math.IsNaN(burden) {
		burden = 0
	}
	return int(math.Round(float64(dodge) * (1 - burdenDodgeLoss*max(0, min(1, burden)))))
}

// blockChance returns the percent chance in [BlockChanceMin, BlockChanceMax]
// that a shield-bearer blocks a strike (Phase 30g2): the minimum, plus the
// shield's own armor, moved up or down by half the range at a full Strength
// edge. Even Strength with a 5-armor shield blocks 20%.
func blockChance(shieldArmor, defStr, atkStr int) int {
	cfg := configs.GetCombatConfig()
	minBlock := int(cfg.BlockChanceMin)
	maxBlock := int(cfg.BlockChanceMax)
	strength := StatEdge(defStr, atkStr) * float64(maxBlock-minBlock) / 2
	return max(minBlock, min(maxBlock, minBlock+shieldArmor+int(math.Round(strength))))
}

// parryModifier is a weapon's parry modifier in percent (Phase 30g2,
// decision 16): swords +5, polearms (reach) +5, daggers −5, axes, maces
// and whips 0, plus the item's own `parry` (a staff's +5). ok is false
// for what can't parry: no weapon, claws, or a ranged weapon.
func parryModifier(weapon items.Item) (mod int, ok bool) {
	if weapon.ItemId == 0 {
		return 0, false
	}
	spec := weapon.GetSpec()
	if spec.Type != items.Weapon {
		return 0, false
	}
	switch spec.Subtype {
	case items.Claws, items.Shooting:
		return 0, false
	case items.Slashing:
		mod = 5
	case items.Stabbing:
		mod = -5
	}
	if spec.Reach {
		mod = 5
	}
	return mod + spec.Parry, true
}

// parryChance returns the percent chance that a melee strike is parried:
// the minimum grown by the Speed advantage up to ParryChanceMax, then the
// weapon's modifier, which moves the whole range (a sword parries 10–35%,
// a dagger 0–25%). A ParryChanceMax of 0 turns parrying off.
func parryChance(defSpeed, atkSpeed, weaponMod int) int {
	cfg := configs.GetCombatConfig()
	minParry := int(cfg.ParryChanceMin)
	maxParry := int(cfg.ParryChanceMax)
	if maxParry <= 0 {
		return 0
	}
	base := floorChance(advantageChance(float64(minParry), float64(maxParry), statAdvantage(defSpeed, atkSpeed)))
	base = max(minParry, min(maxParry, base))
	return max(0, min(100, base+weaponMod))
}

// rollDefense rolls a defense's chance, logging it under name.
func rollDefense(name string, chance int) bool {
	roll := util.Rand(100)
	util.LogRoll(name, roll, chance)
	return roll < chance
}

// BashChance returns the probability in [BashChanceMin, BashChanceMax] that
// a shield-bearer can bash when a melee strike is blocked (Phase 30g2): the
// minimum, grown by the bearer's Strength advantage over the attacker.
func BashChance(defStr, atkStr int) int {
	cfg := configs.GetCombatConfig()
	return floorChance(advantageChance(float64(cfg.BashChanceMin), float64(cfg.BashChanceMax), statAdvantage(defStr, atkStr)))
}

// dualWieldHitPenalty returns the negative hit modifier applied to the offhand
// weapon based on the attacker's dual-wield skill level.
func dualWieldHitPenalty(dwLevel int) int {
	if dwLevel < 4 {
		return -35
	}
	return -25
}

// dualWieldActiveWeaponCount returns how many weapons fire this round based on
// dual-wield skill level and whether both equipped weapons are claws.
func dualWieldActiveWeaponCount(dwLevel int, bothClaws bool) int {
	if bothClaws || dwLevel >= 3 {
		return 2
	}
	if dwLevel == 2 {
		if util.Rand(100) < 50 {
			return 2
		}
		return 1
	}
	return 1
}

// applyDefenseReduction applies a stochastic defense roll to incoming damage,
// returning the final damage and the amount reduced.
func applyDefenseReduction(damage, defenseRating int) (finalDamage, reduction int) {
	defenseAmt := util.Rand(defenseRating)
	if defenseAmt > 0 {
		reduction = int(math.Round((float64(defenseAmt) / 100) * float64(damage)))
		finalDamage = damage - reduction
		return finalDamage, reduction
	}
	return damage, 0
}

// damagePercentOfMax returns the dealt damage expressed as a percentage of the
// theoretical maximum damage for the given dice configuration.
func damagePercentOfMax(damage, dCount, dSides, dBonus int) int {
	maxDmg := dCount*dSides + dBonus
	if maxDmg < 1 {
		maxDmg = 1
	}
	return int(math.Ceil(float64(damage) / float64(maxDmg) * 100))
}

// AlignmentChange returns the alignment delta for a killer after slaying a target.
func AlignmentChange(killerAlignment int8, killedAlignment int8) int {

	isKillerGood := killerAlignment > characters.AlignmentNeutralHigh
	isKillerEvil := killerAlignment < characters.AlignmentNeutralLow
	isKillerNeutral := killerAlignment >= characters.AlignmentNeutralLow && killerAlignment <= characters.AlignmentNeutralHigh

	isKilledGood := killedAlignment > characters.AlignmentNeutralHigh
	isKilledEvil := killedAlignment < characters.AlignmentNeutralLow
	isKilledNeutral := killedAlignment >= characters.AlignmentNeutralLow && killedAlignment <= characters.AlignmentNeutralHigh

	deltaAbs := math.Abs(math.Max(float64(killerAlignment), float64(killedAlignment))-math.Min(float64(killerAlignment), float64(killedAlignment))) * 0.5

	changeAmt := 0
	if deltaAbs <= 10 {
		changeAmt = 0
	} else if deltaAbs <= 30 {
		changeAmt = 1
	} else if deltaAbs <= 60 {
		changeAmt = 2
	} else if deltaAbs <= 80 {
		changeAmt = 3
	} else {
		changeAmt = 4
	}

	factor := 0

	if isKillerGood {
		if isKilledGood {
			factor = -2
			changeAmt = int(math.Max(float64(changeAmt), 1))
		} else if isKilledEvil {
			factor = 1
		} else if isKilledNeutral {
			factor = -1
		}
	} else if isKillerEvil {
		if isKilledGood {
			factor = -1
		} else if isKilledEvil {
			factor = 2
			changeAmt = int(math.Max(float64(changeAmt), 1))
		} else if isKilledNeutral {
			factor = -1
		}
	} else if isKillerNeutral {
		if isKilledGood {
			factor = -1
		} else if isKilledEvil {
			factor = 1
		} else if isKilledNeutral {
			factor = 0
		}
	}

	return factor * changeAmt
}

// expectedDPS estimates average damage per round without randomness.
func expectedDPS(atkChar characters.Character, defChar characters.Character) float64 {

	rate := math.Min(Tempo(&atkChar), float64(configs.GetCombatConfig().MaxTurnsPerRound))

	statDmgBonus := damageBonus(atkChar.Stats.Strength.ValueAdj, defChar.Stats.Strength.ValueAdj)

	attackWeapons := resolveAttackWeapons(atkChar)

	weaponWeight := make([]float64, len(attackWeapons))
	if len(attackWeapons) == 1 {
		weaponWeight[0] = 1.0
	} else {
		dwLevel := atkChar.GetSkillLevel(`dual-wield`)
		alwaysDual := atkChar.Equipment.Weapon.GetSpec().Subtype == items.Claws &&
			atkChar.Equipment.Offhand.GetSpec().Subtype == items.Claws
		switch {
		case alwaysDual || dwLevel >= 3:
			weaponWeight[0] = 1.0
			weaponWeight[1] = 1.0
		case dwLevel == 2:
			weaponWeight[0] = 1.0
			weaponWeight[1] = 0.5
		default:
			weaponWeight[0] = 1.0
			weaponWeight[1] = 0.0
		}
	}

	// hitChance already enforces [ToHitMin, ToHitMax].
	hitPct := float64(hitChance(atkChar.Stats.Speed.ValueAdj, defChar.Stats.Speed.ValueAdj)) / 100.0

	dwLevel := atkChar.GetSkillLevel(`dual-wield`)
	dwPenalty := 0.0
	if len(attackWeapons) > 1 {
		dwPenalty = float64(-dualWieldHitPenalty(dwLevel)) / 100.0
	}

	cfg := configs.GetCombatConfig()
	minHitPct := float64(cfg.ToHitMin) / 100.0

	critPct := float64(critChance(
		atkChar.Stats.Smarts.ValueAdj,
		defChar.Stats.Smarts.ValueAdj,
		false,
		false,
	)) / 100.0

	// A hit that lands is still negated if the defender dodges.
	// Expected damage probability per attack = hitPct * (1 - dodgePct).
	dodgePct := float64(burdenedDodge(dodgeChance(
		defChar.Stats.Perception.ValueAdj,
		atkChar.Stats.Perception.ValueAdj,
	), defChar.Burden())) / 100.0

	// Defense reduces damage by an expected fraction of defenseRating/200
	// (average of a uniform roll over [0, defenseRating) divided by 100).
	defenseFraction := float64(defChar.GetDefense()) / 200.0
	if defenseFraction > 0.95 {
		defenseFraction = 0.95
	}

	totalDPS := 0.0

	for wIdx, weapon := range attackWeapons {
		wWeight := weaponWeight[wIdx]
		if wWeight <= 0 {
			continue
		}

		var attacks, dCount, dSides, dBonus int
		if weapon.ItemId > 0 {
			attacks, dCount, dSides, dBonus, _ = weapon.GetDiceRoll()
		} else {
			attacks, dCount, dSides, dBonus, _ = atkChar.GetDefaultDiceRoll()
		}
		dBonus += statDmgBonus

		avgRoll := float64(dCount) * float64(dSides+1) / 2.0
		avgDmg := avgRoll + float64(dBonus)
		if avgDmg < 0 {
			avgDmg = 0
		}

		critBonusAmt := float64(critDamageBonus(dCount, dSides, dBonus,
			atkChar.Stats.Perception.ValueAdj, defChar.Stats.Perception.ValueAdj))
		critBonus := critBonusAmt * critPct

		effHit := hitPct
		if wIdx > 0 {
			effHit = math.Max(minHitPct, hitPct-dwPenalty)
		}
		// Subtract the expected fraction of hits that get dodged.
		effHit *= (1.0 - dodgePct)

		rawDmg := (avgDmg + critBonus) * effHit
		netDmg := rawDmg * (1.0 - defenseFraction)

		for atkIdx := 0; atkIdx < attacks; atkIdx++ {
			totalDPS += netDmg * wWeight
		}
	}

	return totalDPS * rate
}

// ExpectedDamage is the attacker's average damage per round against the
// defender, without dice (Ashveil Phase 33i1: the company assessment). It
// reads only; no state changes and no random number is drawn.
func ExpectedDamage(atk, def *characters.Character) float64 {
	if atk == nil || def == nil {
		return 0
	}
	return expectedDPS(*atk, *def)
}

// CombatOdds returns the ratio of rounds-for-attacker-to-kill-defender to
// rounds-for-defender-to-kill-attacker. Values above 1.0 favor the attacker.
func CombatOdds(atkChar characters.Character, defChar characters.Character) float64 {
	atkDPS := expectedDPS(atkChar, defChar)
	defDPS := expectedDPS(defChar, atkChar)

	defHP := float64(defChar.Health)
	if defHP < 1 {
		defHP = 1
	}
	atkHP := float64(atkChar.Health)
	if atkHP < 1 {
		atkHP = 1
	}

	if atkDPS < 0.001 {
		return 0
	}

	atkRoundsToKill := defHP / atkDPS

	if defDPS < 0.001 {
		return math.MaxFloat64
	}

	defRoundsToKill := atkHP / defDPS

	return defRoundsToKill / atkRoundsToKill
}

// DodgeRetentionPct is the share of dodge retained under personal burden.
// The preview uses the same rounding and rule as combat.
func DodgeRetentionPct(c *characters.Character) int {
	if c == nil {
		return 100
	}
	return burdenedDodge(100, c.Burden())
}
