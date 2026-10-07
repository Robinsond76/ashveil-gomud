package assessment

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/coordination"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/stance"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Report is a player's assessment of one enemy group, with what explains
// it. Everything in it is something the player can already see.
type Report struct {
	Result
	Counted    []string // "you" first, then companions walking with the leader
	Missing    []string // "Kell (fled)"
	Hurt       []string // "Bram (badly wounded)": wounded or worse
	Burdened   []string // "you (burdened)"
	NoReach    []string // own members who can reach none of them
	OutOfReach []string // foes none of the company can reach
	Allies     bool     // a party ally is here, and isn't counted
	// Phase 33i2: how the group fights together ("a drilled company"; ""
	// for a lone foe), and the roles its visible members show ("a healer").
	Coordination string
	Roles        []string
}

// Gather assesses group g for user from the live game. Call it on the game
// loop. The foes are g's members the user can see (scout's rule); the
// caller has already refused darkness. ok is false when none is visible.
func Gather(user *users.UserRecord, room *rooms.Room, g enemyparty.Group) (Report, bool) {
	if user == nil || user.Character == nil || room == nil {
		return Report{}, false
	}
	visible := g.Visible()
	if len(visible) == 0 {
		return Report{}, false
	}
	foes := Side{Formation: g.Party.Formation, HasFormation: true, Alive: enemyparty.Alive(g.Party)}
	foeNames := map[company.MemberKey]string{}
	for _, m := range visible {
		key := mobparty.MemberKeyFor(m.InstanceId)
		foes.Members = append(foes.Members, Member{Key: key, Name: m.Character.Name, Char: &m.Character, Reach: combat.ResolveReach(&m.Character, m.Reach)})
		foeNames[key] = m.Character.Name
	}

	var rep Report
	own := Side{}
	own.Formation, own.HasFormation = company.FormationFor(user.UserId)
	names := map[company.MemberKey]string{}
	add := func(key company.MemberKey, name string, char *characters.Character, reach bool) {
		own.Members = append(own.Members, Member{Key: key, Name: name, Char: withStance(char, user.UserId, key), Reach: combat.ResolveReach(char, reach)})
		names[key] = name
		rep.Counted = append(rep.Counted, name)
		if word := enemyparty.HealthWord(char.Health, char.HealthMax.Value); word != "unhurt" && word != "scratched" {
			rep.Hurt = append(rep.Hurt, fmt.Sprintf("%s (%s)", name, word))
		}
		if word := char.BurdenWord(); word != characters.BurdenNone {
			rep.Burdened = append(rep.Burdened, fmt.Sprintf("%s (%s)", name, word))
		}
	}
	if why := outOfFight(user.Character); why == "" {
		add(company.LeaderMemberKey, "you", user.Character, false)
	} else {
		rep.Missing = append(rep.Missing, "you ("+why+")")
	}
	present := map[int]bool{}
	here := map[int]string{} // in the room, but not fighting: why
	for _, id := range company.CompanionsWithLeader(user.UserId) {
		instanceId, ok := company.InstanceFor(user.UserId, id)
		if !ok {
			continue
		}
		m := mobs.GetInstance(instanceId)
		if m == nil || m.Character.RoomId != room.RoomId {
			continue
		}
		if why := outOfFight(&m.Character); why != "" {
			here[id] = why
			continue
		}
		present[id] = true
		add(company.CompanionMemberKey(id), m.Character.Name, &m.Character, m.Reach)
	}
	if views, ok := company.CompanyMembers(user.UserId); ok {
		for _, v := range views {
			if present[v.ID] {
				continue
			}
			why, ok := here[v.ID]
			if !ok {
				why = absence(v.Status)
			}
			rep.Missing = append(rep.Missing, fmt.Sprintf("%s (%s)", v.Name, why))
		}
	}

	rep.Result = Estimate(own, foes, combat.ExpectedDamage)
	for _, key := range rep.Result.Unreaching {
		rep.NoReach = append(rep.NoReach, names[key])
	}
	for _, key := range rep.Result.OutOfReach {
		rep.OutOfReach = append(rep.OutOfReach, "the "+foeNames[key])
	}
	rep.Allies = alliesHere(user, room)
	if len(g.Party.Members) > 1 && !g.Solo() {
		rep.Coordination = coordination.SpecOf(groupTier(user.UserId, g, visible)).Word
		rep.Roles = roleWords(visible)
	}
	return rep, true
}

// groupTier is the group's coordination: its battle's, fixed when the
// battle began, when the user is fighting it; else the tier of the members
// the user can see (33i2 review finding 5: a hidden member's level never
// shows).
func groupTier(userId int, g enemyparty.Group, visible []*mobs.Mob) coordination.Tier {
	if b, ok := battle.Current(userId); ok {
		if _, mine := enemyparty.BattleParty(b, []mobparty.Party{g.Party}); mine {
			if tier, ok := enemyparty.BattleTier(userId); ok {
				return tier
			}
		}
	}
	var levels, explicit []int
	for _, m := range visible {
		levels = append(levels, m.Character.Level)
		explicit = append(explicit, m.Coordination)
	}
	return coordination.Of(levels, explicit)
}

// roleWords names the roles the visible members show, beyond fighting:
// "a healer", "two guardians".
func roleWords(visible []*mobs.Mob) []string {
	count := map[string]int{}
	for _, m := range visible {
		count[m.EnemyRole()]++
	}
	var out []string
	for _, role := range []string{"healer", "caster", "guardian"} {
		switch n := count[role]; {
		case n == 1:
			out = append(out, "a "+role)
		case n > 1:
			out = append(out, numberWord(n)+" "+role+"s")
		}
	}
	return out
}

func numberWord(n int) string {
	if words := []string{"", "one", "two", "three", "four", "five"}; n < len(words) {
		return words[n]
	}
	return fmt.Sprint(n)
}

// CoordinationLine says how the group fights together: "They fight as a
// drilled company: a healer and a guardian among them." "" for a lone foe.
func (r Report) CoordinationLine() string {
	if r.Coordination == "" {
		return ""
	}
	if len(r.Roles) == 0 {
		return fmt.Sprintf("They fight as %s.", r.Coordination)
	}
	return fmt.Sprintf("They fight as %s: %s among them.", r.Coordination, list(r.Roles))
}

// outOfFight says why a member standing here can't fight ("" when it can):
// down, or out of the fight (surrendered or withdrawing). 33i1 review
// finding 2.
func outOfFight(c *characters.Character) string {
	switch {
	case c.Health < 1:
		return "down"
	case c.CombatWithdrawn:
		return "out of the fight"
	}
	return ""
}

// absence says why a companion who isn't here isn't counted.
func absence(s company.MemberStatus) string {
	switch s {
	case company.MemberFled:
		return "fled"
	case company.MemberDead:
		return "fallen"
	case company.MemberAwaiting:
		return "awaiting"
	case company.MemberSeparated:
		return "separated"
	}
	return "away"
}

// alliesHere: another player of the user's party stands in the room.
func alliesHere(user *users.UserRecord, room *rooms.Room) bool {
	p := parties.Get(user.UserId)
	if p == nil {
		return false
	}
	for _, id := range p.GetMembers() {
		if id == user.UserId {
			continue
		}
		if u := users.GetByUserId(id); u != nil && u.Character != nil && u.Character.RoomId == room.RoomId {
			return true
		}
	}
	return false
}

// Headline is the assessment's first sentence, without markup.
func (r Report) Headline() string {
	odds := "the odds look clear"
	if r.Close {
		odds = "it could go either way"
	}
	return fmt.Sprintf("%s for your company; %s.", capitalize(r.Risk.Phrase()), odds)
}

// Lines is the assessment as text, for scout and consider.
func (r Report) Lines() []string {
	lines := []string{`<ansi fg="yellow-bold">Assessment:</ansi> ` + r.Headline()}
	counted := "Counted: " + list(r.Counted) + "."
	if len(r.Counted) == 0 {
		counted = "Counted: nobody who can fight."
	}
	if len(r.Missing) > 0 {
		counted += " Not with you: " + list(r.Missing) + "."
	}
	lines = append(lines, "  "+counted)
	if line := r.CoordinationLine(); line != "" {
		lines = append(lines, "  "+line)
	}
	if len(r.Hurt) > 0 {
		lines = append(lines, "  Hurt: "+list(r.Hurt)+".")
	}
	if len(r.Burdened) > 0 {
		lines = append(lines, "  Burdened among you: "+list(r.Burdened)+".")
	}
	if len(r.NoReach) > 0 {
		lines = append(lines, "  Can't reach any of them from where they stand: "+list(r.NoReach)+".")
	}
	if len(r.OutOfReach) > 0 {
		lines = append(lines, "  Out of your company's reach: "+list(r.OutOfReach)+".")
	}
	if r.Allies {
		lines = append(lines, "  Allies here aren't counted.")
	}
	lines = append(lines, "  Not judged: spells, healing, guards and abilities, hidden foes, and anyone yet to come.")
	return lines
}

// Text is Lines joined.
func (r Report) Text() string { return strings.Join(r.Lines(), "\n") }

func list(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// withStance is the member as the estimate should read it: before a battle
// its weapon stance (Phase 69) is only in the store, since the round sets it
// on the battle state, so a copy carries it. The live character is left
// alone.
func withStance(char *characters.Character, leaderUserID int, key company.MemberKey) *characters.Character {
	st := stance.For(leaderUserID, string(key))
	if st == stance.None || (char.RT != nil && char.RT.Stance == st) {
		return char
	}
	cp := *char
	rt := characters.ClassRT{}
	if char.RT != nil {
		rt = *char.RT
	}
	rt.Stance = st
	cp.RT = &rt
	return &cp
}
