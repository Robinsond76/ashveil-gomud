package camping

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Camp activities. The Camp tab lists the chores a company can do at its
// camp before it sleeps: sharpen with a whetstone, coat blades with the
// assigned poisons, and cook. Each row says who does it and what it uses,
// or why nothing can be done now, and its button runs the same command
// the player could type. Reading the rows changes nothing. Once the
// company rests the chores close: the rows vanish and the commands refuse.

const restingActivityText = "The company is resting; do this before you sleep."

// restingNow is true while a camp rest or an inn stay runs for the leader.
func (m *CampingModule) restingNow(leaderUserID int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if camp, ok := m.camps[leaderUserID]; ok && camp.Rest != nil && camp.Rest.State == camping.Resting {
		return true
	}
	stay, ok := m.stays[leaderUserID]
	return ok && stay.Resting()
}

// campActivities are the Camp tab's chore rows for a leader standing at
// their camp; nil while the company rests or is travelling.
func (m *CampingModule) campActivities(user *users.UserRecord, camp camping.Camp) []camping.ActivityRow {
	if user == nil || user.Character == nil || m.restingNow(user.UserId) || m.isTravelling(user.UserId) {
		return nil
	}
	return []camping.ActivityRow{
		m.sharpenActivity(user),
		m.poisonActivity(user),
		m.cookActivity(user, camp),
	}
}

func (m *CampingModule) sharpenActivity(user *users.UserRecord) camping.ActivityRow {
	row := camping.ActivityRow{Key: "sharpen", Label: "Sharpen", Command: "camp sharpen"}
	settings := m.sharpenSettings()
	targets, fighting := m.sharpenTargets(user)
	_, uses := whetstones(user.Character, settings.WhetstoneItemId)
	members := make([]camping.SharpenMember, len(targets))
	for i, t := range targets {
		members[i] = t.member
	}
	plan := camping.PlanSharpen(members, uses)
	wanted := plan.Count(camping.Sharpened) + plan.Count(camping.LeftOut)
	switch {
	case fighting:
		row.Note = "Not while your company is fighting."
	case wanted == 0:
		row.Note = "No blades are dull enough to need the whetstone."
	case uses == 0:
		row.Note = fmt.Sprintf("%s have dull blades, but you have no whetstone.", strings.Join(plan.Names(camping.LeftOut), ", "))
	default:
		who := "You hone"
		if smith, ok := m.fieldSmith(user); ok {
			who = smith.Subject() + " " + smith.Verb("hone", "hones")
		}
		row.Ready = true
		row.Note = fmt.Sprintf("%s the blades of %s, using %s (%s left).", who, strings.Join(plan.Names(camping.Sharpened), ", "), plural(plan.UsesSpent, "whetstone use"), plural(uses, "use"))
	}
	return row
}

func (m *CampingModule) poisonActivity(user *users.UserRecord) camping.ActivityRow {
	row := camping.ActivityRow{Key: "poison", Label: "Poison", Command: "camp poison apply"}
	if len(m.poisonPlan(user.UserId)) == 0 {
		row.Note = "No poison is assigned. Set blades with camp poison assign <member|self> <main|off> <poison>."
		return row
	}
	text, rows, ok := m.poisonCheck(user)
	if !ok {
		row.Note = firstLine(text)
		return row
	}
	coat := 0
	for _, r := range rows {
		if !r.skip {
			coat++
		}
	}
	if coat == 0 {
		row.Note = "Every assigned blade already carries its poison."
		return row
	}
	row.Ready = true
	row.Note = fmt.Sprintf("You coat %s from your vials; each lasts a few minutes or wounding blows.", plural(coat, "blade"))
	return row
}

func (m *CampingModule) cookActivity(user *users.UserRecord, camp camping.Camp) camping.ActivityRow {
	row := camping.ActivityRow{Key: "cook", Label: "Cook", Command: "camp cook"}
	recipes := m.campSettings().Recipes
	cook := m.bestCook(user, recipes)
	chosen, blocked := selectCampRecipe(user, cook, recipes)
	who := "You cook"
	if !cook.IsLeader {
		who = cook.Name + " cooks"
	}
	switch {
	case len(recipes) == 0:
		row.Note = "There is nothing to cook over a campfire."
	case m.inBattle != nil && m.inBattle(user.UserId):
		row.Note = "Not while your company is fighting."
	case chosen == nil && blocked != nil:
		row.Note = fmt.Sprintf("%s needs %s rank %d.", itemName(blocked.Output), blocked.Skill, blocked.MinLevel)
	case chosen == nil:
		row.Note = "Nothing to cook with what you carry and know; try a new mix with cook <ingredients>."
	case !camp.FireLit:
		row.Note = fmt.Sprintf("%s %s once the fire is lit.", who, itemName(chosen.Output))
	default:
		row.Ready = true
		row.Note = fmt.Sprintf("%s %s from the ingredients you carry.", who, itemName(chosen.Output))
	}
	return row
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
