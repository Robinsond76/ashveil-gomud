package testarea

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actionpolicy"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/creatures"
	"github.com/GoMudEngine/GoMud/internal/encounters"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/weather"
)

// The tools work only on a trip: whatever they do is undone by the return.

type toolFunc func(m *Module, user *users.UserRecord, args []string) string

var tools = map[string]toolFunc{
	"class":     toolClass,
	"level":     toolLevel,
	"companion": toolCompanion,
	"fight":     toolFight,
	"clear":     toolClear,
	"heal":      toolHeal,
	"gold":      toolGold,
	"give":      toolGive,
	"kit":       toolKit,
	"items":     toolItems,
	"catalog":   toolCatalog,
	"weather":   toolWeather,
}

func (m *Module) tool(name string) (toolFunc, bool) {
	f, ok := tools[name]
	return f, ok
}

func (m *Module) runTool(user *users.UserRecord, name string, args []string) string {
	if _, in := m.session(user.UserId); !in {
		return "Start a trip first (" + cmd("testarea") + "); these tools change your character and are undone by " + cmd("testarea return") + "."
	}
	f, _ := m.tool(name)
	return f(m, user, args)
}

// --- class and level -------------------------------------------------------

func classListing() string {
	var b strings.Builder
	b.WriteString("Classes: a base archetype (e.g. warrior) or an advanced or elite class (e.g. knight, paladin).\n")
	for _, lineage := range classes.Lineages() {
		var ids []string
		for _, c := range classes.Advanced(lineage) {
			ids = append(ids, c.ID)
			if e, ok := classes.Elite(c.ID); ok {
				ids = append(ids, e.ID)
			}
		}
		fmt.Fprintf(&b, "  %s: %s\n", lineage, strings.Join(ids, ", "))
	}
	return strings.TrimRight(b.String(), "\n")
}

func toolClass(m *Module, user *users.UserRecord, args []string) string {
	if len(args) == 0 {
		return "Usage: " + cmd("testarea class <class>") + "\n" + classListing()
	}
	name, err := classes.AdminSetClass(user.UserId, args[0])
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("You are now a %s. (Spells were re-taught for the class; your level is %d.)", name, user.Character.Level)
}

func parseLevel(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > users.MaxAdminLevel {
		return 0, fmt.Errorf("the level must be a number from 1 to %d", users.MaxAdminLevel)
	}
	return n, nil
}

func toolLevel(m *Module, user *users.UserRecord, args []string) string {
	if len(args) != 1 {
		return "Usage: " + cmd("testarea level <1-100>")
	}
	n, err := parseLevel(args[0])
	if err != nil {
		return err.Error()
	}
	got, err := user.AdminSetLevel(n)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("You are now level %d, with full health and mana.", got)
}

// --- companions ------------------------------------------------------------

func toolCompanion(m *Module, user *users.UserRecord, args []string) string {
	usage := "Usage: " + cmd("testarea companion add <class> [level]") + ", " + cmd("testarea companion class <member> <class>") + ", " + cmd("testarea companion level <member> <level>")
	if len(args) == 0 {
		return usage
	}
	roomID := user.Character.RoomId
	var text string
	var err error
	switch strings.ToLower(args[0]) {
	case "add", "recruit":
		if len(args) < 2 || len(args) > 3 {
			return usage
		}
		level := 0
		if len(args) == 3 {
			if level, err = parseLevel(args[2]); err != nil {
				return err.Error()
			}
		}
		text, err = company.AdminRecruit(user.UserId, roomID, args[1], level)
	case "class":
		if len(args) != 3 {
			return usage
		}
		text, err = company.AdminSetMember(user.UserId, roomID, args[1], args[2], 0)
	case "level":
		if len(args) != 3 {
			return usage
		}
		var level int
		if level, err = parseLevel(args[2]); err != nil {
			return err.Error()
		}
		text, err = company.AdminSetMember(user.UserId, roomID, args[1], "", level)
	default:
		return usage
	}
	if err != nil {
		return err.Error()
	}
	return text
}

// --- fights ----------------------------------------------------------------

// foeTypes are the enemies `testarea fight` knows by name; a number is a mob
// id. Each is a real mob of the shipped world.
var foeTypes = map[string]int{
	"skeleton": 15, "acolyte": 18, "imp": 33, "spider": 36, "wolf": 56, "hexer": 70,
	"brigand": 86, "warden": 92, "chanter": 93, "shaman": 97,
	"ogre": 85, "ent": 34, "lich": 14,
}

const (
	maxFoes     = 6
	defaultFoe  = "skeleton"
	fightsUsage = "Usage: testarea fight <level> <count> <type> [<count> <type> ...]  or  testarea fight <level> <size 2-6> [type ...]"
)

func foeList() string {
	return "Types: " + strings.Join(sortedKeys(foeTypes), ", ") + " (or a mob id). A boss type (ogre, ent, lich) leads the group as its boss; healers (chanter, shaman) tend the rest."
}

// foeID resolves a foe type word (a name or a mob id) to a mob id.
func foeID(w string) (int, bool) {
	if id, ok := foeTypes[strings.ToLower(w)]; ok {
		return id, true
	}
	n, err := strconv.Atoi(w)
	if err != nil || mobs.GetMobSpec(mobs.MobId(n)) == nil {
		return 0, false
	}
	return n, true
}

// parseFoes reads what follows the level of `testarea fight` into the mob
// ids of the group, in order. Two forms:
//   - counted kinds: `2 imp 1 skeleton` is two imps and a skeleton (the
//     form when the words after the level pair a number with a type and at
//     least one type is a name);
//   - the older `<size> [type ...]`: the types repeat in turn until the
//     group has size foes (`3 skeleton imp` is skeleton, imp, skeleton).
//
// The error is the text to show the admin.
func parseFoes(words []string) ([]int, string) {
	if len(words) == 0 {
		return nil, fightsUsage + "\n" + foeList()
	}
	if counted(words) {
		var ids []int
		for i := 0; i < len(words); i += 2 {
			n, _ := strconv.Atoi(words[i])
			id, ok := foeID(words[i+1])
			if !ok {
				return nil, fmt.Sprintf("There is no foe type %q.\n%s", words[i+1], foeList())
			}
			if n < 1 || len(ids)+n > maxFoes {
				return nil, fmt.Sprintf("The group must be from 2 to %d foes.\n%s", maxFoes, fightsUsage)
			}
			for ; n > 0; n-- {
				ids = append(ids, id)
			}
		}
		if len(ids) < 2 {
			return nil, fmt.Sprintf("The group must be from 2 to %d foes.\n%s", maxFoes, fightsUsage)
		}
		return ids, ""
	}
	size, err := strconv.Atoi(words[0])
	if err != nil || size < 2 || size > maxFoes {
		return nil, fmt.Sprintf("The size must be from 2 to %d.\n%s", maxFoes, fightsUsage)
	}
	typeWords := words[1:]
	if len(typeWords) == 0 {
		typeWords = []string{defaultFoe}
	}
	var types []int
	for _, w := range typeWords {
		id, ok := foeID(w)
		if !ok {
			return nil, fmt.Sprintf("There is no foe type %q.\n%s", w, foeList())
		}
		types = append(types, id)
	}
	ids := make([]int, size)
	for i := range ids {
		ids[i] = types[i%len(types)]
	}
	return ids, ""
}

// counted reports whether the words are count-and-type pairs naming at
// least one type: an even number of words, a number first in each pair.
func counted(words []string) bool {
	if len(words)%2 != 0 {
		return false
	}
	named := false
	for i := 0; i < len(words); i += 2 {
		if _, err := strconv.Atoi(words[i]); err != nil {
			return false
		}
		if _, ok := foeTypes[strings.ToLower(words[i+1])]; ok {
			named = true
		}
	}
	return named
}

func isBossMob(id int) bool {
	spec := mobs.GetMobSpec(mobs.MobId(id))
	return spec != nil && spec.Boss
}

func toolFight(m *Module, user *users.UserRecord, args []string) string {
	if len(args) < 2 {
		return fightsUsage + "\n" + foeList()
	}
	level, err := parseLevel(args[0])
	if err != nil {
		return err.Error()
	}
	ids, bad := parseFoes(args[1:])
	if bad != "" {
		return bad
	}
	size := len(ids)
	// A boss leads: its escorts are the foes after it.
	sort.SliceStable(ids, func(i, j int) bool { return isBossMob(ids[i]) && !isBossMob(ids[j]) })
	if actionpolicy.InBattle(user) {
		return actionpolicy.BattleUnderWay
	}
	var foes []encounters.Foe
	boss := false
	for _, id := range ids {
		isBoss := false
		if spec := mobs.GetMobSpec(mobs.MobId(id)); spec != nil && spec.Boss && !boss {
			isBoss, boss = true, true
		}
		foes = append(foes, encounters.Foe{MobID: id, Level: level, Boss: isBoss, Escort: boss && !isBoss})
	}
	if _, err := enemyparty.SpawnEncounter(user.Character.RoomId, user.UserId, foes); err != nil {
		return "The foes would not come: " + err.Error()
	}
	return fmt.Sprintf("%d foes of level %d close in.", size, level)
}

func toolClear(m *Module, user *users.UserRecord, args []string) string {
	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		return "There is no room."
	}
	removed := 0
	for _, id := range room.GetMobs() {
		mob := mobs.GetInstance(id)
		if mob == nil || mob.EncounterOwner != user.UserId {
			continue
		}
		room.RemoveMob(id)
		mobs.DestroyInstance(id)
		removed++
	}
	if removed == 0 {
		return "No foes of yours are here."
	}
	return fmt.Sprintf("%d foes are gone.", removed)
}

func toolHeal(m *Module, user *users.UserRecord, args []string) string {
	c := user.Character
	c.Wounds = nil
	c.RecalculateStats()
	c.Health, c.Mana = c.HealthMax.Value, c.ManaMax.Value
	return "You are fully restored. (" + cmd("healcompany") + " restores the whole company and raises the fallen.)"
}

// --- gold and gear ---------------------------------------------------------

func toolGold(m *Module, user *users.UserRecord, args []string) string {
	n := 10000
	if len(args) > 0 {
		var err error
		if n, err = strconv.Atoi(args[0]); err != nil || n < 1 || n > 10_000_000 {
			return "Usage: " + cmd("testarea gold [amount]")
		}
	}
	user.Character.Gold += n
	return fmt.Sprintf("You have %d gold.", user.Character.Gold)
}

const maxGive = 20

func toolGive(m *Module, user *users.UserRecord, args []string) string {
	if len(args) == 0 {
		return "Usage: " + cmd("testarea give <item name or id> [count]") + "; " + cmd("testarea items <word>") + " searches."
	}
	count := 1
	query := args
	if n, err := strconv.Atoi(args[len(args)-1]); err == nil && len(args) > 1 {
		count, query = n, args[:len(args)-1]
	}
	if count < 1 || count > maxGive {
		return fmt.Sprintf("The count must be from 1 to %d.", maxGive)
	}
	id := items.FindItem(strings.Join(query, " "))
	if id == 0 {
		return fmt.Sprintf("No item matches %q. %s searches.", strings.Join(query, " "), cmd("testarea items <word>"))
	}
	for i := 0; i < count; i++ {
		user.Character.StoreItem(items.New(id))
	}
	spec := items.GetItemSpec(id)
	return fmt.Sprintf("You take %d x %s (#%d).", count, spec.Name, id)
}

func toolItems(m *Module, user *users.UserRecord, args []string) string {
	if len(args) == 0 {
		return "Usage: " + cmd("testarea items <word>")
	}
	word := strings.ToLower(strings.Join(args, " "))
	var lines []string
	for _, spec := range items.GetAllItemSpecs() {
		hay := strings.ToLower(spec.Name + " " + spec.Family + " " + string(spec.Type) + " " + string(spec.Subtype))
		if strings.Contains(hay, word) {
			lines = append(lines, fmt.Sprintf("  #%d %s (%s%s)", spec.ItemId, spec.Name, spec.Type, familySuffix(spec)))
		}
	}
	if len(lines) == 0 {
		return fmt.Sprintf("No item matches %q.", word)
	}
	more := ""
	if len(lines) > 30 {
		more = fmt.Sprintf("\n  ...and %d more; narrow the word.", len(lines)-30)
		lines = lines[:30]
	}
	return strings.Join(lines, "\n") + more
}

func familySuffix(s items.ItemSpec) string {
	if s.Family != "" {
		return ", " + s.Family
	}
	return ""
}

// kit is one item of a kind for each family or slot: the best tier of each.
func kitItems(kind string) ([]int, bool) {
	specs := items.GetAllItemSpecs()
	best := map[string]items.ItemSpec{}
	consider := func(key string, s items.ItemSpec) {
		if cur, ok := best[key]; !ok || s.Tier > cur.Tier || (s.Tier == cur.Tier && s.ItemId < cur.ItemId) {
			best[key] = s
		}
	}
	var direct []int
	for _, s := range specs {
		switch kind {
		case "weapons":
			if s.Type == items.Weapon {
				fam := s.Family
				if fam == "" {
					fam = string(s.Subtype)
				}
				consider(fam+"|"+string(s.Subtype), s)
			}
		case "armor", "armour":
			switch s.Type {
			case items.Offhand, items.Head, items.Neck, items.Body, items.Belt, items.Gloves, items.Legs, items.Feet:
				fam := s.Family
				if fam == "" {
					fam = s.Name
				}
				consider(string(s.Type)+"|"+fam, s)
			}
		case "supplies", "consumables":
			if s.ItemId == creatures.RepairItemID { // Phase 38e review: mortar repairs a stone golem
				direct = append(direct, s.ItemId)
			}
			switch s.Type {
			case items.Potion, items.Food, items.Drink, items.Scroll, items.Grenade:
				consider(string(s.Type)+"|"+s.Name, s)
			}
		case "saddles":
			if s.Saddle != "" || s.Type == items.Pack {
				consider(string(s.Type)+"|"+string(s.Saddle)+"|"+s.Name, s)
			}
		case "camp":
			if s.ItemId == 30 || (s.ItemId >= 45 && s.ItemId <= 50) {
				direct = append(direct, s.ItemId)
			}
		default:
			return nil, false
		}
	}
	ids := direct
	for _, s := range best {
		ids = append(ids, s.ItemId)
	}
	sort.Ints(ids)
	return ids, true
}

func toolKit(m *Module, user *users.UserRecord, args []string) string {
	usage := "Usage: " + cmd("testarea kit <weapons|armor|camp|supplies|saddles>") + " gives one of each kind."
	if len(args) != 1 {
		return usage
	}
	ids, ok := kitItems(strings.ToLower(args[0]))
	if !ok {
		return usage
	}
	if len(ids) == 0 {
		return "This world has none of those."
	}
	for _, id := range ids {
		user.Character.StoreItem(items.New(id))
	}
	return fmt.Sprintf("You take %d items. Everything is gone again when you return.", len(ids))
}

// --- weather ---------------------------------------------------------------

func toolWeather(m *Module, user *users.UserRecord, args []string) string {
	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil || !inTestZone(room.Zone) {
		return "Weather is set for the zone you stand in, and only in the test area's zones."
	}
	names := weather.AdminConditions(room.Zone)
	if len(names) == 0 {
		return room.Zone + " has no weather."
	}
	if len(args) == 0 {
		return fmt.Sprintf("%s weather: %s, or %s to let it change by itself.", room.Zone, strings.Join(names, ", "), cmd("auto"))
	}
	now, err := weather.AdminSetCondition(room.Zone, args[0])
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("%s weather is now %s.", room.Zone, now)
}

func inTestZone(zone string) bool {
	for _, z := range Zones {
		if z == zone {
			return true
		}
	}
	return false
}
