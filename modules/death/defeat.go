package death

// Phase 53: defeat scenarios. A new death that a scenario claims (ClaimDefeat,
// asked by the engine's suicide command before the death's penalties) wakes
// the company in a situation that fits what beat it, instead of at the church
// and a level down: rescued, captured, left for dead, or robbed. The claim is
// saved on the character (MiscData) with the death's pending mark, so a
// restart mid-defeat resumes the same scenario and never rolls again.

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	domain "github.com/GoMudEngine/GoMud/internal/death"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/wounds"
)

const (
	// foeGroupKey and foeInstanceKey remember the foe that landed the
	// killing blow, from the claim until the company wakes, so a retry
	// after a restart still clears them.
	foeGroupKey    = "defeat-group"
	foeInstanceKey = "defeat-foe"

	// boundBuffID is the buff a captured leader wakes under: it holds them
	// in place (no-go) for a few rounds while they work loose.
	boundBuffID = 9301

	defaultRescuedHunger  = 45
	defaultRescuedFatigue = 20
)

var _ domain.ScenarioProvider = (*DeathModule)(nil)

func scenarioOf(c *characters.Character) string {
	id, _ := c.GetMiscData(domain.ScenarioKey).(string)
	return id
}

// raceAndGroups names what a killer is: its race and the groups it
// identifies with, lower case.
func raceAndGroups(k domain.Killer) (string, []string) {
	var mob *mobs.Mob
	if k.InstanceID > 0 {
		mob = mobs.GetInstance(k.InstanceID)
	}
	if mob == nil && k.MobID > 0 {
		mob = mobs.GetMobSpec(mobs.MobId(k.MobID))
	}
	if mob == nil {
		return "", nil
	}
	race := ""
	if r := races.GetRace(mob.Character.RaceId); r != nil {
		race = strings.ToLower(r.Name)
	}
	return race, mob.Groups
}

// ClaimDefeat implements domain.ScenarioProvider. It rolls a scenario for a
// new death from the table, by the killer's kind and the zone they fell in,
// and marks the character with it. It claims nothing where the church must
// stand: no scenario fits, a capture room that doesn't load, or a room the
// admin test area keeps its dead in.
func (m *DeathModule) ClaimDefeat(userID int, killer domain.Killer) bool {
	user := m.lookupUser(userID)
	if user == nil || user.Character == nil {
		return false
	}
	c := user.Character
	c.SetMiscData(domain.ScenarioKey, nil)
	c.SetMiscData(foeGroupKey, nil)
	c.SetMiscData(foeInstanceKey, nil)
	cfg := m.config()
	if len(cfg.scenarios) == 0 {
		return false
	}
	if _, overridden := domain.WakeOverride(c.RoomId); overridden {
		return false
	}
	fell := m.loadRoom(c.RoomId)
	if fell == nil {
		return false
	}
	race, groups := raceAndGroups(killer)
	table := make([]domain.Scenario, 0, len(cfg.scenarios))
	for _, s := range cfg.scenarios {
		// A scenario whose place can't be reached is never rolled.
		if s.Kind == domain.Captured && !m.roomLoads(s.Room) {
			continue
		}
		// A protected death (new characters, perma-gear) never costs goods.
		if killer.Protected && s.Kind.TakesGoods() {
			continue
		}
		table = append(table, s)
	}
	sc, ok := domain.Pick(table, fell.Zone, race, groups, m.roll)
	if !ok {
		return false
	}
	c.SetMiscData(domain.ScenarioKey, sc.ID)
	if killer.InstanceID > 0 {
		if foe := mobs.GetInstance(killer.InstanceID); foe != nil {
			c.SetMiscData(foeInstanceKey, killer.InstanceID)
			if foe.SpawnGroup != "" {
				c.SetMiscData(foeGroupKey, foe.SpawnGroup)
			}
		}
	}
	mudlog.Info("death: defeat scenario claimed", "user", userID, "scenario", sc.ID, "kind", sc.Kind, "zone", fell.Zone, "race", race)
	return true
}

// scenarioFor is the scenario a character's death was claimed by, if the
// table still has it.
func (m *DeathModule) scenarioFor(c *characters.Character) (domain.Scenario, bool) {
	id := scenarioOf(c)
	if id == "" {
		return domain.Scenario{}, false
	}
	return domain.ByID(m.config().scenarios, id)
}

// scenarioDestination is where a scenario wakes the company. ok is false
// when that place can't be had, and the death waits like any other.
func (m *DeathModule) scenarioDestination(sc domain.Scenario, c *characters.Character, cfg settings) (int, bool) {
	switch sc.Kind {
	case domain.Captured:
		return sc.Room, m.roomLoads(sc.Room)
	case domain.LeftForDead, domain.Robbed:
		return c.RoomId, m.roomLoads(c.RoomId)
	}
	// Rescued: the nearest settlement is one in the zone they fell in;
	// failing that, the church that would have taken them.
	if fell := m.loadRoom(c.RoomId); fell != nil {
		for _, s := range cfg.registry.Settlements() {
			if s.Zone == fell.Zone && m.roomLoads(s.ServiceRoomID) {
				return s.ServiceRoomID, true
			}
		}
	}
	return domain.Destination(checkpoint(c), cfg.fallbackID, m.validChurch, m.roomLoads)
}

// defaultText is what the leader reads on waking when the scenario writes
// none.
func defaultText(sc domain.Scenario, where string) string {
	switch sc.Kind {
	case domain.Rescued:
		return "A passing traveller found your company and carried you to " + where + ". You wake hungry and worn out."
	case domain.Captured:
		return "You wake bound in " + where + ". Your pack is locked in a chest, and your captors stand guard over it."
	case domain.LeftForDead:
		return "You wake where you fell. The foes have gone, thinking you finished, and you are badly hurt."
	default:
		return "You wake where you fell, stripped of what the robbers could carry. They are gone."
	}
}

// applyScenario carries out a scenario's effects once the company is where
// it wakes. It runs in the same step that clears the death's pending mark,
// after the move, so nothing is taken from a company that never woke.
// Returns the lines for the leader.
func (m *DeathModule) applyScenario(user *users.UserRecord, sc domain.Scenario, room *rooms.Room) []string {
	c := user.Character
	var lines []string
	switch sc.Kind {
	case domain.Rescued:
		m.wearDown(user.UserId, sc)
	case domain.Captured:
		lines = append(lines, m.capture(user, sc, room)...)
	case domain.LeftForDead:
		pct := sc.WoundPct
		if pct <= 0 {
			pct = 20
		}
		c.AddWound(wounds.Beaten(c.HealthMax.Value, pct, m.roll))
		names := m.woundCompany(user.UserId, pct, m.roll)
		lines = append(lines, "Each of you carries a lasting wound (help wounds).")
		if len(names) == 0 {
			lines[0] = "You carry a lasting wound (help wounds)."
		}
		m.clearFoes(user, room)
	case domain.Robbed:
		lines = append(lines, m.rob(user, sc)...)
		m.clearFoes(user, room)
	}
	c.SetMiscData(domain.ScenarioKey, nil)
	c.SetMiscData(foeGroupKey, nil)
	c.SetMiscData(foeInstanceKey, nil)
	return lines
}

// wearDown leaves every member of a rescued company Hungry and Exhausted:
// their hunger and fatigue fall to the scenario's values, never up.
func (m *DeathModule) wearDown(leaderUserID int, sc domain.Scenario) {
	hunger, fatigue := sc.Hunger, sc.Fatigue
	if hunger <= 0 {
		hunger = defaultRescuedHunger
	}
	if fatigue <= 0 {
		fatigue = defaultRescuedFatigue
	}
	for _, member := range m.needs(leaderUserID) {
		cost := survival.Exertion{
			Hunger:  max(0, member.Needs.Hunger-hunger),
			Fatigue: max(0, member.Needs.Fatigue-fatigue),
		}
		if cost == (survival.Exertion{}) {
			continue
		}
		if _, err := m.drain(leaderUserID, member.Key, cost); err != nil {
			mudlog.Warn("death: rescued company not worn down", "leader", leaderUserID, "member", member.Key, "error", err)
		}
	}
}

// capture binds the leader, locks their pack and most of their gold in the
// captors' chest (the leader's Seized goods, saved with their items), and
// posts guards over it.
func (m *DeathModule) capture(user *users.UserRecord, sc domain.Scenario, room *rooms.Room) []string {
	c := user.Character
	c.Seized = append(c.Seized, c.Items...)
	c.Items = nil
	lost := domain.GoldShare(c.Gold, sc.GoldLossPct)
	held := c.Gold - lost
	c.Gold = 0
	if prior := seizedGold(c); prior > 0 {
		held += prior
	}
	if held > 0 {
		c.SetMiscData(domain.SeizedGoldKey, held)
	}
	c.SetMiscData(domain.SeizedRoomKey, sc.Room)
	c.SetMiscData(domain.SeizedByKey, sc.ID)
	if err := c.AddBuff(boundBuffID, false); err != nil {
		mudlog.Warn("death: bound buff not applied", "user", user.UserId, "error", err)
	}
	setSeizedGuards(c, m.postGuards(user.UserId, sc, room, sc.GuardCount))
	events.AddToQueue(events.CompanyAssetsChanged{UserId: user.UserId})
	lines := []string{"Your pack lies in a locked chest here. Defeat the guards, then <ansi fg=\"command\">reclaim</ansi> it (<ansi fg=\"command\">help defeat</ansi>)."}
	if lost > 0 {
		lines = append(lines, fmt.Sprintf("%d gold is gone for good.", lost))
	}
	return lines
}

// setSeizedGuards saves how many capture guards still stand.
func setSeizedGuards(c *characters.Character, n int) {
	if n <= 0 {
		c.SetMiscData(domain.SeizedGuardsKey, nil)
		return
	}
	c.SetMiscData(domain.SeizedGuardsKey, n)
}

// postGuards puts the capture room's guards in it: want of the scenario's
// guard mob, less any already standing from this company's capture, and
// returns how many now stand. They guard; they do not attack.
func (m *DeathModule) postGuards(userID int, sc domain.Scenario, room *rooms.Room, want int) int {
	if sc.GuardMob <= 0 || want <= 0 || room == nil {
		return 0
	}
	group := domain.CaptorGroup(room.RoomId, userID)
	need := want - len(m.guardsIn(room, group))
	var made []*mobs.Mob
	for ; need > 0; need-- {
		var guard *mobs.Mob
		if sc.GuardLevel > 0 {
			guard = mobs.NewMobById(mobs.MobId(sc.GuardMob), room.RoomId, sc.GuardLevel)
		} else {
			guard = mobs.NewMobById(mobs.MobId(sc.GuardMob), room.RoomId)
		}
		if guard == nil {
			mudlog.Warn("death: guard mob template unavailable", "mob", sc.GuardMob)
			break
		}
		guard.Hostile = false
		guard.MaxWander = 0
		// A guard that flees or yields would never fall, and so would
		// return over the chest: guards stand to the end.
		guard.NeverBreak = true
		guard.SpawnGroup = group
		made = append(made, guard)
	}
	name := ""
	if len(made) > 1 {
		summaries := make([]mobparty.MobSummary, len(made))
		for i, g := range made {
			g.Solitary = false
			summaries[i] = rooms.GroupSummary(g)
		}
		name = mobparty.Generate(summaries).Name
	}
	for _, g := range made {
		g.GroupName = name
		room.AddMob(g.InstanceId)
	}
	m.trackGuards(userID, group, made)
	return len(m.guardsIn(room, group))
}

// onMobDeath counts down a capture's guards as they fall, so the chest
// opens only once all of them have been defeated.
func (m *DeathModule) onMobDeath(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.MobDeath)
	if !ok {
		return events.Continue
	}
	m.mu.Lock()
	owner, guard := m.guardOwner[evt.InstanceId]
	delete(m.guardOwner, evt.InstanceId)
	m.mu.Unlock()
	if !guard {
		return events.Continue
	}
	user := m.lookupUser(owner.userID)
	if user == nil || user.Character == nil {
		return events.Continue
	}
	c := user.Character
	// Only a guard of the capture holding the pack now counts down.
	if owner.group == domain.CaptorGroup(seizedRoom(c), owner.userID) {
		setSeizedGuards(c, domain.SeizedGuards(c)-1)
	}
	return events.Continue
}

// trackGuards remembers whose pack newly posted guards stand over, and
// forgets guards that are gone without a death (a restart's or an unloaded
// room's).
func (m *DeathModule) trackGuards(userID int, group string, made []*mobs.Mob) {
	m.mu.Lock()
	known := make([]int, 0, len(m.guardOwner))
	for id := range m.guardOwner {
		known = append(known, id)
	}
	m.mu.Unlock()
	var gone []int
	for _, id := range known {
		if mobs.GetInstance(id) == nil {
			gone = append(gone, id)
		}
	}
	m.mu.Lock()
	for _, id := range gone {
		delete(m.guardOwner, id)
	}
	for _, g := range made {
		m.guardOwner[g.InstanceId] = guardOf{userID: userID, group: group}
	}
	m.mu.Unlock()
}

// restoreGuards posts a capture's undefeated guards again when the leader
// is in the capture room and none stand there: guards are not saved with
// the room, so a restart or an unloaded room would otherwise leave the
// chest unguarded. Reports whether it posted any.
func (m *DeathModule) restoreGuards(user *users.UserRecord, room *rooms.Room) bool {
	if user == nil || user.Character == nil || room == nil {
		return false
	}
	c := user.Character
	want := domain.SeizedGuards(c)
	if want <= 0 || seizedRoom(c) != room.RoomId {
		return false
	}
	group := domain.CaptorGroup(room.RoomId, user.UserId)
	for _, mobID := range room.GetMobs() {
		// Any guard still here, even one falling this round whose death is
		// not yet counted, means they were never lost.
		if mob := mobs.GetInstance(mobID); mob != nil && mob.SpawnGroup == group {
			return false
		}
	}
	id, _ := c.GetMiscData(domain.SeizedByKey).(string)
	sc, ok := domain.ByID(m.config().scenarios, id)
	if !ok || sc.Kind != domain.Captured {
		// The table no longer has the capture: nobody is left to guard it.
		setSeizedGuards(c, 0)
		return false
	}
	if m.postGuards(user.UserId, sc, room, want) == 0 {
		return false
	}
	user.SendText("Your captors stand over the chest again.")
	return true
}

// guardsIn lists the living guards of a capture group standing in a room.
func (m *DeathModule) guardsIn(room *rooms.Room, group string) []int {
	var out []int
	for _, id := range room.GetMobs() {
		if mob := mobs.GetInstance(id); mob != nil && mob.SpawnGroup == group && mob.Character.Health >= 1 {
			out = append(out, id)
		}
	}
	return out
}

// rob takes a share of the leader's gold and loose goods for good.
func (m *DeathModule) rob(user *users.UserRecord, sc domain.Scenario) []string {
	c := user.Character
	var lines []string
	if lost := domain.GoldShare(c.Gold, sc.GoldLossPct); lost > 0 {
		c.Gold -= lost
		lines = append(lines, fmt.Sprintf("%d gold is gone.", lost))
	}
	limit := sc.ItemLossMax
	if limit <= 0 {
		limit = 5
	}
	kept, taken := domain.Rob(c.Items, sc.ItemLossPct, limit, m.roll, nil)
	if len(taken) > 0 {
		c.Items = kept
		names := make([]string, 0, len(taken))
		for _, itm := range taken {
			names = append(names, itm.Name())
		}
		lines = append(lines, "Taken: "+strings.Join(names, ", ")+".")
	}
	if len(lines) == 0 {
		lines = append(lines, "The robbers found nothing worth taking.")
	}
	events.AddToQueue(events.CompanyAssetsChanged{UserId: user.UserId})
	return lines
}

// clearFoes sends away the foes that beat the company: the killing mob and
// its spawn group, or the ambush or encounter set on the leader, standing in
// the room they woke in. Companions are never touched. Their room spawns
// refill on the room's own timer.
func (m *DeathModule) clearFoes(user *users.UserRecord, room *rooms.Room) {
	if room == nil {
		return
	}
	c := user.Character
	group, _ := c.GetMiscData(foeGroupKey).(string)
	foeID, _ := configInt(c.GetMiscData(foeInstanceKey))
	for _, id := range room.GetMobs() {
		mob := mobs.GetInstance(id)
		if mob == nil {
			continue
		}
		if _, _, member := company.LeaderAndKeyForInstance(id); member {
			continue
		}
		if !(id == foeID || (group != "" && mob.SpawnGroup == group) || mob.AmbushOwner == user.UserId || mob.EncounterOwner == user.UserId || m.threatens(mob, user.UserId)) {
			continue
		}
		mobs.DestroyInstance(id)
		room.RemoveMob(id)
		if home := rooms.LoadRoom(mob.HomeRoomId); home != nil {
			// Despawned with its cooldown: it returns on the room's
			// respawn timer, not the next round.
			home.CleanupMobSpawns(false)
		}
	}
}

// threatens reports whether a mob would set on the woken company at once:
// it is hostile, it is fighting the leader or a companion, or its group
// holds a grudge against the leader. The foes "gone" include these.
func (m *DeathModule) threatens(mob *mobs.Mob, userID int) bool {
	if mob.Hostile {
		return true
	}
	if a := mob.Character.Aggro; a != nil {
		if a.UserId == userID {
			return true
		}
		if leader, _, member := company.LeaderAndKeyForInstance(a.MobInstanceId); member && leader == userID {
			return true
		}
	}
	for _, g := range mob.Groups {
		if mobs.IsHostile(g, userID) {
			return true
		}
	}
	return false
}

// --- reclaim ---

func seizedGold(c *characters.Character) int { return domain.SeizedGold(c) }

func seizedRoom(c *characters.Character) int { return domain.SeizedRoom(c) }

// reclaimCommand is "reclaim": at the capture room, with its guards down,
// the chest gives back the pack and gold the captors held.
func (m *DeathModule) reclaimCommand(_ string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	c := user.Character
	if len(c.Seized) == 0 && seizedGold(c) == 0 {
		user.SendText("Nothing of yours is being held.")
		return true, nil
	}
	if room == nil {
		room = m.loadRoom(c.RoomId)
	}
	held := seizedRoom(c)
	holdingRoom := m.loadRoom(held)
	// A chest whose room is gone gives its goods back anywhere: they are
	// never lost.
	if holdingRoom != nil && (room == nil || room.RoomId != held) {
		user.SendText(fmt.Sprintf("Your belongings are locked in a chest in %s.", roomTitle(holdingRoom)))
		return true, nil
	}
	if c.Aggro != nil {
		user.SendText("Not while you are fighting.")
		return true, nil
	}
	m.restoreGuards(user, room)
	if domain.SeizedGuards(c) > 0 || (room != nil && len(m.guardsIn(room, domain.CaptorGroup(room.RoomId, user.UserId))) > 0) {
		user.SendText("The guards stand over the chest. Defeat them first.")
		return true, nil
	}
	for _, itm := range c.Seized {
		c.StoreItem(itm)
	}
	count := len(c.Seized)
	gold := seizedGold(c)
	c.Seized = nil
	c.Gold += gold
	c.SetMiscData(domain.SeizedGoldKey, nil)
	c.SetMiscData(domain.SeizedRoomKey, nil)
	c.SetMiscData(domain.SeizedByKey, nil)
	c.SetMiscData(domain.SeizedGuardsKey, nil)
	events.AddToQueue(events.CompanyAssetsChanged{UserId: user.UserId})
	text := fmt.Sprintf("You break open the chest and take back your pack (%d items)", count)
	if gold > 0 {
		text += fmt.Sprintf(" and %d gold", gold)
	}
	user.SendText(text + ".")
	return true, nil
}

func (m *DeathModule) woundCompany(leaderUserID, pct int, roll func(int) int) []string {
	if m.wound == nil {
		return company.WoundCompany(leaderUserID, pct, roll)
	}
	return m.wound(leaderUserID, pct, roll)
}

// --- config ---

func stringList(raw any) []string {
	list, _ := raw.([]any)
	var out []string
	for _, item := range list {
		if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
			out = append(out, strings.ToLower(strings.TrimSpace(text)))
		}
	}
	return out
}

// parseScenarios reads the Scenarios config. An entry that isn't a map, or
// that fails Valid, or repeats an ID, is skipped with a warning.
func parseScenarios(raw any) []domain.Scenario {
	list, _ := raw.([]any)
	var out []domain.Scenario
	seen := map[string]bool{}
	for _, entry := range list {
		f := lowerKeys(entry)
		if f == nil {
			mudlog.Warn("death: scenario entry is not a map; skipped")
			continue
		}
		num := func(key string) int { n, _ := configInt(f[key]); return n }
		id, _ := f["id"].(string)
		kind, _ := f["kind"].(string)
		text, _ := f["text"].(string)
		sc := domain.Scenario{
			ID: strings.TrimSpace(id), Kind: domain.ScenarioKind(strings.ToLower(strings.TrimSpace(kind))),
			Weight: num("weight"), Foes: stringList(f["foes"]), Zones: stringList(f["zones"]), Text: strings.TrimSpace(text),
			Room: num("room"), GuardMob: num("guardmob"), GuardCount: num("guardcount"), GuardLevel: num("guardlevel"),
			GoldLossPct: num("goldlosspct"), ItemLossPct: num("itemlosspct"), ItemLossMax: num("itemlossmax"),
			WoundPct: num("woundpct"), Hunger: num("hunger"), Fatigue: num("fatigue"),
		}
		if err := sc.Valid(); err != nil {
			mudlog.Warn("death: scenario skipped", "error", err)
			continue
		}
		if seen[sc.ID] {
			mudlog.Warn("death: scenario listed twice; skipped", "id", sc.ID)
			continue
		}
		// Zones match the zone name as written in the world, so keep its case
		// for display but compare lower case: listHas lowers both sides.
		seen[sc.ID] = true
		out = append(out, sc)
	}
	return out
}
