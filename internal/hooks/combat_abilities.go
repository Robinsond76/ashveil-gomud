package hooks

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 33e: automatic class abilities. After the strategy pass has started
// this round's spells and before any blow, every player and company
// companion in a battle that will swing this round may use one ability
// instead (strategy.DecideAbility): a warrior tackles, a rogue strikes an
// opening, a ranger aims a shot. The ability is the member's turn: a tackle
// takes it whole (no swing), and an opening strike or aimed shot is the
// round's swing itself, its first blow that lands a critical hit (the
// backstab crit, through the shared hit, defense, armor, status, and
// wound rules).
//
// Everything here is runtime only, on the game loop. Cooldowns count combat
// rounds and are never saved: a restart or copyover, which ends every
// battle, leaves every ability ready.

// abilityKey is one actor's one ability.
type abilityKey struct {
	who caster
	id  strategy.Ability
}

var (
	// abilityRounds counts the combat rounds the pass has seen.
	abilityRounds int
	// abilityReady is the combat round from which an actor's ability is
	// ready again; an actor and ability not listed is ready.
	abilityReady = map[abilityKey]int{}
	// abilityTurns are the actors whose turn this round an ability took
	// (a tackle): they don't swing.
	abilityTurns = map[caster]bool{}
	// abilityStrikes are the actors whose swing this round is an opening
	// strike or aimed shot, set back to a plain attack after the blows.
	abilityStrikes = map[caster]bool{}
	// abilityRoll rolls a tackle; tests replace it.
	abilityRoll = util.Rand
)

// UseAbilityRollForTest replaces the tackle's dice until the returned
// restore is called.
func UseAbilityRollForTest(roll func(int) int) (restore func()) {
	prev := abilityRoll
	abilityRoll = roll
	return func() { abilityRoll = prev }
}

// ResetAbilitiesForTest forgets every cooldown.
func ResetAbilitiesForTest() {
	clear(abilityReady)
	clear(abilityTurns)
	clear(abilityStrikes)
}

// abilityPass lets every member of every battle that is about to swing use
// an ability, by its strategy.
func abilityPass() {
	abilityRounds++
	clear(abilityTurns)
	clear(abilityStrikes)
	for k, round := range abilityReady {
		if round <= abilityRounds {
			delete(abilityReady, k)
		}
	}
	for _, uid := range battle.Players() {
		b, ok := battle.Current(uid)
		u := users.GetByUserId(uid)
		if !ok || u == nil || u.Character == nil || u.Character.RoomId != b.RoomId {
			continue
		}
		room := rooms.LoadRoom(b.RoomId)
		if room == nil {
			continue
		}
		p, found := battleParty(b, enemyparty.Parties(room))
		if !found {
			continue
		}
		// 33e review: a company preparing to withdraw uses no abilities
		// (its members only hold the line until it goes).
		if a := u.Character.Aggro; a != nil && a.Type == characters.Retreat {
			continue
		}
		foes := map[int]bool{}
		for _, id := range standingFoes(enemyparty.Group{Party: p}, room) {
			foes[id] = true
		}
		for _, a := range sideActors(u, room) {
			if surprised(a.who.userId, a.who.mobId) {
				continue
			}
			foe, ok := abilityFoe(a, u, foes)
			if !ok {
				continue
			}
			// 33e review: the formation decides, as for a swing: a foe
			// the front row would shield is not reached by an ability,
			// and a tackle needs the foe within hand-to-hand reach.
			reached, close := abilityReach(a, u, foe, room)
			if !reached {
				continue
			}
			sit := abilitySituation(a, u, foe)
			sit.Close = close
			id, use := strategy.DecideAbility(sit)
			if !use {
				continue
			}
			useAbility(a, foe, id, room)
		}
	}
}

// abilityFoe is the foe an actor would swing at this round, when it may
// use an ability at all: standing (a downed player can't), ready to act,
// its weapon not still waiting, and aimed by a plain attack at a standing
// foe of its own battle.
func abilityFoe(a actor, u *users.UserRecord, foes map[int]bool) (*mobs.Mob, bool) {
	if a.char.Health < 1 || !readyToCast(a, u) {
		return nil, false
	}
	agg := a.char.Aggro
	if !plainAttack(agg) || agg.MobInstanceId <= 0 || !foes[agg.MobInstanceId] || agg.RoundsWaiting > 0 {
		return nil, false
	}
	// A foe that surrendered (withdrawn) is no longer fought.
	foe := mobs.GetInstance(agg.MobInstanceId)
	return foe, foe != nil && !foe.Character.CombatWithdrawn
}

// abilityReach reports whether an actor's blow at foe would land on foe
// itself, by the same formation rules as its swing (no front-row foe takes
// it instead), and whether it would with no reach at all (hand to hand,
// for a tackle). An attacker or foe outside any formation fails open, as
// the swing does.
func abilityReach(a actor, u *users.UserRecord, foe *mobs.Mob, room *rooms.Room) (reached, close bool) {
	var col int
	var placed bool
	innate := false
	if a.who.userId > 0 {
		col, placed = resolvePlayerColumn(u.UserId)
	} else {
		if f, ok := enemyparty.CompanyFormation(u.UserId); ok {
			_, col, placed = f.Find(a.key)
		}
		if a.holder.mob != nil {
			innate = a.holder.mob.Reach
		}
	}
	if !placed {
		return true, true
	}
	lands := func(reach formationcombat.Reach) bool {
		t, ok := resolveEnemyAttack(col, foe.InstanceId, room, reach)
		return ok && t != nil && t.InstanceId == foe.InstanceId
	}
	return lands(combat.ResolveReach(a.char, innate)), lands(formationcombat.ReachNone)
}

// abilitySituation is what an actor's ability is chosen from.
func abilitySituation(a actor, u *users.UserRecord, foe *mobs.Mob) strategy.AbilitySituation {
	var known []strategy.Ability
	if a.who.userId > 0 {
		known = strategy.PlayerAbilities(a.char.GetSkillLevel)
	} else {
		known = strategy.CompanionAbilities(a.archetype)
	}
	weapon, backstab := wielding(a.char)
	return strategy.AbilitySituation{
		Known: known,
		Off:   enemyparty.MemberStrategy(u.UserId, a.key).NoAbilities,
		Ready: func(id strategy.Ability) bool {
			_, waiting := abilityReady[abilityKey{who: a.who, id: id}]
			return !waiting
		},
		Weapon:       weapon,
		Backstab:     backstab,
		FoeDown:      status.Live(&foe.Character, status.KnockedDown),
		FoeStunned:   status.Live(&foe.Character, status.Stunned),
		FoeStaggered: status.Live(&foe.Character, status.Staggered),
		FoeExposed:   status.Live(&foe.Character, status.Exposed),
	}
}

// wielding is what a character strikes with, and whether it wields a
// weapon and every weapon it wields can backstab (a blade or claws).
func wielding(c *characters.Character) (strategy.WeaponKind, bool) {
	if c.Equipment.Weapon.ItemId == 0 {
		return strategy.Unarmed, false
	}
	main := c.Equipment.Weapon.GetSpec()
	kind := strategy.Melee
	if main.Subtype == items.Shooting {
		kind = strategy.Shooting
	}
	backstab := items.CanBackstab(main.Subtype)
	if c.Equipment.Offhand.ItemId != 0 {
		if off := c.Equipment.Offhand.GetSpec(); off.Type == items.Weapon && !items.CanBackstab(off.Subtype) {
			backstab = false
		}
	}
	return kind, backstab
}

// useAbility carries out an ability: its cooldown, its line, its event,
// and its effect.
func useAbility(a actor, foe *mobs.Mob, id strategy.Ability, room *rooms.Room) {
	spec, ok := strategy.SpecOf(id)
	if !ok {
		return
	}
	abilityReady[abilityKey{who: a.who, id: id}] = abilityRounds + spec.Cooldown
	target := mobHolder(foe)
	event := combatstream.Event{Kind: combatstream.Ability, RoomId: room.RoomId, Source: a.ref, Target: target.ref, Status: spec.Name}
	switch id {
	case strategy.Tackle:
		abilityTurns[a.who] = true
		chance := strategy.TackleChance(a.char.Stats.Speed.ValueAdj, foe.Character.Stats.Perception.ValueAdj)
		roll := abilityRoll(100)
		util.LogRoll(`Tackle`, roll, chance)
		if roll >= chance {
			event.Outcome = combatstream.OutcomeFailed
			emitCombat(event)
			a.holder.say(fmt.Sprintf(`You lunge to tackle %s, and miss.`, target.tag()),
				`%s lunges to tackle `+verbatim(target.tag())+`, and misses.`, ` (tackle missed)`)
			return
		}
		event.Outcome = combatstream.OutcomeSucceeded
		emitCombat(event)
		a.holder.say(fmt.Sprintf(`You tackle %s to the ground.`, target.tag()),
			`%s tackles `+verbatim(target.tag())+` to the ground.`, ` (knocked down)`)
		foe.AddBuff(status.KnockedDown, `combat`)
		// A tackle is heavy force: it breaks a chant (an enemy starts
		// again) and a wind-up, as a knockdown blow would.
		if !interruptsOff && target.chanting() {
			breakChant(a.holder, target)
		}
		if WindingUp(foe.InstanceId) {
			breakWindUp(a.holder, foe)
		}
	case strategy.OpeningStrike, strategy.AimedShot:
		abilityStrikes[a.who] = true
		a.char.Aggro.Type = characters.BackStab
		emitCombat(event)
		if id == strategy.OpeningStrike {
			a.holder.say(fmt.Sprintf(`You see an opening on %s.`, target.tag()),
				`%s sees an opening on `+verbatim(target.tag())+`.`, ` (opening strike)`)
		} else {
			a.holder.say(fmt.Sprintf(`You take careful aim at %s.`, target.tag()),
				`%s takes careful aim at `+verbatim(target.tag())+`.`, ` (aimed shot)`)
		}
	}
}

// verbatim escapes text for a say template, so it prints as it is.
func verbatim(s string) string { return strings.ReplaceAll(s, "%", "%%") }

// endAbilityStrikes sets a readied strike that never landed (its foe fell
// first, or its turn was lost) back to a plain attack, after the round's
// blows.
func endAbilityStrikes() {
	for who := range abilityStrikes {
		var c *characters.Character
		if who.userId > 0 {
			if u := users.GetByUserId(who.userId); u != nil {
				c = u.Character
			}
		} else if m := mobs.GetInstance(who.mobId); m != nil {
			c = &m.Character
		}
		if c == nil || c.Aggro == nil || c.Aggro.Type != characters.BackStab {
			continue
		}
		c.Aggro.Type = characters.DefaultAttack
		if c.Equipment.Weapon.GetSpec().Subtype == items.Shooting {
			c.Aggro.Type = characters.Shooting
		}
	}
	clear(abilityStrikes)
}
