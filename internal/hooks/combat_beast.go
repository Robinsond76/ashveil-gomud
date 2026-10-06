package hooks

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/beasts"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 39e: the Beast Tamer. A Tamer's bonded beast stands in the formation
// for one battle (internal/beasts makes and keeps it) and, unlike a doll,
// takes its own turn through the ordinary mob paths. This file adds what the
// Tamer's class gives it: Sic, Rally and Pack Sense (the Tamer's), Breath
// (the drake's), the hobbling bite (the warhound's) and a fall that wounds
// instead of killing. All of it is runtime only, on the game loop.

// ResetBeastsForTest forgets the pass's runtime state (there is none beyond
// the beasts themselves, which the company registry holds).
func ResetBeastsForTest() {}

// beastTamers are the Beast Tamers among a leader's side in a battle.
func beastTamers(side []actor) []actor {
	var out []actor
	for _, a := range side {
		if beasts.IsTamer(a.char) && a.char.Health > 0 {
			out = append(out, a)
		}
	}
	return out
}

// beastPass stands every Tamer's beast in its battle's room, gives the
// Tamer's Pack Sense, takes away the beasts of a battle that is over, and
// wounds a beast whose health ran out. It runs at the round's start, right
// after the dolls'.
func beastPass() {
	live := map[int]bool{}
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
		for _, a := range beastTamers(sideActors(u, room)) {
			tamer, found := beasts.Of(uid, a.key)
			if !found {
				continue
			}
			_, err := beasts.Spawn(tamer)
			if err != nil && err != beasts.ErrNoBattle && err != beasts.ErrWounded {
				continue
			}
			rt := a.char.RTState()
			// Review fix: a wounded beast's absence is said once a battle,
			// so the company knows why it fights without it.
			if err == beasts.ErrWounded && !rt.Benched && a.char.Beast != nil {
				rt.Benched = true
				name := a.char.Beast.Name
				a.holder.say(fmt.Sprintf("%s is still wounded and sits this battle out. (rest to mend it)", name),
					"%s's beast "+name+" is still wounded and sits this battle out.", "")
			}
			rt.PackSense = 0
			if mob, standing := beasts.Live(tamer); standing {
				live[mob.InstanceId] = true
				mob.Character.RTState().Sic = 0
				if mob.Character.Health > 0 {
					rt.PackSense = a.char.ClassEffects().Int(classes.PackSense)
				}
			}
		}
	}
	for _, id := range company.BeastInstances() {
		if !live[id] {
			beasts.Dismiss(id)
		}
	}
	// A beast that fell outside a blow's own round handling (a spell, a
	// burning) is wounded here.
	for _, id := range company.BeastInstances() {
		if m := mobs.GetInstance(id); m != nil && m.Character.Health < 1 {
			beastFalls(m)
		}
	}
	for _, id := range company.BeastInstances() {
		beasts.Sync(id)
	}
}

// dismissBeasts takes a leader's beasts away: its battle ended.
func dismissBeasts(leaderID int) {
	beasts.DismissAll(leaderID)
}

// isBeastInstance reports whether an instance is a live bonded beast.
func isBeastInstance(instanceID int) bool {
	_, ok := company.BeastOf(instanceID)
	return ok
}

// beastOf is a Tamer's standing, able beast, if it has one.
func beastOf(a actor, u *users.UserRecord) (*mobs.Mob, bool) {
	tamer, found := beasts.Of(u.UserId, a.key)
	if !found {
		return nil, false
	}
	mob, ok := beasts.Live(tamer)
	if !ok || mob.Character.Health < 1 {
		return nil, false
	}
	return mob, true
}

// sicBeast is Sic: at the start of the Tamer's turn its beast is sent at the
// Tamer's foe with the Tamer's Attack behind its strike. The Tamer's own
// blow is still struck. Rally comes first when the beast is hurt.
func sicBeast(a actor, u *users.UserRecord, foes map[int]bool, room *rooms.Room) {
	fx := a.char.ClassEffects()
	if !fx.Has(classes.BeastSic) || !beasts.IsTamer(a.char) {
		return
	}
	// Review fix: "strategy [member] abilities off" holds the beast back
	// (no Sic, no Rally); it still fights on its own turn.
	if enemyparty.MemberStrategy(u.UserId, a.key).NoAbilities {
		return
	}
	beast, ok := beastOf(a, u)
	if !ok {
		return
	}
	rallyBeast(a, beast, u, room)
	foe, ok := abilityFoe(a, u, foes)
	if !ok || !canFight(&beast.Character) {
		return
	}
	if _, lost := status.LostAction(&beast.Character); lost {
		return
	}
	agg := beast.Character.Aggro
	if agg == nil || !plainAttack(agg) || agg.MobInstanceId != foe.InstanceId {
		beast.Character.Aggro = &characters.Aggro{Type: characters.DefaultAttack, MobInstanceId: foe.InstanceId}
		beast.PreventIdle = true
	}
	beast.Character.RTState().Sic = fx.Int(classes.SicAttack)
	a.holder.say(fmt.Sprintf("You send %s at %s.", verbatimTag(beast), verbatimTag(foe)),
		"%s sends "+verbatim(mobHolder(beast).tag())+" at "+verbatim(mobHolder(foe).tag())+".", ` (sic)`)
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: room.RoomId, Source: a.ref, Target: mobRef(foe), Status: `Sic`})
}

func verbatimTag(m *mobs.Mob) string { return verbatim(mobHolder(m).tag()) }

// rallyBeast is Rally: the Tamer heals its beast for a Minor Heal's worth,
// with no mana, a few times a battle, once the beast is below the company's
// healing threshold.
func rallyBeast(a actor, beast *mobs.Mob, u *users.UserRecord, room *rooms.Room) {
	uses := a.char.ClassEffects().Int(classes.Rally)
	rt := a.char.RTState()
	if uses < 1 || rt.Rallies >= uses {
		return
	}
	limit := max(1, beast.Character.HealthLimit())
	below := strategy.TacticsFor(u.UserId).Healing
	if below <= 0 {
		below = strategy.DefaultHealing
	}
	if beast.Character.Health*100 >= below*limit || beast.Character.Health >= limit {
		return
	}
	rt.Rallies++
	healed := beast.Character.ApplyHealthChange(minorHealRoll(a.char, true))
	if healed < 1 {
		return
	}
	roundExtraMobs = append(roundExtraMobs, beast.InstanceId)
	a.holder.say(fmt.Sprintf("You call %s back to heel and rally it. (rally, %d health, %d left)", verbatimTag(beast), healed, uses-rt.Rallies),
		"%s calls "+verbatim(mobHolder(beast).tag())+fmt.Sprintf(" back to heel and rallies it. (rally, %d health, %d left)", healed, uses-rt.Rallies), ` (rally)`)
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: room.RoomId, Source: a.ref, Target: mobRef(beast), Status: `Rally`, Outcome: combatstream.OutcomeSucceeded})
}

// drakeBreath is a drake hatchling's Breath: every few rounds, in place of
// its bite, fire on its foe and up to two other foes beside it. It reports
// whether the turn was spent.
func drakeBreath(a actor, u *users.UserRecord, foes map[int]bool, room *rooms.Room) bool {
	bi := a.char.RT
	if bi == nil || bi.Beast == nil || bi.Beast.BreathEvery < 1 || a.char.Health < 1 {
		return false
	}
	info := bi.Beast
	round := combatRound.Load()
	if info.BreathNext == 0 {
		info.BreathNext = round + uint64(info.BreathEvery)
		if info.BreathFirst {
			info.BreathNext = round // Phase 39i2: a Dragon Lord's drake breathes at once
		}
	}
	if round < info.BreathNext {
		return false
	}
	foe, ok := abilityFoe(a, u, foes)
	if !ok {
		return false
	}
	targets := []*mobs.Mob{foe}
	ids := make([]int, 0, len(foes))
	for id := range foes {
		if id != foe.InstanceId {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	for _, id := range ids {
		if len(targets) >= max(3, info.BreathFoes) {
			break
		}
		if m := mobs.GetInstance(id); m != nil && m.Character.Health > 0 && !m.Character.CombatWithdrawn && !m.Character.HasBuffFlag("hidden") {
			targets = append(targets, m)
		}
	}
	info.BreathNext = round + uint64(info.BreathEvery)
	owner := breathOwnerLevel(info, a.char)
	burned := 0
	for _, t := range targets {
		dmg := beasts.BreathDamage(owner, info.Damage, util.RollDice(1, beasts.BreathDice))
		if -t.Character.ApplyHealthChange(-dmg) > 0 {
			burned++
			roundExtraMobs = append(roundExtraMobs, t.InstanceId)
			// Phase 39i2: a Dragon Lord's Breath leaves its foes alight.
			if info.BreathBurn && t.Character.Health > 0 {
				events.AddToQueue(events.Buff{MobInstanceId: t.InstanceId, BuffId: status.Burning, Source: `combat`})
			}
		}
	}
	abilityTurns[a.who] = true
	note := fmt.Sprintf("breath, %d foes", burned)
	if info.BreathBurn {
		note += ", burning"
	}
	a.holder.say("", fmt.Sprintf("%%s breathes fire across %s. (%s)", verbatimTag(foe), note), ` (breath)`)
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: room.RoomId, Source: a.ref, Target: mobRef(foe), Status: `Breath`})
	return true
}

// breathOwnerLevel is the level Breath scales with: the beast's, which is its
// Tamer's.
func breathOwnerLevel(_ *characters.BeastInfo, c *characters.Character) int { return c.Level }

// beastBlow is what a beast's landed bite does: a warhound's bite hobbles a
// foe below half health (a Packlord's, three quarters) and, for a Packlord,
// exposes a foe it had hobbled already; a Beastlord's bear rakes the foe
// beside its target too.
func beastBlow(attacker, defender statusHolder, r combat.AttackResult) {
	rt := attacker.char.RT
	if rt == nil || rt.Beast == nil || !r.Hit || r.DamageToTarget <= 0 {
		return
	}
	info := rt.Beast
	d := defender.char
	if info.Swipe > 0 && defender.mob != nil {
		beastSwipe(attacker, defender, max(1, r.DamageToTarget*info.Swipe/100))
	}
	if !info.Hobble || d.Health < 1 || d.HealthMax.Value <= 0 {
		return
	}
	ev := events.Buff{BuffId: status.Hobbled, Source: `combat`}
	if defender.user != nil {
		ev.UserId = defender.user.UserId
	} else if defender.mob != nil {
		ev.MobInstanceId = defender.mob.InstanceId
	} else {
		return
	}
	// Phase 39i2: the Packlord's pack hunts a foe it has crippled.
	if info.Hunt && status.Live(d, status.Hobbled) && !status.Live(d, status.Exposed) {
		ev.BuffId = status.Exposed
		events.AddToQueue(ev)
		attacker.say("", "%s's bite finds "+verbatim(defender.tag())+"'s lame leg, and it is left exposed. (exposed)", ` (exposed)`)
		return
	}
	at := info.HobbleAt
	if at <= 0 {
		at = beasts.HobbleBelow
	}
	if d.Health*100 >= at*d.HealthMax.Value || status.Live(d, status.Hobbled) {
		return
	}
	events.AddToQueue(ev)
	attacker.say("", "%s's bite lays "+verbatim(defender.tag())+" open at the legs. (hobbled)", ` (hobble)`)
}

// beastSwipe is a Beastlord's bear raking the foe standing beside its target
// (the nearest in the same row) for a share of the blow.
func beastSwipe(attacker, defender statusHolder, dmg int) {
	room := rooms.LoadRoom(defender.roomId)
	if room == nil {
		return
	}
	beside := foeBeside(defender.mob, room, nil)
	if beside == nil || beside.Character.Health < 1 {
		return
	}
	dealt := -beside.Character.ApplyHealthChange(-dmg)
	if dealt < 1 {
		return
	}
	roundExtraMobs = append(roundExtraMobs, beside.InstanceId)
	attacker.say("", "%s's swipe rakes "+verbatim(mobHolder(beside).tag())+fmt.Sprintf(" too. (swipe, %d damage)", dealt), ` (swipe)`)
}

// beastGuardsLeft is how many guards a bonded bear has left this battle (none
// for any other character).
func beastGuardsLeft(c *characters.Character) int {
	if c == nil || c.RT == nil || c.RT.Beast == nil {
		return 0
	}
	return max(0, c.RT.Beast.Guards-c.RT.Beast.GuardsUsed)
}

// beastFalls is a beast whose health ran out: it is wounded, out of the
// battle and out of fights until the company rests. It never dies.
func beastFalls(m *mobs.Mob) {
	room := rooms.LoadRoom(m.Character.RoomId)
	// Phase 39i2: a Beastlord's bear stands back up once a battle.
	if rt := m.Character.RT; rt != nil && rt.Beast != nil && rt.Beast.Rise > 0 && !rt.Beast.RiseUsed {
		rt.Beast.RiseUsed = true
		m.Character.Health = max(1, m.Character.HealthLimit()*rt.Beast.Rise/100)
		if room != nil {
			room.SendText(fmt.Sprintf(`%s staggers, shakes itself and stands again. (stands again, %d health)`, named(mobTag(mobName(m.InstanceId))), m.Character.Health))
		}
		emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: m.Character.RoomId, Source: mobRef(m), Status: `Stands again`, Outcome: combatstream.OutcomeSucceeded})
		return
	}
	emitCombat(combatstream.Event{Kind: combatstream.Death, RoomId: m.Character.RoomId, Target: mobRef(m), Outcome: combatstream.OutcomeIncapacitated})
	if room != nil {
		room.SendText(fmt.Sprintf(`%s yelps and goes down, wounded. (it will mend after a rest)`, named(mobTag(mobName(m.InstanceId)))))
	}
	m.DeathProcessed = true
	beasts.Dismiss(m.InstanceId)
}
