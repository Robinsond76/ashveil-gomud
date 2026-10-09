package camping

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/flasks"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 51 rest duties (help campduties). Each member at the camp sleeps
// (the default) or takes a duty for the rest:
//
//   - Duties are standing assignments on the camp (Camp.Duties), locked on
//     the rest when it starts (RestSession.Duties) for the members then at
//     the camp, and settled when it ends. A restart never re-rolls them.
//   - They are extra hands on top of today's automatic specialist work,
//     which is unchanged: a player who never sets a duty sees no change.
//   - A member on any duty misses the Rested buff; a watcher also ends the
//     rest no better than Ready (fatigue is capped at survival's Ready band).
//   - A spoiled rest (raiders caught the camp asleep) settles no duties.

// presentKeys are the keys of the leader and the live companions in the
// leader's room: the members who take their duties this rest.
func (m *CampingModule) presentKeys(user *users.UserRecord) map[string]bool {
	targets, _ := m.prepTargets(user)
	out := make(map[string]bool, len(targets))
	for _, t := range targets {
		out[t.key] = true
	}
	return out
}

// dutyOptions lists the duties a member can take: brew only for an
// Alchemist, whose satchel it fills.
func dutyOptions(c *characters.Character) []camping.Duty {
	out := make([]camping.Duty, 0, len(camping.AllDuties))
	for _, info := range camping.AllDuties {
		if info.Duty == camping.DutyBrew && (c == nil || !flasks.IsAlchemist(c)) {
			continue
		}
		out = append(out, info.Duty)
	}
	return out
}

func dutyUsage() string {
	words := make([]string, 0, len(camping.AllDuties))
	for _, info := range camping.AllDuties {
		words = append(words, string(info.Duty))
	}
	return "Usage: camp duties | camp duties [member|all] " + strings.Join(words, "|") + " | camp duties clear (help campduties)"
}

// dutyTargets resolves the member word to the targets it names.
func (m *CampingModule) dutyTargets(user *users.UserRecord, word string) ([]prepTarget, string) {
	all, _ := m.prepTargets(user)
	return resolveTargets(word, all)
}

// dutiesCommand is "camp duties": it shows the duties, or sets them for
// the next rest.
func (m *CampingModule) dutiesCommand(user *users.UserRecord, room *rooms.Room, args []string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	m.mu.Lock()
	camp, ok := m.camps[user.UserId]
	m.mu.Unlock()
	if !ok {
		return `You have no camp. Use "camp" to make one, then set its duties.`
	}
	if len(args) == 0 {
		return m.dutiesView(user, camp)
	}
	if camp.Rest != nil && camp.Rest.State == camping.Resting {
		return "The company is resting: duties were fixed when the rest began. Set them for the next rest once it is over."
	}
	if room == nil || camp.RoomID != room.RoomId {
		return "Your camp is not here: set its duties at the camp."
	}
	if len(args) == 1 && (strings.EqualFold(args[0], "clear") || strings.EqualFold(args[0], "reset")) {
		return m.assignDuties(user, camp, nil, camping.DutySleep, true)
	}
	if len(args) == 1 {
		// 51 review: a bare duty word is the leader's own ("camp duties
		// watch"), as "camp prepare" takes no member for the leader.
		if _, ok := camping.ParseDuty(args[0]); !ok {
			return dutyUsage()
		}
		args = []string{"me", args[0]}
	}
	duty, valid := camping.ParseDuty(args[len(args)-1])
	if !valid {
		return fmt.Sprintf("There is no duty called %q. %s", args[len(args)-1], dutyUsage())
	}
	targets, refusal := m.dutyTargets(user, strings.Join(args[:len(args)-1], " "))
	if refusal != "" {
		return refusal
	}
	return m.assignDuties(user, camp, targets, duty, false)
}

// assignDuties applies duty to targets (or clears every duty). It is saved
// before anything is reported, and put back if the save fails.
func (m *CampingModule) assignDuties(user *users.UserRecord, camp camping.Camp, targets []prepTarget, duty camping.Duty, clear bool) string {
	var lines []string
	all, _ := m.prepTargets(user)
	next := camp.Duties
	if clear {
		next = nil
		lines = append(lines, "Everyone will sleep through the next rest.")
	}
	for _, t := range targets {
		if duty == camping.DutyBrew && !flasks.IsAlchemist(t.char) {
			lines = append(lines, fmt.Sprintf("%s has no flask satchel to fill, so can't take the brew duty.", t.name))
			continue
		}
		next = camping.WithDuty(next, t.key, duty)
		lines = append(lines, dutyLine(dutyName(all, t), duty))
	}
	m.mu.Lock()
	current, ok := m.camps[user.UserId]
	if !ok || (current.Rest != nil && current.Rest.State == camping.Resting) {
		m.mu.Unlock()
		return "The duties couldn't be set right now."
	}
	updated := current
	updated.Duties = next
	m.camps[user.UserId] = updated
	if err := m.saveLocked(); err != nil {
		m.camps[user.UserId] = current
		m.mu.Unlock()
		mudlog.Error("camping: duties save", "leader", user.UserId, "error", err)
		return "The duties couldn't be saved; nothing changed."
	}
	m.mu.Unlock()
	return strings.Join(lines, "\n")
}

func dutyLine(name string, duty camping.Duty) string {
	for _, info := range camping.AllDuties {
		if info.Duty == duty {
			return fmt.Sprintf("%s %s.", name, info.Does)
		}
	}
	return name + " sleeps."
}

// dutiesView lists each member's duty: the locked ones while resting, the
// standing ones otherwise.
func (m *CampingModule) dutiesView(user *users.UserRecord, camp camping.Camp) string {
	duties, resting := camp.Duties, false
	if camp.Rest != nil && camp.Rest.State == camping.Resting {
		duties, resting = camp.Rest.Duties, true
	}
	targets, _ := m.prepTargets(user)
	head := "Rest duties for the next rest (help campduties):"
	if resting {
		head = "Rest duties, fixed for this rest (help campduties):"
	}
	lines := []string{head}
	for _, t := range targets {
		d := camping.DutyOf(duties, t.key)
		label := string(d)
		if d == camping.DutySleep {
			label += " (default)"
		}
		lines = append(lines, fmt.Sprintf("  %s: %s", dutyName(targets, t), label))
	}
	if !resting {
		lines = append(lines, "Set one with: camp duties [member|all] "+dutyWords()+". Anyone on a duty misses the Rested buff; a watcher also ends no better than Ready.")
	}
	return strings.Join(lines, "\n")
}

func dutyWords() string {
	words := make([]string, 0, len(camping.AllDuties))
	for _, info := range camping.AllDuties {
		words = append(words, string(info.Duty))
	}
	return strings.Join(words, "|")
}

// dutyNames maps member keys to roster names.
func (m *CampingModule) dutyNames(leaderUserID int) map[string]string {
	names := map[string]string{}
	for _, member := range m.survival.CompanyNeeds(leaderUserID) {
		names[string(member.Key)] = member.Name
	}
	return names
}

// dutyStartText is the rest-start line naming who works.
func (m *CampingModule) dutyStartText(user *users.UserRecord, duties map[string]string) string {
	if len(duties) == 0 {
		return ""
	}
	return "On duty tonight: " + m.dutySummary(user.UserId, duties) + "."
}

// dutySummary is "Mira (watch), Brannoc (cook)" in key order.
func (m *CampingModule) dutySummary(leaderUserID int, duties map[string]string) string {
	names := m.dutyNames(leaderUserID)
	keys := make([]string, 0, len(duties))
	for key := range duties {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return dutyKeyLess(keys[i], keys[j]) })
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		name := names[key]
		if name == "" {
			name = "a member"
		}
		parts = append(parts, fmt.Sprintf("%s (%s)", name, camping.DutyOf(duties, key)))
	}
	return strings.Join(parts, ", ")
}

// dutyKeyLess orders the leader first, then companions by id.
func dutyKeyLess(a, b string) bool {
	ia, ib := dutyKeyID(a), dutyKeyID(b)
	return ia < ib
}

// dutyKeyID is 0 for the leader and the companion id otherwise.
func dutyKeyID(key string) int {
	if id, ok := company.CompanionIDFromMemberKey(survival.MemberKey(key)); ok {
		return id
	}
	return 0
}

// announceDuties is the Phase 49 hook: a rest began with duties locked.
func (m *CampingModule) announceDuties(leaderUserID int) {
	if m.onDuties == nil {
		return
	}
	if duties := m.restDuties(leaderUserID); len(duties) > 0 {
		m.onDuties(leaderUserID, duties)
	}
}

// restDuties is a copy of the duties locked on the leader's current rest.
func (m *CampingModule) restDuties(leaderUserID int) map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	camp, ok := m.camps[leaderUserID]
	if !ok || camp.Rest == nil || len(camp.Rest.Duties) == 0 {
		return nil
	}
	out := make(map[string]string, len(camp.Rest.Duties))
	for k, v := range camp.Rest.Duties {
		out[k] = v
	}
	return out
}

// pendingDuties are the duties saved with a pending Rested grant (51
// review: not the camp's current rest, which a break or a new rest may
// have replaced before the grant). An inn's Well Rested has none.
func (m *CampingModule) pendingDuties(leaderUserID int, tier camping.Tier) map[string]string {
	if tier != camping.TierRested {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.restedDuties[leaderUserID]
}

// utilityLevel is a member's own level at a utility.
func (m *CampingModule) utilityLevel(leaderUserID int, key, utility string) int {
	level := archetypes.MemberUtilityLevel
	if m.memberLevel != nil {
		level = m.memberLevel
	}
	return level(leaderUserID, dutyKeyID(key), utility)
}

// --- watch ---

// watchChances are the spot chances of the members on watch: their own
// utility level at the usual rate, never less than WatcherBasePct, so even a
// watcher with no eye for it adds something.
func (m *CampingModule) watchChances(leaderUserID int, watchers []string) []int {
	cfg := m.campSettings()
	out := make([]int, 0, len(watchers))
	for _, key := range watchers {
		pct := archetypes.PctByLevel(m.utilityLevel(leaderUserID, key, archetypes.UtilityWatch), cfg.WatchPctPerLevel, 100)
		out = append(out, max(pct, cfg.WatcherBasePct))
	}
	return out
}

// combineChances is the chance that at least one independent watch spots
// the raiders: 1 - the product of each missing it.
func combineChances(chances ...int) int {
	miss := 1.0
	for _, c := range chances {
		miss *= 1 - float64(min(max(c, 0), 100))/100
	}
	return int(100*(1-miss) + 0.5)
}

// addDutyWatchers folds the members on the watch duty into the raid's
// spot chance. The specialist's own posting is dropped when they are on
// the duty too, so one person never counts twice. It returns the member
// named when the raiders are spotted (the surest eye), whether any watch is
// posted, and the combined chance before bells.
func (m *CampingModule) addDutyWatchers(leader *users.UserRecord, duties map[string]string, auto archetypes.Specialist, hasAuto bool, autoPct int) (archetypes.Specialist, bool, int) {
	workers := m.dutyWorkers(leader, duties, camping.DutyWatch)
	if len(workers) == 0 {
		return auto, hasAuto, autoPct
	}
	keys := make([]string, len(workers))
	for i, w := range workers {
		keys[i] = w.key
	}
	chances := m.watchChances(leader.UserId, keys)
	var chances2 []int
	spotter, best := archetypes.Specialist{}, -1
	for i, w := range workers {
		if hasAuto && isSpecialist(auto, w.key, w.name) {
			hasAuto = false // the specialist is on watch duty: counted below
		}
		chances2 = append(chances2, chances[i])
		if chances[i] > best {
			best = chances[i]
			spotter = archetypes.Specialist{Name: w.name, IsLeader: w.key == string(survival.LeaderMemberKey), Level: m.utilityLevel(leader.UserId, w.key, archetypes.UtilityWatch)}
		}
	}
	if hasAuto {
		chances2 = append(chances2, autoPct)
		if autoPct > best {
			spotter = auto
		}
	}
	return spotter, true, combineChances(chances2...)
}

// isSpecialist reports whether the member with key and name is sp (a
// Specialist carries no companion id, so a companion is matched by name).
func isSpecialist(sp archetypes.Specialist, key, name string) bool {
	if sp.IsLeader {
		return key == string(survival.LeaderMemberKey)
	}
	return key != string(survival.LeaderMemberKey) && sp.Name == name
}

// --- the rest ends ---

// dutyWorker is a member on duty with the character to act for them.
type dutyWorker struct {
	key  string
	name string
	char *characters.Character
	cook campCook
}

// dutyWorkers resolves the members holding duty who are still at the camp
// (the leader and live companions in the leader's room).
func (m *CampingModule) dutyWorkers(user *users.UserRecord, duties map[string]string, duty camping.Duty) []dutyWorker {
	keys := camping.DutyMembers(duties, duty)
	if len(keys) == 0 {
		return nil
	}
	targets, _ := m.prepTargets(user)
	byKey := map[string]prepTarget{}
	for _, t := range targets {
		byKey[t.key] = t
	}
	sort.Slice(keys, func(i, j int) bool { return dutyKeyLess(keys[i], keys[j]) })
	var out []dutyWorker
	for _, key := range keys {
		t, ok := byKey[key]
		if !ok || t.char == nil || t.char.Health < 1 {
			continue
		}
		out = append(out, dutyWorker{key: key, name: t.name, char: t.char})
	}
	return out
}

func (w dutyWorker) isLeader() bool { return w.key == string(survival.LeaderMemberKey) }

// subject names the worker at the start of a sentence.
func (w dutyWorker) subject() string {
	if w.isLeader() {
		return "You"
	}
	return w.name
}

// verb agrees with subject: "you tend", "Mira tends".
func (w dutyWorker) verb(you, other string) string {
	if w.isLeader() {
		return you
	}
	return other
}

// settleDuties runs the work duties of a finished, unspoiled rest once
// (the caller is the grant that cleared the rest's marker) and returns the
// lines to report. Forage is settled with the camp rewards, under their
// cooldown, and watch was settled when the raiders came.
func (m *CampingModule) settleDuties(user *users.UserRecord, live map[int]*characters.Character, duties map[string]string) []string {
	if len(duties) == 0 {
		return nil
	}
	var lines []string
	if watchers := camping.DutyMembers(duties, camping.DutyWatch); len(watchers) > 0 {
		names := m.workerNames(user, watchers)
		lines = append(lines, fmt.Sprintf("%s kept watch through the night.", names))
	}
	if m.inBattle != nil && m.inBattle(user.UserId) {
		return append(lines, "The company was fighting when the rest ended; the rest of its duties were left undone.")
	}
	lines = append(lines, m.settleTending(user, duties)...)
	lines = append(lines, m.settleCooking(user, duties)...)
	lines = append(lines, m.settleForageNote(user, duties)...)
	if len(lines) == 0 {
		return nil
	}
	return append([]string{"Rest duties:"}, lines...)
}

// workerNames is "you", "Mira" or "Mira and Brannoc" for member keys.
func (m *CampingModule) workerNames(user *users.UserRecord, keys []string) string {
	names := m.dutyNames(user.UserId)
	sort.Slice(keys, func(i, j int) bool { return dutyKeyLess(keys[i], keys[j]) })
	var out []string
	for _, key := range keys {
		name := names[key]
		if key == string(survival.LeaderMemberKey) {
			name = "You"
		} else if name == "" {
			name = "A member"
		}
		out = append(out, name)
	}
	switch len(out) {
	case 0:
		return "Nobody"
	case 1:
		return out[0]
	}
	return strings.Join(out[:len(out)-1], ", ") + " and " + out[len(out)-1]
}

// settleTending has each tender treat one lasting wound with the surgeon's
// kit packed for the rest, or else sharpen one member's blades with the
// leader's whetstone (the existing field-surgery and whetstone rules).
func (m *CampingModule) settleTending(user *users.UserRecord, duties map[string]string) []string {
	tenders := m.dutyWorkers(user, duties, camping.DutyTend)
	var lines []string
	var idle []dutyWorker
	kit := m.restKit(user.UserId)
	for _, w := range tenders {
		if kit {
			if treated, ok := m.fieldSurgery(user.UserId); ok {
				lines = append(lines, treated...)
				lines = append(lines, fmt.Sprintf("%s %s the lantern while the surgeon's kit does its work; the kit wears a little with it.", w.subject(), w.verb("hold", "holds")))
				continue
			}
		}
		if text := m.sharpenOne(user); text != "" {
			lines = append(lines, fmt.Sprintf("%s %s the company's gear: %s", w.subject(), w.verb("tend", "tends"), text))
			continue
		}
		idle = append(idle, w)
	}
	// Tenders with nothing to do say so once between them, not once each.
	switch len(idle) {
	case 0:
	case 1:
		lines = append(lines, fmt.Sprintf("%s %s nothing to tend: no wound the kit can treat, and no blade a whetstone can help.", idle[0].subject(), idle[0].verb("find", "finds")))
	default:
		keys := make([]string, len(idle))
		for i, w := range idle {
			keys[i] = w.key
		}
		lines = append(lines, fmt.Sprintf("%s find nothing to tend: no wound the kit can treat, and no blade a whetstone can help.", m.workerNames(user, keys)))
	}
	return lines
}

// settleCooking has each cook turn out one dish from the pack and cargo.
func (m *CampingModule) settleCooking(user *users.UserRecord, duties map[string]string) []string {
	recipes := m.campSettings().Recipes
	var lines []string
	for _, w := range m.dutyWorkers(user, duties, camping.DutyCook) {
		if len(recipes) == 0 {
			lines = append(lines, "There is nothing to cook over a campfire.")
			break
		}
		cook := m.cookFor(user, w, recipes)
		lines = append(lines, m.cookDish(user, rooms.LoadRoom(user.Character.RoomId), cook))
	}
	return lines
}

// cookFor is the camp cook for a worker: their own ranks.
func (m *CampingModule) cookFor(user *users.UserRecord, w dutyWorker, recipes []campRecipe) campCook {
	if w.key == string(survival.LeaderMemberKey) {
		return campCook{Name: user.Character.Name, IsLeader: true, Ranks: skillsOf(user.Character.GetSkillLevel, recipes)}
	}
	return campCook{Name: w.name, ID: dutyKeyID(w.key), Ranks: skillsOf(w.char.GetSkillLevel, recipes)}
}

// settleForageNote tells foragers when the ground was picked over: the
// find itself is granted with the camp rewards (grantCampRewards).
func (m *CampingModule) settleForageNote(user *users.UserRecord, duties map[string]string) []string {
	foragers := camping.DutyMembers(duties, camping.DutyForage)
	if len(foragers) == 0 {
		return nil
	}
	m.mu.Lock()
	reward, owed := m.campRewards[user.UserId]
	m.mu.Unlock()
	if owed && reward.Foragers != "" {
		return nil // reported when the finds are collected
	}
	return []string{fmt.Sprintf("%s went foraging, but the ground here is still picked over from the last forage.", m.workerNames(user, foragers))}
}

// restBrewers maps the characters on the brew duty.
func (m *CampingModule) restBrewers(user *users.UserRecord, duties map[string]string) map[*characters.Character]bool {
	out := map[*characters.Character]bool{}
	for _, w := range m.dutyWorkers(user, duties, camping.DutyBrew) {
		out[w.char] = true
	}
	return out
}

// restedExcluded are the member keys on a duty, who miss the Rested buff.
func restedExcluded(duties map[string]string) map[string]bool {
	out := map[string]bool{}
	for key := range duties {
		if camping.OnDuty(duties, key) {
			out[key] = true
		}
	}
	return out
}

// fatigueCeilings caps the fatigue of the members who kept watch.
func fatigueCeilings(duties map[string]string) map[survival.MemberKey]int {
	var out map[survival.MemberKey]int
	for _, key := range camping.DutyMembers(duties, camping.DutyWatch) {
		if out == nil {
			out = map[survival.MemberKey]int{}
		}
		out[survival.MemberKey(key)] = camping.WatchFatigueCap
	}
	return out
}

// dutyRows are the Camp tab's picker rows for the leader's members at the
// camp.
func (m *CampingModule) dutyRows(user *users.UserRecord, camp camping.Camp) []camping.DutyRow {
	duties := camp.Duties
	if camp.Rest != nil && camp.Rest.State == camping.Resting {
		duties = camp.Rest.Duties
	}
	targets, _ := m.prepTargets(user)
	rows := make([]camping.DutyRow, 0, len(targets))
	for _, t := range targets {
		var options []string
		for _, d := range dutyOptions(t.char) {
			options = append(options, string(d))
		}
		name := dutyName(targets, t)
		command := t.name
		if t.key == string(survival.LeaderMemberKey) {
			command = "me"
		} else if sameName(targets, t) {
			// Two members of one name: the picker names each by #id.
			if id, ok := company.CompanionIDFromMemberKey(survival.MemberKey(t.key)); ok {
				command = "#" + strconv.Itoa(id)
			}
		}
		rows = append(rows, camping.DutyRow{Key: t.key, Name: name, Command: command, Duty: string(camping.DutyOf(duties, t.key)), Options: options})
	}
	return rows
}

// sameName reports whether another target shares t's name.
func sameName(targets []prepTarget, t prepTarget) bool {
	for _, o := range targets {
		if o.key != t.key && strings.EqualFold(o.name, t.name) {
			return true
		}
	}
	return false
}

// dutyName is a member's name, with its #id when another shares the name,
// so the member can be told apart and named in a command.
func dutyName(targets []prepTarget, t prepTarget) string {
	if !sameName(targets, t) {
		return t.name
	}
	if id, ok := company.CompanionIDFromMemberKey(survival.MemberKey(t.key)); ok {
		return t.name + " (#" + strconv.Itoa(id) + ")"
	}
	return t.name
}
