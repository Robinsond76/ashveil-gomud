package strategy

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
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
	// col and placed are its place in the formation (Phase 30c2: a
	// guardian's reach).
	col    int
	placed bool
	// abilities are the class abilities it has (Phase 33e).
	abilities []domain.Ability
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
func CompanionKnows(archetype, class string, level int) func(string) bool {
	known := map[string]bool{}
	for _, id := range archetypes.CompanionKnownSpells(archetype, class, level) {
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
		abilities: domain.AtLevel(domain.PlayerAbilities(user.Character.GetSkillLevel), user.Character.Level),
	}}
	form, hasForm := company.FormationFor(user.UserId)
	if hasForm {
		_, out[0].col, out[0].placed = form.Find(company.LeaderMemberKey)
	}
	views, ok := company.CompanyMembers(user.UserId)
	for _, v := range views {
		mb := member{
			key:       string(company.CompanionMemberKey(v.ID)),
			name:      v.Name,
			archetype: v.Archetype,
			knows:     CompanionKnows(v.Archetype, v.Class, v.Level),
			abilities: domain.AtLevel(domain.CompanionAbilities(v.Archetype), v.Level),
		}
		if hasForm {
			_, mb.col, mb.placed = form.Find(company.CompanionMemberKey(v.ID))
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

const usage = `Change one with <ansi fg="command">strategy [who] [role]</ansi>, <ansi fg="command">strategy [who] target [rule]</ansi>, <ansi fg="command">strategy [who] guard [other]</ansi>, <ansi fg="command">strategy [who] abilities on|off</ansi>, or <ansi fg="command">strategy [who] reserve [percent]</ansi>. See <ansi fg="command">help strategy</ansi> and <ansi fg="command">help abilities</ansi>.`

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
		return m.describe(user.UserId, mb, members)
	}
	if m.env.inBattle(user) {
		return usercommands.BattleUnderWay
	}
	stored := m.Stored(user.UserId, mb.key)
	change := args[1:]
	targetOnly := false
	if change[0] == "target" || change[0] == "rule" {
		change = change[1:]
		if len(change) == 0 {
			return `Target which way? ` + rulesList()
		}
		// Phase 30c2: after "target" only a rule is read ("target guard"
		// is not the guardian).
		if _, ok := domain.ParseRule(change[0]); !ok {
			return fmt.Sprintf(`"%s" is not a target. %s`, change[0], rulesList())
		}
		targetOnly = true
	}
	word := change[0]
	var next domain.Strategy
	var said string
	// Phase 33e: abilities on|off and a mana reserve keep the rest.
	if !targetOnly && (word == "abilities" || word == "ability") {
		return m.setAbilities(user.UserId, mb, stored, change[1:])
	}
	if !targetOnly && word == "reserve" {
		return m.setReserve(user.UserId, mb, stored, change[1:])
	}
	switch {
	case word == "default" || word == "reset":
		d := domain.Default(mb.archetype)
		if err := m.set(user.UserId, mb.key, domain.Strategy{}); err != nil {
			return err.Error()
		}
		return fmt.Sprintf(`%s %s back to the default: %s, going for %s.`, mb.name, verb(mb, "are", "is"), d.Role, d.Rule.Describe())
	default:
		if role, ok := domain.ParseRole(word); ok && !targetOnly && role == domain.Guardian {
			// Phase 30c2: strategy <who> guard [<other>].
			ward := member{}
			if len(change) > 1 {
				sel := strings.Join(change[1:], " ")
				found := false
				if ward, found = resolve(members, sel); !found {
					return fmt.Sprintf(`No one in your company answers to "%s". Type <ansi fg="command">strategy</ansi> to see them.`, sel)
				}
				if ward.key == mb.key {
					return `A guardian guards someone else: name another member of your company, or none for whoever is most hurt.`
				}
			}
			next = domain.Strategy{Role: role, Rule: stored.Rule, Ward: ward.key, NoAbilities: stored.NoAbilities, Reserve: stored.Reserve}
			if ward.key == "" {
				said = fmt.Sprintf(`%s %s now a guardian, guarding whoever is most hurt within reach (<ansi fg="command">help guardian</ansi>).`, mb.name, verb(mb, "are", "is"))
			} else {
				said = fmt.Sprintf(`%s will guard %s, stepping in to take blows meant for %s (<ansi fg="command">help guardian</ansi>).`, mb.name, object(ward), object(ward))
				if warn := reachWarning(mb, ward); warn != "" {
					said += "\n" + warn
				}
			}
		} else if ok && !targetOnly {
			next = domain.Strategy{Role: role, Rule: stored.Rule, NoAbilities: stored.NoAbilities, Reserve: stored.Reserve}
			said = fmt.Sprintf(`%s %s now a %s: %s.`, mb.name, verb(mb, "are", "is"), role, role.Describe())
			if warn := m.cantYet(mb, role); warn != "" {
				said = warn + "\n" + said
			}
		} else if rule, ok := domain.ParseRule(word); ok {
			next = domain.Strategy{Role: stored.Role, Rule: rule, Ward: stored.Ward, NoAbilities: stored.NoAbilities, Reserve: stored.Reserve}
			if err := next.ValidFor(mb.isPlayer); err != nil {
				return `Only a companion can assist you: you are the one they assist.`
			}
			said = fmt.Sprintf(`%s will go for %s, else the nearest foe in reach.`, mb.name, rule.Describe())
		} else {
			return fmt.Sprintf(`"%s" is neither a role nor a target. Roles: %s. %s`, word, rolesList(), rulesList())
		}
	}
	// A setting equal to the default is stored as blank, except a warrior's
	// fighter role (Phase 35d): a blank warrior guards the healer by default,
	// so choosing to fight is kept.
	d := domain.Default(mb.archetype)
	if next.Role == d.Role && !(mb.archetype == "warrior" && next.Role == domain.Fighter && !mb.isPlayer) {
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

// setAbilities turns a member's automatic class abilities on or off
// (Phase 33e).
func (m *StrategyModule) setAbilities(userID int, mb member, stored domain.Strategy, args []string) string {
	if len(args) == 0 || (args[0] != "on" && args[0] != "off") {
		return `Abilities on or off? <ansi fg="command">strategy [who] abilities on|off</ansi>.`
	}
	next := stored
	next.NoAbilities = args[0] == "off"
	if err := m.set(userID, mb.key, next); err != nil {
		return err.Error()
	}
	if next.NoAbilities {
		return fmt.Sprintf(`%s will use no class abilities in battle, only plain blows and spells.`, mb.name)
	}
	names := domain.Names(mb.abilities)
	if len(names) == 0 {
		return fmt.Sprintf(`%s will use class abilities in battle, though %s none yet (<ansi fg="command">help abilities</ansi>).`, mb.name, verb(mb, "have", "has"))
	}
	return fmt.Sprintf(`%s will use %s in battle when the moment comes (<ansi fg="command">help abilities</ansi>).`, mb.name, strings.Join(names, " and "))
}

// setReserve sets the share of its mana a member keeps back from attack
// spells (Phase 33e).
func (m *StrategyModule) setReserve(userID int, mb member, stored domain.Strategy, args []string) string {
	if len(args) == 0 {
		return fmt.Sprintf(`Keep how much mana back? A percent from 0 to %d: <ansi fg="command">strategy [who] reserve 30</ansi>.`, domain.MaxReserve)
	}
	n, ok := domain.ParseReserve(args[0])
	if !ok {
		return fmt.Sprintf(`"%s" is not a reserve. Give a percent from 0 to %d.`, args[0], domain.MaxReserve)
	}
	next := stored
	next.Reserve = n
	if err := m.set(userID, mb.key, next); err != nil {
		return err.Error()
	}
	if n == 0 {
		return fmt.Sprintf(`%s will spend mana on attack spells down to the last drop.`, mb.name)
	}
	return fmt.Sprintf(`%s will cast attack spells only while %d%% of %s mana would be left; heals ignore the reserve.`, mb.name, n, verb(mb, "your", "their"))
}

// abilityLine says what a member's abilities are, for list and describe.
func abilityLine(mb member, s domain.Strategy) string {
	names := domain.Names(mb.abilities)
	switch {
	case s.NoAbilities:
		return "abilities off"
	case len(names) == 0:
		return ""
	}
	return strings.Join(names, ", ")
}

// object names a member as the object of a sentence: "you" for the
// player.
func object(mb member) string {
	if mb.isPlayer {
		return "you"
	}
	return mb.name
}

// reachWarning says when a guardian stands too far from its ward to step
// in (Phase 30c2, the owner's decision 11): a guardian must stand in its
// ward's column or the next. Unplaced members fail open.
func reachWarning(guardian, ward member) string {
	if !guardian.placed || !ward.placed || formationcombat.InLateralRange(guardian.col, ward.col) {
		return ""
	}
	return fmt.Sprintf(`Out of reach: %s can't step in for %s from there. A guardian must stand in its ward's column or the next (<ansi fg="command">formation</ansi>).`, guardian.name, object(ward))
}

// wardOf is the member a guardian's stored ward names, if it is one of
// members.
func wardOf(s domain.Strategy, members []member) (member, bool) {
	if s.Role != domain.Guardian || s.Ward == "" {
		return member{}, false
	}
	for _, mb := range members {
		if mb.key == s.Ward {
			return mb, true
		}
	}
	return member{}, false
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
		uses = []domain.Use{domain.UseWeather, domain.UseStorm, domain.UseAttack, domain.UseAttackAll}
	case domain.Controller:
		uses = []domain.Use{domain.UseHex, domain.UseAttack}
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
	if (role != domain.Healer && role != domain.Caster && role != domain.Controller) || len(m.spellsFor(mb, role)) > 0 {
		return ""
	}
	what := "attack spell"
	switch role {
	case domain.Healer:
		what = "healing spell"
	case domain.Controller:
		what = "hex"
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
	var warnings []string
	for _, mb := range members {
		s := m.Stored(userID, mb.key).Resolve(mb.archetype)
		arch := "-"
		if mb.archetype != "" {
			arch = m.env.archName(mb.archetype)
		}
		line := fmt.Sprintf("  %-*s  %-8s %-8s %-9s", width, mb.name, arch, s.Role, s.Rule)
		if s.Role == domain.Guardian {
			if ward, ok := wardOf(s, members); ok {
				line += " guards " + object(ward)
				if warn := reachWarning(mb, ward); warn != "" {
					warnings = append(warnings, warn)
				}
			} else {
				line += " guards the most hurt"
			}
		} else if sp := m.spellsFor(mb, s.Role); len(sp) > 0 {
			line += " " + strings.Join(sp, ", ")
		} else if s.Role != domain.Fighter {
			line += " (knows no spell for it: fights)"
		}
		// Phase 33e: its abilities and mana reserve.
		if al := abilityLine(mb, s); al != "" {
			line += "; " + al
		}
		if s.Reserve > 0 {
			line += fmt.Sprintf("; keeps %d%% mana", s.Reserve)
		}
		b.WriteString(strings.TrimRight(line, " ") + "\n")
	}
	for _, w := range warnings {
		b.WriteString(w + "\n")
	}
	// Phase 30c: a company focus overrides every target rule above.
	if focus, ok := domain.TacticsFor(userID).FocusRule(); ok {
		fmt.Fprintf(&b, "Your company's focus is %s (<ansi fg=\"command\">company tactics</ansi>): instead of these rules, everyone goes for %s.\n", focus, focus.Describe())
	}
	b.WriteString(usage)
	return b.String()
}

func (m *StrategyModule) describe(userID int, mb member, members []member) string {
	s := m.Stored(userID, mb.key).Resolve(mb.archetype)
	var b strings.Builder
	arch := "no archetype"
	if mb.archetype != "" {
		arch = m.env.archName(mb.archetype)
	}
	fmt.Fprintf(&b, "%s (%s): %s, %s.\n", mb.name, arch, s.Role, s.Role.Describe())
	fmt.Fprintf(&b, "  Goes for %s; out of reach, the nearest foe in reach.\n", s.Rule.Describe())
	if s.Role == domain.Guardian {
		if ward, ok := wardOf(s, members); ok {
			fmt.Fprintf(&b, "  Guards %s (<ansi fg=\"command\">help guardian</ansi>).\n", object(ward))
			if warn := reachWarning(mb, ward); warn != "" {
				b.WriteString("  " + warn + "\n")
			}
		} else {
			b.WriteString("  Guards whoever is most hurt within reach (<ansi fg=\"command\">help guardian</ansi>).\n")
		}
	} else if s.Role != domain.Fighter {
		if sp := m.spellsFor(mb, s.Role); len(sp) > 0 {
			fmt.Fprintf(&b, "  Casts %s.\n", strings.Join(sp, ", then "))
		} else {
			b.WriteString("  Knows no spell for it, so fights.\n")
		}
	}
	// Phase 33e: its class abilities, each with its condition.
	switch {
	case s.NoAbilities:
		b.WriteString("  Abilities off: uses none in battle.\n")
	default:
		for _, id := range mb.abilities {
			if spec, ok := domain.SpecOf(id); ok {
				fmt.Fprintf(&b, "  %s, when %s: %s (at most once every %d combat rounds).\n", spec.Name, spec.When, spec.Does, spec.Cooldown)
			}
		}
	}
	if s.Reserve > 0 {
		fmt.Fprintf(&b, "  Keeps %d%% of %s mana back from attack spells.\n", s.Reserve, verb(mb, "your", "its"))
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
