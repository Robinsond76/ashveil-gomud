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

// hexStatuses are the statuses a Witch's hexes leave on a foe.
var hexStatuses = []int{status.Asleep, status.Paralyzed, status.Blighted}

func hexed(c *characters.Character) bool {
	for _, id := range hexStatuses {
		if c.HasBuff(id) {
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
	fx := src.ClassEffects()
	if fx == nil || dmg <= 0 || tgt.HealthMax.Value <= 0 {
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
	if p := fx.Int(classes.HexedDamage); p > 0 && hexed(tgt) {
		pct += p
	}
	if pct > 0 {
		dmg += (dmg*pct + 50) / 100
	}
	return dmg
}
