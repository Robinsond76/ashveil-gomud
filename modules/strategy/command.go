package strategy

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	domain "github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// member is one character a player sets a strategy for: themselves, or a
// companion on their company record (present, awaiting, or dead).
type member struct {
	key       string
	name      string
	archetype string
	isPlayer  bool
	mana      int
	manaMax   int
	present   bool // a player, or a companion here to read mana from
	knows     func(spellID string) bool
}

// env is the engine the command reads. Tests replace it.
type env struct {
	members   func(user *users.UserRecord) ([]member, bool) // ok false: the company can't be read
	inBattle  func(user *users.UserRecord) bool
	spellInfo func(id string) (name string, cost int, ok bool)
	archName  func(id string) string
}

func nativeEnv() env {
	return env{
		members:   nativeMembers,
		inBattle:  usercommands.InBattle,
		spellInfo: nativeSpellInfo,
		archName: func(id string) string {
			if name, ok := archetypes.Name(id); ok {
				return strings.ToLower(name)
			}
			return id
		},
	}
}

func nativeSpellInfo(id string) (string, int, bool) {
	sp := spells.GetSpell(id)
	if sp == nil {
		return "", 0, false
	}
	return sp.Name, sp.Cost, true
}

// PlayerKnows reports whether a player can cast a spell by themselves: the
// cast skill and the spell in their book.
func PlayerKnows(u *users.UserRecord) func(string) bool {
	return func(id string) bool {
		return u.Character.GetSkillLevel(`cast`) > 0 && u.Character.HasSpell(id)
	}
}

// CompanionKnows reports the spells a companion knows: its archetype's at
// its level (Phase 32d).
func CompanionKnows(archetype string, level int) func(string) bool {
	known := map[string]bool{}
	for _, id := range archetypes.CompanionSpells(archetype, level) {
		known[id] = true
	}
	return func(id string) bool { return known[id] }
}

func nativeMembers(user *users.UserRecord) ([]member, bool) {
	arch, _ := archetypes.PlayerArchetype(user.UserId)
	out := []member{{
		key:       string(company.LeaderMemberKey),
		name:      "You",
		archetype: arch,
		isPlayer:  true,
		mana:      user.Character.Mana,
		manaMax:   user.Character.ManaMax.Value,
		present:   true,
		knows:     PlayerKnows(user),
	}}
	views, ok := company.CompanyMembers(user.UserId)
	for _, v := range views {
		mb := member{
			key:       string(company.CompanionMemberKey(v.ID)),
			name:      v.Name,
			archetype: v.Archetype,
			knows:     CompanionKnows(v.Archetype, v.Level),
		}
		if instanceId, live := company.InstanceFor(user.UserId, v.ID); live && v.Status == company.MemberPresent {
			if mob := mobs.GetInstance(instanceId); mob != nil {
				mb.present, mb.mana, mb.manaMax = true, mob.Character.Mana, mob.Character.ManaMax.Value
			}
		}
		out = append(out, mb)
	}
	return out, ok
}

const usage = `Change one with <ansi fg="command">strategy [who] [role]</ansi> or <ansi fg="command">strategy [who] target [rule]</ansi>. See <ansi fg="command">help strategy</ansi>.`

func (m *StrategyModule) userCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	user.SendText(m.run(user, strings.Fields(strings.ToLower(rest))))
	return true, nil
}

// run carries out the command and returns what to show.
func (m *StrategyModule) run(user *users.UserRecord, args []string) string {
	members, ok := m.env.members(user)
	if ok {
		keep := map[string]bool{}
		for _, mb := range members {
			keep[mb.key] = true
		}
		m.prune(user.UserId, keep)
	}
	if len(args) == 0 {
		return m.list(user.UserId, members)
	}
	mb, found := resolve(members, args[0])
	if !found {
		return fmt.Sprintf(`No one in your company answers to "%s". Type <ansi fg="command">strategy</ansi> to see them.`, args[0])
	}
	if len(args) == 1 {
		return m.describe(user.UserId, mb)
	}
	if m.env.inBattle(user) {
		return usercommands.BattleUnderWay
	}
	stored := m.Stored(user.UserId, mb.key)
	change := args[1:]
	if change[0] == "target" || change[0] == "rule" {
		change = change[1:]
		if len(change) == 0 {
			return `Target which way? ` + rulesList()
		}
	}
	word := change[0]
	var next domain.Strategy
	var said string
	switch {
	case word == "default" || word == "reset":
		d := domain.Default(mb.archetype)
		if err := m.set(user.UserId, mb.key, domain.Strategy{}); err != nil {
			return err.Error()
		}
		return fmt.Sprintf(`%s %s back to the default: %s, going for %s.`, mb.name, verb(mb, "are", "is"), d.Role, d.Rule.Describe())
	default:
		if role, ok := domain.ParseRole(word); ok {
			next = domain.Strategy{Role: role, Rule: stored.Rule}
			said = fmt.Sprintf(`%s %s now a %s: %s.`, mb.name, verb(mb, "are", "is"), role, role.Describe())
			if warn := m.cantYet(mb, role); warn != "" {
				said = warn + "\n" + said
			}
		} else if rule, ok := domain.ParseRule(word); ok {
			next = domain.Strategy{Role: stored.Role, Rule: rule}
			if err := next.ValidFor(mb.isPlayer); err != nil {
				return `Only a companion can assist you: you are the one they assist.`
			}
			said = fmt.Sprintf(`%s will go for %s, else the nearest foe in reach.`, mb.name, rule.Describe())
		} else {
			return fmt.Sprintf(`"%s" is neither a role nor a target. Roles: %s. %s`, word, rolesList(), rulesList())
		}
	}
	// A setting equal to the default is stored as blank.
	d := domain.Default(mb.archetype)
	if next.Role == d.Role {
		next.Role = ""
	}
	if next.Rule == d.Rule {
		next.Rule = ""
	}
	if err := m.set(user.UserId, mb.key, next); err != nil {
		return err.Error()
	}
	return said
}

func verb(mb member, you, other string) string {
	if mb.isPlayer {
		return you
	}
	return other
}

func rolesList() string {
	names := make([]string, len(domain.Roles))
	for i, r := range domain.Roles {
		names[i] = string(r)
	}
	return strings.Join(names, ", ")
}

func rulesList() string {
	names := make([]string, len(domain.Rules))
	for i, r := range domain.Rules {
		names[i] = string(r)
	}
	return `Targets: ` + strings.Join(names, ", ") + `.`
}

// resolve finds a member: me/self/you, a companion's #id, its whole name,
// or a unique part of it (as company commands name one).
func resolve(members []member, selector string) (member, bool) {
	switch selector {
	case "me", "self", "you", "myself":
		return members[0], true
	}
	if id, err := strconv.Atoi(strings.TrimPrefix(selector, "#")); err == nil {
		key := string(company.CompanionMemberKey(id))
		for _, mb := range members[1:] {
			if mb.key == key {
				return mb, true
			}
		}
		return member{}, false
	}
	var exact, partial []member
	for _, mb := range members[1:] {
		name := strings.ToLower(mb.name)
		if name == selector {
			exact = append(exact, mb)
		} else if strings.Contains(name, selector) {
			partial = append(partial, mb)
		}
	}
	if len(exact) == 1 {
		return exact[0], true
	}
	if len(exact) == 0 && len(partial) == 1 {
		return partial[0], true
	}
	return member{}, false
}

// spellsFor lists the automatic spells a member knows for a role, with
// costs: "Magic Missile (6 mana)".
func (m *StrategyModule) spellsFor(mb member, role domain.Role) []string {
	var uses []domain.Use
	switch role {
	case domain.Healer:
		uses = []domain.Use{domain.UseHeal, domain.UseHealAll}
	case domain.Caster:
		uses = []domain.Use{domain.UseAttack, domain.UseAttackAll}
	default:
		return nil
	}
	var out []string
	for _, sp := range m.AutoSpells() {
		wanted := false
		for _, u := range uses {
			wanted = wanted || sp.Use == u
		}
		if !wanted || mb.knows == nil || !mb.knows(sp.ID) {
			continue
		}
		if name, cost, ok := m.env.spellInfo(sp.ID); ok {
			out = append(out, fmt.Sprintf(`<ansi fg="spellname">%s</ansi> (%d mana)`, name, cost))
		}
	}
	return out
}

// cantYet warns when a member knows no spell its new role would cast.
func (m *StrategyModule) cantYet(mb member, role domain.Role) string {
	if role == domain.Fighter || len(m.spellsFor(mb, role)) > 0 {
		return ""
	}
	what := "attack spell"
	if role == domain.Healer {
		what = "healing spell"
	}
	return fmt.Sprintf(`%s %s no %s yet, and will fight until learning one.`, mb.name, verb(mb, "know", "knows"), what)
}

func (m *StrategyModule) list(userID int, members []member) string {
	var b strings.Builder
	b.WriteString("How your company fights (set before a battle; it plays out by these):\n")
	width := 0
	for _, mb := range members {
		if len(mb.name) > width {
			width = len(mb.name)
		}
	}
	for _, mb := range members {
		s := m.Stored(userID, mb.key).Resolve(mb.archetype)
		arch := "-"
		if mb.archetype != "" {
			arch = m.env.archName(mb.archetype)
		}
		line := fmt.Sprintf("  %-*s  %-8s %-8s %-9s", width, mb.name, arch, s.Role, s.Rule)
		if sp := m.spellsFor(mb, s.Role); len(sp) > 0 {
			line += " " + strings.Join(sp, ", ")
		} else if s.Role != domain.Fighter {
			line += " (knows no spell for it: fights)"
		}
		b.WriteString(strings.TrimRight(line, " ") + "\n")
	}
	// Phase 30c: a company focus overrides every target rule above.
	if focus, ok := domain.TacticsFor(userID).FocusRule(); ok {
		fmt.Fprintf(&b, "Your company's focus is %s (<ansi fg=\"command\">company tactics</ansi>): instead of these rules, everyone goes for %s.\n", focus, focus.Describe())
	}
	b.WriteString(usage)
	return b.String()
}

func (m *StrategyModule) describe(userID int, mb member) string {
	s := m.Stored(userID, mb.key).Resolve(mb.archetype)
	var b strings.Builder
	arch := "no archetype"
	if mb.archetype != "" {
		arch = m.env.archName(mb.archetype)
	}
	fmt.Fprintf(&b, "%s (%s): %s, %s.\n", mb.name, arch, s.Role, s.Role.Describe())
	fmt.Fprintf(&b, "  Goes for %s; out of reach, the nearest foe in reach.\n", s.Rule.Describe())
	if s.Role != domain.Fighter {
		if sp := m.spellsFor(mb, s.Role); len(sp) > 0 {
			fmt.Fprintf(&b, "  Casts %s.\n", strings.Join(sp, ", then "))
		} else {
			b.WriteString("  Knows no spell for it, so fights.\n")
		}
	}
	if mb.present && mb.manaMax > 0 {
		fmt.Fprintf(&b, "  Mana %d of %d.\n", mb.mana, mb.manaMax)
	}
	if st := m.Stored(userID, mb.key); st.IsZero() {
		b.WriteString("  Unchanged from the archetype's default.\n")
	}
	b.WriteString(usage)
	return b.String()
}
