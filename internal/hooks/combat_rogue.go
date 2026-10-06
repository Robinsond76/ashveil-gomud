package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 38c2: the rogue elites (Pathfinder, Swordmaster, Nightblade). Like
// every class ability it is runtime only, on the game loop: marks, spent
// uses and the Shadowstep clock live on the character's class state for the
// battle and are never saved.

// poisonedBuff is the shipped poison buff (buffs/13-poisoned.yaml).
const poisonedBuff = 13

// shadowstepAbility keys the Shadowstep's cooldown among the abilities'.
const shadowstepAbility strategy.Ability = "shadowstep"

// abilityKind is the ability whose blow an actor is making this round
// (Opening Strike or Aimed Shot), so what lands after it can tell them apart.
var abilityKind = map[caster]strategy.Ability{}

// noteActed records that a fighter has acted in this combat round: a foe
// that has not is one a Pathfinder may still open.
func noteActed(c *characters.Character) {
	c.RTState().ActedRound = combatRound.Load()
}

// foeActed reports whether a foe has acted in the battle that began in
// startRound.
func foeActed(foe *mobs.Mob, startRound uint64) bool {
	rt := foe.Character.RT
	return rt != nil && rt.ActedRound != 0 && rt.ActedRound >= startRound
}

// pathOpens reports whether a Pathfinder's Eye lets its Opening Strike open
// this foe now: it has not acted yet this battle and the uses are not spent.
func pathOpens(a actor, foe *mobs.Mob, b battle.Battle) bool {
	n := a.char.ClassEffects().Int(classes.PathOpens)
	if n <= 0 || foeActed(foe, b.StartRound) {
		return false
	}
	return a.char.RT == nil || a.char.RT.OpensUsed < n
}

// companyActor finds the company member a holder is, with its leader and
// the room it fights in.
func companyActor(h statusHolder) (actor, *users.UserRecord, *rooms.Room, bool) {
	var u *users.UserRecord
	switch {
	case h.user != nil:
		u = h.user
	case h.mob != nil:
		if owner, _, ok := company.LeaderAndKeyForInstance(h.mob.InstanceId); ok {
			u = users.GetByUserId(owner)
		}
	}
	room := rooms.LoadRoom(h.roomId)
	if u == nil || u.Character == nil || room == nil {
		return actor{}, nil, nil, false
	}
	for _, a := range sideActors(u, room) {
		if a.char == h.char {
			return a, u, room, true
		}
	}
	return actor{}, nil, nil, false
}

// elitePass runs at a round's start, before the abilities: it clears what
// lasted a turn (Vanish, the Shadowstep, the Overwatch hold), marks and
// re-marks a Nightblade's victim, and starts a Shadowstep that needs it.
func elitePass() {
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
		foes := map[int]bool{}
		for _, id := range standingFoes(enemyparty.Group{Party: p}, room) {
			foes[id] = true
		}
		side := sideActors(u, room)
		round := combatRound.Load()
		markFoes(side, p)
		for _, a := range side {
			rt := a.char.RT
			if rt == nil || a.char.Health < 1 {
				continue
			}
			rt.Stepping = false
			rt.Watching, rt.WatchFired = 0, 0
			if rt.VanishEvade > 0 && round > rt.VanishRound+1 {
				rt.VanishEvade = 0 // Vanish lasts the rest of that round and the next
			}
			fx := a.char.ClassEffects()
			if fx.Has(classes.DeathMark) {
				deathMark(a, u, room, foes)
			}
			if fx.Has(classes.Shadowstep) {
				shadowstep(a, u, room, foes, p)
			}
		}
	}
}

// markFoes tells each foe of a battle what the blow code can't see from the
// defender alone: whether it is a boss (Coup de Grace) and whether it stands
// in its group's back row (a Marksman's Back-line eye). It runs only while a
// company member has a use for it.
func markFoes(side []actor, p mobparty.Party) {
	need := false
	for _, a := range side {
		if fx := a.char.ClassEffects(); fx.Has(classes.Coup) || fx.Has(classes.BackAttack) {
			need = true
		}
	}
	if !need {
		return
	}
	last := 0
	for _, id := range p.Members {
		if row, _, ok := p.Formation.Find(mobparty.MemberKeyFor(id)); ok {
			last = max(last, row)
		}
	}
	for _, id := range p.Members {
		m := mobs.GetInstance(id)
		if m == nil {
			continue
		}
		rt := m.Character.RTState()
		rt.Boss = m.Boss
		row, _, ok := p.Formation.Find(mobparty.MemberKeyFor(id))
		rt.BackRow = ok && last > 0 && row == last
	}
}

// deathMark marks a Nightblade's first target of the battle, and passes the
// mark to the most hurt foe in reach when its foe falls.
func deathMark(a actor, u *users.UserRecord, room *rooms.Room, foes map[int]bool) {
	rt := a.char.RTState()
	var foe *mobs.Mob
	if rt.MarkSet {
		cur := mobs.GetInstance(rt.MarkedFoe)
		if cur != nil && cur.Character.Health >= 1 && foes[cur.InstanceId] && !cur.Character.CombatWithdrawn {
			return // the mark holds
		}
		best := -1.0
		for id := range foes {
			m := mobs.GetInstance(id)
			if m == nil || m.Character.Health < 1 || m.Character.CombatWithdrawn || id == rt.MarkedFoe {
				continue
			}
			if reached, _ := abilityReach(a, u, m, room); !reached {
				continue
			}
			frac := float64(m.Character.Health) / float64(max(1, m.Character.HealthMax.Value))
			if foe == nil || frac < best || frac == best && id < foe.InstanceId {
				foe, best = m, frac
			}
		}
		if foe == nil {
			rt.DeathMark, rt.MarkedFoe = nil, 0
			return
		}
	} else if agg := a.char.Aggro; plainAttack(agg) && agg.MobInstanceId > 0 && foes[agg.MobInstanceId] {
		foe = mobs.GetInstance(agg.MobInstanceId)
	}
	if foe == nil || foe.Character.Health < 1 {
		return
	}
	rt.MarkSet, rt.MarkedFoe, rt.DeathMark = true, foe.InstanceId, foe.Character.RTState()
	target := mobHolder(foe)
	pct := a.char.ClassEffects().Int(classes.DeathMark)
	a.holder.say(fmt.Sprintf("You mark %s for death.", target.tag()),
		"%s marks "+verbatim(target.tag())+" for death.", fmt.Sprintf(" (death mark: +%d%% damage from you)", pct))
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: a.char.RoomId, Source: a.ref, Target: target.ref, Status: "Death Mark", Outcome: combatstream.OutcomeSucceeded})
}

// shadowstep lets a Nightblade strike its marked foe in the middle row as if
// it had extended reach, once every few rounds. Guardians still intercept:
// only the reach changes, never who the blow lands on.
func shadowstep(a actor, u *users.UserRecord, room *rooms.Room, foes map[int]bool, p mobparty.Party) {
	rt := a.char.RT
	every := a.char.ClassEffects().Int(classes.Shadowstep)
	round := combatRound.Load()
	if rt == nil || rt.MarkedFoe == 0 || rt.StepRound != 0 && round < rt.StepRound+uint64(every) {
		return
	}
	agg := a.char.Aggro
	if !plainAttack(agg) || agg.MobInstanceId != rt.MarkedFoe || !foes[agg.MobInstanceId] || tempoActive && tempoTurns[a.who] == 0 {
		return
	}
	foe := mobs.GetInstance(rt.MarkedFoe)
	if foe == nil {
		return
	}
	if row, _, ok := p.Formation.Find(mobparty.MemberKeyFor(foe.InstanceId)); !ok || row != 1 {
		return
	}
	if reached, _ := abilityReach(a, u, foe, room); reached {
		return // no step needed
	}
	rt.Stepping = true
	if reached, _ := abilityReach(a, u, foe, room); !reached {
		rt.Stepping = false
		return
	}
	rt.StepRound = round
	target := mobHolder(foe)
	a.holder.say(fmt.Sprintf("You slip between the shadows and strike at %s.", target.tag()),
		"%s slips between the shadows and strikes at "+verbatim(target.tag())+".", " (shadowstep)")
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: a.char.RoomId, Source: a.ref, Target: target.ref, Status: "Shadowstep", Outcome: combatstream.OutcomeSucceeded})
}

// endShadowsteps puts every reach back after a round's blows.
func endShadowsteps() {
	for _, uid := range battle.Players() {
		if u := users.GetByUserId(uid); u != nil && u.Character != nil {
			if room := rooms.LoadRoom(u.Character.RoomId); room != nil {
				for _, a := range sideActors(u, room) {
					if a.char.RT != nil {
						a.char.RT.Stepping = false
					}
				}
			}
		}
	}
}

// eliteBlow is what a resolved blow does for the rogue and ranger elites,
// on either side of it: the attacker's riders (an opening's exposure, a
// pinning shot, Rend, poison, Hunt Down, the kills that quicken or terrify),
// and the defender's Vanish.
func eliteBlow(attacker, defender statusHolder, r combat.AttackResult) {
	if rt := attacker.char.RT; rt != nil {
		rt.Spoil = 0 // a spoiled blow is spent
	}
	if attacker.mob != nil {
		noteActed(attacker.char)
	}
	if !r.Hit || r.DamageToTarget <= 0 {
		return
	}
	if fx := defender.char.ClassEffects(); fx.Has(classes.Vanish) {
		vanish(defender, fx.Int(classes.Vanish))
	}
	fx := attacker.char.ClassEffects()
	if fx == nil || defender.mob == nil {
		return
	}
	who := caster{userId: attacker.ref.UserId, mobId: attacker.ref.MobInstanceId}
	kind := abilityKind[who]
	felled := defender.char.Health < 1
	if fx.Has(classes.ExposeWeak) && kind == strategy.OpeningStrike && !felled {
		events.AddToQueue(events.Buff{MobInstanceId: defender.mob.InstanceId, BuffId: status.Exposed, Source: `combat`, Triggers: 2})
		attacker.say(fmt.Sprintf("You leave %s open.", defender.tag()), "%s leaves "+verbatim(defender.tag())+" open.", " (expose weakness: exposed, 2 rounds)")
	}
	if n := fx.Int(classes.PinCrit); n > 0 && kind == strategy.AimedShot && !felled {
		events.AddToQueue(events.Buff{MobInstanceId: defender.mob.InstanceId, BuffId: status.Hobbled, Source: `combat`, Triggers: n})
		attacker.say(fmt.Sprintf("Your arrow pins %s's leg.", defender.tag()), "%s's arrow pins "+verbatim(defender.tag())+"'s leg.", fmt.Sprintf(" (pinning crit: hobbled, %d rounds)", n))
	}
	if fx.Has(classes.RendArmor) && r.CritLanded && !felled {
		events.AddToQueue(events.Buff{MobInstanceId: defender.mob.InstanceId, BuffId: status.ArmorBroken, Source: `combat`, Triggers: 2})
		attacker.say(fmt.Sprintf("You rend %s's armor.", defender.tag()), "%s rends "+verbatim(defender.tag())+"'s armor.", " (rend: armor broken, 2 rounds)")
	}
	if n := fx.Int(classes.Envenom); n > 0 && !felled && envenomRoll(100) < n && !defender.char.HasBuff(poisonedBuff) {
		events.AddToQueue(events.Buff{MobInstanceId: defender.mob.InstanceId, BuffId: poisonedBuff, Source: `combat`})
		attacker.say(fmt.Sprintf("Poison from your blade seeps into %s.", defender.tag()), "Poison from %s's blade seeps into "+verbatim(defender.tag())+".", " (envenom: poisoned)")
	}
	if fx.Has(classes.HuntDown) {
		huntDown(attacker, defender, fx)
	}
	if felled {
		onFell(attacker, defender, fx, kind, who)
	}
}

// envenomRoll rolls Envenom's chance; tests replace it.
var envenomRoll = abilityRoll

// UseEnvenomRollForTest replaces Envenom's dice until the returned restore
// is called.
func UseEnvenomRollForTest(roll func(int) int) (restore func()) {
	prev := envenomRoll
	envenomRoll = roll
	return func() { envenomRoll = prev }
}

// vanish gives a struck Pathfinder its Evasion once a battle, below 30% of
// its health.
func vanish(h statusHolder, evade int) {
	rt := h.char.RT
	if rt == nil || rt.Vanished || h.char.Health < 1 || h.char.Health*100 >= 30*max(1, h.char.HealthLimit()) {
		return
	}
	rt.Vanished, rt.VanishEvade, rt.VanishRound = true, evade, combatRound.Load()
	h.say("You slip out of sight.", "%s slips out of sight.", fmt.Sprintf(" (vanish: +%d Evasion until your next turn)", evade))
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: h.char.RoomId, Source: h.ref, Status: "Vanish", Outcome: combatstream.OutcomeSucceeded})
}

// onFell is what a kill by a rogue or ranger elite does: Killing Spree
// quickens a Nightblade that fells its marked foe, Apex frightens the felled
// foe's group, and Second Nock sends a Marksman's shot after another target.
func onFell(attacker, defender statusHolder, fx classes.Effects, kind strategy.Ability, who caster) {
	rt := attacker.char.RTState()
	round := combatRound.Load()
	if n := fx.Int(classes.Spree); n > 0 && rt.MarkedFoe == defender.mob.InstanceId && rt.SpreeRound != round {
		if pushMeter(who, n) {
			rt.SpreeRound = round
			attacker.say("The kill feeds your hunger for the next.", "The kill feeds %s's hunger for the next.", fmt.Sprintf(" (killing spree: +%d action meter)", n))
			emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: attacker.char.RoomId, Source: attacker.ref, Status: "Killing Spree", Outcome: combatstream.OutcomeSucceeded})
		}
	}
	if fx.Has(classes.Apex) {
		apex(attacker, defender)
	}
	if fx.Has(classes.SecondNock) && kind == strategy.AimedShot && rt.NockRound != round {
		secondNock(attacker, defender, rt, round)
	}
}
