package company

// Phase 35c: "company train". Companions spend training points, derived
// from their level, on optional skills (archetypes.OptionalSkills). Class
// skills and specialists stay automatic. Only the ranks are saved; the
// points are always worked out again, so nothing can mint them.

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/creatures"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const trainUsage = "Usage: company train | company train [member] | company train [member] [skill] [rank] [confirm]"

// campTrainMaxRank is the highest rank a companion can learn at its
// leader's own camp; higher ranks need a trainer.
const campTrainMaxRank = 2

// skillSetter is implemented by the native runtime: it writes a trained
// rank onto a live companion at once, so a companion that is out uses it
// before its next spawn.
type skillSetter interface {
	SetSkill(instanceID int, skill string, rank int) bool
}

// SetSkill writes a rank onto a live mob (never lowering a higher one).
func (nativeRuntime) SetSkill(instanceID int, skill string, rank int) bool {
	mob := mobs.GetInstance(instanceID)
	if mob == nil {
		return false
	}
	if mob.Character.GetSkillLevel(skill) < rank {
		mob.Character.SetSkill(skill, rank)
	}
	return true
}

// applySkills writes a companion's optional-skill ranks onto its live mob.
func applySkills(mob *mobs.Mob, ranks map[string]int) {
	for _, skill := range sortedSkillIDs(ranks) {
		if mob.Character.GetSkillLevel(skill) < ranks[skill] {
			mob.Character.SetSkill(skill, ranks[skill])
		}
	}
}

func sortedSkillIDs(ranks map[string]int) []string {
	ids := make([]string, 0, len(ranks))
	for id, rank := range ranks {
		if rank > 0 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// skillName is a skill's display name.
func skillName(skillID string) string {
	if s := skills.GetSkill(skillID); s != nil && s.Name != "" {
		return s.Name
	}
	return skillID
}

// earnedTrainingPoints is what a level has earned at the configured rate.
func earnedTrainingPoints(level int) int {
	return configs.GetProgressionConfig().TrainingPointsAt(max(level, 1))
}

// trainingLevel is the companion's level: the live mob's when it is out
// (it is ahead of its record between snapshots), else its record's.
func (m *CompanyModule) trainingLevel(leaderUserID int, c domain.Companion) int {
	level := companionLevelNumber(c)
	if id, ok := m.instance(leaderUserID, c.ID); ok && m.runtime.IsLive(id) {
		if live, _, _, ok := m.runtime.Progress(id); ok && live > 0 {
			level = live
		}
	}
	return max(level, 1)
}

// trainingPoints is the companion's points left; negative while a lost
// level is being repaid.
func (m *CompanyModule) trainingPoints(leaderUserID int, c domain.Companion) int {
	if creatures.Is(c.Archetype) {
		return 0 // Phase 38e: a creature learns no optional skills
	}
	return c.TrainingPoints(earnedTrainingPoints(m.trainingLevel(leaderUserID, c)))
}

// train runs "company train".
func (m *CompanyModule) train(user *users.UserRecord, room *rooms.Room, args []string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	if len(args) == 0 {
		return m.trainView(user.UserId)
	}
	confirm := strings.EqualFold(args[len(args)-1], "confirm")
	if confirm {
		args = args[:len(args)-1]
	}
	record, _ := m.registry.Get(user.UserId)
	if len(args) == 0 {
		return trainUsage
	}
	// "company train <member>": that member's choices. A member's full
	// name may have spaces, so try the whole rest as a name first.
	if !confirm {
		if c, ok := resolveCompanion(record, strings.Join(args, " ")); ok {
			return m.trainMemberView(user.UserId, c)
		}
	}
	if len(args) < 2 {
		if _, ok := resolveCompanion(record, strings.Join(args, " ")); ok {
			return "Name a skill to train: " + trainUsage
		}
		if _, ambiguous := ambiguousCompanion(record, strings.Join(args, " ")); ambiguous {
			return "More than one companion answers to that; use their number (#N)."
		}
		return "You have no companion like that."
	}
	// An optional rank follows the skill: the preview's confirm names it,
	// so repeating that confirm finds the rank reached and does nothing.
	want := 0
	if len(args) >= 3 {
		if n, err := strconv.Atoi(args[len(args)-1]); err == nil && n > 0 {
			want, args = n, args[:len(args)-1]
		}
	}
	// The skill is the last word, so member names may have spaces.
	selector, skill := strings.Join(args[:len(args)-1], " "), strings.ToLower(args[len(args)-1])
	return m.trainSkill(user, room, record, selector, skill, want, confirm)
}

// ambiguousCompanion reports whether a selector matches more than one
// companion by name.
func ambiguousCompanion(record domain.Record, selector string) (int, bool) {
	selector = strings.ToLower(strings.TrimSpace(selector))
	if selector == "" {
		return 0, false
	}
	if _, err := strconv.Atoi(strings.TrimPrefix(selector, "#")); err == nil {
		return 0, false
	}
	exact, partial := domain.SplitNameMatches(record.Companions, selector, func(c domain.Companion) string { return nameOf(c, "") })
	if len(exact) > 1 || (len(exact) == 0 && len(partial) > 1) {
		return len(exact) + len(partial), true
	}
	return 0, false
}

// resolveSkill matches an optional skill by id or display name prefix.
func resolveSkill(word string) (archetypes.OptionalSkill, bool) {
	word = strings.ToLower(strings.TrimSpace(word))
	if o, ok := archetypes.Optional(word); ok {
		return o, true
	}
	var match []archetypes.OptionalSkill
	for _, o := range archetypes.OptionalSkills() {
		if strings.HasPrefix(o.Skill, word) || strings.HasPrefix(strings.ToLower(skillName(o.Skill)), word) {
			match = append(match, o)
		}
	}
	if len(match) == 1 {
		return match[0], true
	}
	return archetypes.OptionalSkill{}, false
}

// trainView lists each companion's points and what each could learn next.
func (m *CompanyModule) trainView(leaderUserID int) string {
	record, ok := m.registry.Get(leaderUserID)
	if !ok || len(record.Companions) == 0 {
		return "You have no companions to train."
	}
	if len(archetypes.OptionalSkills()) == 0 {
		return "There are no skills your companions can be trained in right now."
	}
	lines := []string{"Company training (optional skills; class skills come with level):"}
	for _, c := range record.Companions {
		lines = append(lines, m.trainLines(leaderUserID, c)...)
	}
	lines = append(lines,
		"Ranks 1-2 train at your own camp or a trainer; ranks 3-4 only at a trainer who teaches that rank.",
		"Preview with company train [member] [skill], then add confirm. See help company-train.")
	return strings.Join(lines, "\n")
}

// trainMemberView is "company train <member>".
func (m *CompanyModule) trainMemberView(leaderUserID int, c domain.Companion) string {
	lines := m.trainLines(leaderUserID, c)
	return strings.Join(append(lines, fmt.Sprintf("Preview with company train %s [skill], then add confirm.", trainSelector(c))), "\n")
}

// trainLines is one companion's heading and its optional skills.
func (m *CompanyModule) trainLines(leaderUserID int, c domain.Companion) []string {
	if creatures.Is(c.Archetype) {
		return []string{fmt.Sprintf("  #%d %s, %s, level %d: a creature; it learns by living and has nothing to train", c.ID, nameOf(c, strconv.Itoa(c.MobTemplateID)), archetypeLabel(c.Archetype), m.trainingLevel(leaderUserID, c))}
	}
	points := m.trainingPoints(leaderUserID, c)
	head := fmt.Sprintf("  #%d %s, %s, level %d: %s", c.ID, nameOf(c, strconv.Itoa(c.MobTemplateID)), archetypeLabel(c.Archetype), m.trainingLevel(leaderUserID, c), pointsLabel(points))
	lines := []string{head}
	for _, o := range archetypes.OptionalSkills() {
		rank := c.SkillRank(o.Skill)
		name := skillName(o.Skill)
		switch {
		case !o.Allows(c.Archetype):
			if rank > 0 {
				lines = append(lines, fmt.Sprintf("      %s rank %d (no further training for a %s)", name, rank, archetypeLabel(c.Archetype)))
			}
		case rank >= o.MaxRank:
			lines = append(lines, fmt.Sprintf("      %s rank %d (mastered)", name, rank))
		default:
			lines = append(lines, fmt.Sprintf("      %s rank %d: rank %d costs %d", name, rank, rank+1, domain.SkillRankCost(rank+1)))
		}
	}
	if len(lines) == 1 {
		lines = append(lines, "      nothing they can learn")
	}
	return lines
}

func pointsLabel(points int) string {
	switch {
	case points < 0:
		return fmt.Sprintf("0 training points (%d owed from a lost level)", -points)
	case points == 1:
		return "1 training point"
	}
	return fmt.Sprintf("%d training points", points)
}

// trainSelector is how a companion is named in a typed command.
func trainSelector(c domain.Companion) string {
	return "#" + strconv.Itoa(c.ID)
}

// trainPlace says where the next rank can be learned here, or why not.
func trainPlace(leaderUserID int, room *rooms.Room, skill string, rank int) (string, bool) {
	if room != nil {
		if r, ok := room.SkillTraining[skill]; ok && rank >= r.Min && rank <= r.Max {
			return "the trainer here", true
		}
	}
	if rank <= campTrainMaxRank && room != nil {
		if s, ok := camping.CampStateOf(leaderUserID, room.RoomId, room.GetTags()); ok && s.HasCamp && s.Here {
			return "your camp", true
		}
	}
	if rank <= campTrainMaxRank {
		return fmt.Sprintf("Rank %d can be learned at your own camp or from a trainer who teaches it.", rank), false
	}
	return fmt.Sprintf("Rank %d can only be learned from a trainer who teaches it.", rank), false
}

// trainSkill previews, or with confirm commits, one rank. Confirm checks
// everything again, so a preview that has gone stale can't commit, and a
// repeated confirm finds the rank already reached.
func (m *CompanyModule) trainSkill(user *users.UserRecord, room *rooms.Room, record domain.Record, selector, skillWord string, want int, confirm bool) string {
	leader := user.UserId
	c, found := resolveCompanion(record, selector)
	if !found {
		if _, ambiguous := ambiguousCompanion(record, selector); ambiguous {
			return "More than one companion answers to that; use their number (#N)."
		}
		return "You have no companion like that."
	}
	name := nameOf(c, "#"+strconv.Itoa(c.ID))
	o, ok := resolveSkill(skillWord)
	if !ok {
		return fmt.Sprintf("%s can't be trained in that. Companions learn only optional skills; see company train.", name)
	}
	skill, label := o.Skill, skillName(o.Skill)
	rank := c.SkillRank(skill)
	target := rank + 1
	switch {
	case want > 0 && rank >= want:
		return fmt.Sprintf("%s already has %s rank %d; there is nothing to do.", name, label, rank)
	case want > target:
		return fmt.Sprintf("%s must learn %s rank %d first.", name, label, target)
	}

	// The skill first, then the company's state, then the member's.
	if !o.Allows(c.Archetype) {
		return fmt.Sprintf("A %s can't learn %s.", archetypeLabel(c.Archetype), label)
	}
	if rank >= o.MaxRank {
		return fmt.Sprintf("%s already has %s rank %d, the most a companion can learn.", name, label, rank)
	}
	if usercommands.InBattle(user) {
		return "Not in the middle of a battle. Train your company once the fighting is done."
	}
	if blocked, _ := expedition.MovementBlocked(leader); blocked {
		return "Not while your company is travelling. Train once you have arrived."
	}
	if blocked, _ := camping.MovementBlocked(leader); blocked {
		return "Not while your company is resting. Train once the rest is over."
	}
	switch {
	case c.Dead():
		return fmt.Sprintf("%s has fallen and can't train until raised.", name)
	case c.Separated():
		return fmt.Sprintf("%s is separated from the company and can't train until they rejoin.", name)
	}
	instanceID, tracked := m.instance(leader, c.ID)
	if c.PendingReturn || !tracked || !m.runtime.IsLive(instanceID) || !m.runtime.IsAttached(leader, instanceID) || !m.runtime.WithLeader(leader, instanceID) {
		return fmt.Sprintf("%s isn't here with you to train.", name)
	}
	if _, fighting, ok := m.runtime.Standing(instanceID); !ok || fighting {
		return fmt.Sprintf("%s is busy fighting.", name)
	}
	cost := domain.SkillRankCost(target)
	points := m.trainingPoints(leader, c)
	if points < cost {
		return fmt.Sprintf("%s needs %d training points for %s rank %d and has %s.", name, cost, label, target, pointsLabel(points))
	}
	place, here := trainPlace(leader, room, skill, target)
	if !here {
		return fmt.Sprintf("%s can't learn %s rank %d here. %s", name, label, target, place)
	}
	if !confirm {
		lines := []string{
			fmt.Sprintf("%s can learn %s rank %d at %s for %d training point%s (%d now, %d after).",
				name, label, target, place, cost, plural(cost), points, points-cost),
		}
		if s := skills.GetSkill(skill); s != nil && s.Description != "" {
			lines = append(lines, label+": "+s.Description)
		}
		lines = append(lines, fmt.Sprintf("Type company train %s %s %d confirm to train.", trainSelector(c), skill, target))
		return strings.Join(lines, "\n")
	}

	before, _ := m.registry.Get(leader)
	// The points came from the live level; record it in the same save, so
	// a crash can't leave the ranks above a lower saved level.
	if c.State != nil {
		if level, _, _, ok := m.runtime.Progress(instanceID); ok && level > c.State.Level {
			state := c.State.Clone()
			state.Level = level
			if live, ok := m.runtime.Snapshot(instanceID); ok {
				state.Experience = live.Experience
			}
			_ = m.registry.SetState(leader, c.ID, state)
		}
	}
	if err := m.registry.SetSkillRank(leader, c.ID, skill, target); err != nil {
		return "Your company can't be changed right now."
	}
	if err := m.save(); err != nil {
		m.registry.Put(before)
		return "Your company's records couldn't be saved; nothing was trained. Please try again."
	}
	if setter, ok := m.runtime.(skillSetter); ok {
		setter.SetSkill(instanceID, skill, target)
	}
	return fmt.Sprintf("%s learns %s rank %d at %s (%s left).", name, label, target, place, pointsLabel(points-cost))
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// inspectMember is "company inspect" of the leader's own companion (Phase
// 35c): its level, trained optional skills, and training points. exact
// matches only #N or a full name; otherwise a unique partial name too.
func (m *CompanyModule) inspectMember(leaderUserID int, selector string, exact bool) (string, bool) {
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return "", false
	}
	sel := strings.ToLower(strings.TrimSpace(selector))
	if exact && !strings.HasPrefix(sel, "#") {
		found := false
		for _, c := range record.Companions {
			if strings.ToLower(nameOf(c, "")) == sel {
				found = true
			}
		}
		if !found {
			return "", false
		}
	}
	if !exact && strings.HasPrefix(sel, "#") {
		return "", false
	}
	c, ok := resolveCompanion(record, sel)
	if !ok {
		return "", false
	}
	level := m.trainingLevel(leaderUserID, c)
	lines := []string{fmt.Sprintf("#%d %s, %s, level %d.", c.ID, nameOf(c, strconv.Itoa(c.MobTemplateID)), archetypeLabel(c.Archetype), level)}
	if f, ok := familyOf(c); ok { // Phase 38e: a creature learns no optional skills
		lines = append(lines, familyLine(f))
		return strings.Join(lines, "\n"), true
	}
	var trained []string
	for _, skill := range sortedSkillIDs(c.Skills) {
		text := fmt.Sprintf("%s rank %d", skillName(skill), c.Skills[skill])
		if g := c.GrantedSkills[skill]; g > 0 {
			text += fmt.Sprintf(" (arrived with rank %d)", g)
		}
		trained = append(trained, text)
	}
	if len(trained) == 0 {
		lines = append(lines, "Optional skills: none yet.")
	} else {
		lines = append(lines, "Optional skills: "+strings.Join(trained, ", ")+".")
	}
	lines = append(lines, "Training: "+pointsLabel(m.trainingPoints(leaderUserID, c))+". See company train.")
	if line := m.opinionLine(leaderUserID, c.ID); line != "" { // Phase 64
		lines = append(lines, line)
	}
	if line := m.bondLine(leaderUserID, c.ID); line != "" { // Phase 65
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n"), true
}
