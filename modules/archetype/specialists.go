package archetype

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 33f2 expedition specialists: the specialist resolver other modules
// ask through internal/archetypes, the specialists view, Read the Trail
// (on each step and on a bare "track"), and Keen Eye (on each step). Every
// entry point runs on the game loop; the module lock only guards config and
// the registry, never engine calls.

var _ archetypes.SpecialistProvider = (*ArchetypeModule)(nil)

const trackUsage = `Usage: track (read the trail around you). See <ansi fg="command">help trail</ansi>.`

// specialistOrder is the specialists view's order, with display names.
var specialistOrder = []struct{ utility, name, does string }{
	{archetypes.UtilityTrail, "Read the Trail", "warns of enemy groups beyond the exits"},
	{archetypes.UtilityPathfinder, "Pathfinder", "eases walking strain on rough ground"},
	{archetypes.UtilityKeenEye, "Keen Eye", "spots secret exits"},
	{utilityTraps, "Traps", "finds and disarms traps"},
	{archetypes.UtilityHaggle, "Haggle", "better market prices"},
	{archetypes.UtilityWeather, "Weather Sense", "forecasts the weather"},
	{utilityLight, "Light", "lights the way in the dark"},
	{archetypes.UtilityWatch, "Camp Watch", "spots raiders before they reach the camp"},
	{archetypes.UtilityFieldSmith, "Field Smith", "puts a longer edge on sharpened blades"},
	{archetypes.UtilityVigil, "Vigil", "steadies companions' loyalty at camp"},
	{archetypes.UtilityForage, "Forage", "finds food at camp"},
}

// registerSpecialists wires 33f2's commands. The step listener is the 17b
// one (onStep), which calls autoTrail and autoKeenEye.
func (m *ArchetypeModule) registerSpecialists() {
	m.plug.AddUserCommand("track", m.trackCommand, false, false)
	m.plug.AddUserCommand("specialists", m.specialistsCommand, true, false)
}

// BestSpecialist implements archetypes.SpecialistProvider.
func (m *ArchetypeModule) BestSpecialist(leaderUserID int, utility string, roomIDs ...int) (archetypes.Specialist, bool) {
	user := users.GetByUserId(leaderUserID)
	if user == nil || user.Character == nil || !m.autoskillOn(leaderUserID, utility) {
		return archetypes.Specialist{}, false
	}
	if _, busy := battle.Current(leaderUserID); busy {
		return archetypes.Specialist{}, false // none acts in the leader's own battle
	}
	if len(roomIDs) == 0 {
		roomIDs = []int{user.Character.RoomId}
	}
	best, ok := m.bestMember(user, utility, roomIDs...)
	if !ok {
		return archetypes.Specialist{}, false
	}
	return best.specialist(), true
}

// MemberUtilityLevel implements archetypes.MemberUtilityProvider (Phase 51).
func (m *ArchetypeModule) MemberUtilityLevel(leaderUserID, companionID int, utility string) int {
	user := users.GetByUserId(leaderUserID)
	if user == nil || user.Character == nil {
		return 0
	}
	members := companyMembers(user, user.Character.RoomId)
	m.levels(members, utility)
	for _, mb := range members {
		if (companionID == 0 && mb.IsLeader) || (companionID != 0 && mb.CompanionID == companionID) {
			return mb.Level
		}
	}
	return 0
}

var _ archetypes.MemberUtilityProvider = (*ArchetypeModule)(nil)

func (mb member) specialist() archetypes.Specialist {
	return archetypes.Specialist{Name: mb.Name, IsLeader: mb.user != nil, Level: mb.Level}
}

// SpecialistsView implements archetypes.SpecialistProvider: who in the
// leader's company, here with them, performs each capability.
func (m *ArchetypeModule) SpecialistsView(leaderUserID int) string {
	user := users.GetByUserId(leaderUserID)
	if user == nil || user.Character == nil {
		return ""
	}
	lines := []string{`<ansi fg="yellow-bold">Company specialists</ansi> (the best one here does the work):`}
	for _, s := range specialistOrder {
		state := ""
		if !m.autoskillOn(leaderUserID, s.utility) {
			state = ` <ansi fg="black-bold">(autoskill off)</ansi>`
		}
		who := fmt.Sprintf(`<ansi fg="black-bold">nobody here (%s)</ansi>`, m.performers(s.utility))
		if best, ok := m.bestMember(user, s.utility, user.Character.RoomId); ok {
			name := best.Name
			if best.user != nil {
				name = "you"
			}
			who = fmt.Sprintf(`%s, level %d`, name, best.Level)
		}
		lines = append(lines, fmt.Sprintf(`  %-15s %s%s`, s.name, who, state))
		lines = append(lines, fmt.Sprintf(`  %-15s <ansi fg="black-bold">%s</ansi>`, "", s.does))
	}
	lines = append(lines, `See <ansi fg="command">help specialists</ansi>; <ansi fg="command">autoskill</ansi> turns each on or off.`)
	return strings.Join(lines, "\n")
}

// performers names the archetypes that perform a utility: "a ranger".
func (m *ArchetypeModule) performers(utility string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var names []string
	for _, a := range m.table.List() {
		if a.HasUtility(utility) {
			names = append(names, "a "+strings.ToLower(a.Name))
		}
	}
	if len(names) == 0 {
		return "no archetype"
	}
	sort.Strings(names)
	return strings.Join(names, " or ") + "'s skill"
}

func (m *ArchetypeModule) specialistsCommand(_ string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	user.SendText(m.SpecialistsView(user.UserId))
	return true, nil
}

// --- Read the Trail -----------------------------------------------------------

// trailSign is one enemy group found beyond an exit.
type trailSign struct {
	path  []string // exits from the reader's room: one, or two when far
	name  string   // the group's name, "a band of ruffians"
	count int      // living hostile members
}

// hostileCount counts a group's living, hostile, uncharmed members. Tracks
// need no sight, so hidden members count.
func hostileCount(g enemyparty.Group) int {
	n := 0
	for _, id := range g.Party.Members {
		mob := mobs.GetInstance(id)
		if mob == nil || mob.Character.Health < 1 || !mob.Hostile || mob.Character.IsCharmed() {
			continue
		}
		n++
	}
	return n
}

// readableExits are the exits a reader can follow: secret ones only when
// the leader can see them.
func readableExits(user *users.UserRecord, room *rooms.Room) []string {
	var out []string
	for name, ex := range room.Exits {
		if ex.Secret {
			target := rooms.LoadedRoom(ex.RoomId)
			if target == nil || !user.Character.SeesSecretExit(room.RoomId, name, ex.RoomId, target.Zone) {
				continue
			}
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// signsBeyond lists the enemy groups in the rooms beyond room's exits, and
// beyond those when far, never counting room itself. Only rooms already in
// memory are read (a room nobody is near holds no live mobs anyway).
func signsBeyond(user *users.UserRecord, room *rooms.Room, far bool) []trailSign {
	var out []trailSign
	seen := map[int]bool{room.RoomId: true}
	type step struct {
		path []string
		room *rooms.Room
	}
	var next []step
	for _, exitName := range readableExits(user, room) {
		target := rooms.LoadedRoom(room.Exits[exitName].RoomId)
		if target == nil || seen[target.RoomId] {
			continue
		}
		seen[target.RoomId] = true
		next = append(next, step{[]string{exitName}, target})
	}
	collect := func(s step) {
		for _, g := range enemyparty.Groups(s.room) {
			if n := hostileCount(g); n > 0 {
				out = append(out, trailSign{path: s.path, name: g.Name, count: n})
			}
		}
	}
	for _, s := range next {
		collect(s)
	}
	if far {
		for _, s := range next {
			for _, exitName := range readableExits(user, s.room) {
				target := rooms.LoadedRoom(s.room.Exits[exitName].RoomId)
				if target == nil || seen[target.RoomId] {
					continue
				}
				seen[target.RoomId] = true
				collect(step{[]string{s.path[0], exitName}, target})
			}
		}
	}
	return out
}

// describeSigns renders what a tracker of level reads: level 1 only the
// direction, 2 the group, 3 its size, 4 also two rooms away (signsBeyond).
// Groups of one name seen the same way are counted together.
func describeSigns(signs []trailSign, level int) string {
	type key struct{ where, name string }
	var order []key
	counts := map[key]int{}
	for _, s := range signs {
		where := s.path[0]
		if len(s.path) > 1 {
			where = fmt.Sprintf("beyond %s, then %s", s.path[0], s.path[1])
		}
		k := key{where, s.name}
		if level <= 1 {
			k.name = ""
		}
		if _, seen := counts[k]; !seen {
			order = append(order, k)
		}
		counts[k] += s.count
	}
	parts := make([]string, 0, len(order))
	for _, k := range order {
		switch {
		case level <= 1:
			parts = append(parts, "fresh tracks, "+k.where)
		case level == 2:
			parts = append(parts, fmt.Sprintf("%s, %s", k.name, k.where))
		default:
			parts = append(parts, fmt.Sprintf("%s (%d), %s", k.name, counts[k], k.where))
		}
	}
	return strings.Join(parts, "; ")
}

// readTrail is the trail around the leader's room as the company's best
// tracker reads it; "" with no tracker or nothing to read.
func (m *ArchetypeModule) readTrail(user *users.UserRecord, room *rooms.Room, roomIDs ...int) (string, bool) {
	best, ok := m.bestMember(user, archetypes.UtilityTrail, roomIDs...)
	if !ok {
		return "", false
	}
	m.mu.Lock()
	farLevel := m.config.TrailFarLevel
	m.mu.Unlock()
	signs := signsBeyond(user, room, best.Level >= farLevel)
	if len(signs) == 0 {
		return "", true
	}
	sp := best.specialist()
	return fmt.Sprintf(`<ansi fg="yellow">%s %s the trail: %s.</ansi>`, sp.Subject(), sp.Verb("read", "reads"), describeSigns(signs, best.Level)), true
}

// autoTrail reads the trail when the company walks into a room.
func (m *ArchetypeModule) autoTrail(user *users.UserRecord, room *rooms.Room, fromRoomID int) {
	if !m.autoskillOn(user.UserId, archetypes.UtilityTrail) {
		return
	}
	if _, busy := battle.Current(user.UserId); busy {
		return
	}
	if text, _ := m.readTrail(user, room, room.RoomId, fromRoomID); text != "" {
		user.SendText(text)
	}
}

// trackCommand is a bare "track": read the trail now.
func (m *ArchetypeModule) trackCommand(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	if room == nil {
		return true, nil
	}
	if strings.TrimSpace(rest) != "" {
		user.SendText(trackUsage)
		return true, nil
	}
	if _, busy := battle.Current(user.UserId); busy {
		user.SendText("You can't read the trail in the middle of a battle.")
		return true, nil
	}
	if _, ok := m.bestMember(user, archetypes.UtilityTrail, room.RoomId); !ok {
		user.SendText(fmt.Sprintf("Nobody here in your company can read a trail (%s).", m.performers(archetypes.UtilityTrail)))
		return true, nil
	}
	m.mu.Lock()
	cooldown := m.config.TrailCooldownRounds
	m.mu.Unlock()
	if !user.Character.TryCooldown("trail", fmt.Sprintf("%d rounds", cooldown)) {
		user.SendText(fmt.Sprintf("You need to wait %d more rounds to read the trail again.", user.Character.GetCooldown("trail")))
		return true, nil
	}
	text, _ := m.readTrail(user, room, room.RoomId)
	if text == "" {
		text = "There are no fresh tracks of enemies nearby."
	}
	user.SendText(text)
	return true, nil
}

// EvadeAmbush implements archetypes.AmbushEvader: a tracker in the leader's
// company leads it around an ambush on a travel route with
// AmbushEvadePerLevel% per level. It names the tracker when it succeeds.
func (m *ArchetypeModule) EvadeAmbush(leaderUserID int, roomIDs ...int) (archetypes.Specialist, bool) {
	sp, ok := m.BestSpecialist(leaderUserID, archetypes.UtilityTrail, roomIDs...)
	if !ok {
		return archetypes.Specialist{}, false
	}
	m.mu.Lock()
	perLevel := m.config.AmbushEvadePerLevel
	m.mu.Unlock()
	if m.roll() > archetypes.PctByLevel(sp.Level, perLevel, 100) {
		return sp, false
	}
	return sp, true
}

// --- Keen Eye -----------------------------------------------------------------

// autoKeenEye gives the company one roll per unseen secret exit when it
// walks into a room. Without a rogue, the leader's own eye rolls at level 0.
func (m *ArchetypeModule) autoKeenEye(user *users.UserRecord, room *rooms.Room, fromRoomID int) {
	if !m.autoskillOn(user.UserId, archetypes.UtilityKeenEye) {
		return
	}
	if _, busy := battle.Current(user.UserId); busy {
		return
	}
	var hidden []string
	for name, ex := range room.Exits {
		if !ex.Secret {
			continue
		}
		target := rooms.LoadRoom(ex.RoomId)
		if target == nil || user.Character.SeesSecretExit(room.RoomId, name, ex.RoomId, target.Zone) {
			continue
		}
		hidden = append(hidden, name)
	}
	if len(hidden) == 0 {
		return
	}
	sort.Strings(hidden)
	spotter, ok := m.bestMember(user, archetypes.UtilityKeenEye, room.RoomId, fromRoomID)
	if !ok {
		spotter = member{UtilityMember: archetypes.UtilityMember{IsLeader: true}, Name: user.Character.Name, Perception: user.Character.Stats.Perception.ValueAdj, user: user}
	}
	m.mu.Lock()
	perLevel, target := m.config.KeenEyePerLevel, m.config.KeenEyeTarget
	m.mu.Unlock()
	var found []string
	for _, name := range hidden {
		if archetypes.CheckScore(spotter.Level, perLevel, spotter.Perception)+m.roll() < target {
			continue
		}
		if user.Character.LearnSecretExit(room.RoomId, name) {
			found = append(found, fmt.Sprintf(`<ansi fg="exit">%s</ansi>`, name))
		}
	}
	if len(found) == 0 {
		return
	}
	sp := spotter.specialist()
	user.SendText(fmt.Sprintf(`<ansi fg="yellow-bold">%s %s a hidden way: %s.</ansi>`, sp.Subject(), sp.Verb("spot", "spots"), strings.Join(found, " and ")))
}
