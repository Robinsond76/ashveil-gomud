package combat

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/strategy"
)

// Phase 38b: what a class's conditional blows add to a landed strike. They
// read the attacker's class effects and the target's state when the blow
// lands, and do nothing for a character with no class.

// unholyRaces are what a Paladin's Smite is for: the dead and the damned.
var unholyRaces = map[string]bool{"undead": true, "ghostly spirit": true, "demon": true}

// holyRaces are what a Demon's rending claws are for.
var holyRaces = map[string]bool{"angel": true}

// hexStatuses are the statuses only a Witch's hexes leave on a foe.
var hexStatuses = []int{status.Asleep, status.Paralyzed, status.Blighted}

// sharedHexStatuses are statuses a hex leaves that other blows leave too
// (a Tackle's knockdown, a weapon's poison, Grave Chill's hobble): they
// count as a hex only while a hex laid that status on the foe this battle.
//
// Phase 38c3: every status a hex leaves counts (a foe that is poisoned by
// Miasma, hobbled, knocked down or exposed by a hex is "hexed" too), or the
// Hag's and Crone of Ash's curse would only ever meet sleepers.
var sharedHexStatuses = []int{status.Poisoned, status.KnockedDown, status.Hobbled, status.Exposed}

// Hexed reports whether a character carries a status a hex leaves.
func Hexed(c *characters.Character) bool { return hexed(c) }

func hexed(c *characters.Character) bool {
	for _, id := range hexStatuses {
		// A status buff left behind by its expiry is not a hex.
		if status.Live(c, id) {
			return true
		}
	}
	if c.RT == nil || len(c.RT.HexBuffs) == 0 {
		return false
	}
	for _, id := range sharedHexStatuses {
		if !c.RT.HexBuffs[id] {
			continue
		}
		// Poison is the shipped buff, which carries no status spec.
		if id == status.Poisoned && c.HasBuff(id) || id != status.Poisoned && status.Live(c, id) {
			return true
		}
	}
	return false
}

// classBlowDamage is a blow's damage after the attacker's class riders:
// Finisher adds an Opening Strike's damage against a foe at or below its
// threshold, and Wounded, Smite, Rend Holy and Hexed Damage raise it by a
// percent. Only a blow that has damage to raise is changed.
func classBlowDamage(src, tgt *characters.Character, dmg int) int {
	// Phase 39a: a Sweep's or a held blow's own share of the damage.
	if src.RT != nil && src.RT.BlowPct > 0 && dmg > 0 {
		dmg = max(1, (dmg*src.RT.BlowPct+50)/100)
	}
	fx := src.ClassEffects()
	var summon *characters.SummonInfo
	if src.RT != nil {
		summon = src.RT.Summon
	}
	// Phase 38c3: a Crone of Ash's Ashen Curse raises every ally's blows
	// against the foe while it is hexed (the attacker's own Hexed Damage,
	// a Hag's, counts instead when it is larger).
	curse := 0
	if tgt.RT != nil && tgt.RT.CurseDmg > 0 {
		curse = tgt.RT.CurseDmg
	}
	if (fx == nil && summon == nil && curse == 0) || dmg <= 0 || tgt.HealthMax.Value <= 0 {
		return dmg
	}
	hpPct := tgt.Health * 100 / tgt.HealthMax.Value
	if f := fx.Int(classes.Finisher); f > 0 {
		if hpPct <= 30+10*f {
			dmg += strategy.OpeningStrikeBonus(src.Level)
		}
	}
	pct := 0
	if hpPct <= 50 {
		pct += fx.Int(classes.Wounded)
	}
	race := strings.ToLower(tgt.Race())
	if unholyRaces[race] {
		pct += fx.Int(classes.Smite)
	}
	if holyRaces[race] {
		pct += fx.Int(classes.RendHoly)
	}
	if summon != nil {
		if unholyRaces[race] {
			pct += summon.Smite
		}
		if holyRaces[race] {
			pct += summon.Rend
		}
	}
	if p := max(fx.Int(classes.HexedDamage), curse); p > 0 && hexed(tgt) {
		pct += p
	}
	// Phase 39b: a Ronin's Vengeance grows with each fallen ally.
	if p := fx.Int(classes.Vengeance); p > 0 {
		pct += p * src.Aura.Fallen
	}
	if pct > 0 {
		dmg += (dmg*pct + 50) / 100
	}
	return dmg
}

// attackRating is the attacker's Attack against this defender: raised by a
// Warlord's Battle Cry and by its mark on the defender, lowered by a
// Dread Knight's Aura of Dread when it is the one aimed at, and by a foe's
// Intimidation (a Blackguard's wound) against anyone but the one who made
// it.
func attackRating(atk, def *characters.Character) int {
	rating := atk.AttackSkill() + atk.Aura.Attack - def.ClassEffects().Int(classes.AuraDread)
	// Phase 38c1: a foe the Warlord has marked is easier for everyone to hit.
	if def.RT != nil && def.RT.Mark > 0 {
		rating += def.RT.Mark
	}
	// Phase 38c3: a Crone of Ash's Ashen Curse makes a hexed foe easier to hit.
	if def.RT != nil && def.RT.CurseAtk > 0 && hexed(def) {
		rating += def.RT.CurseAtk
	}
	if rt := atk.RT; rt != nil && rt.Intim > 0 && def.RT != rt.IntimOwner {
		rating -= rt.Intim
	}
	return rating
}
