package combat

import (
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 43b weapon poisons. A coated blade that wounds spends one contact;
// each contact has a delivery chance (items.DeliveryPct) to leave the
// poison's combat status on the victim, who carries one at a time.

const (
	mirethornDodgeCut = 10 // percentage points off the victim's dodge
	leadrootCutPct    = 15 // percent off the physical blows it deals
)

// leadrootDamage cuts a strike's damage by 15% (rounded down, keeping at
// least 1) while its dealer carries leadroot.
func leadrootDamage(c *characters.Character, dmg int) int {
	if dmg <= 0 || !c.HasBuffFlag(status.FlagLeadroot) {
		return dmg
	}
	return max(1, dmg-dmg*leadrootCutPct/100)
}

// fareDamage applies a member's battle condition (Phase 50) to a blow: the
// percent hunger and a meal buff put on what it deals, then the percent
// less (or, thirsty, more) its target takes. At least 1 stays.
func fareDamage(src, tgt *characters.Character, dmg int) int {
	if dmg <= 0 {
		return dmg
	}
	if src.RT != nil && src.RT.FareDamage != 0 {
		dmg = max(1, (dmg*(100+src.RT.FareDamage)+50)/100)
	}
	if tgt.RT != nil && tgt.RT.FareGuard != 0 {
		dmg = max(1, (dmg*(100-min(tgt.RT.FareGuard, 90))+50)/100)
	}
	return dmg
}

// FareSpellFactor is a spell's multiplier from the caster's and target's
// battle condition (Phase 50), as fareDamage does for blows.
func FareSpellFactor(src, tgt *characters.Character) float64 {
	f := 1.0
	if src != nil && src.RT != nil && src.RT.FareDamage != 0 {
		f *= float64(100+src.RT.FareDamage) / 100
	}
	if tgt != nil && tgt.RT != nil && tgt.RT.FareGuard != 0 {
		f *= float64(100-min(tgt.RT.FareGuard, 90)) / 100
	}
	return f
}

// poisonSusceptibility is the target creature's weapon-poison susceptibility.
func poisonSusceptibility(targetType SourceTarget, targetMob []*mobs.Mob) string {
	if targetType == Mob && len(targetMob) > 0 && targetMob[0] != nil {
		return targetMob[0].PoisonSusceptibility
	}
	return items.PoisonNormal
}

// deliverWeaponPoison spends a contact of weapon's coating for a blow that
// wounded target, rolls the delivery, and queues the poison's status on the
// result. It returns the status word for the hit line, or "" when nothing
// was delivered.
func deliverWeaponPoison(r *AttackResult, slot items.ItemType, weapon items.Item, target *characters.Character, susceptibility string, now time.Time) string {
	if slot == `` || !weapon.Coated(now) || weapon.CoatContacts-r.PoisonSpent[slot] <= 0 {
		return ``
	}
	if r.PoisonSpent == nil {
		r.PoisonSpent = map[items.ItemType]int{}
	}
	r.PoisonSpent[slot]++ // a contact is spent whether or not it delivers
	poison, ok := items.PoisonByID(weapon.CoatKind)
	if !ok || susceptibility == items.PoisonImmune {
		return ``
	}
	// A victim carries one poison at a time, never stacked or refreshed, and
	// a contact against one performs no delivery roll.
	if status.PoisonLive(target) {
		return ``
	}
	for _, id := range r.BuffTarget {
		if status.IsWeaponPoison(id) {
			return ``
		}
	}
	chance := items.DeliveryPct
	if susceptibility == items.PoisonResistant {
		chance = items.ResistantPct
	}
	roll := util.Rand(100)
	util.LogRoll(`Poison`, roll, chance)
	if roll >= chance {
		return ``
	}
	r.BuffTarget = append(r.BuffTarget, poison.BuffID)
	return status.Word(poison.BuffID)
}
