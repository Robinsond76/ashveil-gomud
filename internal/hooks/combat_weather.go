package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/stormcraft"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 39c: a Shaman's battle weather. The weather is battle state
// (internal/battle) and the statuses it puts on foes (Fogbound, Windchilled)
// do the rest through the combat round; this file decides when a weather is
// worth calling and tells the room when one passes.

// shoots reports whether a foe is a threat to a fog or a chill: it fires a
// ranged weapon, or it is a spell-caster or healer of its group.
func shoots(id int) bool {
	m := mobs.GetInstance(id)
	if m == nil || m.Character.Health < 1 {
		return false
	}
	if m.Character.Equipment.Weapon.GetSpec().Subtype == items.Shooting {
		return true
	}
	switch strategy.Role(m.EnemyRole()) {
	case strategy.Caster, strategy.Healer, strategy.Controller:
		return true
	}
	return false
}

// weatherReady is strategy.Situation.CanWeather for a caster: Fog and Chill
// Wind are worth their mana only against a foe that shoots or casts, and
// Rain only for a caster that has Lightning to feed on it.
func weatherReady(a actor, foes []int) func(string) bool {
	return func(spellId string) bool {
		kind, ok := stormcraft.KindOf(spellId)
		if !ok {
			return false
		}
		if kind == stormcraft.Rain {
			return a.knows("lightning")
		}
		for _, id := range foes {
			if shoots(id) {
				return true
			}
		}
		return false
	}
}

// weatherPass counts a round off every battle's weather, and tells the
// room's company when one passes. Called once per combat round.
func weatherPass() {
	for _, end := range battle.TickWeather() {
		u := users.GetByUserId(end.UserId)
		if u == nil || u.Character == nil {
			continue
		}
		if room := rooms.LoadRoom(u.Character.RoomId); room != nil {
			room.SendText(end.Kind.EndLine())
		}
	}
}
