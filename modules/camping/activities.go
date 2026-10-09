package camping

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/ansitags"
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
	rows := []camping.ActivityRow{
		m.sharpenActivity(user),
		m.poisonActivity(user),
		m.cookActivity(user, camp),
	}
	// Review: the supplies set by for the rest itself are chores before
	// sleep too, listed only when the company carries them. Draughts and
	// salves stay off the list (they guard a crossing, not a rest), and so
	// does the remedy (the Ailing line keeps its own button).
	if row, ok := m.brothActivity(user, camp); ok {
		rows = append(rows, row)
	}
	if row, ok := m.incenseActivity(user, camp); ok {
		rows = append(rows, row)
	}
	return rows
}

func (m *CampingModule) brothActivity(user *users.UserRecord, camp camping.Camp) (camping.ActivityRow, bool) {
	supply, _ := camping.FindSupply("broth")
	have := m.gearCount(user.UserId, supply.ItemID)
	if have == 0 {
		return camping.ActivityRow{}, false
	}
	row := camping.ActivityRow{Key: "broth", Label: "Broth", Command: "camp prepare broth all"}
	targets, _ := m.prepTargets(user)
	var names []string
	for _, t := range targets {
		if _, held := m.personalBuff(t.char); held || camp.Prepared.HasBroth(t.key) {
			continue
		}
		names = append(names, t.name)
	}
	queued := 0
	if camp.Prepared != nil {
		queued = len(camp.Prepared.Broth)
	}
	switch {
	case len(names) == 0:
		row.Note = "Everyone has broth set by or a draught working."
	case have < len(names)+queued:
		row.Note = fmt.Sprintf("You carry %s; the company needs %d. Set it by for one member with camp prepare broth <member>.", fmt.Sprintf("%d fortifying broth", have), len(names)+queued)
	default:
		row.Ready = true
		row.Note = fmt.Sprintf("Sets fortifying broth by for %s: spent as the rest begins, it fortifies them when the rest is done.", joinNames(leaderAsYou(user, names)))
	}
	return row, true
}

func (m *CampingModule) incenseActivity(user *users.UserRecord, camp camping.Camp) (camping.ActivityRow, bool) {
	supply, _ := camping.FindSupply("incense")
	if m.gearCount(user.UserId, supply.ItemID) == 0 {
		return camping.ActivityRow{}, false
	}
	row := camping.ActivityRow{Key: "incense", Label: "Incense", Command: "camp prepare incense"}
	switch {
	case camp.Prepared != nil && camp.Prepared.Incense:
		row.Note = "Watch incense is already set for the next rest."
	case !m.hasWatch(user.UserId, camp.RoomID):
		row.Note = "Needs a Camp Watch: a member with the watch skill, here at the camp."
	default:
		row.Ready = true
		row.Note = "Sets watch incense by: lit as the rest begins, it sharpens the watch's eye for raiders by ten points."
	}
	return row, true
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
		dull := leaderAsYou(user, plan.Names(camping.LeftOut))
		dull[0] = strings.ToUpper(dull[0][:1]) + dull[0][1:]
		verb := "has"
		if len(dull) > 1 || dull[0] == "You" {
			verb = "have"
		}
		row.Note = fmt.Sprintf("%s %s dull blades, but you have no whetstone.", joinNames(dull), verb)
	default:
		who := "You hone"
		names := leaderAsYou(user, plan.Names(camping.Sharpened))
		if smith, ok := m.fieldSmith(user); ok {
			who = smith.Subject() + " " + smith.Verb("hone", "hones")
		}
		if who == "You hone" {
			for i := range names {
				if names[i] == "you" {
					names[i] = "yourself"
				}
			}
		}
		row.Ready = true
		row.Note = fmt.Sprintf("%s blades for %s, using %s (%s left).", who, joinNames(names), plural(plan.UsesSpent, "whetstone use"), plural(uses, "use"))
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
		row.Note = poisonReason(text)
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
	row.Note = fmt.Sprintf("You coat %s from your vials; each coat lasts a few minutes or a few wounding blows.", plural(coat, "blade"))
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

// poisonReason is the one line of a refused poison check that says why:
// the first blocker, else its closing line (the plan's per-blade rows come
// first and are not a reason). Markup is stripped for the Camp tab.
func poisonReason(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	reason := lines[len(lines)-1]
	for i, line := range lines {
		if strings.HasPrefix(line, "Blocked, nothing spent") && i+1 < len(lines) {
			reason = lines[i+1]
			break
		}
	}
	return strings.TrimSpace(ansitags.Parse(reason, ansitags.StripTags))
}

// joinNames lists names as "A", "A and B" or "A, B and C".
func joinNames(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// leaderAsYou names the leader "you" in a list of member names.
func leaderAsYou(user *users.UserRecord, names []string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		if n == user.Character.Name {
			n = "you"
		}
		out[i] = n
	}
	return out
}
