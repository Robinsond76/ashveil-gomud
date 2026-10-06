package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 39f: the Gryphon Rider. Dive is the rider's whole turn: a stooping
// blow that passes a standing front-row foe to strike the middle or back
// row, from open sky only. All of it is runtime on the game loop, like every
// class ability; the cost in Evasion lives on the round's aura and is gone
// when the next round's auras are set.

// buffPoisoned is the shipped Poisoned buff (_datafiles/world/default/buffs).
const buffPoisoned = 13

// diveGround reports whether the rider can fly here: open sky, not indoors
// (caves are indoor biomes) and not on narrow ground.
func diveGround(room *rooms.Room) bool {
	return room != nil && !room.IsIndoor() && !enemyparty.Narrow(room)
}

// diveOpen reports whether a dive at foe is possible: the ground allows it
// and the foe stands in the rider's column or the next, in its group's
// formation. A rider outside any formation fails open, like a swing.
func diveOpen(a actor, u *users.UserRecord, foe *mobs.Mob, room *rooms.Room) bool {
	if !diveGround(room) || a.char.Health < 1 {
		return false
	}
	party, ok := enemyparty.PartyOf(room, foe.InstanceId)
	if !ok {
		return false
	}
	_, foeCol, found := party.Formation.Find(mobparty.MemberKeyFor(foe.InstanceId))
	if !found {
		return false
	}
	var col int
	var placed bool
	if a.who.userId > 0 {
		col, placed = resolvePlayerColumn(u.UserId)
	} else if f, ok := enemyparty.CompanyFormation(u.UserId); ok {
		_, col, placed = f.Find(a.key)
	}
	return !placed || formationcombat.InLateralRange(col, foeCol)
}

// useDive is a Dive: the whole turn, one blow at the foe (or at the guardian
// that steps in for it), at 100% of a blow's damage and the class's bonus,
// then the rider's Evasion cost. A landed blow can leave the foe bleeding,
// knocked down, exposed or poisoned, by the rider's ranks.
func useDive(a actor, u *users.UserRecord, foe *mobs.Mob, room *rooms.Room) {
	abilityTurns[a.who] = true
	fx := a.char.ClassEffects()
	target, _ := guardedTarget(u.UserId, room, foe, true)
	if target == nil {
		target = foe
	}
	a.holder.say(fmt.Sprintf(`You fold your wings and stoop on %s.`, mobHolder(foe).tag()),
		`%s stoops from the sky on `+verbatim(mobHolder(foe).tag())+`.`, ` (dive)`)
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: room.RoomId, Source: a.ref, Target: mobRef(target), Status: `Dive`})
	r := extraBlow(a, target, room, 100+fx.Int(classes.DiveDmg))
	if !fx.Has(classes.DiveSteady) {
		a.char.Aura.Evasion -= strategy.DiveEvasionCost
	}
	if !r.Hit || r.DamageToTarget < 1 || target.Character.Health < 1 {
		return
	}
	tag := mobHolder(target).tag()
	apply := func(id int, you, others, suffix string) {
		events.AddToQueue(events.Buff{MobInstanceId: target.InstanceId, BuffId: id, Source: `combat`})
		a.holder.say(fmt.Sprintf(you, tag), others, suffix)
	}
	if fx.Has(classes.Talons) && !status.Live(&target.Character, status.Bleeding) {
		apply(status.Bleeding, `Your talons rake %s, and it bleeds.`, `%s's talons rake `+verbatim(tag)+`.`, ` (bleeding)`)
	}
	if fx.Has(classes.DiveDown) && !status.Live(&target.Character, status.KnockedDown) && lanceHeld(a.char) {
		apply(status.KnockedDown, `The stoop knocks %s to the ground.`, `The stoop knocks `+verbatim(tag)+` to the ground.`, ` (knocked down)`)
	}
	if fx.Has(classes.DiveExpo) && !status.Live(&target.Character, status.Exposed) {
		apply(status.Exposed, `You mark %s, and it is left exposed.`, `%s marks `+verbatim(tag)+`, leaving it exposed.`, ` (exposed)`)
	}
	if fx.Has(classes.DivePois) {
		apply(buffPoisoned, `Your venom sinks into %s.`, `%s's venom sinks into `+verbatim(tag)+`.`, ` (poisoned)`)
	}
}

// lanceHeld reports whether the rider carries a two-handed reach weapon: a
// war spear, the lance of the design until a lance family ships.
func lanceHeld(c *characters.Character) bool {
	if c.Equipment.Weapon.ItemId == 0 {
		return false
	}
	spec := c.Equipment.Weapon.GetSpec()
	return spec.Reach && spec.Hands == items.TwoHanded
}
