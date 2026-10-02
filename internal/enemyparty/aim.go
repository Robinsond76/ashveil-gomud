package enemyparty

import (
	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 32d: each character aims by its strategy's target rule (the
// owner's rule 6: a character's strategy sets its target, and if it can't
// reach that one it attacks one it can reach, by formation).

// Attacker is a character aiming at an enemy group: a player (Key
// company.LeaderMemberKey) or one of their companions.
type Attacker struct {
	LeaderId int               // the player whose side it fights on
	Key      company.MemberKey // its place in the player's formation
	Char     *characters.Character
	MobReach bool          // a companion mob's own reach flag
	Rule     strategy.Rule // its target rule
	AssistId int           // the player's target, for strategy.Assist
	Spell    bool          // aiming a spell: every foe is in reach
}

// PlayerAttacker is a player aiming by their company's focus, else their
// own strategy.
func PlayerAttacker(u *users.UserRecord) Attacker {
	return Attacker{
		LeaderId: u.UserId,
		Key:      company.LeaderMemberKey,
		Char:     u.Character,
		Rule:     AimRule(u.UserId, company.LeaderMemberKey),
	}
}

// CompanionAttacker is a player's companion aiming by the company's focus,
// else its own strategy; assistId is the player's target.
func CompanionAttacker(leaderId int, key company.MemberKey, mob *mobs.Mob, assistId int) Attacker {
	return Attacker{
		LeaderId: leaderId,
		Key:      key,
		Char:     &mob.Character,
		MobReach: mob.Reach,
		Rule:     AimRule(leaderId, key),
		AssistId: assistId,
	}
}

// Focus is the company focus leaderId's side aims by now (Phase 30c): the
// focus ordered for their battle, else their saved tactics' focus. ok is
// false for none (each member by its own rule).
func Focus(leaderId int) (strategy.Rule, bool) {
	if r, set := battle.Focus(leaderId); set {
		rule := strategy.Rule(r)
		if rule == "" || rule == strategy.NoFocus {
			return "", false
		}
		return rule, true
	}
	return strategy.TacticsFor(leaderId).FocusRule()
}

// AimRule is the target rule a company member aims by: the company's
// focus when one is set, else the member's own strategy rule. Roles are
// untouched (a healer still heals).
func AimRule(leaderId int, key company.MemberKey) strategy.Rule {
	if rule, ok := Focus(leaderId); ok {
		return rule
	}
	return MemberStrategy(leaderId, key).Rule
}

// MemberStrategy is a company member's strategy, resolved against its
// archetype's default.
func MemberStrategy(leaderId int, key company.MemberKey) strategy.Strategy {
	arch := ""
	if key == company.LeaderMemberKey {
		arch, _ = archetypes.PlayerArchetype(leaderId)
	} else if id, ok := company.CompanionIDFromMemberKey(key); ok {
		arch, _ = company.CompanionArchetype(leaderId, id)
	}
	return strategy.For(leaderId, string(key), arch)
}

// Foes are g's living, visible members as the attacker sees them: where
// they stand, whether it can reach them, the group's leader, and whom of
// the attacker's side each strikes.
func Foes(g Group, a Attacker) []strategy.Foe {
	alive := Alive(g.Party)
	col, placed := 0, false
	if f, ok := CompanyFormation(a.LeaderId); ok {
		_, col, placed = f.Find(a.Key)
	}
	reach := formationcombat.Reach(0)
	if a.Char != nil {
		reach = combat.ResolveReach(a.Char, a.MobReach)
	}
	var room *rooms.Room
	if a.Char != nil {
		room = rooms.LoadRoom(a.Char.RoomId)
	}
	leaderFound := false
	var out []strategy.Foe
	for _, id := range g.Party.Members {
		m := mobs.GetInstance(id)
		if m == nil || m.Character.Health < 1 || m.Character.HasBuffFlag("hidden") {
			continue
		}
		key := mobparty.MemberKeyFor(id)
		row, mcol, _ := g.Party.Formation.Find(key)
		f := strategy.Foe{
			ID:         id,
			HP:         m.Character.Health,
			MaxHP:      m.Character.HealthMax.Value,
			Row:        row,
			Col:        mcol,
			Reachable:  a.Spell || !placed || Legal(room, a.LeaderId, col, g.Party.Formation, key, alive, reach),
			StrikesPct: StrikesPct(m.Character.Aggro, a.LeaderId),
			Chanting:   m.Character.Aggro != nil && m.Character.Aggro.Type == characters.SpellCast,
			Caster:     len(m.Character.SpellBook) > 0,
		}
		// Members are ranked toughest first: the first standing is the
		// leader, and the next steps up when it falls.
		if !leaderFound {
			f.Leader, leaderFound = true, true
		}
		out = append(out, f)
	}
	return out
}

// StrikesPct is the health percentage of the member of leaderId's side an
// enemy's aim strikes, or -1 when it strikes none of them.
func StrikesPct(a *characters.Aggro, leaderId int) int {
	if a == nil {
		return -1
	}
	// A foe chanting a harmful spell strikes its first target (32d review).
	if a.Type == characters.SpellCast {
		spell := *a
		spell.UserId, spell.MobInstanceId = 0, 0
		if len(a.SpellInfo.TargetUserIds) > 0 {
			spell.UserId = a.SpellInfo.TargetUserIds[0]
		} else if len(a.SpellInfo.TargetMobInstanceIds) > 0 {
			spell.MobInstanceId = a.SpellInfo.TargetMobInstanceIds[0]
		} else {
			return -1
		}
		if !harmfulSpell(a.SpellInfo.SpellId) {
			return -1
		}
		a = &spell
	}
	if a.UserId > 0 {
		if a.UserId != leaderId {
			return -1
		}
		if u := users.GetByUserId(leaderId); u != nil && u.Character.Health > 0 {
			return strategy.Percent(u.Character.Health, u.Character.HealthMax.Value)
		}
		return -1
	}
	if owner, _, ok := company.LeaderAndKeyForInstance(a.MobInstanceId); ok && owner == leaderId {
		if m := mobs.GetInstance(a.MobInstanceId); m != nil && m.Character.Health > 0 {
			return strategy.Percent(m.Character.Health, m.Character.HealthMax.Value)
		}
	}
	return -1
}

// Aim is the member of g the attacker strikes, by its rule: the rule's
// choice among the foes it can reach, else the nearest it can reach, else
// the front-most. ok is false when no member stands.
func Aim(g Group, a Attacker) (int, bool) {
	return strategy.Pick(a.Rule, Foes(g, a), a.AssistId, a.Spell)
}

// RuleChoice is the rule's own choice among the foes the attacker can
// reach, with no fallback: ok is false when the rule has none (its choice
// is out of reach, or it follows something that isn't there). The upkeep
// uses it to move a rule that follows something (assist, defend) only
// when the rule itself points elsewhere.
//
// current is the foe the attacker is on: under defend, a current foe that
// strikes someone as hurt as the rule's choice does is kept (32d review:
// no turning on ties).
func RuleChoice(g Group, a Attacker, current int) (int, bool) {
	var pool []strategy.Foe
	pct := map[int]int{}
	for _, f := range Foes(g, a) {
		if f.Reachable {
			pool = append(pool, f)
			pct[f.ID] = f.StrikesPct
		}
	}
	if len(pool) == 0 {
		return 0, false
	}
	choice, ok := strategy.Choose(a.Rule, pool, a.AssistId)
	if ok && a.Rule == strategy.Defend && choice != current {
		if p, here := pct[current]; here && p >= 0 && p == pct[choice] {
			return current, true
		}
	}
	return choice, ok
}

func harmfulSpell(id string) bool {
	s := spells.GetSpell(id)
	return s != nil && (s.Type == spells.HarmSingle || s.Type == spells.HarmMulti || s.Type == spells.HarmArea)
}
