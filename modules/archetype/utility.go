package archetype

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/walking"
)

// Phase 17b utility skills: autoskill toggles, rogue traps (sense, disarm,
// auto-sense on entry and before picklock) and wizard auto-light. Every
// entry point runs on the game loop (commands, the walking step listener,
// the picklock seam). The module lock only guards the registry and config;
// live members are resolved before taking it.

const (
	utilityLight = "light"
	utilityTraps = "traps"

	autoskillUsage = `Usage: autoskill | autoskill <utility> on|off`
	trapUsage      = `Usage: trap sense [exit|container] | trap disarm <exit|container>`
)

var (
	_ archetypes.TrapProvider = (*ArchetypeModule)(nil)
	_ archetypes.TrapSenser   = (*ArchetypeModule)(nil)
)

// utilityConfig holds Phase 17b's utility-skill balance. All values are
// module config (files/data-overlays/config.yaml).
type utilityConfig struct {
	// UtilitySkills maps a utility (e.g. "light") to the skill whose level a
	// player uses for it (e.g. "cast").
	UtilitySkills map[string]string

	SensePerLevel          int
	SenseDifficultyFactor  int
	SenseCooldownRounds    int
	AutoSensePenalty       int
	DisarmPerLevel         int
	DisarmDifficultyFactor int
	DisarmBackfireMargin   int
	DisarmRounds           int

	AutoLightSpell         string
	AutoLightBelow         int
	AutoLightCooldown      int
	CompanionLightBuffID   int
	CompanionLightManaCost int
}

func defaultUtilityConfig() utilityConfig {
	return utilityConfig{
		UtilitySkills:          map[string]string{utilityLight: "cast", utilityTraps: "skulduggery"},
		SensePerLevel:          20,
		SenseDifficultyFactor:  5,
		SenseCooldownRounds:    2,
		AutoSensePenalty:       20,
		DisarmPerLevel:         20,
		DisarmDifficultyFactor: 5,
		DisarmBackfireMargin:   25,
		DisarmRounds:           900,
		AutoLightSpell:         "floatinglight",
		AutoLightBelow:         1,
		AutoLightCooldown:      10,
		CompanionLightBuffID:   1000,
		CompanionLightManaCost: 10,
	}
}

// parseUtilityConfig overlays configured values on the defaults. A missing
// or non-positive number keeps its default (AutoSensePenalty may be 0).
func parseUtilityConfig(get func(string) any) utilityConfig {
	cfg := defaultUtilityConfig()
	if get == nil {
		return cfg
	}
	positive := func(key string, into *int) {
		if v := configInt(get(key)); v > 0 {
			*into = v
		}
	}
	positive("SensePerLevel", &cfg.SensePerLevel)
	positive("SenseDifficultyFactor", &cfg.SenseDifficultyFactor)
	positive("SenseCooldownRounds", &cfg.SenseCooldownRounds)
	positive("DisarmPerLevel", &cfg.DisarmPerLevel)
	positive("DisarmDifficultyFactor", &cfg.DisarmDifficultyFactor)
	positive("DisarmBackfireMargin", &cfg.DisarmBackfireMargin)
	positive("DisarmRounds", &cfg.DisarmRounds)
	positive("AutoLightCooldown", &cfg.AutoLightCooldown)
	positive("CompanionLightBuffID", &cfg.CompanionLightBuffID)
	positive("CompanionLightManaCost", &cfg.CompanionLightManaCost)
	if raw := get("AutoSensePenalty"); raw != nil {
		if v := configInt(raw); v >= 0 {
			cfg.AutoSensePenalty = v
		}
	}
	if raw := get("AutoLightBelow"); raw != nil {
		if v := configInt(raw); v >= 1 && v <= 3 {
			cfg.AutoLightBelow = v
		}
	}
	if s := configString(get("AutoLightSpell")); s != "" {
		cfg.AutoLightSpell = strings.ToLower(s)
	}
	return cfg
}

// parseUtilitySkills reads the Utilities list ([{Utility, Skill}]); an
// empty or missing list keeps the defaults.
func parseUtilitySkills(raw any, defaults map[string]string) map[string]string {
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		return defaults
	}
	out := map[string]string{}
	for _, entry := range list {
		fields := stringMap(entry)
		if fields == nil {
			continue
		}
		utility := strings.ToLower(configString(fields["utility"]))
		skill := strings.ToLower(configString(fields["skill"]))
		if utility == "" || skill == "" {
			continue
		}
		out[utility] = skill
	}
	if len(out) == 0 {
		return defaults
	}
	return out
}

// registerUtility wires Phase 17b's commands and step listener.
func (m *ArchetypeModule) registerUtility() {
	m.plug.AddUserCommand("autoskill", m.autoskillCommand, true, false)
	m.plug.AddUserCommand("trap", m.trapCommand, false, false)
	walking.AddStepListener(m.onStep)
}

func nativeRoll() int { return util.Rand(100) + 1 }

func nativeNow() uint64 { return util.GetRoundCount() }

// nativeCast starts a spell through the real user cast command, so mana,
// wait rounds, and spell scripts behave exactly as a typed cast.
func nativeCast(user *users.UserRecord, room *rooms.Room, spellID string) {
	if _, err := usercommands.Cast(spellID, user, room, 0); err != nil {
		mudlog.Warn("archetype: auto-cast", "user", user.UserId, "spell", spellID, "error", err)
	}
}

// --- company members ---------------------------------------------------------

// member is one present company member considered for a utility.
type member struct {
	archetypes.UtilityMember
	Name       string
	Perception int
	user       *users.UserRecord
	mob        *mobs.Mob
	archetype  string
}

// companyMembers resolves the leader plus each spawned companion standing
// in any of roomIDs (a walking step counts companions still in the origin
// room, since they follow). It reads engine and company state only and
// must not be called with the module lock held.
func companyMembers(user *users.UserRecord, roomIDs ...int) []member {
	inRooms := func(roomID int) bool {
		for _, id := range roomIDs {
			if id == roomID {
				return true
			}
		}
		return false
	}
	out := []member{{
		UtilityMember: archetypes.UtilityMember{IsLeader: true},
		Name:          user.Character.Name,
		Perception:    user.Character.Stats.Perception.ValueAdj,
		user:          user,
	}}
	for _, ref := range survival.CurrentRoster(user.UserId) {
		companionID, ok := company.CompanionIDFromMemberKey(ref.Key)
		if !ok {
			continue
		}
		instanceID, ok := company.InstanceFor(user.UserId, companionID)
		if !ok {
			continue
		}
		mob := mobs.GetInstance(instanceID)
		if mob == nil || !inRooms(mob.Character.RoomId) || mob.Character.IsDisabled() {
			continue // a downed companion can't sense, disarm, or conjure
		}
		archetype, _ := company.CompanionArchetype(user.UserId, companionID)
		out = append(out, member{
			UtilityMember: archetypes.UtilityMember{CompanionID: companionID},
			Name:          mob.Character.Name,
			Perception:    mob.Character.Stats.Perception.ValueAdj,
			mob:           mob,
			archetype:     archetype,
		})
	}
	return out
}

// levels fills in each member's utility level. Engine reads (skill and
// character levels) happen before the module lock is taken.
func (m *ArchetypeModule) levels(members []member, utility string) {
	m.mu.Lock()
	skill := m.config.UtilitySkills[utility]
	m.mu.Unlock()
	skillLevels := make([]int, len(members))
	charLevels := make([]int, len(members))
	for i, mb := range members {
		if mb.user != nil {
			skillLevels[i] = mb.user.Character.GetSkillLevel(skill)
		} else {
			charLevels[i] = mb.mob.Character.Level
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range members {
		mb := &members[i]
		if mb.user != nil {
			id, chosen := m.registry.Players[mb.user.UserId]
			a, known := m.table.Get(id)
			mb.Level = archetypes.PlayerUtilityLevel(a, chosen && known, utility, skillLevels[i])
			continue
		}
		a, known := m.table.Get(mb.archetype)
		mb.Level = archetypes.CompanionUtilityLevel(a, known && mb.archetype != "", utility, charLevels[i])
	}
}

// bestMember picks the company member best at a utility among those in
// roomIDs.
func (m *ArchetypeModule) bestMember(user *users.UserRecord, utility string, roomIDs ...int) (member, bool) {
	return m.bestMemberWhere(user, utility, nil, roomIDs...)
}

// bestMemberWhere is bestMember restricted to members eligible says can act
// right now (nil: everyone), so an unable member never masks an able one.
func (m *ArchetypeModule) bestMemberWhere(user *users.UserRecord, utility string, eligible func(member) bool, roomIDs ...int) (member, bool) {
	members := companyMembers(user, roomIDs...)
	m.levels(members, utility)
	plain := make([]archetypes.UtilityMember, len(members))
	for i, mb := range members {
		plain[i] = mb.UtilityMember
		if eligible != nil && !eligible(mb) {
			plain[i].Level = 0
		}
	}
	best, ok := archetypes.BestMember(plain)
	if !ok {
		return member{}, false
	}
	for _, mb := range members {
		if mb.UtilityMember == best {
			return mb, true
		}
	}
	return member{}, false
}

func (mb member) label() string {
	if mb.user != nil {
		return "You"
	}
	return mb.Name
}

// --- autoskill ---------------------------------------------------------------

// autoskillOn reports a user's toggle for a utility; the default is on.
func (m *ArchetypeModule) autoskillOn(userID int, utility string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	on, set := m.registry.Autoskill[userID][utility]
	return !set || on
}

func (m *ArchetypeModule) utilitiesLocked() []string {
	out := make([]string, 0, len(m.config.UtilitySkills))
	for u := range m.config.UtilitySkills {
		out = append(out, u)
	}
	sort.Strings(out)
	return out
}

func (m *ArchetypeModule) autoskillList(userID int) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	lines := []string{"Automatic skills (your companions follow your settings):"}
	for _, u := range m.utilitiesLocked() {
		on, set := m.registry.Autoskill[userID][u]
		state := "on"
		if set && !on {
			state = "off"
		}
		lines = append(lines, fmt.Sprintf("  %-8s %s", u, state))
	}
	return strings.Join(lines, "\n")
}

func (m *ArchetypeModule) setAutoskill(userID int, utility string, on bool) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	utility = strings.ToLower(strings.TrimSpace(utility))
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.config.UtilitySkills[utility]; !ok {
		return fmt.Sprintf(`There is no automatic skill called "%s". Choose from: %s.`, utility, strings.Join(m.utilitiesLocked(), ", "))
	}
	toggles := m.registry.Autoskill[userID]
	prev, hadPrev := toggles[utility]
	if toggles == nil {
		toggles = map[string]bool{}
		m.registry.Autoskill[userID] = toggles
	}
	toggles[utility] = on
	if err := m.saveLocked(); err != nil {
		if hadPrev {
			toggles[utility] = prev
		} else {
			delete(toggles, utility)
		}
		return err.Error()
	}
	state := "off"
	if on {
		state = "on"
	}
	return fmt.Sprintf("Automatic %s is now %s.", utility, state)
}

func (m *ArchetypeModule) autoskillCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	args := strings.Fields(strings.ToLower(rest))
	switch {
	case len(args) == 0:
		user.SendText(m.autoskillList(user.UserId))
	case len(args) == 2 && (args[1] == "on" || args[1] == "off"):
		user.SendText(m.setAutoskill(user.UserId, args[0], args[1] == "on"))
	default:
		user.SendText(autoskillUsage)
	}
	return true, nil
}

// --- traps -------------------------------------------------------------------

// trappedLock is a locked exit or container carrying trap buffs.
type trappedLock struct {
	ID         string
	Name       string
	Container  bool
	Difficulty int
	Buffs      []int
}

func (l trappedLock) describe() string {
	if l.Container {
		return fmt.Sprintf(`the <ansi fg="container">%s</ansi>`, l.Name)
	}
	return fmt.Sprintf(`the <ansi fg="exit">%s</ansi> exit`, l.Name)
}

// lockID matches picklock's key format.
func lockID(roomID int, name string) string { return fmt.Sprintf(`%d-%s`, roomID, name) }

// trappedLocks lists a room's locked, trapped exits and containers, sorted.
func trappedLocks(room *rooms.Room) []trappedLock {
	var out []trappedLock
	for name, ex := range room.Exits {
		if ex.Lock.IsLocked() && len(ex.Lock.TrapBuffIds) > 0 {
			out = append(out, trappedLock{ID: lockID(room.RoomId, name), Name: name, Difficulty: int(ex.Lock.Difficulty), Buffs: ex.Lock.TrapBuffIds})
		}
	}
	for name, c := range room.Containers {
		if c.Lock.IsLocked() && len(c.Lock.TrapBuffIds) > 0 {
			out = append(out, trappedLock{ID: lockID(room.RoomId, name), Name: name, Container: true, Difficulty: int(c.Lock.Difficulty), Buffs: c.Lock.TrapBuffIds})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// findLock resolves a player's target to a room exit or container name,
// reporting whether it exists and whether it is a locked, trapped lock.
func findLock(room *rooms.Room, target string) (name string, exists bool, lock trappedLock, trapped bool) {
	if c := room.FindContainerByName(target); c != "" {
		name, exists = c, true
	} else if e, _ := room.FindExitByName(target); e != "" {
		name, exists = e, true
	}
	if !exists {
		return "", false, trappedLock{}, false
	}
	for _, l := range trappedLocks(room) {
		if l.Name == name {
			return name, true, l, true
		}
	}
	return name, true, trappedLock{}, false
}

// pruneDisarmedLocked drops expired disarms (not persisted until the next
// save). Caller holds m.mu.
func (m *ArchetypeModule) pruneDisarmedLocked(now uint64) {
	for id, until := range m.registry.Disarmed {
		if archetypes.TrapArmedAt(until, now) {
			delete(m.registry.Disarmed, id)
		}
	}
}

// TrapArmed implements archetypes.TrapProvider.
func (m *ArchetypeModule) TrapArmed(lockID string) bool {
	now := m.now()
	m.mu.Lock()
	defer m.mu.Unlock()
	return archetypes.TrapArmedAt(m.registry.Disarmed[lockID], now)
}

// armedLocks filters trapped locks down to those still armed.
func (m *ArchetypeModule) armedLocks(locks []trappedLock) []trappedLock {
	now := m.now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneDisarmedLocked(now)
	out := locks[:0:0]
	for _, l := range locks {
		if archetypes.TrapArmedAt(m.registry.Disarmed[l.ID], now) {
			out = append(out, l)
		}
	}
	return out
}

func (m *ArchetypeModule) senseSucceeds(mb member, l trappedLock, penalty int) bool {
	m.mu.Lock()
	cfg := m.config
	m.mu.Unlock()
	score := archetypes.CheckScore(mb.Level, cfg.SensePerLevel, mb.Perception) - penalty
	return archetypes.CheckMargin(score, m.roll(), l.Difficulty, cfg.SenseDifficultyFactor) >= 0
}

// sense is "trap sense [target]".
func (m *ArchetypeModule) sense(user *users.UserRecord, room *rooms.Room, target string) string {
	best, ok := m.bestMember(user, utilityTraps, room.RoomId)
	if !ok {
		return "Nobody in your company knows how to find traps."
	}
	locks := trappedLocks(room)
	if target = strings.TrimSpace(target); target != "" {
		_, exists, lock, trapped := findLock(room, target)
		if !exists {
			return "There is no such exit or container."
		}
		locks = nil
		if trapped {
			locks = []trappedLock{lock}
		}
	}
	m.mu.Lock()
	cooldown := m.config.SenseCooldownRounds
	m.mu.Unlock()
	if !user.Character.TryCooldown("trapsense", fmt.Sprintf("%d rounds", cooldown)) {
		return fmt.Sprintf("You need to wait %d more rounds to search for traps again.", user.Character.GetCooldown("trapsense"))
	}
	var found []string
	for _, l := range m.armedLocks(locks) {
		if m.senseSucceeds(best, l, 0) {
			found = append(found, l.describe())
		}
	}
	if len(found) == 0 {
		return "You find no traps."
	}
	verb := "finds"
	if best.user != nil {
		verb = "find"
	}
	return fmt.Sprintf("%s %s a trap on %s.", best.label(), verb, strings.Join(found, " and "))
}

// disarm is "trap disarm <target>".
func (m *ArchetypeModule) disarm(user *users.UserRecord, room *rooms.Room, target string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	if strings.TrimSpace(target) == "" {
		return trapUsage
	}
	_, exists, lock, trapped := findLock(room, target)
	if !exists {
		return "There is no such exit or container."
	}
	if !trapped {
		return "You find nothing to disarm there."
	}
	if !m.TrapArmed(lock.ID) {
		return "That trap is already disarmed."
	}
	best, ok := m.bestMember(user, utilityTraps, room.RoomId)
	if !ok {
		return "Nobody in your company knows how to disarm traps."
	}
	m.mu.Lock()
	cfg := m.config
	m.mu.Unlock()
	if !user.Character.TryCooldown("trapdisarm", fmt.Sprintf("%d rounds", cfg.SenseCooldownRounds)) {
		return fmt.Sprintf("You need to wait %d more rounds before trying again.", user.Character.GetCooldown("trapdisarm"))
	}
	score := archetypes.CheckScore(best.Level, cfg.DisarmPerLevel, best.Perception)
	margin := archetypes.CheckMargin(score, m.roll(), lock.Difficulty, cfg.DisarmDifficultyFactor)
	switch {
	case margin >= 0:
		now := m.now()
		m.mu.Lock()
		m.pruneDisarmedLocked(now)
		m.registry.Disarmed[lock.ID] = archetypes.DisarmedUntil(now, cfg.DisarmRounds)
		if err := m.saveLocked(); err != nil {
			delete(m.registry.Disarmed, lock.ID)
			m.mu.Unlock()
			return err.Error()
		}
		m.mu.Unlock()
		verb := "disarms"
		if best.user != nil {
			verb = "disarm"
		}
		return fmt.Sprintf("%s carefully %s the trap on %s.", best.label(), verb, lock.describe())
	case margin < -cfg.DisarmBackfireMargin:
		for _, buffID := range lock.Buffs {
			if best.user != nil {
				best.user.AddBuff(buffID, "trap")
			} else {
				best.mob.AddBuff(buffID, "trap")
			}
		}
		room.SendText(fmt.Sprintf(`<ansi fg="alert-3">%s triggered a trap!</ansi>`, best.Name), user.UserId)
		return fmt.Sprintf(`<ansi fg="alert-5">%s the trap on %s!</ansi>`, springSubject(best), lock.describe())
	default:
		verb := "fails"
		if best.user != nil {
			verb = "fail"
		}
		return fmt.Sprintf("%s %s to disarm the trap, but nothing happens.", best.label(), verb)
	}
}

func springSubject(mb member) string {
	if mb.user != nil {
		return "You spring"
	}
	return mb.Name + " fumbles and springs"
}

func (m *ArchetypeModule) trapCommand(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	if room == nil {
		return true, nil
	}
	args := strings.Fields(strings.ToLower(rest))
	if len(args) == 0 {
		user.SendText(trapUsage)
		return true, nil
	}
	target := strings.Join(args[1:], " ")
	switch args[0] {
	case "sense", "search", "find":
		user.SendText(m.sense(user, room, target))
	case "disarm":
		user.SendText(m.disarm(user, room, target))
	default:
		user.SendText(trapUsage)
	}
	return true, nil
}

// SenseBeforePick implements archetypes.TrapSenser: one free sense (no
// cooldown) the first time a user starts picking a given armed, trapped
// lock this session.
func (m *ArchetypeModule) SenseBeforePick(userID, roomID int, lock string) {
	user := users.GetByUserId(userID)
	room := rooms.LoadRoom(roomID)
	if user == nil || room == nil || !m.autoskillOn(userID, utilityTraps) {
		return
	}
	key := fmt.Sprintf("%d|%s", userID, lock)
	m.mu.Lock()
	_, done := m.pickSensed[key]
	m.mu.Unlock()
	if done {
		return
	}
	var target *trappedLock
	for _, l := range m.armedLocks(trappedLocks(room)) {
		if l.ID == lock {
			l := l
			target = &l
		}
	}
	if target == nil {
		return
	}
	best, ok := m.bestMember(user, utilityTraps, roomID)
	if !ok {
		return // nobody could have sensed it; don't spend the free look
	}
	m.mu.Lock()
	if m.pickSensed == nil {
		m.pickSensed = map[string]struct{}{}
	}
	m.pickSensed[key] = struct{}{}
	m.mu.Unlock()
	if !m.senseSucceeds(best, *target, 0) {
		return
	}
	user.SendText(fmt.Sprintf(`<ansi fg="yellow-bold">Your instincts prickle: %s is trapped.</ansi>`, target.describe()))
}

// --- step listener: auto-sense and auto-light ---------------------------------

func (m *ArchetypeModule) onStep(userID, fromRoomID, toRoomID int) {
	user := users.GetByUserId(userID)
	if user == nil {
		return
	}
	room := rooms.LoadRoom(toRoomID)
	if room == nil {
		return
	}
	m.autoSense(user, room, fromRoomID)
	m.autoLight(user, room, fromRoomID)
}

// autoSense gives the company one passive roll per trapped lock when it
// walks into a room.
func (m *ArchetypeModule) autoSense(user *users.UserRecord, room *rooms.Room, fromRoomID int) {
	if !m.autoskillOn(user.UserId, utilityTraps) {
		return
	}
	locks := m.armedLocks(trappedLocks(room))
	if len(locks) == 0 {
		return
	}
	best, ok := m.bestMember(user, utilityTraps, room.RoomId, fromRoomID)
	if !ok {
		return
	}
	m.mu.Lock()
	penalty := m.config.AutoSensePenalty
	m.mu.Unlock()
	var found []string
	for _, l := range locks {
		if m.senseSucceeds(best, l, penalty) {
			found = append(found, l.describe())
		}
	}
	if len(found) == 0 {
		return
	}
	who := "Your instincts prickle"
	if best.user == nil {
		who = best.Name + " stops you"
	}
	user.SendText(fmt.Sprintf(`<ansi fg="yellow-bold">%s: there is a trap on %s.</ansi>`, who, strings.Join(found, " and ")))
}

// autoLight conjures a floating light when a step leaves the walker in
// darkness with no party light up.
func (m *ArchetypeModule) autoLight(user *users.UserRecord, room *rooms.Room, fromRoomID int) {
	if user.Character.Aggro != nil || user.Character.IsDisabled() {
		return
	}
	if !m.autoskillOn(user.UserId, utilityLight) {
		return
	}
	m.mu.Lock()
	cfg := m.config
	m.mu.Unlock()
	if room.VisibilityForUser(user) >= cfg.AutoLightBelow || room.HasPartyLightFor(user.UserId) {
		return
	}
	members := companyMembers(user, room.RoomId, fromRoomID)
	for _, mb := range members {
		if mb.mob != nil && mb.mob.Character.HasBuffFlag(rooms.FlagPartyLight) {
			return // a following companion already carries a party light
		}
	}
	// Only members able to conjure right now are considered, so a companion
	// that is fighting or out of mana never masks a wizard who could cast.
	spell := spells.GetSpell(cfg.AutoLightSpell)
	able := func(mb member) bool {
		if mb.user != nil {
			return spell != nil && mb.user.Character.HasSpell(cfg.AutoLightSpell) && mb.user.Character.Mana >= spell.Cost
		}
		return mb.mob.Character.Aggro == nil && mb.mob.Character.Mana >= cfg.CompanionLightManaCost
	}
	best, ok := m.bestMemberWhere(user, utilityLight, able, room.RoomId, fromRoomID)
	if !ok {
		return
	}
	// The cooldown starts only once an able member makes the attempt.
	if !user.Character.TryCooldown("autolight", fmt.Sprintf("%d rounds", cfg.AutoLightCooldown)) {
		return
	}
	if best.user != nil {
		user.SendText("Darkness closes in. You begin conjuring a floating light.")
		m.cast(user, room, cfg.AutoLightSpell)
		return
	}
	best.mob.Character.Mana -= cfg.CompanionLightManaCost
	best.mob.AddBuff(cfg.CompanionLightBuffID, "spell")
	user.SendText(fmt.Sprintf("%s conjures a floating light against the dark.", best.Name))
}
