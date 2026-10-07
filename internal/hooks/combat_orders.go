package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/flasks"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/orders"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 61: battle orders. Each round, before its role and target rule, a
// member reads its orders in order (internal/orders); the first whose
// condition holds and whose action it can carry out fires, and takes the
// member's turn (a hold only keeps its mana). The battle log notes it, "as
// ordered". Runtime state only, on the game loop.

// orderedWardKey is a member of a player's company holding a guard order.
type orderedWardKey struct {
	leader int
	guard  company.MemberKey
}

// orderedWards is, this round, whom each guard order guards. Rebuilt every
// round by strategyPass; guardianFor reads it.
var orderedWards = map[orderedWardKey]company.MemberKey{}

// orderStreak is the last round a member's guard or hold order fired, and
// for whom, so an order that keeps holding is noted once, not every round.
var orderStreak = map[caster]orderStreakEntry{}

type orderStreakEntry struct {
	round uint64
	fight uint64
	tag   string
}

// fightOpened is the round each battle's orders were first read: a battle
// that begins at a round's end (the next group stepping up) has its first
// orders round after its StartRound, so "first" is the first round read,
// not StartRound (61 review).
var fightOpened = map[uint64]uint64{}

// firstOrdersRound is whether this is the first round the battle's orders
// are read.
func firstOrdersRound(fightID uint64) bool {
	round := combatRound.Load()
	at, seen := fightOpened[fightID]
	if !seen {
		fightOpened[fightID], at = round, round
	}
	return at == round
}

// orderedWard is whom a guard order has the member guard this round.
func orderedWard(leader int, guard company.MemberKey) (company.MemberKey, bool) {
	w, ok := orderedWards[orderedWardKey{leader, guard}]
	return w, ok
}

// orderTurn is what a member's orders came to this round.
type orderTurn struct {
	handled bool // an order took the member's turn
	hold    bool // a hold order: no attack spell this round
}

// orderCtx is the round's battle as one member's orders see it.
type orderCtx struct {
	a          actor
	u          *users.UserRecord
	side       []actor
	allies     []strategy.Ally
	autoSpells []strategy.Spell
	g          enemyparty.Group
	foes       []int
	room       *rooms.Room
	b          battle.Battle
	// meleeFirst is whether the member's role fights with weapon or fists
	// (a fighter or guardian), so a break order strikes rather than casts.
	meleeFirst bool
	// first is whether this is the battle's first round of orders.
	first bool
}

func (c orderCtx) caster() orders.Caster {
	return orders.Caster{Spells: classCosted(c.a.char, c.autoSpells), Knows: c.a.knows, Mana: c.a.char.Mana, Flasks: flasks.Remaining(c.a.char)}
}

// runOrders reads one member's orders and carries out the one that fires.
func runOrders(c orderCtx, list []orders.Order) orderTurn {
	foeList := enemyparty.Foes(c.g, c.a.att)
	reach := map[int]bool{}
	snap := orders.Snapshot{FirstRound: c.first}
	for _, f := range foeList {
		reach[f.ID] = f.Reachable
		boss := false
		if m := mobs.GetInstance(f.ID); m != nil {
			boss = m.Boss
		}
		snap.Foes = append(snap.Foes, orders.FoeInfo{ID: f.ID, Chanting: f.Chanting, Caster: f.Caster, Healer: f.Healer, Boss: boss})
	}
	for i, al := range c.allies {
		snap.Allies = append(snap.Allies, orders.Ally{HP: al.HP, MaxHP: al.MaxHP, Down: al.Downed, Pending: al.Pending, Self: c.side[i].who == c.a.who})
	}
	sp := c.caster()
	// plan is how a fired order would be carried out; usable and the
	// execution read the same one, so an order never fires and does nothing.
	plan := func(f orders.Fire) (act strategy.Action, melee, ok bool) {
		switch f.Order.Do {
		case orders.Heal:
			al := snap.Allies[f.Ally]
			spell, ok := sp.Heal(orderShare(al))
			return strategy.Action{Kind: strategy.Heal, Spell: spell.ID, Ally: f.Ally}, false, ok
		case orders.Guard:
			// Only with a guard left and the ward within guarding ground
			// (61 review), so a guard that cannot happen gives way.
			battle.CaptureGuards(c.u.UserId, string(c.a.key), c.a.char.Level)
			if battle.GuardsLeft(c.u.UserId, string(c.a.key)) < 1 || !ableToGuard(guardMember{key: c.a.key, char: c.a.char}) {
				return strategy.Action{}, false, false
			}
			form, _ := company.FormationFor(c.u.UserId)
			return strategy.Action{}, false, formationcombat.GuardGround(form, c.a.key, c.side[f.Ally].key, enemyparty.Narrow(c.room))
		case orders.Hold:
			// A fighter casts nothing by its role, so a hold changes nothing.
			return strategy.Action{}, false, !c.meleeFirst && sp.KnowsAttack()
		case orders.Strongest:
			spell, kind, ok := sp.Attack(len(c.foes), false)
			return strategy.Action{Kind: kind, Spell: spell.ID}, false, ok
		case orders.Break:
			canMelee := reach[f.Foe] && retargetable(c.a.char.Aggro) && !aimedAt(c.a.char.Aggro, f.Foe)
			if c.meleeFirst && canMelee {
				return strategy.Action{}, true, true
			}
			if spell, kind, ok := sp.Attack(len(c.foes), true); ok && !aimedByCast(c.a.char, f.Foe) {
				return strategy.Action{Kind: kind, Spell: spell.ID}, false, true
			}
			// A healer or caster with no attack spell to send does not
			// give up its turn to a swing (61 review).
			return strategy.Action{}, false, false
		}
		return strategy.Action{}, false, false
	}
	fire, ok := orders.Evaluate(list, snap, func(f orders.Fire) bool { _, _, ok := plan(f); return ok })
	if !ok {
		return orderTurn{}
	}
	act, melee, _ := plan(fire)
	switch fire.Order.Do {
	case orders.Hold:
		c.noteOnce(fire, "hold", nil)
		return orderTurn{hold: true}
	case orders.Guard:
		ward := c.side[fire.Ally].key
		orderedWards[orderedWardKey{c.u.UserId, c.a.key}] = ward
		c.noteOnce(fire, "guard:"+string(ward), &c.side[fire.Ally])
		return orderTurn{handled: false}
	case orders.Heal, orders.Strongest:
		info, ok := autoSpellTargets(act, c.a, c.side, c.g, c.foes)
		if !ok || !startCast(c.a, act.Spell, info, c.room.RoomId) {
			return orderTurn{}
		}
		if act.Kind == strategy.Heal {
			coverHeal(c.allies, act)
			c.note(fire, &c.side[fire.Ally], 0)
		} else {
			c.note(fire, nil, 0)
		}
		return orderTurn{handled: true}
	case orders.Break:
		if melee {
			prev := 0
			if plainAttack(c.a.char.Aggro) {
				prev = c.a.char.Aggro.MobInstanceId
			}
			c.a.char.SetAggro(0, fire.Foe, attackType(c.a.char.Aggro))
			emitTargetChange(c.a.ref, mobRefById(prev), mobRefById(fire.Foe), c.room.RoomId)
			if c.a.who.userId > 0 {
				events.AddToQueue(events.AggroChanged{UserId: c.a.who.userId, RoomId: c.a.char.RoomId})
			} else {
				events.AddToQueue(events.AggroChanged{MobInstanceId: c.a.who.mobId, RoomId: c.a.char.RoomId})
			}
			c.note(fire, nil, fire.Foe)
			return orderTurn{handled: true}
		}
		info := characters.SpellAggroInfo{SpellId: act.Spell, TargetUserIds: []int{}, TargetMobInstanceIds: []int{fire.Foe}}
		if !startCast(c.a, act.Spell, info, c.room.RoomId) {
			return orderTurn{}
		}
		c.note(fire, nil, fire.Foe)
		return orderTurn{handled: true}
	}
	return orderTurn{}
}

// orderShare is an ally's health in thousandths.
func orderShare(a orders.Ally) int {
	return a.HP * 1000 / max(a.MaxHP, 1)
}

func aimedAt(a *characters.Aggro, foe int) bool {
	return plainAttack(a) && a.MobInstanceId == foe
}

// aimedByCast is whether the member is already chanting at the foe.
func aimedByCast(c *characters.Character, foe int) bool {
	return c.Aggro != nil && c.Aggro.Type == characters.SpellCast && len(c.Aggro.SpellInfo.TargetMobInstanceIds) == 1 && c.Aggro.SpellInfo.TargetMobInstanceIds[0] == foe
}

// noteOnce notes an order that holds over rounds (a guard, a hold) when it
// starts or changes, not again every round it keeps holding.
func (c orderCtx) noteOnce(f orders.Fire, tag string, subject *actor) {
	round := combatRound.Load()
	last, had := orderStreak[c.a.who]
	orderStreak[c.a.who] = orderStreakEntry{round: round, fight: c.b.FightID, tag: tag}
	if had && last.tag == tag && last.fight == c.b.FightID && last.round+1 >= round {
		return
	}
	c.note(f, subject, 0)
}

// note tells the battle log an order fired: a line for the player and the
// room, and an event on the combat stream.
func (c orderCtx) note(f orders.Fire, ally *actor, foe int) {
	a := c.a
	subject := combatstream.Ref{}
	view := func(leader bool) string {
		who := func(t actor) string {
			if t.who.userId > 0 {
				if leader {
					return `you`
				}
				return userTag(t.char.Name)
			}
			return named(mobTag(mobName(t.who.mobId)))
		}
		self := ""
		if a.who.userId > 0 {
			self = userTag(a.char.Name)
			if leader {
				self = `You`
			}
		} else {
			self = named(mobTag(mobName(a.who.mobId)))
		}
		verb := func(you, other string) string {
			if self == `You` {
				return you
			}
			return other
		}
		var body string
		switch f.Order.Do {
		case orders.Heal:
			if ally != nil && ally.who == a.who {
				body = verb(`look to your own wounds`, `looks to a wound of their own`)
			} else if ally != nil {
				body = verb(`tend `, `tends `) + who(*ally)
			}
		case orders.Break:
			target := named(mobTag(mobName(foe)))
			body = verb(`turn on `, `turns on `) + target
			if f.Order.When == orders.Chanting {
				body += ` to break its chant`
			}
		case orders.Guard:
			if ally != nil {
				body = verb(`move to guard `, `moves to guard `) + who(*ally)
			}
		case orders.Strongest:
			body = verb(`put everything into your next spell`, `puts everything into the next spell`)
		case orders.Hold:
			body = verb(`hold back your mana`, `holds back mana`)
		}
		return util.CapitalizeFirst(fmt.Sprintf(`%s %s, as ordered.`, self, body))
	}
	c.u.SendText(view(true))
	c.room.SendText(view(false), c.u.UserId)
	switch {
	case ally != nil:
		subject = ally.ref
	case foe > 0:
		subject = mobRefById(foe)
	}
	emitCombat(combatstream.Event{Kind: combatstream.OrderFired, FightID: c.b.FightID, RoomId: c.room.RoomId, Source: a.ref, Target: subject, Status: string(f.Order.Do)})
}

// pruneOrderStreaks forgets members whose guard or hold order has not
// fired for a few rounds, and battles that have ended.
func pruneOrderStreaks() {
	round := combatRound.Load()
	for who, e := range orderStreak {
		if e.round+3 < round {
			delete(orderStreak, who)
		}
	}
	live := map[uint64]bool{}
	for _, uid := range battle.Players() {
		if b, ok := battle.Current(uid); ok {
			live[b.FightID] = true
		}
	}
	for id := range fightOpened {
		if !live[id] {
			delete(fightOpened, id)
		}
	}
}
