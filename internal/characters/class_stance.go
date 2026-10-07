package characters

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/stance"
)

// StanceGear is what the character holds, as a weapon stance reads it
// (Phase 69).
func (c *Character) StanceGear() stance.Gear {
	return c.stanceGearWith(c.Equipment.Weapon)
}

func (c *Character) stanceGearWith(w items.Item) stance.Gear {
	g := stance.Gear{}
	if w.ItemId > 0 {
		spec := w.GetSpec()
		g.TwoHanded = spec.Hands == items.TwoHanded
		g.Shooting = spec.Subtype == items.Shooting
		g.Class = strings.ToLower(spec.WeaponClass)
		g.Family = strings.ToLower(spec.Family)
	}
	if o := c.Equipment.Offhand; o.ItemId > 0 {
		spec := o.GetSpec()
		g.Shield = spec.Type != items.Weapon && spec.DamageReduction > 0
	}
	return g
}

// Stance is the weapon stance the character fights in this battle, or none
// when it has no stance or no longer holds what the stance needs.
func (c *Character) Stance() stance.Stance {
	if c.RT == nil || !stance.Fits(c.RT.Stance, c.StanceGear()) {
		return stance.None
	}
	return c.RT.Stance
}

// StanceEffect is what the character's stance changes now: nothing without
// one.
func (c *Character) StanceEffect() stance.Effect {
	if c.RT == nil || c.RT.Stance == stance.None {
		return stance.Effect{}
	}
	return stance.EffectFor(c.RT.Stance, c.StanceGear())
}

// StanceEffectWith is what the character's stance does for blows struck
// with weapon, as if it were in the main hand: a dual-wielder's second
// weapon takes the stance only when the stance fits it.
func (c *Character) StanceEffectWith(weapon items.Item) stance.Effect {
	if c.RT == nil || c.RT.Stance == stance.None {
		return stance.Effect{}
	}
	return stance.EffectFor(c.RT.Stance, c.stanceGearWith(weapon))
}
