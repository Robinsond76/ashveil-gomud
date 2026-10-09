package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/stance"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 69: weapon stances. A company member's stance is chosen between
// battles (modules/strategy); the round's tempo fill copies it onto the
// member's battle state, where internal/combat reads its effect. Only the
// player and company companions have one: an enemy never does. Runtime state
// only, on the game loop.

// applyStance sets a fighter's battle stance from the store, or clears it.
func applyStance(who caster, c *characters.Character) {
	if c.RT != nil && c.RT.StanceRead {
		return // read once a battle: set between battles only
	}
	leader, chosen, ok := chosenStance(who, c)
	if !ok {
		return
	}
	was := c.RTState().Stance
	c.RTState().Stance = chosen
	c.RT.StanceRead = true
	// Said once, as the battle's first round reads it, and only when the
	// stance can work: the log names what the member is doing.
	if chosen != was && stance.Fits(chosen, c.StanceGear()) {
		d, _ := stance.Lookup(chosen)
		name, verb := c.Name, "takes"
		if who.mobId == 0 {
			name, verb = "You", "take"
		}
		if u := users.GetByUserId(leader); u != nil {
			u.SendText(fmt.Sprintf(`%s %s the <ansi fg="yellow">%s</ansi> stance: %s, but %s.`, name, verb, d.Name, d.Gain, d.Cost))
		}
	}
}

// chosenStance is the stance the store holds for the member, with the
// leader it belongs to; ok is false when there is nothing to apply (not a
// company member, or no stance chosen and none set).
func chosenStance(who caster, c *characters.Character) (leader int, chosen stance.Stance, ok bool) {
	leader, key := who.userId, company.LeaderMemberKey
	if who.mobId > 0 {
		id, k, found := company.LeaderAndKeyForInstance(who.mobId)
		if !found {
			return 0, stance.None, false
		}
		leader, key = id, k
	}
	chosen = stance.For(leader, string(key))
	if chosen == stance.None && (c.RT == nil || c.RT.Stance == stance.None) {
		return leader, stance.None, false
	}
	return leader, chosen, true
}
