package hooks

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hexes"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/summons"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 38c3: the wizard and witch elites. The Archon (Counterspell, Mana
// Shield, Reflection, Aegis), the Archmage (Overchannel, Storm), the
// Necromancer (thralls, Death's Harvest, Lich's Bargain), the Wise One's
// ward gifts and the Crone of Ash's curses. Everything here is runtime
// only, on the game loop; the spells' own numbers live in the spell scripts
// and scripting.ScriptActor.

func init() {
	mobcommands.OnFoeFall.Register(eliteFall)
}

var (
	// heldTurns are the casters whose turn this round an Archon's held turn
	// took (a Counterspell, an Aegis): abilityPass hands them to abilityTurns,
	// so they don't swing.
	heldTurns = map[caster]bool{}
	// counterRoll rolls a Counterspell (0..99); tests replace it.
	counterspellRoll = util.Rand
	// aegisWards is the mana an Archon's Aegis costs, in Arcane Wards.
	aegisWards = 2
)

// UseCounterspellRollForTest replaces the Counterspell dice until the
// returned restore is called.
func UseCounterspellRollForTest(roll func(int) int) (restore func()) {
	prev := counterspellRoll
	counterspellRoll = roll
	return func() { counterspellRoll = prev }
}

// ResetEliteForTest forgets held turns and fallen foes, and the rogue and
// ranger elites' marks.
func ResetEliteForTest() {
	resetRogueRangerElite()
	clear(heldTurns)
	summons.ResetFallenForTest()
}

// takeHeldTurns moves this round's held turns into the ability turns.
func takeHeldTurns() {
	for who := range heldTurns {
		abilityTurns[who] = true
	}
	clear(heldTurns)
}

// archonTurn is an Archon's held turn: it counters the first chant it sees,
// else spreads its Aegis. It reports whether it used the turn.
func archonTurn(a actor, side []actor, u *users.UserRecord, foes []int) bool {
	fx := a.char.ClassEffects()
	if fx == nil || a.char.Health < 1 {
		return false
	}
	if enemyparty.MemberStrategy(u.UserId, a.key).NoAbilities {
		return false
	}
	if fx.Has(classes.Counter) && counterspell(a, fx, foes) {
		heldTurns[a.who], tempoBlocked[a.who] = true, true
		return true
	}
	if fx.Has(classes.Aegis) && aegis(a, side) {
		heldTurns[a.who], tempoBlocked[a.who] = true, true
		return true
	}
	return false
}

// counterspell tries to break the chant of the first foe that is chanting: its
// Mysticism against the caster's (the hexes' edge roll, 65 in 100 at even
// stats, 25 to 90, a boss 25 less), for the spell's mana. A countered caster
// starts again at its next turn, and may lose some of its mana.
func counterspell(a actor, fx classes.Effects, foes []int) bool {
	cost := fx.Int(classes.Counter)
	if cost < 1 || a.char.Mana < cost {
		return false
	}
	var chanter *mobs.Mob
	for _, id := range foes {
		if m := mobs.GetInstance(id); m != nil && m.Character.Health >= 1 && mobHolder(m).chanting() {
			chanter = m
			break
		}
	}
	if chanter == nil {
		return false
	}
	a.char.Mana -= cost
	if a.who.userId > 0 {
		events.AddToQueue(events.CharacterVitalsChanged{UserId: a.who.userId})
	}
	resist := 0
	if chanter.Boss {
		resist = hexes.BossResist
	}
	edge := combat.StatEdge(a.char.Stats.Mysticism.ValueAdj, chanter.Character.Stats.Mysticism.ValueAdj)
	chance := hexes.LandChanceWith(edge, resist, fx.Int(classes.CounterBonus))
	spellId := chanter.Character.Aggro.SpellInfo.SpellId
	name, _ := spellName(spellId)
	foe := named(mobTag(mobName(chanter.InstanceId)))
	if counterspellRoll(100) >= chance {
		emitCombat(combatstream.Event{Kind: combatstream.Interrupt, RoomId: a.char.RoomId, Source: a.ref, Target: mobRef(chanter), SpellId: spellId, Status: name, Outcome: combatstream.OutcomeFailed})
		a.holder.say("You hold your turn and speak the counter-word, but "+foe+"'s chant shrugs it off. (counterspell failed, "+name+" chant holds)",
			"%s holds a turn and speaks a counter-word, but "+verbatim(foe)+"'s chant shrugs it off. (counterspell failed, "+name+" chant holds)", "")
		return true
	}
	chantRestarts[chanter.InstanceId] = true
	emitCombat(combatstream.Event{Kind: combatstream.Interrupt, RoomId: a.char.RoomId, Source: a.ref, Target: mobRef(chanter), SpellId: spellId, Status: name, Outcome: combatstream.OutcomeSucceeded})
	emitCast(combatstream.CastComplete, mobRef(chanter), spellId, combatstream.OutcomeInterrupted, a.char.RoomId)
	drained := ""
	if pct := fx.Int(classes.CounterDrain); pct > 0 {
		lost := min(chanter.Character.Mana, max(1, chanter.Character.ManaMax.Value*pct/100))
		chanter.Character.Mana -= lost
		if lost > 0 {
			drained = fmt.Sprintf(", %d mana lost", lost)
		}
	}
	a.holder.say("You hold your turn and speak the counter-word, and "+foe+"'s chant comes apart. (counterspell, "+name+" interrupted"+drained+")",
		"%s holds a turn and speaks a counter-word, and "+verbatim(foe)+"'s chant comes apart. (counterspell, "+name+" interrupted"+drained+")", "")
	return true
}

// aegis wards every ally at once, once a battle, when two or more of the
// company stand without a ward.
func aegis(a actor, side []actor) bool {
	rt := a.char.RTState()
	cost := aegisWards * a.char.SpellCost(spells.GetSpell("arcaneward"))
	if rt.AegisUsed || a.char.Mana < cost {
		return false
	}
	bare := 0
	for _, t := range side {
		if t.char.Health >= 1 && (t.char.RT == nil || t.char.RT.Ward <= 0) {
			bare++
		}
	}
	if bare < 2 {
		return false
	}
	self := scripting.GetActor(a.who.userId, a.who.mobId)
	if self == nil {
		return false
	}
	cap := max(1, int(self.SpellPower("arcaneward")))
	blows := max(1, self.ClassEffect("wardblows"))
	warded := 0
	for _, t := range side {
		if t.char.Health < 1 {
			continue
		}
		ally := scripting.GetActor(t.who.userId, t.who.mobId)
		if ally != nil && ally.GrantWard(cap, blows) {
			self.WardGifts(ally)
			warded++
		}
	}
	if warded == 0 {
		return false
	}
	rt.AegisUsed = true
	a.char.Mana -= cost
	if a.who.userId > 0 {
		events.AddToQueue(events.CharacterVitalsChanged{UserId: a.who.userId})
	}
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: a.char.RoomId, Source: a.ref, Status: "Archon's Aegis", Outcome: combatstream.OutcomeSucceeded})
	suffix := fmt.Sprintf(" (archon's aegis, %d warded)", warded)
	a.holder.say("You spread a great ward over the whole company."+suffix, "%s spreads a great ward over the whole company."+suffix, "")
	return true
}

// overchannel decides a damage spell's Overchannel or Storm for a cast:
// the percent more damage it deals and the mana it costs. cost is its
// ordinary cost.
func overchannel(c *characters.Character, sp *spells.SpellData, cost int) (pct, mana int, storm bool) {
	fx := c.ClassEffects()
	if fx == nil || sp == nil || (sp.Type != spells.HarmSingle && sp.Type != spells.HarmMulti) {
		return 0, cost, false
	}
	rt := c.RTState()
	if fx.Has(classes.Storm) && !rt.StormUsed && sp.SpellId == "sparks" {
		return 100, cost, true
	}
	every := fx.Int(classes.Overchannel)
	if every < 1 || (rt.OverSpent && combatRound.Load() < rt.OverRound+uint64(every)) {
		return 0, cost, false
	}
	more := cost + (cost+1)/2
	if c.Mana < more {
		return 0, cost, false
	}
	return 50, more, false
}

// announceOverchannel tells of a cast that swells with Overchannel or Storm.
func announceOverchannel(a actor, pct, mana int, storm bool) {
	if storm {
		suffix := " (archmage's storm, double damage)"
		a.holder.say("A storm of arcane power gathers around you."+suffix, "A storm of arcane power gathers around %s."+suffix, "")
		return
	}
	suffix := fmt.Sprintf(" (overchannel, +%d%% damage, %d mana)", pct, mana)
	a.holder.say("You pour more of yourself into the spell."+suffix, "%s pours more of itself into the spell."+strings.ReplaceAll(suffix, "%", "%%"), "")
}

// announceInstantLance tells of a Lance that needs no chant (Phase 38d).
func announceInstantLance(a actor) {
	suffix := " (instant lance, no chant)"
	a.holder.say("The lance forms at once."+suffix, "%s's lance forms at once."+suffix, "")
	// Review: the battle screen names it, as it does Overwatch or Aegis.
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: a.char.RoomId, Source: a.ref, Status: "Instant Lance", Outcome: combatstream.OutcomeSucceeded})
}

// applyEliteAuras gives each standing member the round's Mana Shield (the
// best of its row's holders) and Hearth's Peace (its ward's Evasion).
func applyEliteAuras(side []actor, rowOf func(actor) int) {
	shield := map[int]int{}
	for _, a := range side {
		if fx := a.char.ClassEffects(); fx != nil && a.char.Health >= 1 {
			if row := rowOf(a); row >= 0 {
				shield[row] = max(shield[row], fx.Int(classes.SpellShield))
			}
		}
	}
	for _, a := range side {
		a.char.Aura.SpellResolve = shield[rowOf(a)]
		if rt := a.char.RT; rt != nil && rt.Ward > 0 && rt.WardPeace > 0 && a.char.Health >= 1 {
			a.char.Aura.Evasion += rt.WardPeace
		}
	}
}

// wardAfterBlow finishes what a ward did to a blow: a Reflection that struck
// back, a ward that broke (Mend Charm, a cleansing break), and a fall a
// Ward of Life or a Lich's Bargain turned aside.
func wardAfterBlow(attacker, defender statusHolder, r combat.AttackResult) {
	if r.Ward.Reflect > 0 && attacker.char.Health >= 0 {
		suffix := fmt.Sprintf(" (reflection, %d damage)", r.Ward.Reflect)
		defender.say("Your ward turns the blow back on "+verbatim(attacker.tag())+"."+suffix, "%s's ward turns the blow back on "+verbatim(attacker.tag())+"."+suffix, "")
		if attacker.user != nil {
			events.AddToQueue(events.CharacterVitalsChanged{UserId: attacker.user.UserId})
			roundExtraPlayers = append(roundExtraPlayers, attacker.user.UserId)
		} else {
			roundExtraMobs = append(roundExtraMobs, attacker.mob.InstanceId)
		}
	}
	rt := defender.char.RT
	if r.Ward.Broke && rt != nil {
		var notes []string
		if n := rt.WardMend; n > 0 && defender.char.Health >= 1 {
			if healed := defender.char.ApplyHealthChange(n); healed > 0 {
				notes = append(notes, fmt.Sprintf("mend charm, %d healed", healed))
			}
		}
		if rt.WardCleanse && defender.char.Health >= 1 {
			if word := status.CleanseOne(defender.char); word != "" {
				notes = append(notes, word+" removed")
			}
		}
		rt.WardMend, rt.WardCleanse, rt.WardPeace, rt.WardLifeBy, rt.WardReflectBy = 0, false, 0, nil, nil
		if len(notes) > 0 {
			suffix := " (ward broken"
			for _, n := range notes {
				suffix += ", " + n
			}
			suffix += ")"
			defender.say("The ward gives way, and its charm closes over you."+suffix, "The ward over %s gives way, and its charm closes over them."+suffix, "")
			if defender.user != nil {
				events.AddToQueue(events.CharacterVitalsChanged{UserId: defender.user.UserId})
			}
		}
	}
	if r.Ward.Saved == "" || rt == nil {
		return
	}
	saved := rt.Saved
	rt.Saved = ""
	switch saved {
	case "ward of life":
		defender.say("The ward of life holds you at 1 health.", "The ward of life holds %s at 1 health.", " (ward of life)")
	case "elixir":
		// Phase 39i2: the blow left the ally at 1 health; the elixir brings it
		// up to the Panacean's share of its health.
		by := rt.ElixirBy
		if by == nil || defender.char.Health < 1 {
			return
		}
		target := max(1, defender.char.HealthLimit()*by.ElixirPct/100)
		if defender.char.Health < target {
			defender.char.ApplyHealthChange(target - defender.char.Health)
		}
		note := fmt.Sprintf(" (elixir, %d health, %d of %d left)", defender.char.Health, by.ElixirMax-by.ElixirSpent, by.ElixirMax)
		defender.say("A Panacean's elixir burns down your throat as the blow lands, and you stay on your feet."+note, "A Panacean's elixir keeps %s on their feet."+note, "")
		emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: defender.roomId, Source: defender.ref, Status: `Elixir`, Outcome: combatstream.OutcomeSucceeded})
		if defender.user != nil {
			events.AddToQueue(events.CharacterVitalsChanged{UserId: defender.user.UserId})
		}
	case "bargain":
		crumbled := ""
		for _, id := range company.SummonInstances() {
			m := mobs.GetInstance(id)
			sm := summonInfo(characterOf(m))
			if sm == nil || sm.Kind != summons.Thrall {
				continue
			}
			if (defender.user != nil && sm.OwnerUser == defender.user.UserId) || (defender.mob != nil && sm.OwnerMob == defender.mob.InstanceId) {
				crumbled = named(mobTag(m.Character.Name))
				summons.Dismiss(id)
				break
			}
		}
		suffix := " (lich's bargain, 1 health left)"
		if crumbled != "" {
			suffix = " (lich's bargain, 1 health left, thrall crumbles)"
			defender.say("The bargain is struck: "+verbatim(crumbled)+" crumbles, and you stand at 1 health."+suffix, "The bargain is struck: "+verbatim(crumbled)+" crumbles, and %s stands at 1 health."+suffix, "")
		} else {
			defender.say("The bargain is struck, and you stand at 1 health."+suffix, "The bargain is struck, and %s stands at 1 health."+suffix, "")
		}
	}
}

// eliteFall is what a foe's fall means to the elites in its battle: the
// Necromancer may raise it, Death's Harvest feeds the casters, and a Crone's
// Soul Rot frightens its group.
func eliteFall(m *mobs.Mob) *mobs.Mob {
	if m == nil || summons.IsSummon(m) {
		return m
	}
	if _, _, isCompany := company.LeaderAndKeyForInstance(m.InstanceId); isCompany {
		return m
	}
	room := rooms.LoadRoom(m.Character.RoomId)
	if room == nil {
		return m
	}
	for _, uid := range battle.Players() {
		b, ok := battle.Current(uid)
		u := users.GetByUserId(uid)
		if !ok || u == nil || u.Character == nil || b.RoomId != room.RoomId || u.Character.RoomId != room.RoomId || !b.Has(m.InstanceId) {
			continue
		}
		if !m.Boss {
			summons.NoteFallen(uid, summons.Fallen{MobID: m.MobId, Level: m.Character.Level})
		}
		for _, a := range sideActors(u, room) {
			fx := a.char.ClassEffects()
			if a.char.Health < 1 || fx == nil {
				continue
			}
			if pct := fx.Int(classes.Harvest); pct > 0 {
				if gained := a.char.ApplyManaChange(max(1, a.char.ManaMax.Value*pct/100)); gained > 0 {
					if a.who.userId > 0 {
						events.AddToQueue(events.CharacterVitalsChanged{UserId: a.who.userId})
					}
					suffix := fmt.Sprintf(" (death's harvest, %d mana)", gained)
					a.holder.say("The fallen's last breath feeds you."+suffix, "The fallen's last breath feeds %s."+suffix, "")
				}
			}
		}
		if m.Character.RT != nil && m.Character.RT.SoulRot && combat.Hexed(&m.Character) {
			soulRot(uid, m, room)
		}
	}
	return m
}

// soulRot shakes the group of a hexed foe that fell: each of its standing
// members takes a morale check.
func soulRot(leader int, fallen *mobs.Mob, room *rooms.Room) {
	for _, p := range enemyparty.Parties(room) {
		in := false
		for _, id := range p.Members {
			if id == fallen.InstanceId {
				in = true
			}
		}
		if !in {
			continue
		}
		checked := 0
		for _, id := range standingFoes(enemyparty.Group{Party: p}, room) {
			if id == fallen.InstanceId {
				continue
			}
			events.AddToQueue(events.MoraleCheck{LeaderUserId: leader, MobInstanceId: id})
			checked++
		}
		if checked > 0 {
			room.SendText(fmt.Sprintf("The rot in %s's soul spreads to its fellows. (soul rot, %d morale checks)", named(mobTag(mobName(fallen.InstanceId))), checked))
		}
	}
}

// cursePass runs each round after the statuses tick: a foe a Crone of Ash
// hexed is left exposed a round after its hex ends (Lingering Curse), and
// one hexed three rounds running falls (Crone's Doom).
func cursePass() {
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
		for _, id := range standingFoes(enemyparty.Group{Party: p}, room) {
			foe := mobs.GetInstance(id)
			if foe == nil || foe.Character.RT == nil {
				continue
			}
			rt := foe.Character.RT
			hexed := combat.Hexed(&foe.Character)
			if rt.LingerAt > 0 && uint64(hexes.Default.Round()) >= rt.LingerAt && !hexed {
				rt.LingerAt = 0
				events.AddToQueue(events.Buff{MobInstanceId: id, BuffId: status.Exposed, Source: `spell`, Triggers: hexes.Triggers(1)})
				room.SendText(fmt.Sprintf("The curse lifts from %s, and it stays open. (lingering curse, exposed)", named(mobTag(mobName(id)))))
			}
			if hexed {
				rt.HexStreak++
			} else {
				rt.HexStreak = 0
			}
			owner := rt.CurseBy
			if owner == nil || rt.HexStreak < 3 || !owner.ClassEffects().Has(classes.Doom) {
				continue
			}
			own := owner.RTState()
			if own.DoomUsed || owner.Health < 1 {
				continue
			}
			own.DoomUsed = true
			var dealt int
			if foe.Boss {
				dealt = -foe.Character.ApplyHealthChange(-max(1, foe.Character.HealthMax.Value/10))
			} else {
				dealt = -foe.Character.ApplyHealthChange(-foe.Character.Health)
			}
			foe.Character.TrackPlayerDamage(uid, dealt)
			roundExtraMobs = append(roundExtraMobs, id)
			word := "falls"
			if foe.Boss {
				word = "is gutted"
			}
			room.SendText(fmt.Sprintf("The curse on %s comes due, and it %s. (crone's doom, %d damage)", named(mobTag(mobName(id))), word, dealt))
		}
	}
}
