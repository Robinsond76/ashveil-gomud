package hooks

import (
	"fmt"

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
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 39a: the Halberdier. Sweep strikes a foe and the foes beside it in
// its row, Brace holds the turn for a foe's strike, Hook drags a leaping
// foe off its feet, and a Valkyrie's Charged Sweep adds lightning. All of it
// is runtime only, on the game loop, like every class ability.

// hookRoll rolls Hook's chance; tests replace it.
var hookRoll = util.Rand

// UseHookRollForTest replaces Hook's dice until the returned restore is
// called.
func UseHookRollForTest(roll func(int) int) (restore func()) {
	prev := hookRoll
	hookRoll = roll
	return func() { hookRoll = prev }
}

// halberdier reports whether a character is of the Halberdier lineage.
func halberdier(c *characters.Character) bool { return c.ArchetypeID() == "halberdier" }

// extraBlow is one more weapon blow by a company member at a foe, outside
// its ordinary swing: through the same hit, defense, armor, status and wound
// rules and the same lines as any blow, with its damage at pct percent. A
// foe it fells is sent to the round's death handling.
func extraBlow(a actor, foe *mobs.Mob, room *rooms.Room, pct int) combat.AttackResult {
	rt := a.char.RTState()
	rt.BlowPct = pct
	defer func() { rt.BlowPct = 0 }()
	defRoom := rooms.LoadRoom(foe.Character.RoomId)
	if defRoom == nil {
		defRoom = room
	}
	var r combat.AttackResult
	if u := a.holder.user; u != nil {
		r = combat.AttackPlayerVsMob(u, foe)
		emitAttack(userRef(u), mobRef(foe), room.RoomId, u.Character, r)
		for _, id := range r.BuffSource {
			u.AddBuff(id, `combat`)
		}
		for _, id := range r.BuffTarget {
			foe.AddBuff(id, `combat`)
		}
		for _, msg := range r.MessagesToSource {
			u.SendText(msg)
		}
		for _, msg := range r.MessagesToSourceRoom {
			room.SendText(msg, u.UserId)
		}
		for _, msg := range r.MessagesToTargetRoom {
			defRoom.SendText(msg, u.UserId)
		}
		afterBlow(userHolder(u), mobHolder(foe), r)
		if r.Hit {
			scripting.TryMobScriptEvent(`onHurt`, foe.InstanceId, u.UserId, `user`, map[string]any{`damage`: r.DamageToTarget, `crit`: r.Crit})
		}
		if r.DamageToSource != 0 {
			events.AddToQueue(events.CharacterVitalsChanged{UserId: u.UserId})
		}
	} else if m := a.holder.mob; m != nil {
		r = combat.AttackMobVsMob(m, foe)
		emitAttack(mobRef(m), mobRef(foe), room.RoomId, &m.Character, r)
		for _, id := range r.BuffSource {
			m.AddBuff(id, `combat`)
		}
		for _, id := range r.BuffTarget {
			foe.AddBuff(id, `combat`)
		}
		for _, msg := range r.MessagesToSourceRoom {
			room.SendText(msg)
		}
		for _, msg := range r.MessagesToTargetRoom {
			defRoom.SendText(msg)
		}
		afterBlow(mobHolder(m), mobHolder(foe), r)
		if r.Hit {
			scripting.TryMobScriptEvent(`onHurt`, foe.InstanceId, m.InstanceId, `mob`, map[string]any{`damage`: r.DamageToTarget, `crit`: r.Crit})
		}
	}
	if foe.Character.Health < 1 {
		foe.Character.EndAggro()
		events.AddToQueue(events.AggroChanged{MobInstanceId: foe.InstanceId, RoomId: foe.Character.RoomId})
	}
	roundExtraMobs = append(roundExtraMobs, foe.InstanceId)
	return r
}

// sweepTargets are the foes a sweep at foe strikes, foe first: the foe
// nearest it in the same row of its group (the weaker when two stand beside
// it), or every foe in the row from level 8. Only the standing foes of the
// battle count.
func sweepTargets(a actor, foe *mobs.Mob, room *rooms.Room, foes map[int]bool) []*mobs.Mob {
	out := []*mobs.Mob{foe}
	party, ok := enemyparty.PartyOf(room, foe.InstanceId)
	if !ok {
		return out
	}
	row, col, found := party.Formation.Find(mobparty.MemberKeyFor(foe.InstanceId))
	if !found {
		return out
	}
	alive := enemyparty.Alive(party)
	var beside []*mobs.Mob
	var dist []int
	for c := 0; c < company.FormationCols; c++ {
		key := party.Formation.At(row, c)
		if c == col || key == "" || !alive[key] {
			continue
		}
		id, ok := mobparty.InstanceIdFromMemberKey(key)
		if !ok || !foes[id] {
			continue
		}
		m := mobs.GetInstance(id)
		if m == nil || m.Character.Health < 1 || m.Character.CombatWithdrawn || m.Character.HasBuffFlag("hidden") {
			continue
		}
		beside = append(beside, m)
		dist = append(dist, abs(c-col))
	}
	if strategy.SweepWide(a.char.Level) {
		return append(out, beside...)
	}
	best := -1
	for i, m := range beside {
		if best < 0 || dist[i] < dist[best] || dist[i] == dist[best] && m.Character.Health < beside[best].Character.Health {
			best = i
		}
	}
	if best >= 0 {
		out = append(out, beside[best])
	}
	return out
}

// foeStrikesColumn reports whether a standing foe of the battle is aimed at
// a member of the actor's formation column, so a brace has a blow to
// answer. A foe's blow at a member behind the front is caught by the front.
func foeStrikesColumn(a actor, u *users.UserRecord, f company.Formation, foes map[int]bool) bool {
	_, col, placed := f.Find(a.key)
	for id := range foes {
		m := mobs.GetInstance(id)
		if m == nil || m.Character.Health < 1 || m.Character.Aggro == nil {
			continue
		}
		key := company.MemberKey("")
		switch agg := m.Character.Aggro; {
		case agg.UserId == u.UserId:
			key = company.LeaderMemberKey
		case agg.MobInstanceId > 0:
			if owner, k, ok := company.LeaderAndKeyForInstance(agg.MobInstanceId); ok && owner == u.UserId {
				key = k
			}
		}
		if key == "" {
			continue
		}
		if key == a.key {
			return true
		}
		if _, c, ok := f.Find(key); ok && placed && c == col {
			return true
		}
	}
	return false
}

// halberdSituation fills in what Sweep and Brace are chosen from.
func halberdSituation(sit *strategy.AbilitySituation, a actor, u *users.UserRecord, f company.Formation, foe *mobs.Mob, room *rooms.Room, foes map[int]bool) {
	for _, id := range sit.Known {
		switch id {
		case strategy.Sweep:
			sit.SweepFoes = len(sweepTargets(a, foe, room, foes))
		case strategy.Brace:
			// Only the front of a column catches the blows aimed into it
			// (a member outside any formation is struck as itself).
			front := true
			if row, col, ok := f.Find(a.key); ok {
				r, _ := frontOf(f, col, enemyparty.CompanyAlive(u.UserId, f))
				front = row == r
			}
			sit.Struck = front && foeStrikesColumn(a, u, f, foes)
		}
	}
}

// frontOf is the row of the front-most living occupant of a column.
func frontOf(f company.Formation, col int, alive map[company.MemberKey]bool) (int, bool) {
	for r := 0; r < company.FormationRows; r++ {
		if key := f.At(r, col); key != "" && alive[key] {
			return r, true
		}
	}
	return 0, false
}

// useSweep is a Sweep: the whole turn, one blow at each foe struck, each at
// the sweep's share of the damage (80%, or a Sweeper's 100%), and a
// Valkyrie's Charged Sweep adds lightning that ignores armor to each blow
// that lands.
func useSweep(a actor, foe *mobs.Mob, room *rooms.Room, foes map[int]bool) {
	fx := a.char.ClassEffects()
	pct := strategy.SweepPct
	if v := fx.Int(classes.SweepPct); v > 0 {
		pct = v
	}
	abilityTurns[a.who] = true
	targets := sweepTargets(a, foe, room, foes)
	sides := 0
	if cost := fx.Int(classes.ChargedMana); cost > 0 && a.char.Mana >= cost {
		a.char.ApplyManaChange(-cost)
		sides = fx.Int(classes.ChargedDice)
	}
	suffix := ` (sweep)`
	if sides > 0 {
		suffix = ` (charged sweep)`
	}
	var what string
	if len(targets) > 1 {
		what = fmt.Sprintf(`You sweep your weapon across %s and the foes beside it.`, mobHolder(foe).tag())
	} else {
		what = fmt.Sprintf(`You sweep your weapon across %s.`, mobHolder(foe).tag())
	}
	a.holder.say(what, `%s sweeps a weapon across the row.`, suffix)
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: room.RoomId, Source: a.ref, Target: mobRef(foe), Status: `Sweep`})
	struck := map[int]bool{}
	for _, t := range targets {
		if struck[t.InstanceId] || t.Character.Health < 1 {
			continue
		}
		struck[t.InstanceId] = true
		r := extraBlow(a, t, room, pct)
		if sides > 0 && r.Hit && t.Character.Health > 0 {
			bolt := util.RollDice(1, sides)
			t.Character.ApplyHealthChange(-bolt)
			if t.Character.Health < 1 {
				t.Character.EndAggro()
				events.AddToQueue(events.AggroChanged{MobInstanceId: t.InstanceId, RoomId: t.Character.RoomId})
			}
			a.holder.say(fmt.Sprintf(`Lightning leaps from your weapon into %s.`, mobHolder(t).tag()),
				`Lightning leaps from %s's weapon into `+verbatim(mobHolder(t).tag())+`.`, fmt.Sprintf(` (lightning, %d damage)`, bolt))
			if owner := a.holder.char.GetCharmedUserId(); a.holder.user == nil && owner > 0 {
				t.Character.TrackPlayerDamage(owner, bolt)
			} else if a.holder.user != nil {
				t.Character.TrackPlayerDamage(a.holder.user.UserId, bolt)
			}
		}
	}
}

// useBrace is a Brace: the whole turn, and a blow held for the first foe
// that strikes the halberdier's place in the line.
func useBrace(a actor, room *rooms.Room) {
	abilityTurns[a.who] = true
	a.char.RTState().Brace = true
	a.holder.say(`You set your weapon and brace for the first blow.`, `%s sets a weapon and braces for the first blow.`, ` (brace)`)
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: room.RoomId, Source: a.ref, Status: `Brace`})
}

// braceBlow lets a braced halberdier answer a foe that has just struck its
// place in the line (its own, or anyone in its column for a Vanguard): the
// held blow, 25% harder than an ordinary one, once. The foe must be within
// the halberdier's reach.
func braceBlow(attacker, defender statusHolder) {
	if attacker.mob == nil || attacker.char.Health < 1 || attacker.char.Aggro == nil {
		return
	}
	if _, _, companion := company.LeaderAndKeyForInstance(attacker.mob.InstanceId); companion || attacker.char.IsCharmed() {
		return
	}
	var u *users.UserRecord
	var key company.MemberKey
	if defender.user != nil {
		u, key = defender.user, company.LeaderMemberKey
	} else if defender.mob != nil {
		owner, k, ok := company.LeaderAndKeyForInstance(defender.mob.InstanceId)
		if !ok {
			return
		}
		u, key = users.GetByUserId(owner), k
	}
	if u == nil || u.Character == nil {
		return
	}
	room := rooms.LoadRoom(defender.roomId)
	f, _ := company.FormationFor(u.UserId)
	if room == nil {
		return
	}
	_, hitCol, _ := f.Find(key)
	for _, h := range sideActors(u, room) {
		rt := h.char.RT
		if rt == nil || !rt.Brace || h.char.Health < 1 || status.Grounded(h.char) {
			continue
		}
		fx := h.char.ClassEffects()
		if h.key != key {
			_, col, ok := f.Find(h.key)
			if !fx.Has(classes.BraceCol) || !ok || col != hitCol {
				continue
			}
		}
		if reached, _ := abilityReach(h, u, attacker.mob, room); !reached {
			continue
		}
		rt.Brace = false
		h.holder.say(fmt.Sprintf(`Your braced weapon meets %s.`, attacker.tag()),
			`%s's braced weapon meets `+verbatim(attacker.tag())+`.`, ` (brace)`)
		r := extraBlow(h, attacker.mob, room, strategy.BracePct)
		if r.Hit && fx.Has(classes.BraceDown) && attacker.char.Health > 0 && !status.Live(attacker.char, status.KnockedDown) {
			events.AddToQueue(events.Buff{MobInstanceId: attacker.mob.InstanceId, BuffId: status.KnockedDown, Source: `combat`})
			h.holder.say(fmt.Sprintf(`The held blow knocks %s down.`, attacker.tag()),
				`The held blow knocks `+verbatim(attacker.tag())+` down.`, ` (knocked down)`)
		}
	}
}

// hookBlow is the glaive's Hook: from level 6 a hit by a halberdier's reach
// weapon has a chance to drag a leaping foe off its feet.
func hookBlow(attacker, defender statusHolder, r combat.AttackResult) {
	if !r.Hit || r.DamageToTarget < 1 || defender.mob == nil || !defender.mob.Leap || defender.char.Health < 1 {
		return
	}
	if !halberdier(attacker.char) || !attacker.char.Equipment.Weapon.GetSpec().Reach || status.Live(defender.char, status.KnockedDown) {
		return
	}
	chance := strategy.HookChance(attacker.char.Level)
	if chance <= 0 || hookRoll(100) >= chance {
		return
	}
	events.AddToQueue(events.Buff{MobInstanceId: defender.mob.InstanceId, BuffId: status.KnockedDown, Source: `combat`})
	attacker.say(fmt.Sprintf(`You hook %s off its feet.`, defender.tag()),
		`%s hooks `+verbatim(defender.tag())+` off its feet.`, ` (hook, knocked down)`)
}
