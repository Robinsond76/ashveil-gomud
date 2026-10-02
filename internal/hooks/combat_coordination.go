package hooks

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/coordination"
	"github.com/GoMudEngine/GoMud/internal/effecttargets"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/engagement"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 33i2: coordinated enemies. An enemy group fights by its
// coordination tier (internal/coordination), fixed when its battle began:
// its focus (keepPartyEngaged), its healers and casters (enemyStrategyPass),
// and its guardians (enemyGuardianFor). Everything here is runtime only,
// on the game loop; the tier, the focus said aloud, and the guards spent
// live on the player's battle (internal/battle).

// groupLeader is the group's leader: its highest-level member able to
// fight, ties to the lowest instance id. 0 when none can fight.
func groupLeader(p mobparty.Party) int {
	best, bestLevel := 0, -1
	for _, id := range p.Members {
		m := mobs.GetInstance(id)
		if m == nil || !canFight(&m.Character) {
			continue
		}
		if m.Character.Level > bestLevel || (m.Character.Level == bestLevel && id < best) {
			best, bestLevel = id, m.Character.Level
		}
	}
	return best
}

// fighterRole reports whether an enemy fights with its weapon by role: a
// fighter or a guardian (healers and casters swing only when they have
// nothing to cast).
func fighterRole(m *mobs.Mob) bool {
	r := m.EnemyRole()
	return r == string(strategy.Fighter) || r == string(strategy.Guardian)
}

// focusFollowers are the group's fighters, besides its leader, who take
// its focus: the tier's share of all its fighters, the leader counted
// when it is one, the rest by ascending instance id.
func focusFollowers(p mobparty.Party, leaderId int, tier coordination.Tier) map[int]bool {
	var fighters []int
	leaderFights := false
	for _, id := range p.Members {
		m := mobs.GetInstance(id)
		if m == nil || !canFight(&m.Character) || !fighterRole(m) {
			continue
		}
		if id == leaderId {
			leaderFights = true
			continue
		}
		fighters = append(fighters, id)
	}
	sort.Ints(fighters)
	total := len(fighters)
	if leaderFights {
		total++
	}
	n := coordination.FocusCount(tier, total)
	if leaderFights {
		n--
	}
	out := map[int]bool{}
	for i := 0; i < n && i < len(fighters); i++ {
		out[fighters[i]] = true
	}
	return out
}

// enemyRule is the rule an enemy re-aims by (its personality's; "" for
// none, the old weakest pick) and the noise it re-aims with: its own,
// raised to its group's tier's floor.
func enemyRule(m *mobs.Mob, tier coordination.Tier) (strategy.Rule, int) {
	rule, noise, ok := m.Personality()
	if !ok {
		return "", coordination.Noise(tier, 0)
	}
	return strategy.Rule(rule), coordination.Noise(tier, noise)
}

// enemyPick picks the member an enemy aims at: by its rule, when it has
// one; else, noise percent of the time a random member in reach, and
// otherwise the weakest legal one, exactly as before Phase 33i2.
func (s companySide) enemyPick(m *mobs.Mob, rule strategy.Rule, noise int, foes []strategy.Foe, candidates []engagement.Combatant, attackerCol int, reach formationcombat.Reach) (int, bool) {
	if rule != "" {
		return strategy.EnemyPick(rule, foes, noise, aimRoll)
	}
	if noise > 0 {
		var pool []int
		for _, f := range foes {
			if f.Reachable {
				pool = append(pool, f.ID)
			}
		}
		if len(pool) > 0 && aimRoll(100) < noise {
			return pool[aimRoll(len(pool))], true
		}
	}
	_, keys := s.combatants()
	legal := func(_, defender engagement.Combatant) bool {
		return s.legalAgainstCompany(attackerCol, keys[defender.ID], reach)
	}
	return engagement.AssignTarget(engagement.Combatant{Col: attackerCol}, candidates, engagement.Weakest, legal)
}

// anyCaster reports whether any member of the company is a healer or a
// caster, or is chanting.
func anyCaster(foes []strategy.Foe) bool {
	for _, f := range foes {
		if f.Caster || f.Chanting {
			return true
		}
	}
	return false
}

// healChanter is a company member chanting a helpful spell that an enemy
// in attackerCol can reach: a veteran company turns onto them.
func (s companySide) healChanter(attackerCol int, reach formationcombat.Reach) (company.MemberKey, bool) {
	chantingHelp := func(c *characters.Character) bool {
		return c != nil && c.Health > 0 && c.Aggro != nil && c.Aggro.Type == characters.SpellCast &&
			effecttargets.Helpful(spells.GetSpell(c.Aggro.SpellInfo.SpellId))
	}
	if chantingHelp(s.leader.Character) && s.legalAgainstCompany(attackerCol, company.LeaderMemberKey, reach) {
		return company.LeaderMemberKey, true
	}
	for _, id := range s.companionIds {
		key := s.companions[id]
		if m := mobs.GetInstance(id); m != nil && chantingHelp(&m.Character) && s.legalAgainstCompany(attackerCol, key, reach) {
			return key, true
		}
	}
	return "", false
}

// memberTag is a company member's name as narration prints it.
func (s companySide) memberTag(key company.MemberKey) string {
	if key == company.LeaderMemberKey {
		return userTag(s.leader.Character.Name)
	}
	for _, id := range s.companionIds {
		if s.companions[id] == key {
			return named(mobTag(mobName(id)))
		}
	}
	return `someone`
}

// announceFocus says a coordinated group's focus aloud when it changes:
// "The band of ruffians closes in on Aria." A rabble, and a lone enemy,
// say nothing.
func (s companySide) announceFocus(p mobparty.Party, room *rooms.Room, leaderId int, focus company.MemberKey, tier coordination.Tier) {
	if !coordination.SpecOf(tier).Announce || len(p.Members) < 2 || focus == "" {
		return
	}
	if !battle.SetEnemyFocus(s.leader.UserId, string(focus)) {
		return
	}
	g, ok := enemyparty.GroupOf(room, leaderId)
	if !ok || g.Solo() {
		return
	}
	room.SendText(focusLine(g.Name, s.memberTag(focus)))
}

// focusLine is a group's focus line, its name made definite.
func focusLine(groupName, target string) string {
	return util.CapitalizeFirst(fmt.Sprintf(`%s closes in on %s.`, definite(groupName), target))
}

// definite turns "a band of ruffians" into "the band of ruffians".
func definite(name string) string {
	for _, article := range []string{"a ", "an ", "the "} {
		if strings.HasPrefix(strings.ToLower(name), article) {
			return "the " + name[len(article):]
		}
	}
	return "the " + name
}

// enemyGuardianFor finds an enemy guardian who steps in for a blow from
// the player's side about to land on struck, in the player's battle:
// a member of struck's group whose role is guardian, able to guard,
// guarding struck now (the most hurt member within one column of it), with
// one of the group's guards left for the battle (its tier's). It spends
// the guard and says so. ok is false when no one steps in.
func enemyGuardianFor(userId int, room *rooms.Room, struck *mobs.Mob) (*mobs.Mob, bool) {
	if room == nil || struck == nil {
		return nil, false
	}
	b, ok := battle.Current(userId)
	if !ok || b.RoomId != room.RoomId || !b.Has(struck.InstanceId) {
		return nil, false
	}
	tier, _ := enemyparty.BattleTier(userId)
	spec := coordination.SpecOf(tier)
	if spec.Guards < 1 || b.EnemyGuards >= spec.Guards {
		return nil, false
	}
	party, ok := enemyparty.PartyOf(room, struck.InstanceId)
	if !ok {
		return nil, false
	}
	var members []*mobs.Mob
	for _, id := range party.Members {
		if m := mobs.GetInstance(id); m != nil && m.Character.Health > 0 && m.Character.RoomId == room.RoomId {
			members = append(members, m)
		}
	}
	struckKey := string(mobparty.MemberKeyFor(struck.InstanceId))
	for _, g := range members {
		if g.InstanceId == struck.InstanceId || g.EnemyRole() != string(strategy.Guardian) {
			continue
		}
		// One that hasn't joined this battle, or is hidden, never steps in
		// (33i2 review finding 3).
		if !b.Has(g.InstanceId) || g.Character.HasBuffFlag("hidden") {
			continue
		}
		if !ableToGuard(guardMember{char: &g.Character, mob: g}) {
			continue
		}
		gKey := mobparty.MemberKeyFor(g.InstanceId)
		guarded := make([]strategy.Guarded, 0, len(members))
		for _, m := range members {
			mKey := mobparty.MemberKeyFor(m.InstanceId)
			guarded = append(guarded, strategy.Guarded{
				Key: string(mKey), HP: m.Character.Health, MaxHP: m.Character.HealthMax.Value,
				InReach: formationcombat.GuardReach(party.Formation, gKey, mKey),
			})
		}
		if w, ok := strategy.GuardWard(string(gKey), "", guarded); !ok || w != struckKey {
			continue
		}
		left, ok := battle.SpendEnemyGuard(userId, spec.Guards)
		if !ok {
			return nil, false
		}
		guardian, ward := named(mobTag(mobName(g.InstanceId))), named(mobTag(mobName(struck.InstanceId)))
		room.SendText(guardLine(guardian, ward, left))
		emitCombat(combatstream.Event{Kind: combatstream.GuardUsed, FightID: b.FightID, RoomId: room.RoomId, Source: mobRef(g), Target: mobRef(struck)})
		if left == 0 {
			emitCombat(combatstream.Event{Kind: combatstream.GuardExhausted, FightID: b.FightID, RoomId: room.RoomId, Source: mobRef(g)})
		}
		return g, true
	}
	return nil, false
}

// enemyStrategyPass lets every enemy healer and caster in a battle cast by
// its role, as strategyPass does for a company, before any blow: a healer
// heals a member of its group below its tier's threshold, a caster casts at
// the company. They pay mana, chant, and can be interrupted as anyone does.
// A rabble starts at most one heal a round. An enemy fighting in two
// players' battles decides once. Enemy healers use single-target heals
// only: a group heal's friendly scope reaches only a company.
func enemyStrategyPass() {
	autoSpells := costedAutoSpells()
	decided := map[int]bool{}
	healsBy := map[string]int{} // heals started this round, by group (33i2 review finding 2)
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
		targets := companyTargets(u, room)
		if len(targets) == 0 {
			continue
		}
		tier, _ := enemyparty.BattleTier(uid)
		spec := coordination.SpecOf(tier)
		side := enemyActors(b, p, room)
		allies := make([]strategy.Ally, len(side))
		for i, a := range side {
			allies[i] = strategy.Ally{HP: a.char.Health, MaxHP: a.char.HealthLimit()}
		}
		markPendingHeals(allies, side, autoSpells)
		for _, a := range side {
			role := strategy.Role(a.holder.mob.EnemyRole())
			if decided[a.who.mobId] || (role != strategy.Healer && role != strategy.Caster) {
				continue
			}
			if a.char.Aggro == nil || !readyToCast(a, u) {
				continue // not in the fight yet, or busy
			}
			decided[a.who.mobId] = true
			action := strategy.Decide(strategy.Situation{
				Role:      role,
				Mana:      a.char.Mana,
				Knows:     a.knows,
				Spells:    autoSpells,
				Allies:    allies,
				Foes:      len(targets),
				HealBelow: spec.HealBelow,
				MaxMana:   a.char.ManaMax.Value,
			})
			heal := action.Kind == strategy.Heal || action.Kind == strategy.HealAll
			if heal && spec.HealsPerRound > 0 && healsBy[p.ID] >= spec.HealsPerRound {
				continue
			}
			info, ok := enemySpellTargets(action, a, side, b, targets, spec)
			if !ok {
				continue
			}
			if startCast(a, action.Spell, info, room.RoomId) {
				coverHeal(allies, action)
				if heal {
					healsBy[p.ID]++
				}
			}
		}
	}
}

// enemyActors are the battle group's living members in the room, as
// actors: an enemy knows a spell its spellbook holds, but not a group
// heal.
func enemyActors(b battle.Battle, p mobparty.Party, room *rooms.Room) []actor {
	groupHeals := map[string]bool{}
	for _, sp := range strategy.AutoSpells() {
		if sp.Use == strategy.UseHealAll {
			groupHeals[sp.ID] = true
		}
	}
	var out []actor
	for _, id := range p.Members {
		m := mobs.GetInstance(id)
		if m == nil || !b.Has(id) || m.Character.Health < 1 || m.Character.RoomId != room.RoomId {
			continue
		}
		char := &m.Character
		out = append(out, actor{
			who:    caster{mobId: id},
			char:   char,
			ref:    mobRef(m),
			key:    mobparty.MemberKeyFor(id),
			knows:  func(spellId string) bool { return !groupHeals[spellId] && char.HasSpell(spellId) },
			holder: mobHolder(m),
		})
	}
	return out
}

// companyTarget is a member of the player's side an enemy spell may aim at.
type companyTarget struct {
	key    company.MemberKey
	userId int
	mobId  int
}

// companyTargets are the player (standing) and their living companions in
// the room, visible to an enemy.
func companyTargets(u *users.UserRecord, room *rooms.Room) []companyTarget {
	var out []companyTarget
	if u.Character.Health > 0 && !u.Character.HasBuffFlag("hidden") {
		out = append(out, companyTarget{key: company.LeaderMemberKey, userId: u.UserId})
	}
	for _, id := range room.GetMobs(rooms.FindCharmed) {
		leaderId, key, ok := company.LeaderAndKeyForInstance(id)
		m := mobs.GetInstance(id)
		if !ok || leaderId != u.UserId || m == nil || m.Character.Health < 1 || m.Character.HasBuffFlag("hidden") {
			continue
		}
		out = append(out, companyTarget{key: key, mobId: id})
	}
	return out
}

// enemySpellTargets turns an enemy's decision into its spell's targets: a
// heal on the ally chosen, an attack at the group's focus when its tier
// casts there, else at the caster's own aim, else at the player; an area
// spell at the whole company here.
func enemySpellTargets(action strategy.Action, a actor, side []actor, b battle.Battle, targets []companyTarget, spec coordination.Spec) (characters.SpellAggroInfo, bool) {
	info := characters.SpellAggroInfo{SpellId: action.Spell, TargetUserIds: []int{}, TargetMobInstanceIds: []int{}}
	add := func(t companyTarget) {
		if t.userId > 0 {
			info.TargetUserIds = append(info.TargetUserIds, t.userId)
		} else {
			info.TargetMobInstanceIds = append(info.TargetMobInstanceIds, t.mobId)
		}
	}
	switch action.Kind {
	case strategy.Heal:
		if action.Ally < 0 || action.Ally >= len(side) {
			return info, false
		}
		info.TargetMobInstanceIds = append(info.TargetMobInstanceIds, side[action.Ally].who.mobId)
	case strategy.Attack:
		pick := -1
		if spec.SpellsAtFocus && b.EnemyFocus != "" {
			for i, t := range targets {
				if string(t.key) == b.EnemyFocus {
					pick = i
				}
			}
		}
		if pick < 0 && a.char.Aggro != nil {
			for i, t := range targets {
				if (t.userId > 0 && t.userId == a.char.Aggro.UserId) || (t.mobId > 0 && t.mobId == a.char.Aggro.MobInstanceId) {
					pick = i
				}
			}
		}
		if pick < 0 {
			pick = 0
		}
		add(targets[pick])
	case strategy.AttackAll:
		for _, t := range targets {
			add(t)
		}
	default:
		return info, false
	}
	return info, len(info.TargetUserIds)+len(info.TargetMobInstanceIds) > 0
}
