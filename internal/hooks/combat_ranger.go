package hooks

import (
	"fmt"
	"slices"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/morale"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 38c2: the ranger elites (Sentinel, Marksman, Ravager). Runtime only,
// on the game loop: holds, spoiled blows, marks and spent uses live on the
// class state for the battle and are never saved.

// spoiled are the foes whose next blow an Overwatch shot spoiled; the round's
// start (and the blow itself) lifts it.
var spoiled []*characters.ClassRT

// ResetEliteForTest forgets every runtime mark of the rogue and ranger
// elites.
func ResetEliteForTest() {
	clear(spoiled)
	spoiled = nil
	clear(abilityKind)
}

// eliteRoundStart lifts what lasts a round: a spoiled blow that was never
// struck.
func eliteRoundStart() {
	for _, rt := range spoiled {
		rt.Spoil = 0
	}
	clear(spoiled)
	spoiled = spoiled[:0]
}

// battleFoes are the standing foes of a leader's battle and their group.
func battleFoes(u *users.UserRecord, room *rooms.Room) (map[int]bool, bool) {
	b, ok := battle.Current(u.UserId)
	if !ok || b.RoomId != room.RoomId {
		return nil, false
	}
	p, found := battleParty(b, enemyparty.Parties(room))
	if !found {
		return nil, false
	}
	foes := map[int]bool{}
	for _, id := range standingFoes(enemyparty.Group{Party: p}, room) {
		foes[id] = true
	}
	return foes, true
}

// ---- Sentinel -------------------------------------------------------------

// watchThreat reports whether a standing foe could strike an ally of the
// actor's middle or back row now: by the formation's own rules it gets
// through to a member behind the front row.
func watchThreat(a actor, u *users.UserRecord, f company.Formation, foes map[int]bool, room *rooms.Room) bool {
	alive := enemyparty.CompanyAlive(u.UserId, f)
	for id := range foes {
		m := mobs.GetInstance(id)
		if m == nil || m.Character.Health < 1 || m.Character.CombatWithdrawn {
			continue
		}
		// A foe aimed at a middle- or back-row ally is about to go for it,
		// whether or not the front row would take the blow.
		if key, ok := aimedMember(u, m); ok && key != a.key {
			if row, _, placed := f.Find(key); placed && row >= 1 {
				return true
			}
		}
		col, ok := resolveHostileAttackerColumn(room, id)
		if !ok {
			continue
		}
		reach := combat.ResolveReach(&m.Character, m.Reach)
		for r := 1; r < company.FormationRows; r++ {
			for c := 0; c < company.FormationCols; c++ {
				key := f.At(r, c)
				if key == "" || key == a.key || !alive[key] {
					continue
				}
				if final, ok := resolveAttackTarget(col, f, key, alive, reach, groundForMob(m, u.UserId)); ok {
					if row, _, placed := f.Find(final); placed && row >= 1 && final != a.key {
						return true
					}
				}
			}
		}
	}
	return false
}

// aimedMember is the member of a leader's company a foe is aimed at.
func aimedMember(u *users.UserRecord, foe *mobs.Mob) (company.MemberKey, bool) {
	agg := foe.Character.Aggro
	if agg == nil {
		return "", false
	}
	if agg.UserId == u.UserId {
		return company.LeaderMemberKey, true
	}
	if agg.MobInstanceId > 0 {
		if owner, key, ok := company.LeaderAndKeyForInstance(agg.MobInstanceId); ok && owner == u.UserId {
			return key, true
		}
	}
	return "", false
}

// useOverwatch is the Sentinel's hold: its whole turn, its shots ready for
// the first foes that strike a middle- or back-row ally, or begin a chant.
func useOverwatch(a actor, room *rooms.Room, u *users.UserRecord, foes map[int]bool) {

	abilityTurns[a.who] = true
	rt := a.char.RTState()
	rt.Watching = max(1, a.char.ClassEffects().Int(classes.OverwatchMax))
	a.holder.say("You hold your arrow ready and watch the line.", "%s holds an arrow ready and watches the line.", " (overwatch)")
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: room.RoomId, Source: a.ref, Status: "Overwatch", Outcome: combatstream.OutcomeSucceeded})
	// A chant already begun this round is answered at once.
	if a.char.ClassEffects().Has(classes.OverwatchCh) {
		for id := range foes {
			if m := mobs.GetInstance(id); m != nil {
				answerChant(m, a, u, room)
			}
		}
	}
}

// looseHeldShots looses a Sentinel's held arrow at its own foe once the
// round's first blows are struck, when no foe gave it a shot (Phase 38c2
// review): a quiet hold costs the Sentinel its Aimed Shot, never its blow.
func looseHeldShots() {
	for _, uid := range battle.Players() {
		u := users.GetByUserId(uid)
		if u == nil || u.Character == nil {
			continue
		}
		room := rooms.LoadRoom(u.Character.RoomId)
		if room == nil {
			continue
		}
		foes, ok := battleFoes(u, room)
		if !ok {
			continue
		}
		for _, a := range sideActors(u, room) {
			rt := a.char.RT
			if rt == nil || rt.Watching < 1 {
				continue
			}
			fired := rt.WatchFired > 0
			rt.Watching = 0
			if fired || a.char.Health < 1 || status.Grounded(a.char) || a.char.HasBuffFlag("no-combat") {
				continue
			}
			agg := a.char.Aggro
			if agg == nil || agg.MobInstanceId == 0 || !foes[agg.MobInstanceId] {
				continue
			}
			foe := mobs.GetInstance(agg.MobInstanceId)
			if foe == nil || foe.Character.Health < 1 || foe.Character.CombatWithdrawn {
				continue
			}
			if reached, _ := abilityReach(a, u, foe, room); !reached {
				continue
			}
			target := mobHolder(foe)
			a.holder.say(fmt.Sprintf("No blow comes; you loose your held arrow at %s.", target.tag()),
				"%s looses a held arrow at "+verbatim(target.tag())+".", "")
			extraBlow(a, foe, room, 100)
		}
	}
}

// overwatchShot is one held shot at a foe: the Sentinel's weapon blow with
// the rank's Attack, through the same hit, defense, armor and wound rules as
// any blow.
func overwatchShot(a actor, foe *mobs.Mob, room *rooms.Room) combat.AttackResult {
	atk := a.char.ClassEffects().Int(classes.OverwatchAtk)
	a.char.Aura.Attack += atk
	defer func() { a.char.Aura.Attack -= atk }()
	return extraBlow(a, foe, room, 100)
}

// spendWatch uses one held shot of a Sentinel; a second shot in the round
// spends its next turn too.
func spendWatch(a actor) {
	rt := a.char.RTState()
	rt.Watching--
	if a.char.ClassEffects().Int(classes.OverwatchMax) >= 2 && rt.Watching == 0 && rt.WatchFired >= 1 {
		rt.WatchDebt = true
	}
	rt.WatchFired++
}

// overwatchBlow answers a foe about to strike a member of a player's
// company, before the blow resolves: the first Sentinel holding a shot that
// reaches the foe shoots it when the member struck stands in the middle or
// back row. It reports whether the blow is stopped (a Guardian Arrow, or the
// foe fell to the shot).
func overwatchBlow(leader *users.UserRecord, f company.Formation, struck company.MemberKey, mob *mobs.Mob, room *rooms.Room) bool {
	if row, _, ok := f.Find(struck); !ok || row < 1 {
		return false
	}
	for _, h := range sideActors(leader, rooms.LoadRoom(leader.Character.RoomId)) {
		rt := h.char.RT
		if rt == nil || rt.Watching < 1 || h.key == struck || h.char.Health < 1 || status.Grounded(h.char) || h.char.HasBuffFlag("no-combat") {
			continue
		}
		if reached, _ := abilityReach(h, leader, mob, room); !reached {
			continue
		}
		spendWatch(h)
		fx := h.char.ClassEffects()

		target := mobHolder(mob)
		h.holder.say(fmt.Sprintf("Your overwatch arrow flies at %s.", target.tag()),
			"%s's overwatch arrow flies at "+verbatim(target.tag())+".", "")
		r := overwatchShot(h, mob, room)
		emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: room.RoomId, Source: h.ref, Target: target.ref, Status: "Overwatch", Outcome: outcomeOf(r.Hit)})
		if mob.Character.Health < 1 {
			return true
		}
		if !r.Hit || r.DamageToTarget <= 0 {
			return false
		}
		if fx.Has(classes.GuardArrow) {
			h.holder.say(fmt.Sprintf("Your arrow stops %s's blow.", target.tag()),
				"%s's arrow stops "+verbatim(target.tag())+"'s blow.", " (guardian arrow)")
			return true
		}
		rt2 := mob.Character.RTState()
		rt2.Spoil = 50
		spoiled = append(spoiled, rt2)
		h.holder.say(fmt.Sprintf("Your arrow spoils %s's swing.", target.tag()),
			"%s's overwatch arrow spoils "+verbatim(target.tag())+"'s swing.", " (overwatch: the blow lands at half damage)")
		if fx.Has(classes.OverwatchDwn) && mob.Leap && !status.Live(&mob.Character, status.KnockedDown) {
			events.AddToQueue(events.Buff{MobInstanceId: mob.InstanceId, BuffId: status.KnockedDown, Source: `combat`})
			h.holder.say(fmt.Sprintf("The arrow pulls %s out of its leap.", target.tag()),
				"The arrow pulls "+verbatim(target.tag())+" out of its leap.", " (knocked down)")
		}
		return false
	}
	return false
}

func outcomeOf(ok bool) string {
	if ok {
		return combatstream.OutcomeSucceeded
	}
	return combatstream.OutcomeFailed
}

// answerChant lets a holding Sentinel shoot a foe that has begun a chant
// this round, and break it on a hit.
func answerChant(m *mobs.Mob, h actor, u *users.UserRecord, room *rooms.Room) {
	rt := h.char.RT
	foe := m.Character.RT
	if rt == nil || rt.Watching < 1 || foe == nil || foe.ChantRound != combatRound.Load() || foe.ChantAnswered || m.Character.Health < 1 {
		return
	}
	if agg := m.Character.Aggro; agg == nil || agg.Type != characters.SpellCast {
		return
	}
	if reached, _ := abilityReach(h, u, m, room); !reached {
		return
	}
	foe.ChantAnswered = true
	spendWatch(h)
	target := mobHolder(m)
	h.holder.say(fmt.Sprintf("Your overwatch arrow flies at %s as it begins to chant.", target.tag()),
		"%s's overwatch arrow flies at "+verbatim(target.tag())+" as it begins to chant.", "")
	r := overwatchShot(h, m, room)
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: room.RoomId, Source: h.ref, Target: target.ref, Status: "Overwatch", Outcome: outcomeOf(r.Hit)})
	if r.Hit && r.DamageToTarget > 0 && m.Character.Health >= 1 && m.Character.Aggro != nil && m.Character.Aggro.Type == characters.SpellCast {
		breakChant(h.holder, target)
	}
}

// chantBegun is called when an enemy starts a chant: any Sentinel holding
// its shot now may answer it. Holds end with the round's first blows
// (looseHeldShots), so in practice a chant begun before the abilities is
// answered by useOverwatch's own scan.
func chantBegun(m *mobs.Mob) {
	if _, _, companion := company.LeaderAndKeyForInstance(m.InstanceId); companion {
		return
	}
	m.Character.RTState().ChantRound = combatRound.Load()
	m.Character.RT.ChantAnswered = false
	for _, uid := range battle.Players() {
		b, ok := battle.Current(uid)
		u := users.GetByUserId(uid)
		if !ok || !b.Has(m.InstanceId) || u == nil || u.Character == nil {
			continue
		}
		room := rooms.LoadRoom(b.RoomId)
		if room == nil {
			continue
		}
		for _, h := range sideActors(u, room) {
			if h.char.ClassEffects().Has(classes.OverwatchCh) && h.char.Health >= 1 {
				answerChant(m, h, u, room)
			}
		}
	}
}

// ---- Marksman -------------------------------------------------------------

// secondNock sends a Marksman's shot after another target when its Aimed
// Shot fells the first, once a round: a caster first, else the most hurt
// foe it can reach.
func secondNock(attacker, defender statusHolder, rt *characters.ClassRT, round uint64) {
	a, u, room, ok := companyActor(attacker)
	if !ok {
		return
	}
	foes, ok := battleFoes(u, room)
	if !ok {
		return
	}
	ids := make([]int, 0, len(foes))
	for id := range foes {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	var pick *mobs.Mob
	pickCaster, pickFrac := false, 2.0
	for _, id := range ids {
		m := mobs.GetInstance(id)
		if m == nil || m.Character.Health < 1 || m.InstanceId == defender.mob.InstanceId || m.Character.CombatWithdrawn {
			continue
		}
		if reached, _ := abilityReach(a, u, m, room); !reached {
			continue
		}
		caster := len(m.Character.SpellBook) > 0
		frac := float64(m.Character.Health) / float64(max(1, m.Character.HealthMax.Value))
		if pick == nil || caster && !pickCaster || caster == pickCaster && frac < pickFrac {
			pick, pickCaster, pickFrac = m, caster, frac
		}
	}
	if pick == nil {
		return
	}
	rt.NockRound = round
	target := mobHolder(pick)
	a.holder.say(fmt.Sprintf("You nock a second arrow and loose it at %s.", target.tag()),
		"%s nocks a second arrow and looses it at "+verbatim(target.tag())+".", " (second nock)")
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: room.RoomId, Source: a.ref, Target: target.ref, Status: "Second Nock", Outcome: combatstream.OutcomeSucceeded})
	// The second arrow is a plain shot (Phase 38c2 review): not the Aimed
	// Shot's sure critical hit and bonus, and it pins no one.
	delete(abilityKind, a.who)
	if agg := a.char.Aggro; agg != nil && agg.Type == characters.BackStab {
		agg.Type = characters.DefaultAttack
	}
	extraBlow(a, pick, room, 100)
}

// ---- Ravager --------------------------------------------------------------

// huntDown is a Ravager's blow at a foe: it opens a bleeding wound on a foe
// at or below the rank's share of its health (75%), adds a stack to one already bleeding, and leaves
// the foe hunted this round (a harder flight, and a rattled group if it
// breaks).
func huntDown(attacker, defender statusHolder, fx classes.Effects) {
	rt := defender.char.RTState()
	rt.HuntedRound = combatRound.Load()
	rt.HuntPenalty = fx.Int(classes.FleePenalty)
	rt.HarrowPts = fx.Int(classes.Harrow)
	if defender.char.Health < 1 {
		return
	}
	bleeding := status.Live(defender.char, status.Bleeding)
	if !bleeding && defender.char.Health*100 > defender.char.HealthMax.Value*fx.Int(classes.HuntDown) {
		return
	}
	events.AddToQueue(events.Buff{MobInstanceId: defender.mob.InstanceId, BuffId: status.Bleeding, Source: `combat`, ExtraTriggers: fx.Int(classes.BleedLong)})
	word := "opens a wound on"
	if bleeding {
		word = "deepens the wound on"
	}
	attacker.say(fmt.Sprintf("You %s %s.", map[bool]string{false: "open a wound on", true: "deepen the wound on"}[bleeding], defender.tag()),
		"%s "+word+" "+verbatim(defender.tag())+".", " (hunt down: bleeding)")
}

// apex forces a morale check on the rest of a felled foe's group.
func apex(attacker, defender statusHolder) {
	a, u, room, ok := companyActor(attacker)
	if !ok {
		return
	}
	p, found := enemyparty.PartyOf(room, defender.mob.InstanceId)
	if !found {
		return
	}
	checked := 0
	for _, id := range p.Members {
		if id == defender.mob.InstanceId {
			continue
		}
		if m := mobs.GetInstance(id); m != nil && m.Character.Health >= 1 && !m.Character.CombatWithdrawn {
			events.AddToQueue(events.MoraleCheck{LeaderUserId: u.UserId, MobInstanceId: id})
			checked++
		}
	}
	if checked > 0 {
		a.holder.say(fmt.Sprintf("The fall of %s shakes the rest of its band.", defender.tag()),
			"The fall of "+verbatim(defender.tag())+" shakes the rest of its band.", " (apex: morale check)")
		emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: room.RoomId, Source: a.ref, Target: defender.ref, Status: "Apex", Outcome: combatstream.OutcomeSucceeded})
	}
}

// enemyOutcome is an enemy's morale outcome by its temperament: a foe a
// Ravager has struck this round has a smaller chance to lose its nerve and
// flee (never none, when it could).
func enemyOutcome(m *mobs.Mob, temperament string) morale.Outcome {
	roll := moraleRoll(100)
	if rt := m.Character.RT; rt != nil && rt.HuntPenalty > 0 && rt.HuntedRound == combatRound.Load() {
		return morale.EnemyHunted(temperament, roll, rt.HuntPenalty)
	}
	return morale.Enemy(temperament, roll)
}

// harrow rattles the group of a foe that a Ravager wounded and that has now
// lost its nerve: every ally has Attack against them for 2 rounds.
func harrow(m *mobs.Mob, room int) {
	rt := m.Character.RT
	if rt == nil || rt.HarrowPts <= 0 {
		return
	}
	pts := rt.HarrowPts
	rt.HarrowPts = 0
	r := rooms.LoadRoom(room)
	if r == nil {
		return
	}
	p, ok := enemyparty.PartyOf(r, m.InstanceId)
	if !ok {
		return
	}
	rattled := 0
	for _, id := range p.Members {
		if id == m.InstanceId {
			continue
		}
		if o := mobs.GetInstance(id); o != nil && o.Character.Health >= 1 && !o.Character.CombatWithdrawn {
			lendMark(o.Character.RTState(), pts)
			rattled++
		}
	}
	if rattled > 0 {
		r.SendText(fmt.Sprintf("The band falters as one of its own breaks. (harrow: +%d Attack for the company against them, 2 rounds)", pts))
	}
}
