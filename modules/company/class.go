package company

// Phase 38b: class promotion and talents, for the leader and each companion.
// The graph, ranks and talents are internal/classes; a companion's choices
// are on its company record, a player's in the archetype registry. This file
// holds the commands, which preview and confirm like company train, and the
// company half of the class seam.

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/classes"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var _ domain.ClassProvider = (*CompanyModule)(nil)

// classSetter is implemented by the native runtime: it writes a companion's
// class and talents onto its live mob at once.
type classSetter interface {
	SetClass(instanceID int, class string, talents []string) bool
}

// SetClass writes class state onto a live mob and settles its maxima: one
// that falls clamps its health and mana, one that rises never refills.
func (nativeRuntime) SetClass(instanceID int, class string, talents []string) bool {
	mob := mobs.GetInstance(instanceID)
	if mob == nil {
		return false
	}
	mob.Character.SetClassState(class, talents)
	mob.Character.RecalculateStats()
	mob.Character.Health = min(mob.Character.Health, mob.Character.HealthLimit())
	mob.Character.Mana = min(mob.Character.Mana, mob.Character.ManaMax.Value)
	return true
}

// CompanionClassState implements domain.ClassProvider.
func (m *CompanyModule) CompanionClassState(leaderUserID, companionID int) (string, []string, bool) {
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return "", nil, false
	}
	for _, c := range record.Companions {
		if c.ID == companionID {
			return c.Class, slices.Clone(c.Talents), true
		}
	}
	return "", nil, false
}

// PromoteCompanion implements domain.ClassProvider: record the class and
// save, restoring the record if the save fails, then update the live mob.
// The caller has checked the rules.
func (m *CompanyModule) PromoteCompanion(leaderUserID, companionID int, class string) error {
	return m.commitClass(leaderUserID, companionID, func() error {
		return m.registry.SetCompanionClass(leaderUserID, companionID, class)
	})
}

// PickCompanionTalent implements domain.ClassProvider.
func (m *CompanyModule) PickCompanionTalent(leaderUserID, companionID int, talent string) error {
	return m.commitClass(leaderUserID, companionID, func() error {
		return m.registry.AddCompanionTalent(leaderUserID, companionID, talent)
	})
}

func (m *CompanyModule) commitClass(leader, companionID int, change func() error) error {
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	before, ok := m.registry.Get(leader)
	if !ok {
		return domain.ErrUnknownMember
	}
	if err := change(); err != nil {
		return err
	}
	if err := m.save(); err != nil {
		m.registry.Put(before)
		return err
	}
	if instanceID, tracked := m.instance(leader, companionID); tracked {
		if setter, ok := m.runtime.(classSetter); ok {
			class, talents, _ := m.CompanionClassState(leader, companionID)
			setter.SetClass(instanceID, class, talents)
		}
	}
	return nil
}

// --- subjects ---------------------------------------------------------------

// classSubject is one character a class command is about: the leader or a
// companion.
type classSubject struct {
	player    bool
	name      string // "You" for the leader
	lineage   string
	level     int
	alignment int
	class     string
	talents   []string
	c         domain.Companion
	instance  int
	live      bool
}

func (s classSubject) label() string {
	if s.player {
		return "You"
	}
	return s.name
}

// are is "are" for you, "is" for a companion.
func (s classSubject) are() string {
	if s.player {
		return "are"
	}
	return "is"
}

func (s classSubject) your() string {
	if s.player {
		return "your"
	}
	return "their"
}

func (m *CompanyModule) playerSubject(user *users.UserRecord) classSubject {
	lineage, _ := archetypes.PlayerArchetype(user.UserId)
	st := classes.PlayerClass(user.UserId)
	return classSubject{player: true, name: "You", lineage: lineage, level: user.Character.Level,
		alignment: int(user.Character.Alignment), class: st.Class, talents: st.Talents}
}

func (m *CompanyModule) companionSubject(leader int, c domain.Companion) classSubject {
	s := classSubject{name: nameOf(c, "#"+strconv.Itoa(c.ID)), lineage: c.Archetype, level: m.trainingLevel(leader, c),
		alignment: m.companionAlignment(c), class: c.Class, talents: c.Talents, c: c}
	s.instance, s.live = m.instance(leader, c.ID)
	return s
}

// selectSubject finds who a command is about from the words that name them:
// empty, "me", "self" or "you" is the leader; otherwise a companion by number
// or name.
func (m *CompanyModule) selectSubject(user *users.UserRecord, words []string) (classSubject, string) {
	sel := strings.ToLower(strings.TrimSpace(strings.Join(words, " ")))
	switch sel {
	case "", "me", "self", "you":
		return m.playerSubject(user), ""
	}
	record, ok := m.registry.Get(user.UserId)
	if !ok {
		return classSubject{}, "You have no companion like that."
	}
	c, found := resolveCompanion(record, sel)
	if !found {
		if _, ambiguous := ambiguousCompanion(record, sel); ambiguous {
			return classSubject{}, "More than one companion answers to that; use their number (#N)."
		}
		return classSubject{}, "You have no companion like that."
	}
	return m.companionSubject(user.UserId, c), ""
}

// splitSubject splits "[who] <name>" into who and the trailing class or
// talent name, trying the whole line as a name first (the leader).
func splitSubject(words []string, isName func(string) bool) (who []string, name string, ok bool) {
	for i := 0; i < len(words); i++ {
		if n := strings.Join(words[i:], "-"); isName(n) {
			return words[:i], n, true
		}
	}
	return nil, "", false
}

func classWord(s string) bool {
	_, ok := classes.Get(s)
	return ok
}

func talentWord(s string) bool {
	_, ok := classes.TalentByID(s)
	return ok
}

// --- commands ---------------------------------------------------------------

const classUsage = `Use <ansi fg="command">class</ansi> to see your class, <ansi fg="command">class [member]</ansi> for a companion, <ansi fg="command">class paths [member]</ansi> for the routes open to a lineage, and <ansi fg="command">class promote [member] [class]</ansi> to promote. See <ansi fg="command">help promotion</ansi>.`

const talentUsage = `Use <ansi fg="command">talent</ansi> to see your talents, <ansi fg="command">talent [member]</ansi> for a companion, and <ansi fg="command">talent pick [member] [talent]</ansi> to choose one. See <ansi fg="command">help talents</ansi>.`

func (m *CompanyModule) classCommand(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	args := util.SplitButRespectQuotes(strings.ToLower(rest))
	user.SendText(m.runClass(user, room, args))
	return true, nil
}

func (m *CompanyModule) talentCommand(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	args := util.SplitButRespectQuotes(strings.ToLower(rest))
	user.SendText(m.runTalent(user, room, args))
	return true, nil
}

func (m *CompanyModule) runClass(user *users.UserRecord, room *rooms.Room, args []string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	if len(args) > 0 {
		switch args[0] {
		case "paths", "path", "routes":
			s, problem := m.selectSubject(user, args[1:])
			if problem != "" {
				return problem
			}
			return m.pathsView(s)
		case "promote", "promotion":
			return m.promote(user, room, args[1:])
		}
	}
	s, problem := m.selectSubject(user, args)
	if problem != "" {
		return problem + "\n" + classUsage
	}
	return m.classView(s)
}

func (m *CompanyModule) runTalent(user *users.UserRecord, room *rooms.Room, args []string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	if len(args) > 0 {
		switch args[0] {
		case "pick", "choose", "take":
			return m.pickTalent(user, room, args[1:])
		case "list", "all":
			s, problem := m.selectSubject(user, args[1:])
			if problem != "" {
				return problem
			}
			return m.talentView(s, true)
		}
	}
	s, problem := m.selectSubject(user, args)
	if problem != "" {
		return problem + "\n" + talentUsage
	}
	return m.talentView(s, false)
}

// --- views ------------------------------------------------------------------

func className(id string) string {
	if c, ok := classes.Get(id); ok {
		return c.Name
	}
	return id
}

func lineageName(id string) string {
	if id == "" {
		return "no archetype"
	}
	return archetypeLabel(id)
}

// classView is "class" for one character: lineage, class, ranks, talents and
// what comes next.
func (m *CompanyModule) classView(s classSubject) string {
	if s.lineage == "" {
		return fmt.Sprintf("%s %s no archetype yet, so there is no class to promote into. See help archetype.", s.label(), s.are())
	}
	var lines []string
	cur, promoted := classes.Get(s.class)
	switch {
	case promoted:
		lines = append(lines, fmt.Sprintf("%s %s a level %d %s, a %s (%s). Alignment %+d.", s.label(), s.are(), s.level, lineageName(s.lineage), cur.Name, tierName(cur.Tier), s.alignment))
	case s.class != "":
		lines = append(lines, fmt.Sprintf("%s %s a level %d %s with a class this server no longer knows (%q); %s ranks are paused.", s.label(), s.are(), s.level, lineageName(s.lineage), s.class, s.your()))
	default:
		lines = append(lines, fmt.Sprintf("%s %s a level %d %s with no promotion yet. Alignment %+d.", s.label(), s.are(), s.level, lineageName(s.lineage), s.alignment))
	}
	if promoted {
		for _, r := range classes.RanksReached(s.class, s.level) {
			lines = append(lines, fmt.Sprintf("  Rank %d, %s: %s.", r.Level, r.Name, r.Text))
		}
		if len(classes.RanksReached(s.class, s.level)) == 0 {
			lines = append(lines, fmt.Sprintf("  No rank yet at level %d (see class paths).", s.level))
		}
	}
	opts := classes.Options(s.lineage, s.class, s.level, s.alignment)
	for _, o := range opts {
		switch {
		case o.Eligible:
			lines = append(lines, fmt.Sprintf("  Ready to promote: %s. Type class promote%s %s.", o.Class.Name, selectorSuffix(s), classKey(o.Class)))
		case o.Waiting:
			lines = append(lines, fmt.Sprintf("  Waiting: %s %s. %s keeps %s ranks and promotes once %s alignment recovers.", o.Class.Name, o.Reason, s.label(), s.your(), s.your()))
		}
	}
	if next := classes.Milestone(s.class, s.level); next != "" {
		lines = append(lines, "  "+next)
	}
	if owed := classes.TalentsOwed(s.level, s.talents); owed > 0 {
		lines = append(lines, fmt.Sprintf("  %d talent%s to choose: talent pick%s [talent].", owed, plural(owed), selectorSuffix(s)))
	}
	return strings.Join(lines, "\n")
}

func tierName(t classes.Tier) string {
	switch t {
	case classes.TierAdvanced:
		return "advanced class"
	case classes.TierElite:
		return "elite class"
	}
	return "base class"
}

// classKey is a class's id as the commands take it.
func classKey(c classes.Class) string { return c.ID }

// selectorSuffix is the member selector a follow-up command needs.
func selectorSuffix(s classSubject) string {
	if s.player {
		return ""
	}
	return " #" + strconv.Itoa(s.c.ID)
}

// pathsView lists a lineage's routes with their gates and what the
// character may do about each.
func (m *CompanyModule) pathsView(s classSubject) string {
	if s.lineage == "" {
		return fmt.Sprintf("%s %s no archetype yet, so there are no routes to show.", s.label(), s.are())
	}
	adv := classes.Advanced(s.lineage)
	if len(adv) == 0 {
		return fmt.Sprintf("A %s has no promotion routes.", lineageName(s.lineage))
	}
	lines := []string{fmt.Sprintf("Routes for a %s (level %d promotes, level %d for the elite step):", lineageName(s.lineage), classes.AdvancedLevel, classes.EliteLevel)}
	for _, a := range adv {
		mark := ""
		if s.class == a.ID {
			mark = " (current)"
		}
		lines = append(lines, fmt.Sprintf("  %s%s: %s. Needs %s.", a.Name, mark, a.Role, a.Gate.Label()))
		for _, r := range a.Ranks {
			lines = append(lines, fmt.Sprintf("      rank %d, %s", r.Level, r.Name))
		}
		if e, ok := classes.Elite(a.ID); ok {
			if e.Planned {
				lines = append(lines, fmt.Sprintf("    then %s (planned: not open yet)", e.Name))
			} else {
				cur := ""
				if s.class == e.ID {
					cur = " (current)"
				}
				lines = append(lines, fmt.Sprintf("    then %s%s: %s. Ranks at 30, 35, 40, 45, 50, 55 and 60.", e.Name, cur, e.Role))
			}
		}
	}
	lines = append(lines, `Routes are final. See help promotion, and help [class] for a route's ranks.`)
	return strings.Join(lines, "\n")
}

// talentView lists a character's talents, and with all the whole menu.
func (m *CompanyModule) talentView(s classSubject, all bool) string {
	if s.lineage == "" {
		return fmt.Sprintf("%s %s no archetype yet, so there are no talents to choose.", s.label(), s.are())
	}
	menu := classes.TalentsFor(s.lineage)
	if len(menu) == 0 {
		return fmt.Sprintf("A %s has no talents.", lineageName(s.lineage))
	}
	slots := classes.TalentSlots(s.level)
	owed := classes.TalentsOwed(s.level, s.talents)
	lines := []string{fmt.Sprintf("%s %s level %d: %d of %d talent%s earned so far, %d to choose.", s.label(), s.are(), s.level, min(slots, len(s.talents)), len(classes.TalentLevels), plural(len(classes.TalentLevels)), owed)}
	if len(s.talents) > 0 {
		counts := map[string]int{}
		var order []string
		for i, id := range s.talents {
			if counts[id] == 0 {
				order = append(order, id)
			}
			counts[id]++
			_ = i
		}
		for _, id := range order {
			t, ok := classes.TalentByID(id)
			name := id
			text := ""
			if ok {
				name, text = t.Name, t.Text
			}
			times := ""
			if counts[id] > 1 {
				times = fmt.Sprintf(" x%d", counts[id])
			}
			state := ""
			if active := classes.ActiveTalents(s.level, s.talents); classes.Count(active, id) < counts[id] {
				state = " (off until the level is regained)"
			}
			lines = append(lines, fmt.Sprintf("  %s%s: %s%s.", name, times, text, state))
		}
	}
	if next, ok := classes.NextTalentLevel(s.level); ok && owed == 0 {
		lines = append(lines, fmt.Sprintf("  The next talent comes at level %d.", next))
	}
	if owed > 0 || all {
		lines = append(lines, "  Talents to choose from:")
		for _, t := range menu {
			taken := classes.Count(s.talents, t.ID)
			lines = append(lines, fmt.Sprintf("    %s (%s): %s%s.", t.Name, t.ID, t.Text, takenNote(taken, t.Max)))
		}
		if owed > 0 {
			lines = append(lines, fmt.Sprintf("  Type talent pick%s [talent] to choose.", selectorSuffix(s)))
		}
	}
	return strings.Join(lines, "\n")
}

func takenNote(taken, max int) string {
	switch {
	case taken >= max:
		return " [at its limit]"
	case taken > 0:
		return fmt.Sprintf(" [taken %d of %d]", taken, max)
	}
	return ""
}

// --- promotion and talent commits ------------------------------------------

// classGate is why a class change can't happen right now, or "".
func (m *CompanyModule) classGate(user *users.UserRecord, s classSubject) string {
	leader := user.UserId
	if usercommands.InBattle(user) {
		return "Not in the middle of a battle. Once the fighting is done."
	}
	if blocked, _ := expedition.MovementBlocked(leader); blocked {
		return "Not while your company is travelling. Once you have arrived."
	}
	if blocked, _ := camping.MovementBlocked(leader); blocked {
		return "Not while your company is resting. Once the rest is over."
	}
	if s.player {
		if user.Character.Health < 1 {
			return "Not while you are down."
		}
		return ""
	}
	c := s.c
	name := s.name
	switch {
	case c.Dead():
		return fmt.Sprintf("%s has fallen and must be raised first.", name)
	case c.Separated():
		return fmt.Sprintf("%s is separated from the company and must rejoin first.", name)
	}
	instanceID, tracked := m.instance(leader, c.ID)
	if c.PendingReturn || !tracked || !m.runtime.IsLive(instanceID) || !m.runtime.IsAttached(leader, instanceID) || !m.runtime.WithLeader(leader, instanceID) {
		return fmt.Sprintf("%s isn't here with you.", name)
	}
	if _, fighting, ok := m.runtime.Standing(instanceID); !ok || fighting {
		return fmt.Sprintf("%s is busy fighting.", name)
	}
	return ""
}

// promote previews, or with confirm commits, a promotion. Confirm checks
// everything again against the current state (a gate that drifted since the
// preview), and a repeated confirm finds the class already held.
func (m *CompanyModule) promote(user *users.UserRecord, room *rooms.Room, rest []string) string {
	confirm := false
	if n := len(rest); n > 0 && rest[n-1] == "confirm" {
		confirm, rest = true, rest[:n-1]
	}
	if len(rest) == 0 {
		return classUsage
	}
	who, target, ok := splitSubject(rest, classWord)
	if !ok {
		return fmt.Sprintf(`There is no class called "%s". Type class paths to see the routes.`, strings.Join(rest, " "))
	}
	s, problem := m.selectSubject(user, who)
	if problem != "" {
		return problem
	}
	if s.lineage == "" {
		return fmt.Sprintf("%s %s no archetype yet, so there is nothing to promote.", s.label(), s.are())
	}
	want, _ := classes.Get(target)
	if s.class == want.ID {
		return fmt.Sprintf("%s %s already a %s; there is nothing to do.", s.label(), s.are(), want.Name)
	}
	chosen, err := classes.Check(s.lineage, s.class, target, s.level, s.alignment)
	if err != nil {
		return refusal(s, want, err)
	}
	if msg := m.classGate(user, s); msg != "" {
		return msg
	}
	if !confirm {
		lines := []string{fmt.Sprintf("%s can become a %s: %s.", s.label(), chosen.Name, chosen.Role)}
		reached := 0
		for _, r := range chosen.Ranks {
			if s.level >= r.Level {
				reached++
				lines = append(lines, fmt.Sprintf("  Rank %d, %s: %s.", r.Level, r.Name, r.Text))
			}
		}
		if next, ok := nextRankOf(chosen, s.level); ok {
			lines = append(lines, fmt.Sprintf("  Next, at level %d: %s.", next.Level, next.Name))
		}
		_ = reached
		gate := chosen.Gate
		if chosen.Tier == classes.TierElite {
			if p, ok := classes.Get(chosen.Parent); ok {
				gate = p.Gate
			}
		}
		lines = append(lines, fmt.Sprintf("  Needs %s; %s %s %+d.", gate.Label(), s.label(), strings.Replace(s.are(), "are", "have", 1), s.alignment))
		if chosen.Tier == classes.TierElite {
			lines = append(lines, "  An elite promotion keeps every rank so far.")
		} else {
			lines = append(lines, "  If the alignment later drifts, the route is kept; the elite step then waits until it recovers.")
		}
		lines = append(lines, fmt.Sprintf("Routes are final: there is no switching to another. Type class promote%s %s confirm to promote.", selectorSuffix(s), classKey(chosen)))
		return strings.Join(lines, "\n")
	}

	var cerr error
	if s.player {
		cerr = classes.PromotePlayer(user.UserId, chosen.ID)
	} else {
		cerr = m.PromoteCompanion(user.UserId, s.c.ID, chosen.ID)
	}
	if cerr != nil {
		mudlog.Warn("company: promote", "leader", user.UserId, "error", cerr)
		return "Your records couldn't be saved; nothing changed. Please try again."
	}
	text := fmt.Sprintf("%s %s now a %s.", s.label(), s.are(), chosen.Name)
	if s.player {
		text = fmt.Sprintf("You are now a %s.", chosen.Name)
	}
	for _, r := range classes.RanksReached(chosen.ID, s.level) {
		if r.Level >= chosen.Ranks[0].Level {
			text += fmt.Sprintf("\n  Rank %d, %s: %s.", r.Level, r.Name, r.Text)
		}
	}
	return text
}

func nextRankOf(c classes.Class, level int) (classes.Rank, bool) {
	for _, r := range c.Ranks {
		if level < r.Level {
			return r, true
		}
	}
	return classes.Rank{}, false
}

func refusal(s classSubject, want classes.Class, err error) string {
	switch {
	case errors.Is(err, classes.ErrFinalRoute):
		return fmt.Sprintf("%s %s on a final route (%s); there is nothing further to promote to.", s.label(), s.are(), className(s.class))
	case errors.Is(err, classes.ErrNotAvailable):
		return fmt.Sprintf("%s can't become a %s: %v. Routes are final; see class paths.", s.label(), want.Name, err)
	}
	return fmt.Sprintf("%s can't become a %s yet: %v.", s.label(), want.Name, err)
}

// pickTalent previews, or with confirm commits, one talent pick. A talent
// is a permanent choice, so it asks for confirm first.
func (m *CompanyModule) pickTalent(user *users.UserRecord, room *rooms.Room, rest []string) string {
	confirm := false
	if n := len(rest); n > 0 && rest[n-1] == "confirm" {
		confirm, rest = true, rest[:n-1]
	}
	if len(rest) == 0 {
		return talentUsage
	}
	who, id, ok := splitSubject(rest, talentWord)
	if !ok {
		return fmt.Sprintf(`There is no talent called "%s". Type talent list to see them.`, strings.Join(rest, " "))
	}
	s, problem := m.selectSubject(user, who)
	if problem != "" {
		return problem
	}
	t, _ := classes.TalentByID(id)
	if s.lineage == "" {
		return fmt.Sprintf("%s %s no archetype yet.", s.label(), s.are())
	}
	if err := classes.CanPick(s.lineage, s.talents, s.level, id); err != nil {
		switch {
		case errors.Is(err, classes.ErrUnknownTalent):
			return fmt.Sprintf("A %s can't take %s. Type talent list to see its talents.", lineageName(s.lineage), t.Name)
		case errors.Is(err, classes.ErrNoTalentOwed):
			if next, ok := classes.NextTalentLevel(s.level); ok {
				return fmt.Sprintf("%s %s no talent to choose now. The next comes at level %d.", s.label(), strings.Replace(s.are(), "are", "have", 1), next)
			}
			return fmt.Sprintf("%s %s chosen every talent there is.", s.label(), strings.Replace(s.are(), "are", "have", 1))
		case errors.Is(err, classes.ErrTalentMaxed):
			return fmt.Sprintf("%s already has %s as many times as it can be taken.", s.label(), t.Name)
		}
		return err.Error()
	}
	if msg := m.classGate(user, s); msg != "" {
		return msg
	}
	if !confirm {
		return fmt.Sprintf("%s can take %s: %s.\nTalents are permanent. Type talent pick%s %s confirm to take it.", s.label(), t.Name, t.Text, selectorSuffix(s), t.ID)
	}
	var err error
	if s.player {
		err = classes.PickPlayerTalent(user.UserId, user.Character.Level, t.ID)
	} else {
		err = m.PickCompanionTalent(user.UserId, s.c.ID, t.ID)
	}
	if err != nil {
		mudlog.Warn("company: pick talent", "leader", user.UserId, "error", err)
		return "Your records couldn't be saved; nothing changed. Please try again."
	}
	if s.player {
		return fmt.Sprintf("You take %s: %s.", t.Name, t.Text)
	}
	return fmt.Sprintf("%s takes %s: %s.", s.name, t.Name, t.Text)
}
