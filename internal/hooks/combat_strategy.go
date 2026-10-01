package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/effecttargets"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/engagement"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/strategy"
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

// actor is one character on a player's side in a battle.
type actor struct {
	who   caster
	char  *characters.Character
	ref   combatstream.Ref
	key   company.MemberKey
	knows func(spellId string) bool
	att   enemyparty.Attacker
}

// strategyPass lets every healer and caster in a battle cast, by its
// strategy.
func strategyPass() {
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
		}
		for _, a := range side {
			if !readyToCast(a, u) {
				continue
			}
			role := enemyparty.MemberStrategy(uid, a.key).Role
			// A guardian fights as a fighter (Phase 30c2).
			if role == strategy.Fighter || role == strategy.Guardian {
				continue
			}
			action := strategy.Decide(strategy.Situation{
				Role:   role,
				Mana:   a.char.Mana,
				Knows:  a.knows,
				Spells: autoSpells,
				Allies: allies,
				Foes:   len(foes),
				// Phase 30c: the company's healing threshold.
				HealBelow: healBelow,
			})
			info, ok := autoSpellTargets(action, a, side, g, foes)
			if !ok {
				continue
			}
			startCast(a, action.Spell, info, room.RoomId)
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
			out = append(out, sp)
		}
	}
	return out
}

// standingFoes are the battle group's living, visible members in the room.
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
				return u.Character.GetSkillLevel(`cast`) > 0 && u.Character.HasSpell(id)
			},
			att: enemyparty.PlayerAttacker(u),
		})
	}
	assist := 0
	if a := u.Character.Aggro; plainAttack(a) {
		assist = a.MobInstanceId
	}
	for _, instanceId := range room.GetMobs(rooms.FindCharmed) {
		leaderId, key, ok := company.LeaderAndKeyForInstance(instanceId)
		m := mobs.GetInstance(instanceId)
		if !ok || leaderId != u.UserId || m == nil || m.Character.Health < 1 {
			continue
		}
		arch := ""
		if id, ok := company.CompanionIDFromMemberKey(key); ok {
			arch, _ = company.CompanionArchetype(u.UserId, id)
		}
		known := map[string]bool{}
		for _, id := range archetypes.CompanionSpells(arch, m.Character.Level) {
			known[id] = true
		}
		out = append(out, actor{
			who:   caster{mobId: instanceId},
			char:  &m.Character,
			ref:   mobRef(m),
			key:   key,
			knows: func(id string) bool { return known[id] },
			att:   enemyparty.CompanionAttacker(u.UserId, key, m, assist),
		})
	}
	return out
}

// readyToCast reports whether a character can start a spell this round:
// alive, able to fight, not already chanting or fleeing, and (for the
// player) not stood down.
func readyToCast(a actor, u *users.UserRecord) bool {
	if !canFight(a.char) || nerveSkip[a.who.mobId] {
		return false
	}
	// Phase 30a: a status that costs its holder the action costs the cast.
	if _, lost := status.LostAction(a.char); lost {
		return false
	}
	if agg := a.char.Aggro; agg != nil && (agg.Type == characters.SpellCast || agg.Type == characters.Flee) {
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
	case strategy.Attack:
		att := a.att
		att.Spell = true // a spell reaches anyone
		id, ok := enemyparty.Aim(g, att)
		if !ok {
			return info, false
		}
		info.TargetMobInstanceIds = append(info.TargetMobInstanceIds, id)
	case strategy.AttackAll:
		info.TargetMobInstanceIds = append(info.TargetMobInstanceIds, foes...)
	default:
		return info, false
	}
	return info, len(info.TargetUserIds)+len(info.TargetMobInstanceIds) > 0
}

// startCast begins a spell as the cast command does: the spell's onCast
// (its chant line; a script may refuse), the mana, the chant rounds, and
// the cast-start event. The caster's aim is remembered, to turn back to.
func startCast(a actor, spellId string, info characters.SpellAggroInfo, roomId int) {
	sp := spells.GetSpell(spellId)
	if sp == nil || a.char.Mana < sp.Cost {
		return
	}
	info = effecttargets.Resolve(a.who.userId, a.who.mobId, info)
	if effecttargets.Helpful(sp) && len(info.TargetUserIds)+len(info.TargetMobInstanceIds) == 0 {
		return
	}
	proceed := true
	if ok, err := scripting.TrySpellScriptEvent(`onCast`, a.who.userId, a.who.mobId, info); err == nil {
		proceed = ok
	}
	if !proceed {
		return
	}
	if agg := a.char.Aggro; plainAttack(agg) && agg.MobInstanceId > 0 {
		castAims[a.who] = agg.MobInstanceId
	}
	a.char.Mana -= sp.Cost
	a.char.SetCast(sp.WaitRounds, info)
	emitCast(combatstream.CastStart, a.ref, spellId, ``, roomId)
	if a.who.userId > 0 {
		events.AddToQueue(events.SkillUsed{UserId: a.who.userId, Skill: `cast`, Details: spellId})
		events.AddToQueue(events.CharacterVitalsChanged{UserId: a.who.userId})
		events.AddToQueue(events.AggroChanged{UserId: a.who.userId, RoomId: roomId})
		return
	}
	events.AddToQueue(events.AggroChanged{MobInstanceId: a.who.mobId, RoomId: roomId})
}

// endCast ends a character's spell (cast, fizzled, or held): it turns
// back to the foe it was aimed at before an automatic spell, if that foe
// still stands here, and otherwise has no aim, as before (the next round's
// upkeep turns it by its rule).
func endCast(c *characters.Character, who caster) {
	c.Aggro = nil
	foe, ok := castAims[who]
	delete(castAims, who)
	if !ok {
		return
	}
	if m := mobs.GetInstance(foe); m != nil && m.Character.Health > 0 && m.Character.RoomId == c.RoomId && !m.Character.HasBuffFlag("hidden") {
		c.SetAggro(0, foe, characters.DefaultAttack)
	}
}

// pruneCastAims forgets remembered aims of characters no longer chanting
// (gone, or their spell was cut short some other way).
func pruneCastAims() {
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
