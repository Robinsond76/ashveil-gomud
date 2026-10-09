package storyevents

import (
	"encoding/json"
	"fmt"
	"strings"

	"slices"

	"github.com/GoMudEngine/GoMud/internal/actionpolicy"
	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/banter"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encounters"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/storyevents"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/wounds"
)

// world is everything the module touches outside its own state, so tests
// can drive the flow without the whole game. Each effect returns the line
// it earns for the result, or "" when it has nothing to say.
type world interface {
	Lookups() storyevents.Lookups
	User(userID int) *users.UserRecord
	// Members lists the leader first, then each companion present.
	Members(userID int) []storyevents.Facts
	// Company is what requirements read of the whole company.
	Company(userID int, flags map[string]bool) storyevents.Company
	// Zone is a room's zone and tags; ok is false for a missing room.
	Zone(roomID int) (zone string, tags []string, ok bool)
	// Busy reports why a trigger must not open a scene now (a fight, a
	// journey, a rest); the camp trigger passes camp = true, since a rest
	// has just ended.
	Busy(userID, roomID int, camp bool) bool
	// FoeLevel is the level a foe of level 0 takes in a room.
	FoeLevel(userID, roomID int) int

	Wound(userID int, f storyevents.Facts, pct int) string
	Ailment(userID int, f storyevents.Facts, kind string) string
	Need(userID int, f storyevents.Facts, stat string, amount int) string
	AddItems(userID int, op string, item, count int) string
	TakeItems(userID, item, count int) string
	Gold(userID, amount int) string
	Loyalty(userID int, op string, members []storyevents.Facts, delta int) string
	// Opinion reports a choice's stance to the companions with the company
	// (Phase 64) and returns what they say, one line each.
	Opinion(userID int, op, stance, subject string, members []storyevents.Facts) []string
	Battle(userID, roomID int, foes []storyevents.Foe) string
	Move(userID, roomID int) string

	// Send writes text to the leader; Push sends the web client a GMCP
	// message.
	Send(userID int, text string)
	Push(userID int, module string, payload any)
}

type liveWorld struct{}

func (liveWorld) Lookups() storyevents.Lookups {
	return storyevents.Lookups{
		Item:    func(id int) bool { return items.GetItemSpec(id) != nil },
		Mob:     func(id int) bool { return mobs.GetMobSpec(mobs.MobId(id)) != nil },
		Room:    func(id int) bool { return rooms.LoadRoom(id) != nil },
		Skill:   skills.SkillExists,
		Ailment: func(kind string) bool { _, ok := survival.AilmentFor(kind); return ok },
		Class: func(id string) bool {
			if _, ok := classes.Get(id); ok {
				return true
			}
			return slices.Contains(classes.Lineages(), id) || archetypes.Exists(id)
		},
		Personality: func(name string) bool { return slices.Contains(banter.Personalities, name) },
	}
}

func (liveWorld) User(userID int) *users.UserRecord { return users.GetByUserId(userID) }

func lowerSkills(in map[string]int) map[string]int {
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[strings.ToLower(k)] = v
	}
	return out
}

func (liveWorld) Members(userID int) []storyevents.Facts {
	user := users.GetByUserId(userID)
	if user == nil || user.Character == nil {
		return nil
	}
	c := user.Character
	leaderKey := string(survival.LeaderMemberKey)
	out := []storyevents.Facts{{
		Key: leaderKey, Name: c.Name, Leader: true,
		Archetype: c.ArchetypeID(), Class: classes.PlayerClass(userID).Class,
		Alignment: int(c.Alignment), Level: c.Level, Skills: lowerSkills(c.Skills),
		Tags: storyevents.TagsFor(userID, leaderKey),
	}}
	if views, ok := company.CompanyMembers(userID); ok {
		for _, v := range views {
			if v.Status != company.MemberPresent {
				continue
			}
			key := string(survival.CompanionMemberKey(v.ID))
			out = append(out, storyevents.Facts{
				Key: key, Name: v.Name, Archetype: v.Archetype, Class: v.Class,
				Personality: v.Personality, Alignment: v.Alignment, Level: v.Level,
				Skills: lowerSkills(v.Skills), Tags: storyevents.TagsFor(userID, key),
			})
		}
	}
	return out
}

func (liveWorld) Company(userID int, flags map[string]bool) storyevents.Company {
	co := storyevents.Company{Items: map[int]int{}, Flags: flags}
	if user := users.GetByUserId(userID); user != nil && user.Character != nil {
		co.Gold = user.Character.Gold
	}
	for _, stack := range encumbrance.CargoContents(userID) {
		co.Items[stack.ItemId] += stack.Count
	}
	return co
}

func (liveWorld) Zone(roomID int) (string, []string, bool) {
	room := rooms.LoadRoom(roomID)
	if room == nil {
		return "", nil, false
	}
	return room.Zone, room.GetTags(), true
}

func (liveWorld) Busy(userID, roomID int, camp bool) bool {
	user := users.GetByUserId(userID)
	if user == nil || user.Character == nil || user.Character.RoomId != roomID {
		return true
	}
	if user.Character.Health < 1 || user.Character.CombatWithdrawn || actionpolicy.InBattle(user) {
		return true
	}
	if blocked, _ := expedition.MovementBlocked(userID); blocked {
		return true
	}
	if !camp {
		if blocked, _ := camping.MovementBlocked(userID); blocked {
			return true
		}
	}
	if room := rooms.LoadRoom(roomID); room != nil {
		for _, id := range room.GetMobs() {
			mob := mobs.GetInstance(id)
			if mob == nil || mob.Character.Health < 1 {
				continue
			}
			// A group sprung on this company (a random encounter, a scene's
			// own battle) holds the scene back even before it has struck.
			if mob.EncounterOwner == userID {
				return true
			}
			if a := mob.Character.Aggro; a != nil && a.UserId == userID {
				return true
			}
		}
	}
	return false
}

func (liveWorld) FoeLevel(userID, roomID int) int {
	if room := rooms.LoadRoom(roomID); room != nil {
		if cfg := rooms.GetZoneConfig(room.Zone); cfg != nil && cfg.Encounters.Band.Valid() {
			return cfg.Encounters.Band.Low
		}
	}
	if user := users.GetByUserId(userID); user != nil && user.Character != nil {
		return max(user.Character.Level, 1)
	}
	return 1
}

func keyOf(f storyevents.Facts) survival.MemberKey { return survival.MemberKey(f.Key) }

func (liveWorld) Wound(userID int, f storyevents.Facts, pct int) string {
	var char interface {
		AddWound(wounds.Wound)
	}
	maxHealth := 0
	if f.Leader {
		user := users.GetByUserId(userID)
		if user == nil || user.Character == nil {
			return ""
		}
		char, maxHealth = user.Character, user.Character.HealthMax.Value
	} else {
		id, ok := survival.CompanionIDFromMemberKey(keyOf(f))
		if !ok {
			return ""
		}
		instanceID, ok := company.InstanceFor(userID, id)
		if !ok {
			return ""
		}
		mob := mobs.GetInstance(instanceID)
		if mob == nil {
			return ""
		}
		char, maxHealth = &mob.Character, mob.Character.HealthMax.Value
	}
	w := wounds.Beaten(maxHealth, pct, util.Rand)
	char.AddWound(w)
	return fmt.Sprintf("%s is left with %s.", f.Name, wounds.Describe(w))
}

func (liveWorld) Ailment(userID int, f storyevents.Facts, kind string) string {
	caught, err := survival.CatchAilment(userID, keyOf(f), kind)
	if err != nil {
		mudlog.Warn("storyevents: ailment", "user", userID, "error", err)
		return ""
	}
	if !caught {
		return ""
	}
	spec, ok := survival.AilmentFor(kind)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s comes down with a %s.", f.Name, strings.ToLower(spec.Name))
}

func (liveWorld) Need(userID int, f storyevents.Facts, stat string, amount int) string {
	cost := survival.Exertion{}
	word := ""
	switch stat {
	case "hunger":
		cost.Hunger, word = amount, "hungrier"
	case "thirst":
		cost.Thirst, word = amount, "thirstier"
	case "fatigue":
		cost.Fatigue, word = amount, "more tired"
	}
	if _, err := survival.ApplyMemberDrain(userID, keyOf(f), cost); err != nil {
		mudlog.Warn("storyevents: need", "user", userID, "error", err)
		return ""
	}
	return fmt.Sprintf("%s is %s for it.", f.Name, word)
}

func itemName(id int) string {
	if spec := items.GetItemSpec(id); spec != nil {
		if spec.DisplayName != "" {
			return spec.DisplayName
		}
		return spec.Name
	}
	return fmt.Sprintf("item %d", id)
}

func (liveWorld) AddItems(userID int, op string, item, count int) string {
	err := encumbrance.DepositCargo(userID, op, []encumbrance.CargoStack{{ItemId: item, Count: count}})
	if err != nil {
		mudlog.Warn("storyevents: add items", "user", userID, "item", item, "error", err)
		return fmt.Sprintf("You cannot carry the %s, and it is left behind.", itemName(item))
	}
	return fmt.Sprintf("You take %d %s.", count, itemName(item))
}

func (liveWorld) TakeItems(userID, item, count int) string {
	have := 0
	for _, stack := range encumbrance.CargoContents(userID) {
		if stack.ItemId == item {
			have += stack.Count
		}
	}
	count = min(count, have)
	if count < 1 {
		return ""
	}
	if err := encumbrance.WithdrawCargo(userID, item, count); err != nil {
		mudlog.Warn("storyevents: take items", "user", userID, "item", item, "error", err)
		return ""
	}
	return fmt.Sprintf("You give up %d %s.", count, itemName(item))
}

func (liveWorld) Gold(userID, amount int) string {
	user := users.GetByUserId(userID)
	if user == nil || user.Character == nil {
		return ""
	}
	if amount < 0 {
		amount = -min(-amount, user.Character.Gold)
	}
	if amount == 0 {
		return ""
	}
	user.Character.Gold += amount
	if amount > 0 {
		return fmt.Sprintf("You gain %d gold.", amount)
	}
	return fmt.Sprintf("You lose %d gold.", -amount)
}

func (liveWorld) Loyalty(userID int, op string, members []storyevents.Facts, delta int) string {
	var ids []int
	for _, f := range members {
		if id, ok := survival.CompanionIDFromMemberKey(keyOf(f)); ok && !f.Leader {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return ""
	}
	changed, err := company.AdjustLoyaltyOnce(userID, op, ids, delta)
	if err != nil {
		mudlog.Warn("storyevents: loyalty", "user", userID, "error", err)
		return ""
	}
	// Only those whose loyalty moved (one at the bound, or without a
	// disposition, is left out of the line).
	if len(changed) == 0 {
		return ""
	}
	who := strings.Join(changed, ", ")
	if delta > 0 {
		return fmt.Sprintf("%s thinks better of you.", who)
	}
	return fmt.Sprintf("%s thinks less of you.", who)
}

func (liveWorld) Opinion(userID int, op, stance, subject string, members []storyevents.Facts) []string {
	ids := []int{}
	for _, f := range members {
		if id, ok := survival.CompanionIDFromMemberKey(keyOf(f)); ok && !f.Leader {
			ids = append(ids, id)
		}
	}
	said, err := company.Opinion(userID, opinions.Choice{Kind: opinions.Kind(stance), Op: op, Witnesses: ids, Subject: subject})
	if err != nil {
		mudlog.Warn("storyevents: opinion", "user", userID, "error", err)
		return nil
	}
	return said
}

func (liveWorld) Battle(userID, roomID int, foes []storyevents.Foe) string {
	level := liveWorld{}.FoeLevel(userID, roomID)
	planned := make([]encounters.Foe, 0, len(foes))
	for _, f := range foes {
		l := f.Level
		if l <= 0 {
			l = level
		}
		planned = append(planned, encounters.Foe{MobID: f.Mob, Level: l})
	}
	if err := encounters.StartGroup(userID, roomID, planned); err != nil {
		mudlog.Warn("storyevents: battle", "user", userID, "room", roomID, "error", err)
		return "Whatever stirred settles again, and nothing comes."
	}
	return ""
}

func (liveWorld) Move(userID, roomID int) string {
	user := users.GetByUserId(userID)
	if user == nil || user.Character == nil {
		return ""
	}
	origin := user.Character.RoomId
	if err := rooms.MoveToRoom(userID, roomID); err != nil {
		mudlog.Warn("storyevents: move", "user", userID, "room", roomID, "error", err)
		return ""
	}
	company.RelocateCompany(userID, origin, roomID)
	return ""
}

func (liveWorld) Send(userID int, text string) {
	if user := users.GetByUserId(userID); user != nil {
		user.SendText(text)
	}
}

func (liveWorld) Push(userID int, module string, payload any) {
	f, ok := usercommands.GetExportedFunction("SendGMCPEvent")
	if !ok {
		return
	}
	if send, ok := f.(func(int, string, any)); ok {
		if _, err := json.Marshal(payload); err != nil {
			mudlog.Error("storyevents: payload", "error", err)
			return
		}
		send(userID, module, payload)
	}
}
