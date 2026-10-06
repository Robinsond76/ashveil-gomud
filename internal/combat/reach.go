package combat

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// ResolveReach determines a combatant's current formationcombat.Reach from
// their equipped weapon, falling back to innateReach (a mob's own Reach
// flag; always false for a player, who has no reach source but a weapon).
// It is computed at the point of use rather than stored, so it always
// reflects the currently equipped weapon with no second source of truth.
func ResolveReach(c *characters.Character, innateReach bool) formationcombat.Reach {
	if c != nil && c.Equipment.Weapon.ItemId > 0 {
		spec := c.Equipment.Weapon.GetSpec() // ItemSpec by value; zero value if unresolvable
		if spec.Subtype == items.Shooting {
			return formationcombat.ReachAny
		}
		if spec.Reach {
			return formationcombat.ReachExtended
		}
		// Phase 39e: a Beast Tamer's whip reaches like a polearm.
		if spec.Subtype == items.Whipping && c.ClassEffects().Has(classes.BeastSic) {
			return formationcombat.ReachExtended
		}
	}

	if innateReach || c.ReachOverride() { // Phase 38c2: a Shadowstep reaches
		return formationcombat.ReachExtended
	}

	return formationcombat.ReachNone
}
