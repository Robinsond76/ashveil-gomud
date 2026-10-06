package hooks

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"slices"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/hexes"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/strategy"
)

// Phase 38a: the Witch. Its role (controller) casts hexes that take foes'
// turns away. This file chooses which foes a hex is worth casting at and
// which of them it reaches; the spells' scripts roll the resist and apply
// the status (ScriptActor.CastHex), and the status engine does the rest.

// hexEligible reports whether a foe is worth the hex: it doesn't carry its
// status, isn't immune to it, isn't already the target of a hex of the same
// kind being chanted, (Blight) heals, and (Dread) can lose nerve.
func hexEligible(h hexes.Hex, a actor, id int) bool {
	m := mobs.GetInstance(id)
	if m == nil || m.Character.Health < 1 || m.Character.HasBuffFlag("hidden") {
		return false // hexTargets' pool (enemyparty.Foes) leaves these out too
	}
	buff := h.Buff
	if h.Morale {
		buff = -1
	}
	if h.Buff > 0 && m.Character.HasBuff(h.Buff) {
		return false
	}
	if hexes.Default.Immune(fmt.Sprintf(`m%d`, id), buff) {
		return false
	}
	if t := enemyTemperament(m); h.Morale && (t == "" || t == "unbreakable") {
		return false // a dread that can't move it isn't worth the mana (DreadCheck)
	}
	if h.NeedsHealer && strategy.Role(m.EnemyRole()) != strategy.Healer {
		return false
	}
	return !hexChanting(h.Spell, id, a)
}

// hexChanting reports whether a hex of the spell is already being chanted at
// the foe by someone else, so two Witches hex different foes.
func hexChanting(spellId string, foe int, except actor) bool {
	chanting := func(c *characters.Character, who caster) bool {
		if who == except.who || c.Aggro == nil || c.Aggro.Type != characters.SpellCast || c.Aggro.SpellInfo.SpellId != spellId {
			return false
		}
		return slices.Contains(c.Aggro.SpellInfo.TargetMobInstanceIds, foe)
	}
	for _, other := range hexSide {
		if chanting(other.char, other.who) {
			return true
		}
	}
	return false
}

// hexSide is the company's actors for the pass in progress, so a hex
// chanting anywhere on the side is seen. Game loop only.
var hexSide []actor

// hexReady is strategy.Situation.CanHex for a caster: whether any foe of
// the group is worth the spell.
func hexReady(a actor, foes []int) func(string) bool {
	return func(spellId string) bool {
		h, ok := hexes.For(spellId)
		if !ok {
			return false
		}
		for _, id := range foes {
			if hexEligible(h, a, id) {
				return true
			}
		}
		return false
	}
}

// hexTargets are the foes a hex reaches, the first being its primary: a foe
// winding up a blow, else a chanting caster, else the nearest (front-most);
// then, up to the caster's reach (hexes.Reach), the rest of the primary's row,
// then the nearest others. Miasma covers the primary's row only.
func hexTargets(spellId string, a actor, g enemyparty.Group, foes []int) []int {
	h, ok := hexes.For(spellId)
	if !ok {
		return nil
	}
	att := a.att
	att.Spell = true // a spell reaches anyone
	var pool []strategy.Foe
	for _, f := range enemyparty.Foes(g, att) {
		if slices.Contains(foes, f.ID) && hexEligible(h, a, f.ID) {
			pool = append(pool, f)
		}
	}
	if len(pool) == 0 {
		return nil
	}
	return strategy.HexTargets(pool, h.Row, h.ReachOf(a.char.Level)+a.char.ClassEffects().Int(classes.HexReach), WindingUp)
}
