package camping

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 23b whetstones. The leader's whetstones sharpen the company's
// equipped blades on demand: one use per member sharpened, however many
// blades they carry; members with no dull blade cost nothing; members are
// sharpened leader first, then by company number, until the uses run
// out. Never mid-fight. Everything here runs on the game loop (a user
// command, or the NewRound grant pass), so the stones and the edges
// change together in one step; they persist with the user file and the
// company record at the engine's save points. m.mu guards only the auto
// setting and is never held while touching a character.

const sharpenUsage = "Usage: sharpen | sharpen status | sharpen auto on|off"

func (m *CampingModule) sharpenCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	user.SendText(m.sharpenArgs(user, strings.Fields(strings.ToLower(strings.TrimSpace(rest)))))
	return true, nil
}

// sharpenArgs serves both "sharpen ..." and "camp sharpen ...".
func (m *CampingModule) sharpenArgs(user *users.UserRecord, args []string) string {
	switch {
	case len(args) == 0:
		return m.sharpen(user, false)
	case args[0] == "status" && len(args) == 1:
		return m.sharpenPreview(user)
	case args[0] == "auto" && len(args) == 1:
		return autoText(m.autoSharpenOn(user.UserId))
	case args[0] == "auto" && len(args) == 2 && (args[1] == "on" || args[1] == "off"):
		return m.setAutoSharpen(user.UserId, args[1] == "on")
	}
	return sharpenUsage
}

func autoText(on bool) string {
	if on {
		return "Auto-sharpen is on: your company sharpens its blades when a camp rest ends."
	}
	return "Auto-sharpen is off."
}

func (m *CampingModule) autoSharpenOn(leaderUserID int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.autoSharpen[leaderUserID]
}

// setAutoSharpen stores the leader's preference durably.
func (m *CampingModule) setAutoSharpen(leaderUserID int, on bool) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	was := m.autoSharpen[leaderUserID]
	if on {
		m.autoSharpen[leaderUserID] = true
	} else {
		delete(m.autoSharpen, leaderUserID)
	}
	if err := m.saveLocked(); err != nil {
		if was {
			m.autoSharpen[leaderUserID] = true
		} else {
			delete(m.autoSharpen, leaderUserID)
		}
		return err.Error()
	}
	return autoText(on)
}

func (m *CampingModule) hasCamp(leaderUserID int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.camps[leaderUserID]
	return ok
}

// sharpenTarget is one member's character and plan row.
type sharpenTarget struct {
	member camping.SharpenMember
	char   *characters.Character
}

// blades returns a character's equipped bladed weapons.
func blades(c *characters.Character) []*items.Item {
	var out []*items.Item
	for _, itm := range []*items.Item{&c.Equipment.Weapon, &c.Equipment.Offhand} {
		if itm.ItemId <= 0 {
			continue
		}
		spec := itm.GetSpec()
		if spec.Type == items.Weapon && camping.Bladed(string(spec.Subtype)) {
			out = append(out, itm)
		}
	}
	return out
}

func memberFor(id int, c *characters.Character, present bool) sharpenTarget {
	member := camping.SharpenMember{ID: id, Name: c.Name, Present: present}
	for _, blade := range blades(c) {
		if blade.Sharpened() {
			member.Sharp++
		} else {
			member.Dull++
		}
	}
	return sharpenTarget{member: member, char: c}
}

// sharpenTargets lists the leader and every rostered companion, and
// whether any of them is fighting. A live companion is present when in
// the leader's room. A companion with no live mob has its gear on the
// company record, not at hand: it is listed as not here and left alone.
func (m *CampingModule) sharpenTargets(user *users.UserRecord) ([]sharpenTarget, bool) {
	targets := []sharpenTarget{memberFor(camping.LeaderID, user.Character, true)}
	fighting := user.Character.Aggro != nil
	live, roster := m.companions(user.UserId)
	for _, companionID := range sortedIDs(live) {
		c := live[companionID]
		if c.Aggro != nil {
			fighting = true
		}
		targets = append(targets, memberFor(companionID, c, c.RoomId == user.Character.RoomId))
	}
	var names map[int]string
	for _, companionID := range roster {
		if _, ok := live[companionID]; ok {
			continue
		}
		if names == nil {
			names = rosterNames(user.UserId)
		}
		name := names[companionID]
		if name == "" {
			name = fmt.Sprintf("companion #%d", companionID)
		}
		targets = append(targets, sharpenTarget{member: camping.SharpenMember{ID: companionID, Name: name}})
	}
	return targets, fighting
}

// rosterNames maps companion IDs to their roster names.
func rosterNames(leaderUserID int) map[int]string {
	names := map[int]string{}
	for _, ref := range survival.CurrentRoster(leaderUserID) {
		if id, ok := company.CompanionIDFromMemberKey(ref.Key); ok {
			names[id] = ref.Name
		}
	}
	return names
}

func (m *CampingModule) sharpenSettings() innSettings {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.innSettings()
}

// whetstones returns the indexes of the leader's whetstones in c.Items,
// most used first, and their total uses left.
func whetstones(c *characters.Character, stoneID int) ([]int, int) {
	var idx []int
	total := 0
	for i, itm := range c.Items {
		if itm.ItemId == stoneID && itm.Uses > 0 {
			idx = append(idx, i)
			total += itm.Uses
		}
	}
	sort.SliceStable(idx, func(a, b int) bool { return c.Items[idx[a]].Uses < c.Items[idx[b]].Uses })
	return idx, total
}

// spendWhetstones spends n uses, most-used stone first, and removes each
// stone worn away (Validate would refill a stone kept at zero uses). It
// returns the stones removed.
func spendWhetstones(user *users.UserRecord, stones []int, n int) []items.Item {
	c := user.Character
	worn := map[int]bool{}
	for _, i := range stones {
		for n > 0 && c.Items[i].Uses > 0 {
			c.Items[i].Uses--
			n--
		}
		if c.Items[i].Uses <= 0 {
			worn[i] = true
		}
	}
	if len(worn) == 0 {
		return nil
	}
	var spent []items.Item
	kept := make([]items.Item, 0, len(c.Items)-len(worn))
	for i, itm := range c.Items {
		if worn[i] {
			spent = append(spent, itm)
			continue
		}
		kept = append(kept, itm)
	}
	c.Items = kept
	for _, itm := range spent {
		events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: itm, Gained: false})
	}
	return spent
}

// sharpen runs one pass. Auto mode (a camp rest's end) is silent when it
// has nothing to do.
func (m *CampingModule) sharpen(user *users.UserRecord, auto bool) string {
	settings := m.sharpenSettings()
	targets, fighting := m.sharpenTargets(user)
	if fighting {
		if auto {
			return "Your company was fighting when the rest ended; no blades were sharpened."
		}
		return "You can't sharpen blades in the middle of a fight."
	}
	stones, uses := whetstones(user.Character, settings.WhetstoneItemId)
	members := make([]camping.SharpenMember, len(targets))
	byID := map[int]sharpenTarget{}
	for i, t := range targets {
		members[i] = t.member
		byID[t.member.ID] = t
	}
	plan := camping.PlanSharpen(members, uses)
	wanted := plan.Count(camping.Sharpened) + plan.Count(camping.LeftOut)
	switch {
	case wanted == 0 && auto:
		return ""
	case wanted == 0:
		return nothingToSharpen(plan)
	case uses == 0 && auto:
		return ""
	case uses == 0:
		return "You have no whetstone."
	}

	for _, e := range plan.Entries {
		if e.Outcome != camping.Sharpened {
			continue
		}
		for _, blade := range blades(byID[e.Member.ID].char) {
			blade.Sharpen(settings.SharpenedBonus, settings.SharpenedStrikes)
		}
	}
	spent := spendWhetstones(user, stones, plan.UsesSpent)

	lines := []string{fmt.Sprintf("You work a whetstone along your company's blades. Sharpened: %s.", strings.Join(plan.Names(camping.Sharpened), ", "))}
	if names := plan.Names(camping.LeftOut); len(names) > 0 {
		lines = append(lines, fmt.Sprintf("The whetstone ran out before %s.", strings.Join(names, ", ")))
	}
	lines = append(lines, otherOutcomes(plan)...)
	_, left := whetstones(user.Character, settings.WhetstoneItemId)
	summary := fmt.Sprintf("That took %s; %s left.", plural(plan.UsesSpent, "use"), plural(left, "use"))
	if len(spent) > 0 {
		summary += " A whetstone is worn away."
	}
	lines = append(lines, summary)
	return strings.Join(lines, "\n")
}

func nothingToSharpen(plan camping.SharpenPlan) string {
	lines := []string{"No blade in your company needs sharpening."}
	return strings.Join(append(lines, otherOutcomes(plan)...), "\n")
}

func otherOutcomes(plan camping.SharpenPlan) []string {
	var lines []string
	if names := plan.Names(camping.AlreadySharp); len(names) > 0 {
		lines = append(lines, fmt.Sprintf("Already sharp: %s.", strings.Join(names, ", ")))
	}
	if names := plan.Names(camping.NotHere); len(names) > 0 {
		lines = append(lines, fmt.Sprintf("Not here: %s.", strings.Join(names, ", ")))
	}
	return lines
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// sharpenPreview shows what "sharpen" would do now, changing nothing.
func (m *CampingModule) sharpenPreview(user *users.UserRecord) string {
	settings := m.sharpenSettings()
	targets, fighting := m.sharpenTargets(user)
	stones, uses := whetstones(user.Character, settings.WhetstoneItemId)
	members := make([]camping.SharpenMember, len(targets))
	byID := map[int]sharpenTarget{}
	for i, t := range targets {
		members[i] = t.member
		byID[t.member.ID] = t
	}
	plan := camping.PlanSharpen(members, uses)

	lines := []string{fmt.Sprintf("Whetstones: %d (%s left). %s", len(stones), plural(uses, "use"), autoText(m.autoSharpenOn(user.UserId)))}
	for _, e := range plan.Entries {
		lines = append(lines, fmt.Sprintf("  %s: %s", e.Member.Name, previewLine(e, byID[e.Member.ID].char, fighting)))
	}
	switch {
	case fighting:
		lines = append(lines, "Not while your company is fighting.")
	case plan.UsesSpent > 0:
		lines = append(lines, fmt.Sprintf("Sharpening now would use %s.", plural(plan.UsesSpent, "whetstone use")))
	case plan.Count(camping.LeftOut) > 0:
		lines = append(lines, "You have no whetstone.")
	default:
		lines = append(lines, "Sharpening now would use nothing.")
	}
	return strings.Join(lines, "\n")
}

func previewLine(e camping.SharpenEntry, c *characters.Character, fighting bool) string {
	if e.Outcome == camping.NotHere || c == nil {
		return "not here"
	}
	names := []string{}
	for _, blade := range blades(c) {
		name := blade.DisplayName()
		if label := blade.EdgeLabel(); label != "" {
			name += " " + label
		}
		names = append(names, name)
	}
	gear := strings.Join(names, ", ")
	switch e.Outcome {
	case camping.Sharpened:
		if fighting {
			return gear + " (dull)"
		}
		return gear + " (would be sharpened)"
	case camping.AlreadySharp:
		return gear + " (already sharp)"
	case camping.LeftOut:
		return gear + " (no whetstone use left for it)"
	}
	return "no blade"
}
