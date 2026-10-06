package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 39h: the Arbalist. A Piercing Bolt is the whole turn: one heavy blow
// that ignores part of the target's armor, and the turn after it is spent
// winding the crossbow. All of it is runtime battle state on the ClassRT, on
// the game loop, gone with the fight.

// reloadTurn spends an Arbalist's turn winding its crossbow when its last
// bolt left it unloaded. It reports whether the turn is spent. A turn the
// action meter gives it none is not spent (the winding waits for a real one).
func reloadTurn(a actor) bool {
	rt := a.char.RT
	if rt == nil || !rt.Reload || a.char.Health < 1 {
		return false
	}
	if tempoActive && tempoTurns[a.who] == 0 {
		return false
	}
	rt.Reload = false
	abilityTurns[a.who] = true
	a.holder.say("You wind the crossbow for the next bolt.", "%s winds the crossbow for the next bolt.", " (reloading)")
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: a.char.RoomId, Source: a.ref, Status: "Reload", Outcome: combatstream.OutcomeSucceeded})
	return true
}

// useBolt is a Piercing Bolt: the whole turn, one blow at the foe (or at the
// guardian that steps in for it) at the bolt's share of a shot, with the
// rank's armor piercing and Steady Aim, then the winding. A landed bolt can
// hobble the foe and wear its armor down, by the Arbalist's ranks.
func useBolt(a actor, u *users.UserRecord, foe *mobs.Mob, room *rooms.Room) {
	abilityTurns[a.who] = true
	fx := a.char.ClassEffects()
	rt := a.char.RTState()
	target, _ := guardedTarget(u.UserId, room, foe, true)
	if target == nil {
		target = foe
	}
	a.holder.say(fmt.Sprintf(`You set a heavy bolt and loose it at %s.`, mobHolder(foe).tag()),
		`%s looses a heavy bolt at `+verbatim(mobHolder(foe).tag())+`.`, ` (piercing bolt)`)
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: room.RoomId, Source: a.ref, Target: mobRef(target), Status: `Piercing Bolt`})

	atk := steadyAim(rt, fx)
	rt.AimStruck = false
	a.char.Aura.Attack += atk
	rt.BlowPierce = fx.Int(classes.BoltPierce)
	r := extraBlow(a, target, room, strategy.BoltBlowPct+fx.Int(classes.BoltDmg))
	rt.BlowPierce = 0
	a.char.Aura.Attack -= atk

	// The winding: the next turn, unless this is the first bolt of the battle
	// and the Arbalist has practiced its loading.
	if rt.BoltFired || !fx.Has(classes.FirstLoaded) {
		rt.Reload = true
	}
	rt.BoltFired = true

	if !r.Hit || r.DamageToTarget < 1 || target.Character.Health < 1 {
		return
	}
	tag := mobHolder(target).tag()
	if n := fx.Int(classes.BoltCripple); n > 0 && !status.Live(&target.Character, status.Hobbled) {
		events.AddToQueue(events.Buff{MobInstanceId: target.InstanceId, BuffId: status.Hobbled, Source: `combat`, Triggers: n})
		a.holder.say(fmt.Sprintf(`The bolt cripples %s.`, tag), `The bolt cripples `+verbatim(tag)+`.`, ` (hobbled)`)
	}
	if cut := fx.Int(classes.Shred); cut > 0 {
		frt := target.Character.RTState()
		if before := frt.Shred; before < fx.Int(classes.ShredCap) {
			frt.Shred = min(before+cut, fx.Int(classes.ShredCap))
			a.holder.say(fmt.Sprintf(`The bolt tears at %s's armor.`, tag), `The bolt tears at `+verbatim(tag)+`'s armor.`, ` (armor worn)`)
		}
	}
}

// steadyAim is the Attack Steady Aim gives the next bolt: nothing has hurt
// the shooter since its last one.
func steadyAim(rt *characters.ClassRT, fx classes.Effects) int {
	if rt.AimStruck {
		return 0
	}
	return fx.Int(classes.SteadyAim)
}

// arbalistBlow notes a blow that lands on an Arbalist holding Steady Aim: it
// is struck, so its next bolt gets no bonus.
func arbalistBlow(defender statusHolder, r combat.AttackResult) {
	if !r.Hit || r.DamageToTarget <= 0 || defender.char.RT == nil {
		return
	}
	if defender.char.ClassEffects().Has(classes.SteadyAim) {
		defender.char.RT.AimStruck = true
	}
}
