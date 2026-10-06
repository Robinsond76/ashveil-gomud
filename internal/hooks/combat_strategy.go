package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/hexes"
	"slices"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/effecttargets"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/engagement"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/flasks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/summons"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 32d: healers and casters. A battle plays out on its own (the
// owner's rule 5): at the start of each round, after the battle pass and
// the upkeep and before any blow, every character in a battle whose role
// is healer or caster decides whether to cast this round (a heal on the
// hurt, an attack spell at its rule's choice) by strategy.Decide, and
// starts the chant exactly as the cast command would. A fighter, or a
// healer or caster with nothing to cast, swings at its aim.
//
// A caster's aim before the spell is remembered, and it turns back to it,
// without a "turns toward" line, when the spell ends (castAims). Runtime
// only, on the game loop, like the battles themselves.

// caster identifies a character casting in a battle: a player or a mob.
type caster struct {
	userId, mobId int
}

// playerDeathHealth is the health at which a downed player dies
// (NewRound_AutoHeal: at -10 they die).
const playerDeathHealth = -10

// castAims holds the foe each automatic caster was aimed at before its
// spell, to turn back to when the spell ends. Game loop only.
var castAims = map[caster]int{}

// enemyCastAims holds the player each enemy caster (by instance id) was
// aimed at before its automatic spell (Phase 33i2). Game loop only.
var enemyCastAims = map[int]int{}

// actor is one character on a player's side in a battle.
type actor struct {
	who   caster
	char  *characters.Character
	ref   combatstream.Ref
	key   company.MemberKey
	knows func(spellId string) bool
	att   enemyparty.Attacker
	// holder is the actor for the status, interrupt, and narration
	// helpers; archetype a companion's (Phase 33e: its abilities).
	holder    statusHolder
	archetype string
}

// strategyPass lets every healer and caster in a battle cast, by its
// strategy.
func strategyPass() {
	defer func() { hexSide = nil }()
	pruneCastAims()
	autoSpells := costedAutoSpells()
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
		g := enemyparty.Group{Party: p}
		foes := standingFoes(g, room)
		if len(foes) == 0 {
			continue
		}
		side := sideActors(u, room)
		healBelow := strategy.TacticsFor(uid).Healing
		allies := make([]strategy.Ally, len(side))
		for i, a := range side {
			// A player who is down (bleeding out, not yet dead) can still
			// be healed (32d review).
			// Phase 30b: a wounded ally is healed only to its limit.
			allies[i] = strategy.Ally{HP: a.char.Health, MaxHP: a.char.HealthLimit(), Downed: a.who.userId > 0 && a.char.Health < 1}
			if rt := a.char.RT; rt != nil {
				allies[i].Warded, allies[i].Barked, allies[i].Rejuv, allies[i].Blessed = rt.Ward > 0 && !rt.WardSigil, rt.Bark > 0, rt.Rejuv > 0, rt.Bless > 0
			}
			// Phase 39g: poison or bleeding an Alchemist's antidote takes off.
			allies[i].Afflicted = a.char.HasBuffFlag("poison") || a.char.HasBuffFlag("bleeding")
		}
		// Phase 33e: a heal already chanting covers its patients, so a
		// second healer turns to someone else.
		markPendingHeals(allies, side, autoSpells)
		hexSide = side
		for _, a := range side {
			if !readyToCast(a, u) {
				continue
			}
			st := enemyparty.MemberStrategy(uid, a.key)
			role := st.Role
			// A guardian fights as a fighter (Phase 30c2).
			if role == strategy.Fighter || role == strategy.Guardian {
				continue
			}
			// Phase 38c3: an Archon holds its turn to counter a chant, or to
			// ward the whole company.
			if archonTurn(a, side, u, foes) {
				continue
			}
			action := strategy.Decide(strategy.Situation{
				Role:   role,
				Mana:   a.char.Mana,
				Knows:  a.knows,
				Spells: classCosted(a.char, autoSpells),
				Allies: allies,
				Foes:   len(foes),
				// Phase 38b: a summoner calls its summon once a battle.
				Summoned: a.char.ClassEffects().Int(classes.Summon) == 0 || (a.char.RT != nil && a.char.RT.Summoned),
				Boss:     anyBoss(foes),
				// Phase 30c: the company's healing threshold.
				HealBelow: healBelow,
				// Phase 33e: the member's mana reserve.
				MaxMana: a.char.ManaMax.Value,
				Reserve: st.Reserve,
				// Phase 38a: a hex goes only at a foe worth it.
				CanHex: hexReady(a, foes),
				// Phase 38c3: a Necromancer raises a foe that has fallen.
				CanRaise: canRaise(a, uid),
				// Phase 39c: a Shaman calls a weather when none is up.
				Weather:    string(b.Weather.Kind),
				CanWeather: weatherReady(a, foes),
				// Phase 39g: an Alchemist throws flasks while its satchel holds them.
				Flasks: flasks.Remaining(a.char),
			})
			info, ok := autoSpellTargets(action, a, side, g, foes)
			if !ok {
				continue
			}
			if startCast(a, action.Spell, info, room.RoomId) {
				coverHeal(allies, action)
			}
		}
	}
}

// costedAutoSpells is the configured automatic spells with their mana
// costs from the spell files; a spell no longer loaded is left out.
func costedAutoSpells() []strategy.Spell {
	var out []strategy.Spell
	for _, sp := range strategy.AutoSpells() {
		if data := spells.GetSpell(sp.ID); data != nil {
			sp.Cost = data.Cost
			sp.Flask = data.Flask // Phase 39g
			out = append(out, sp)
		}
	}
	return out
}

// classCosted is the automatic spells at the caster's own mana costs, which
// its class may change (Phase 38b). A caster with no class gets the list.
func classCosted(c *characters.Character, list []strategy.Spell) []strategy.Spell {
	if c.ClassEffects() == nil {
		return list
	}
	out := make([]strategy.Spell, len(list))
	for i, sp := range list {
		out[i] = sp
		out[i].Cost = c.SpellCost(spells.GetSpell(sp.ID))
	}
	return out
}

// standingFoes are the battle group's living, visible members in the room.
// anyBoss reports whether one of the foes is a boss (Phase 38b review).
func anyBoss(foes []int) bool {
	for _, id := range foes {
		if m := mobs.GetInstance(id); m != nil && m.Boss {
			return true
		}
	}
	return false
}

func standingFoes(g enemyparty.Group, room *rooms.Room) []int {
	var out []int
	for _, id := range g.Party.Members {
		if m := mobs.GetInstance(id); m != nil && m.Character.Health > 0 && m.Character.RoomId == room.RoomId && !m.Character.HasBuffFlag("hidden") {
			out = append(out, id)
		}
	}
	return out
}

// sideActors are the player (standing, or down but not yet dead: they can
// be healed, though they cast nothing) and their living companions in the
// room.
func sideActors(u *users.UserRecord, room *rooms.Room) []actor {
	var out []actor
	if u.Character.Health > playerDeathHealth {
		out = append(out, actor{
			who:  caster{userId: u.UserId},
			char: u.Character,
			ref:  userRef(u),
			key:  company.LeaderMemberKey,
			knows: func(id string) bool {
				return u.Character.GetSkillLevel(`cast`) > 0 && u.Character.KnowsSpell(id)
			},
			att:    enemyparty.PlayerAttacker(u),
			holder: userHolder(u),
		})
	}
	assist := 0
	if a := u.Character.Aggro; plainAttack(a) {
		assist = a.MobInstanceId
	}
	for _, instanceId := range room.GetMobs(rooms.FindCharmed) {
		leaderId, key, ok := company.LeaderAndKeyForInstance(instanceId)
		m := mobs.GetInstance(instanceId)
		if !ok || leaderId != u.UserId || m == nil || m.Character.Health < 1 || isDollInstance(instanceId) {
			continue // Phase 39d: a doll has no turn of its own
		}
		arch := ""
		if id, ok := company.CompanionIDFromMemberKey(key); ok {
			arch, _ = company.CompanionArchetype(u.UserId, id)
		}
		known := map[string]bool{}
		class, _ := m.Character.ClassState()
		for _, id := range archetypes.CompanionKnownSpells(arch, class, m.Character.Level) {
			known[id] = true
		}
		out = append(out, actor{
			who:       caster{mobId: instanceId},
			char:      &m.Character,
			ref:       mobRef(m),
			key:       key,
			knows:     func(id string) bool { return known[id] },
			att:       enemyparty.CompanionAttacker(u.UserId, key, m, assist),
			holder:    mobHolder(m),
			archetype: arch,
		})
	}
	return out
}

// readyToCast reports whether a character can start a spell this round:
// alive, able to fight, not already chanting or fleeing, and (for the
// player) not stood down.
func readyToCast(a actor, u *users.UserRecord) bool {
	if !canFight(a.char) || nerveSkip[a.who.mobId] || surprised(a.who.userId, a.who.mobId) {
		return false
	}
	// Phase 30a: a status that costs its holder the action costs the cast.
	if _, lost := status.LostAction(a.char); lost {
		return false
	}
	if agg := a.char.Aggro; agg != nil && (agg.Type == characters.SpellCast || agg.Type == characters.Flee || agg.Type == characters.Retreat) {
		return false
	}
	if a.who.userId > 0 && a.char.Aggro == nil && engagement.StoodDown(u.UserId) {
		return false
	}
	return true
}

// autoSpellTargets turns a decision into the spell's targets. ok is false for
// a swing, or when there is no one to cast at.
func autoSpellTargets(action strategy.Action, a actor, side []actor, g enemyparty.Group, foes []int) (characters.SpellAggroInfo, bool) {
	info := characters.SpellAggroInfo{SpellId: action.Spell, TargetUserIds: []int{}, TargetMobInstanceIds: []int{}}
	add := func(t actor) {
		if t.who.userId > 0 {
			info.TargetUserIds = append(info.TargetUserIds, t.who.userId)
		} else {
			info.TargetMobInstanceIds = append(info.TargetMobInstanceIds, t.who.mobId)
		}
	}
	switch action.Kind {
	case strategy.Heal:
		if action.Ally < 0 || action.Ally >= len(side) {
			return info, false
		}
		add(side[action.Ally])
	case strategy.HealAll:
		for _, t := range side {
			add(t)
		}
	case strategy.Buff:
		if action.Ally < 0 || action.Ally >= len(side) {
			return info, false
		}
		add(side[action.Ally])
	case strategy.Summon, strategy.Raise, strategy.Weather:
		add(a) // the call has no target; the caster stands for it
	case strategy.Row:
		if action.Ally < 0 || action.Ally >= len(side) {
			return info, false
		}
		add(side[action.Ally]) // the spell's script reaches the row
	case strategy.Drain:
		att := a.att
		att.Spell = true
		id, ok := enemyparty.Aim(g, att)
		if !ok {
			return info, false
		}
		info.TargetMobInstanceIds = append(info.TargetMobInstanceIds, id)
		if a.char.ClassEffects().Int(classes.Siphon) >= 2 {
			for _, other := range foes {
				if other != id {
					info.TargetMobInstanceIds = append(info.TargetMobInstanceIds, other)
					break
				}
			}
		}
	case strategy.Flame:
		att := a.att
		att.Spell = true // a flask reaches anyone
		id, ok := enemyparty.Aim(g, att)
		if !ok {
			return info, false
		}
		info.TargetMobInstanceIds = append(info.TargetMobInstanceIds, flameTargets(a, g, foes, id)...)
	case strategy.Attack, strategy.Storm:
		att := a.att
		att.Spell = true // a spell reaches anyone
		id, ok := enemyparty.Aim(g, att)
		if !ok {
			return info, false
		}
		info.TargetMobInstanceIds = append(info.TargetMobInstanceIds, id)
		// Phase 38c3: an Archmage's Arcane Barrage reaches a second foe; Phase 39c:
		// a Stormcaller's Lightning chains to a second foe.
		if (action.Spell == "mm" && a.char.ClassEffects().Has(classes.Barrage)) ||
			(action.Kind == strategy.Storm && a.char.ClassEffects().Int(classes.Chain) > 0) ||
			(action.Spell == "arcanelance" && a.char.ClassEffects().Int(classes.LanceTwin) > 0) {
			for _, other := range foes {
				if other != id {
					info.TargetMobInstanceIds = append(info.TargetMobInstanceIds, other)
					break
				}
			}
		}
	case strategy.AttackAll:
		info.TargetMobInstanceIds = append(info.TargetMobInstanceIds, foes...)
	case strategy.Hex:
		info.TargetMobInstanceIds = append(info.TargetMobInstanceIds, hexTargets(action.Spell, a, g, foes)...)
	default:
		return info, false
	}
	return info, len(info.TargetUserIds)+len(info.TargetMobInstanceIds) > 0
}

// startCast begins a spell as the cast command does: the spell's onCast
// (its chant line; a script may refuse), the mana, the chant rounds, and
// the cast-start event. The caster's aim is remembered, to turn back to.
func startCast(a actor, spellId string, info characters.SpellAggroInfo, roomId int) bool {
	sp := spells.GetSpell(spellId)
	if sp == nil || a.char.Mana < a.char.SpellCost(sp) || flasks.Remaining(a.char) < sp.Flask {
		return false
	}
	info = effecttargets.Resolve(a.who.userId, a.who.mobId, info)
	if effecttargets.Helpful(sp) && len(info.TargetUserIds)+len(info.TargetMobInstanceIds) == 0 {
		return false
	}
	proceed := true
	if ok, err := scripting.TrySpellScriptEvent(`onCast`, a.who.userId, a.who.mobId, info); err == nil {
		proceed = ok
	}
	if !proceed {
		return false
	}
	if agg := a.char.Aggro; plainAttack(agg) && agg.MobInstanceId > 0 {
		castAims[a.who] = agg.MobInstanceId
	} else if plainAttack(agg) && agg.UserId > 0 && a.who.mobId > 0 {
		// Phase 33i2: an enemy aimed at a player turns back to them.
		enemyCastAims[a.who.mobId] = agg.UserId
	}
	cost := a.char.SpellCost(sp)
	// Phase 38c3: an Archmage's Overchannel (more damage for more mana) or
	// its Storm (double damage, once a battle, free).
	if pct, mana, storm := overchannel(a.char, sp, cost); pct > 0 {
		info.Over, cost = pct, mana
		if rt := a.char.RTState(); storm {
			rt.StormUsed = true
		} else {
			rt.OverSpent, rt.OverRound = true, combatRound.Load()
		}
		announceOverchannel(a, pct, mana, storm)
	}
	a.char.Mana -= cost
	a.char.FlasksSpent += sp.Flask // Phase 39g: a thrown flask is used up
	wait := sp.WaitRounds
	if sp.Type == spells.HarmSingle || sp.Type == spells.HarmMulti {
		// Phase 38c3: Quick casting trims every other damage spell's chant.
		if trim := a.char.ClassEffects().Int(classes.ChantTrim); trim > 0 {
			rt := a.char.RTState()
			rt.QuickCasts++
			if rt.QuickCasts%2 == 1 {
				wait = max(0, wait-trim)
			}
		}
	}
	if sp.SpellId == "arcanelance" {
		// Phase 38d: a High Sorcerer's Gathered chant trims every other Lance
		// (review: trimming each one made a one-round Lance every round), and
		// its Instant Lance needs no chant, once a battle.
		fx := a.char.ClassEffects()
		rt := a.char.RTState()
		if trim := fx.Int(classes.LanceTrim); trim > 0 {
			rt.QuickCasts++
			if rt.QuickCasts%2 == 1 {
				wait = max(0, wait-trim)
			}
		}
		if fx.Has(classes.LanceFree) && !rt.LanceFreed {
			rt.LanceFreed = true
			wait = 0
			announceInstantLance(a)
		}
	}
	if sp.SpellId == "callhost" || sp.SpellId == "bindfiend" {
		wait = max(0, wait-a.char.ClassEffects().Int(classes.SummonSooner)) // Swift Host, Mastered binding
	}
	if _, isHex := hexes.For(spellId); isHex {
		wait = max(0, wait-a.char.ClassEffects().Int(classes.HexChant)) // Phase 38b: a Witch's quicker chant
		// Phase 38c3: a Crone's Quick curses trims every other hex's chant.
		if trim := a.char.ClassEffects().Int(classes.HexQuick); trim > 0 {
			rt := a.char.RTState()
			rt.QuickCasts++
			if rt.QuickCasts%2 == 1 {
				wait = max(0, wait-trim)
			}
		}
	}
	a.char.SetCast(wait, info)
	if tempoActive {
		tempoChanted[a.who], tempoBlocked[a.who] = true, true
	}
	emitCast(combatstream.CastStart, a.ref, spellId, ``, roomId)
	if a.who.mobId > 0 {
		noteActed(a.char)
		if m := mobs.GetInstance(a.who.mobId); m != nil {
			chantBegun(m)
		}
	}
	if a.who.userId > 0 {
		events.AddToQueue(events.SkillUsed{UserId: a.who.userId, Skill: `cast`, Details: spellId})
		events.AddToQueue(events.CharacterVitalsChanged{UserId: a.who.userId})
		events.AddToQueue(events.AggroChanged{UserId: a.who.userId, RoomId: roomId})
		return true
	}
	events.AddToQueue(events.AggroChanged{MobInstanceId: a.who.mobId, RoomId: roomId})
	return true
}

// markPendingHeals marks the allies an automatic heal already chanting on
// the side covers (Phase 33e): its patients, or everyone for a group heal.
func markPendingHeals(allies []strategy.Ally, side []actor, autoSpells []strategy.Spell) {
	uses := map[string]strategy.Use{}
	for _, sp := range autoSpells {
		uses[sp.ID] = sp.Use
	}
	for _, a := range side {
		agg := a.char.Aggro
		if agg == nil || agg.Type != characters.SpellCast {
			continue
		}
		switch uses[agg.SpellInfo.SpellId] {
		case strategy.UseHealAll:
			coverHeal(allies, strategy.Action{Kind: strategy.HealAll})
		case strategy.UseWard, strategy.UseBark, strategy.UseBless:
			for i, t := range side {
				if (t.who.userId > 0 && slices.Contains(agg.SpellInfo.TargetUserIds, t.who.userId)) ||
					(t.who.mobId > 0 && slices.Contains(agg.SpellInfo.TargetMobInstanceIds, t.who.mobId)) {
					switch uses[agg.SpellInfo.SpellId] {
					case strategy.UseWard:
						allies[i].Warded = true
					case strategy.UseBark:
						allies[i].Barked = true
					default:
						allies[i].Blessed = true
					}
				}
			}
		case strategy.UseHeal, strategy.UseBigHeal, strategy.UseRejuv:
			for i, t := range side {
				if (t.who.userId > 0 && slices.Contains(agg.SpellInfo.TargetUserIds, t.who.userId)) ||
					(t.who.mobId > 0 && slices.Contains(agg.SpellInfo.TargetMobInstanceIds, t.who.mobId)) {
					allies[i].Pending = true
				}
			}
		}
	}
}

// coverHeal marks the allies a heal just started covers.
func coverHeal(allies []strategy.Ally, action strategy.Action) {
	switch action.Kind {
	case strategy.Buff:
		if action.Ally >= 0 && action.Ally < len(allies) {
			switch action.Spell {
			case "ward", "arcaneward":
				allies[action.Ally].Warded = true
			case "barkskin", "stoneskin":
				allies[action.Ally].Barked = true
			case "bless", "tonic":
				allies[action.Ally].Blessed = true
			case "antidote":
				allies[action.Ally].Afflicted = false
			}
		}
	case strategy.Heal:
		if action.Ally >= 0 && action.Ally < len(allies) {
			allies[action.Ally].Pending = true
		}
	case strategy.HealAll:
		for i := range allies {
			allies[i].Pending = true
		}
	}
}

// endCast ends a character's spell (cast, fizzled, or held): it turns
// back to the foe it was aimed at before an automatic spell, if that foe
// still stands here, and otherwise has no aim, as before (the next round's
// upkeep turns it by its rule).
func endCast(c *characters.Character, who caster) {
	c.Aggro = nil
	foe, ok := castAims[who]
	delete(castAims, who)
	player, aimedAtPlayer := enemyCastAims[who.mobId]
	if who.mobId > 0 {
		delete(enemyCastAims, who.mobId)
	}
	if !ok {
		if aimedAtPlayer {
			if u := users.GetByUserId(player); u != nil && u.Character.Health > 0 && u.Character.RoomId == c.RoomId && !u.Character.HasBuffFlag("hidden") {
				c.SetAggro(player, 0, characters.DefaultAttack)
			}
		}
		return
	}
	if m := mobs.GetInstance(foe); m != nil && m.Character.Health > 0 && m.Character.RoomId == c.RoomId && !m.Character.HasBuffFlag("hidden") {
		c.SetAggro(0, foe, characters.DefaultAttack)
	}
}

// pruneCastAims forgets remembered aims of characters no longer chanting
// (gone, or their spell was cut short some other way).
func pruneCastAims() {
	for id := range enemyCastAims {
		if m := mobs.GetInstance(id); m == nil || m.Character.Aggro == nil || m.Character.Aggro.Type != characters.SpellCast {
			delete(enemyCastAims, id)
		}
	}
	for who := range castAims {
		var c *characters.Character
		if who.userId > 0 {
			if u := users.GetByUserId(who.userId); u != nil {
				c = u.Character
			}
		} else if m := mobs.GetInstance(who.mobId); m != nil {
			c = &m.Character
		}
		if c == nil || c.Aggro == nil || c.Aggro.Type != characters.SpellCast {
			delete(castAims, who)
		}
	}
}

// canRaise reports whether a character may raise a fallen foe now: its class
// allows another thrall this battle and a foe has fallen.
func canRaise(a actor, leader int) bool {
	n := a.char.ClassEffects().Int(classes.Raise)
	return n > 0 && n > a.char.RTState().Raised && summons.HasFallen(leader)
}
