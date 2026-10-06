package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/interrupt"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 30d1: broken chants and shield counters (internal/interrupt holds
// the rules). After every weapon blow, once its lines are out, afterBlow
// looks at what it did:
//   - a blow that did damage may break the target's chant (Phase 30d1b:
//     heavy force always does, any other blow by a chance that grows with
//     its damage). A player or a company companion loses the spell with
//     half its mana back; any other mob (an enemy) starts the chant again
//     from the first word at its next turn. A chant that holds is told;
//   - a melee blow a shield blocked may be countered (Phase 30g2; before
//     it, a miss): the bearer slams its shield back into the attacker, by
//     a chance from the two Strengths, and the bash may stun. A bash is a
//     counter strike only: it breaks no chant or wind-up.
//
// Phase 30d2's wind-ups (combat_windup.go) are handled from afterBlow too.
//
// Both sets below are runtime only, on the game loop. Aggro (and so every
// chant) is never saved, so a restart or copyover mid-fight drops them
// with the chant.

var (
	// chantRestarts are the enemy mobs whose chant a blow broke, to start
	// again at their next turn.
	chantRestarts = map[int]bool{}
	// countered holds the bearers (by Ref key) that have countered this
	// combat round.
	countered = map[string]bool{}
	// counterRoll rolls the counters; tests replace it.
	counterRoll = util.Rand
	// breakRoll rolls whether a blow breaks a chant; tests replace it.
	breakRoll = util.Rand
	// interruptsOff turns this file off, for a test whose golden record
	// predates Phase 30d1.
	interruptsOff bool
)

// DisableInterruptsForTest turns broken chants and shield counters off
// until the returned restore is called: only for a test that compares
// against a record of combat captured before Phase 30d1. Wind-ups (30d2)
// are not affected; a foe without `windups` never starts one.
func DisableInterruptsForTest() (restore func()) {
	prev := interruptsOff
	interruptsOff = true
	return func() { interruptsOff = prev }
}

// UseCounterRollForTest replaces the counter's dice until the returned
// restore is called.
func UseCounterRollForTest(roll func(int) int) (restore func()) {
	prev := counterRoll
	counterRoll = roll
	return func() { counterRoll = prev }
}

// UseBreakRollForTest replaces the chant-break dice until the returned
// restore is called.
func UseBreakRollForTest(roll func(int) int) (restore func()) {
	prev := breakRoll
	breakRoll = roll
	return func() { breakRoll = prev }
}

// interruptRound starts a combat round: no one has countered yet, and a
// mob gone, dead, or no longer chanting has no restart owed.
func interruptRound() {
	countered = map[string]bool{}
	windUpRound() // Phase 30d2
	for id := range chantRestarts {
		m := mobs.GetInstance(id)
		if m == nil || m.Character.Health < 1 || m.Character.Aggro == nil || m.Character.Aggro.Type != characters.SpellCast {
			delete(chantRestarts, id)
		}
	}
}

// chanting reports whether the holder is chanting a spell now: casting,
// and not an enemy waiting to start again.
func (h statusHolder) chanting() bool {
	if h.char.Aggro == nil || h.char.Aggro.Type != characters.SpellCast {
		return false
	}
	return h.mob == nil || !chantRestarts[h.mob.InstanceId]
}

// afterBlow applies what a resolved blow does to chants and counters.
// Called at every blow site after the blow's own lines.
func afterBlow(attacker, defender statusHolder, r combat.AttackResult) {
	afterWindUpBlow(attacker, defender, r) // Phase 30d2
	// Phase 38a: the first damage wakes a sleeper (paralysis holds).
	if r.Hit && r.DamageToTarget > 0 && status.Wake(defender.char) {
		spec := status.Get(status.Asleep)
		defender.say(spec.EndYou, spec.EndOther, ` (woken by the blow)`)
		emitCombat(combatstream.Event{Kind: combatstream.StatusExpired, RoomId: defender.roomId, Target: defender.ref, BuffId: spec.Id, Status: spec.Word})
	}
	if rt := attacker.char.RT; rt != nil && rt.Cut > 0 {
		rt.Cut = 0 // Phase 39i: the cut strings weakened this one attack
	}
	samuraiBlow(attacker, defender, r)   // Phase 39b
	arbalistBlow(defender, r)            // Phase 39h
	beastBlow(attacker, defender, r)     // Phase 39e
	eliteBlow(attacker, defender, r)     // Phase 38c2
	wardAfterBlow(attacker, defender, r) // Phase 38c3
	if interruptsOff {
		return
	}
	if defender.char.Health >= 1 && interrupt.CanBreak(r.Hit, r.DamageToTarget, defender.chanting()) {
		chance := interrupt.BreakChanceFor(r.DamageToTarget, defender.char.HealthMax.Value, heavyBlow(r), chantDifficulty(defender), chantResolves(defender))
		// Phase 38b: a class steadies its chants.
		if pct := defender.char.ClassEffects().Int(classes.ChantBreak); pct > 0 && chance < 100 {
			chance = chance * (100 - min(pct, 100)) / 100
		}
		if interrupt.RollBreak(chance, breakRoll) {
			breakChant(attacker, defender)
		} else {
			holdChant(attacker, defender)
		}
	}
	if r.Blocked() {
		counterBlow(attacker, defender)
	}
	if r.Parried() {
		riposteBlow(attacker, defender)
	}
	thornsBlow(attacker, defender, r)
	oathBlow(attacker, defender, r)
	hookBlow(attacker, defender, r) // Phase 39a
	braceBlow(attacker, defender)   // Phase 39a
	if r.CritLanded && attacker.char.ClassEffects().Has(classes.TerrorCrit) && defender.char.Health >= 1 && !status.Live(defender.char, status.Staggered) {
		ev := events.Buff{BuffId: status.Staggered, Source: `combat`}
		if defender.user != nil {
			ev.UserId = defender.user.UserId
		} else {
			ev.MobInstanceId = defender.mob.InstanceId
		}
		events.AddToQueue(ev)
		defender.say(`Dread of the blow staggers you.`, `Dread of the blow staggers %s.`, ` (staggered)`)
	}
}

// riposteBlow lets a parrying Duelist answer the blow at once, with a
// counter-blow the size of an Opening Strike (half again at Riposte 2).
// Once a round, as a shield's counter is.
func riposteBlow(attacker, defender statusHolder) {
	fx := defender.char.ClassEffects()
	n := fx.Int(classes.Riposte)
	if n <= 0 || defender.char.Health < 1 || attacker.char.Health < 1 ||
		attacker.char.RoomId != defender.char.RoomId || defender.char.HasBuffFlag("no-combat") || status.Grounded(defender.char) {
		return
	}
	// Phase 38c2: a Swordmaster may riposte more than once a round (Twin
	// ripostes) or without limit while above half health (Unbroken Guard).
	round := combatRound.Load()
	rt := defender.char.RTState()
	if rt.RiposteRound != round {
		rt.RiposteRound, rt.Ripostes = round, 0
	}
	free := fx.Has(classes.RiposteFree) && defender.char.Health*2 > defender.char.HealthLimit()
	if rt.Ripostes >= 1 {
		if !free && rt.Ripostes >= fx.Int(classes.RiposteRound) {
			return
		}
	} else if countered[defender.ref.Key()] {
		return
	}
	rt.Ripostes++
	countered[defender.ref.Key()] = true
	size := strategy.OpeningStrikeBonus(defender.char.Level)
	if n >= 2 {
		size += size / 2
	}
	size = max(1, size*(100+fx.Int(classes.RipostePct))/100)
	crit := fx.Has(classes.RiposteDaze) && combat.Crits(*defender.char, *attacker.char)
	dealt := -attacker.char.ApplyHealthChange(-size)
	for _, e := range attackEvents(defender.ref, attacker.ref, defender.char.RoomId, defender.char, combat.AttackResult{Hit: true, DamageToTarget: dealt}) {
		if e.Kind == combatstream.Attack {
			e.WeaponType = `riposte`
		}
		emitCombat(e)
	}
	suffix := fmt.Sprintf(` (riposte, %d damage)`, dealt)
	if attacker.mob != nil && attacker.char.Health >= 1 {
		if crit {
			events.AddToQueue(events.Buff{MobInstanceId: attacker.mob.InstanceId, BuffId: status.Staggered, Source: `combat`})
			suffix += ` (disarmed: staggered)`
		}
		if fx.Has(classes.RiposteExpo) {
			events.AddToQueue(events.Buff{MobInstanceId: attacker.mob.InstanceId, BuffId: status.Exposed, Source: `combat`, Triggers: 1})
			suffix += ` (exposed)`
		}
	}
	room := rooms.LoadRoom(defender.char.RoomId)
	var exclude []int
	if defender.user != nil {
		defender.user.SendText(fmt.Sprintf(`You turn the parry into a riposte against %s.`, attacker.tag()) + suffix)
		exclude = append(exclude, defender.user.UserId)
	}
	if attacker.user != nil {
		attacker.user.SendText(util.CapitalizeFirst(fmt.Sprintf(`%s turns the parry into a riposte against you.`, defender.tag())) + suffix)
		exclude = append(exclude, attacker.user.UserId)
	}
	if room != nil {
		room.SendText(util.CapitalizeFirst(fmt.Sprintf(`%s turns the parry into a riposte against %s.`, defender.tag(), attacker.tag()))+suffix, exclude...)
	}
	owner := defender.char.GetCharmedUserId()
	if defender.user != nil {
		owner = defender.user.UserId
	}
	if owner > 0 {
		attacker.char.TrackPlayerDamage(owner, dealt)
	}
	if attacker.user != nil {
		roundExtraPlayers = append(roundExtraPlayers, attacker.user.UserId)
		events.AddToQueue(events.CharacterVitalsChanged{UserId: attacker.user.UserId})
	} else {
		roundExtraMobs = append(roundExtraMobs, attacker.mob.InstanceId)
	}
}

// chantDifficulty is the difficulty of the spell the holder is chanting
// (Phase 35b: harder spells break more easily).
func chantDifficulty(h statusHolder) int {
	if h.char.Aggro == nil {
		return 0
	}
	if sp := spells.GetSpell(h.char.Aggro.SpellInfo.SpellId); sp != nil {
		return sp.Difficulty
	}
	return 0
}

// chantResolves reports whether the holder is chanting a one-round heal,
// which only heavy force breaks (Phase 35d).
func chantResolves(h statusHolder) bool {
	if h.char.Aggro == nil {
		return false
	}
	sp := spells.GetSpell(h.char.Aggro.SpellInfo.SpellId)
	if sp == nil {
		return false
	}
	helpful := sp.Type == spells.HelpSingle || sp.Type == spells.HelpMulti
	return interrupt.ResolvesHeal(sp.WaitRounds, helpful, sp.School == spells.SchoolRestoration)
}

// heavyBlow reports whether a blow lands with heavy force, which always
// breaks a chant: a critical hit that got through the armor, or one that
// staggers, knocks down, or stuns.
func heavyBlow(r combat.AttackResult) bool {
	if r.CritLanded {
		return true
	}
	for _, id := range r.BuffTarget {
		if id == status.Staggered || id == status.KnockedDown || id == status.Stunned {
			return true
		}
	}
	return false
}

// spellName is a spell's name and cost, the id when it is no longer
// loaded.
func spellName(spellId string) (string, int) {
	if sp := spells.GetSpell(spellId); sp != nil {
		return sp.Name, sp.Cost
	}
	return spellId, 0
}

// breakChant breaks the chanter's spell: the events, the lines, and for a
// player or companion the spell's end with half its mana back, for an
// enemy a restart owed.
func breakChant(by, chanter statusHolder) {
	spellId := chanter.char.Aggro.SpellInfo.SpellId
	name, _ := spellName(spellId)
	cost := chanter.char.SpellCost(spells.GetSpell(spellId))
	roomId := chanter.char.RoomId
	emitCombat(combatstream.Event{Kind: combatstream.Interrupt, RoomId: roomId, Source: by.ref, Target: chanter.ref, SpellId: spellId, Status: name, Outcome: combatstream.OutcomeSucceeded})
	emitCast(combatstream.CastComplete, chanter.ref, spellId, combatstream.OutcomeInterrupted, roomId)

	room := rooms.LoadRoom(roomId)
	other := util.CapitalizeFirst(fmt.Sprintf(`%s's chant breaks off under the blow. (%s interrupted)`, chanter.tag(), name))

	if chanter.enemy() {
		// An enemy: it starts again at its next turn.
		chantRestarts[chanter.mob.InstanceId] = true
		if room != nil {
			room.SendText(other)
		}
		return
	}

	refund := chanter.char.ApplyManaChange(interrupt.Refund(cost))
	who := caster{}
	if chanter.user != nil {
		who.userId = chanter.user.UserId
	} else {
		who.mobId = chanter.mob.InstanceId
	}
	endCast(chanter.char, who)

	if chanter.user != nil {
		back := ``
		if refund > 0 {
			back = fmt.Sprintf(`, %d mana back`, refund)
		}
		chanter.user.SendText(fmt.Sprintf(`The blow breaks your chant, and %s is lost. (%s interrupted%s)`, name, name, back))
		if room != nil {
			room.SendText(other, chanter.user.UserId)
		}
		events.AddToQueue(events.CharacterVitalsChanged{UserId: chanter.user.UserId})
		events.AddToQueue(events.AggroChanged{UserId: chanter.user.UserId, RoomId: roomId})
		return
	}
	if room != nil {
		room.SendText(other)
	}
	events.AddToQueue(events.AggroChanged{MobInstanceId: chanter.mob.InstanceId, RoomId: roomId})
}

// holdChant tells of a blow the chant withstood: the lines, and an
// Interrupt event that failed. The chant goes on untouched.
func holdChant(by, chanter statusHolder) {
	spellId := chanter.char.Aggro.SpellInfo.SpellId
	name, _ := spellName(spellId)
	roomId := chanter.char.RoomId
	emitCombat(combatstream.Event{Kind: combatstream.Interrupt, RoomId: roomId, Source: by.ref, Target: chanter.ref, SpellId: spellId, Status: name, Outcome: combatstream.OutcomeFailed})

	var exclude []int
	if chanter.user != nil {
		chanter.user.SendText(fmt.Sprintf(`You flinch, but your chant holds. (%s, chant held)`, name))
		exclude = append(exclude, chanter.user.UserId)
	}
	if room := rooms.LoadRoom(roomId); room != nil {
		room.SendText(util.CapitalizeFirst(fmt.Sprintf(`%s flinches, but the chant holds. (%s, chant held)`, chanter.tag(), name)), exclude...)
	}
}

// restartChant starts an enemy's broken chant again from the first word,
// at its turn: the line, the full chant time, a cast-start event. It
// takes the mob's turn. A mob with no target left standing here gives the
// spell up. ok is false when no restart was owed.
func restartChant(m *mobs.Mob) bool {
	if !chantRestarts[m.InstanceId] {
		return false
	}
	delete(chantRestarts, m.InstanceId)
	agg := m.Character.Aggro
	sp := spells.GetSpell(agg.SpellInfo.SpellId)
	if sp == nil || !spellTargetStands(agg.SpellInfo, m.Character.RoomId) {
		endCast(&m.Character, caster{mobId: m.InstanceId})
		events.AddToQueue(events.AggroChanged{MobInstanceId: m.InstanceId, RoomId: m.Character.RoomId})
		return true
	}
	agg.RoundsWaiting = sp.WaitRounds + m.Character.ColdDelay()
	agg.ColdDelayed, agg.ColdNotice = m.Character.ColdDelay() > 0, false
	rounds := agg.RoundsWaiting + 1
	plural := `rounds`
	if rounds == 1 {
		plural = `round`
	}
	if room := rooms.LoadRoom(m.Character.RoomId); room != nil {
		room.SendText(util.CapitalizeFirst(fmt.Sprintf(`%s starts the chant again from the first word. (chanting: %s, %d %s)`,
			named(mobTag(mobName(m.InstanceId))), sp.Name, rounds, plural)))
	}
	emitCast(combatstream.CastStart, mobRef(m), sp.SpellId, ``, m.Character.RoomId)
	return true
}

// spellTargetStands reports whether any of a spell's targets still stands
// in the room.
func spellTargetStands(info characters.SpellAggroInfo, roomId int) bool {
	for _, id := range info.TargetUserIds {
		if u := users.GetByUserId(id); u != nil && u.Character.RoomId == roomId && u.Character.Health > 0 {
			return true
		}
	}
	for _, id := range info.TargetMobInstanceIds {
		if m := mobs.GetInstance(id); m != nil && m.Character.RoomId == roomId && m.Character.Health > 0 {
			return true
		}
	}
	return false
}

// counterBlow lets the bearer of a shield that blocked a blow counter it.
func counterBlow(attacker, bearer statusHolder) {
	if bearer.char.Health < 1 || attacker.char.Health < 1 {
		return
	}
	c := interrupt.Counter{
		Melee:     weaponType(attacker.char) != string(items.Shooting),
		SameRoom:  attacker.char.RoomId == bearer.char.RoomId,
		Shield:    bearer.char.HasShield(),
		Able:      !bearer.char.HasBuffFlag("no-combat") && !status.Grounded(bearer.char),
		Chanting:  bearer.chanting(),
		Countered: countered[bearer.ref.Key()],
	}
	if !interrupt.CanCounter(c) {
		return
	}
	bashChance := combat.BashChance(bearer.char, attacker.char)
	bash, ok := interrupt.RollCounter(bashChance, counterRoll)
	if !ok {
		return
	}
	countered[bearer.ref.Key()] = true

	dealt := -attacker.char.ApplyHealthChange(-bash.Damage)
	result := combat.AttackResult{Hit: true, DamageToTarget: dealt}
	if bash.Stun {
		result.BuffTarget = []int{status.Stunned}
	}
	roomId := bearer.char.RoomId
	for _, e := range attackEvents(bearer.ref, attacker.ref, roomId, bearer.char, result) {
		if e.Kind == combatstream.Attack {
			e.WeaponType = `shield-bash`
		}
		emitCombat(e)
	}

	suffix := fmt.Sprintf(` (shield bash, %d damage)`, dealt)
	if bash.Stun {
		suffix = fmt.Sprintf(` (shield bash, %d damage, stunned)`, dealt)
	}
	counterLines(bearer, attacker, suffix)

	if bash.Stun {
		if attacker.user != nil {
			attacker.user.AddBuff(status.Stunned, `combat`)
		} else {
			attacker.mob.AddBuff(status.Stunned, `combat`)
		}
	}

	// The damage counts toward the kill (a companion's for its leader, as
	// its blows do), and a fall is resolved this round.
	owner := bearer.char.GetCharmedUserId()
	if bearer.user != nil {
		owner = bearer.user.UserId
	}
	if owner > 0 {
		attacker.char.TrackPlayerDamage(owner, dealt)
	}
	if attacker.user != nil {
		roundExtraPlayers = append(roundExtraPlayers, attacker.user.UserId)
		events.AddToQueue(events.CharacterVitalsChanged{UserId: attacker.user.UserId})
	} else {
		roundExtraMobs = append(roundExtraMobs, attacker.mob.InstanceId)
	}

	// A bash is a counter strike only (owner, 2026-09-30): it breaks
	// neither a chant nor a wind-up.
}

// counterLines tells the bearer, the attacker, and the room of a bash.
func counterLines(bearer, attacker statusHolder, suffix string) {
	room := rooms.LoadRoom(bearer.char.RoomId)
	his := bearer.char.CombatPronouns().Possessive
	roomLine := util.CapitalizeFirst(fmt.Sprintf(`%s drives %s shield back into %s.`, bearer.tag(), his, attacker.tag())) + suffix
	var exclude []int
	if bearer.user != nil {
		bearer.user.SendText(fmt.Sprintf(`You drive your shield back into %s.`, attacker.tag()) + suffix)
		exclude = append(exclude, bearer.user.UserId)
	}
	if attacker.user != nil {
		attacker.user.SendText(util.CapitalizeFirst(fmt.Sprintf(`%s drives %s shield back into you.`, bearer.tag(), his)) + suffix)
		exclude = append(exclude, attacker.user.UserId)
	}
	if room != nil {
		room.SendText(roomLine, exclude...)
	}
}
