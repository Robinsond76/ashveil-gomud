package camping

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 33f3 camp specialists: camp raids and Camp Watch, and the rewards
// of a completed, unbroken camp rest (Forage, Vigil). Raids are rolled when
// a rest starts and saved on it; they and the rewards are resolved on the
// game loop (onNewRound), never on a rest timer's goroutine. Rewards are
// applied through operation IDs the cargo and company records keep, so a
// retry after a restart never doubles or loses them.

// campRaid is one zone's raid table entry.
type campRaid struct {
	MobID     int
	ChancePct int
}

// forageFind is one weighted forage item.
type forageFind struct {
	ItemID int
	Weight int
}

// campRecipe is one camp-cooking recipe (Phase 33f3), the inn hearth's
// shape: an output from inputs, gated by a skill level.
type campRecipe struct {
	Output   int
	Inputs   []int
	Skill    string
	MinLevel int
}

// campSettings is 33f3's configuration.
type campSettings struct {
	Raids              map[string]campRaid
	WatchPctPerLevel   int
	RaidEarliestPct    int
	RaidLatestPct      int
	VigilCap           int
	Forage             map[string][]forageFind
	ForageBase         int
	ForageLevelsPerOne int
	Recipes            []campRecipe
	// RewardCooldown is how often a company may earn Forage and Vigil.
	RewardCooldown time.Duration
}

func defaultCampSettings() campSettings {
	return campSettings{
		Raids:              map[string]campRaid{},
		WatchPctPerLevel:   25,
		RaidEarliestPct:    30,
		RaidLatestPct:      70,
		VigilCap:           60,
		Forage:             map[string][]forageFind{},
		ForageBase:         1,
		ForageLevelsPerOne: 2,
		RewardCooldown:     15 * time.Minute,
	}
}

func listOf(raw any) []any {
	list, _ := raw.([]any)
	return list
}

func fieldsOf(raw any) map[string]any {
	out := map[string]any{}
	switch v := raw.(type) {
	case map[string]any:
		for k, item := range v {
			out[strings.ToLower(k)] = item
		}
	case map[any]any:
		for k, item := range v {
			if name, ok := k.(string); ok {
				out[strings.ToLower(name)] = item
			}
		}
	}
	return out
}

func intsOf(raw any) []int {
	var out []int
	for _, v := range listOf(raw) {
		if n, ok := configInt(v); ok && n > 0 {
			out = append(out, n)
		}
	}
	return out
}

// parseCampSettings reads 33f3's config; malformed entries are skipped.
func parseCampSettings(get func(string) any) campSettings {
	s := defaultCampSettings()
	if get == nil {
		return s
	}
	for _, entry := range listOf(get("CampRaids")) {
		f := fieldsOf(entry)
		zone := configString(f["zone"])
		mobID, okM := configInt(f["mobid"])
		chance, okC := configInt(f["chancepct"])
		if zone == "" || !okM || mobID <= 0 || !okC || chance < 0 || chance > 100 {
			mudlog.Warn("camping: CampRaids entry skipped", "entry", entry)
			continue
		}
		s.Raids[zone] = campRaid{MobID: mobID, ChancePct: chance}
	}
	pct := func(key string, into *int, lo, hi int) {
		if n, ok := configInt(get(key)); ok && n >= lo && n <= hi {
			*into = n
		}
	}
	pct("WatchPctPerLevel", &s.WatchPctPerLevel, 0, 100)
	pct("RaidEarliestPct", &s.RaidEarliestPct, 0, 100)
	pct("RaidLatestPct", &s.RaidLatestPct, 0, 100)
	if s.RaidLatestPct < s.RaidEarliestPct {
		s.RaidEarliestPct, s.RaidLatestPct = 30, 70
	}
	pct("VigilCap", &s.VigilCap, 0, company.MaxLoyalty)
	pct("ForageBase", &s.ForageBase, 0, 10)
	pct("ForageLevelsPerOne", &s.ForageLevelsPerOne, 1, 10)
	if raw := configString(get("CampRewardCooldown")); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d >= 0 {
			s.RewardCooldown = d
		}
	}
	for _, entry := range listOf(get("Forage")) {
		f := fieldsOf(entry)
		zone := configString(f["zone"])
		var finds []forageFind
		for _, it := range listOf(f["items"]) {
			fi := fieldsOf(it)
			id, okI := configInt(fi["itemid"])
			w, okW := configInt(fi["weight"])
			if okI && id > 0 && okW && w > 0 {
				finds = append(finds, forageFind{ItemID: id, Weight: w})
			}
		}
		if zone == "" || len(finds) == 0 {
			mudlog.Warn("camping: Forage entry skipped", "entry", entry)
			continue
		}
		s.Forage[zone] = finds
	}
	for _, entry := range listOf(get("CampRecipes")) {
		f := fieldsOf(entry)
		out, okO := configInt(f["output"])
		inputs := intsOf(f["inputs"])
		level, _ := configInt(f["minlevel"])
		if !okO || out <= 0 || len(inputs) == 0 {
			mudlog.Warn("camping: CampRecipes entry skipped", "entry", entry)
			continue
		}
		s.Recipes = append(s.Recipes, campRecipe{Output: out, Inputs: inputs, Skill: strings.ToLower(configString(f["skill"])), MinLevel: level})
	}
	return s
}

func (m *CampingModule) campSettings() campSettings {
	if !m.campCfgLoaded {
		return defaultCampSettings()
	}
	return m.campCfg
}

// rollPct is 0..99.
func (m *CampingModule) rollPct() int {
	if m.roll == nil {
		return 99
	}
	return m.roll(100)
}

// planRaidLocked (rest start) rolls whether raiders come to a camp in this
// zone and when, as a share of the rest.
func (m *CampingModule) planRaidLocked(room *rooms.Room, started time.Time) *camping.Raid {
	cfg := m.campSettings()
	raid, ok := cfg.Raids[room.Zone]
	if !ok || raid.ChancePct <= 0 || m.rollPct() >= raid.ChancePct {
		return nil
	}
	span := cfg.RaidLatestPct - cfg.RaidEarliestPct
	pct := cfg.RaidEarliestPct
	if span > 0 && m.roll != nil {
		pct += m.roll(span + 1)
	}
	at := started.UTC().Add(camping.RestDuration * time.Duration(pct) / 100)
	return &camping.Raid{AtUTC: at, MobID: raid.MobID}
}

// raidOutcome is a resolved raid, acted on outside the module lock.
type raidOutcome struct {
	leader  *users.UserRecord
	roomID  int
	mobID   int
	spotted bool
	watch   archetypes.Specialist
}

// fireDueRaids resolves every raid whose time has come (game loop). With
// the leader at the camp, the company's watch may spot it; unspotted, it
// breaks the rest. A leader offline when the raiders come is robbed of
// the rest (33f3 review: logging out must not dodge a raid); one who is
// away from the camp (moved by an admin or a death) is not raided. The raid
// is saved as resolved before the raiders spawn, so it never comes twice.
// The watch is looked up outside the module lock.
func (m *CampingModule) fireDueRaids() {
	now := m.clock().UTC()
	cfg := m.campSettings()
	m.mu.Lock()
	due := map[int]int{} // leader -> camp room
	for leaderUserID, camp := range m.camps {
		if camp.Rest != nil && camp.Rest.RaidDue(now) {
			due[leaderUserID] = camp.RoomID
		}
	}
	m.mu.Unlock()
	if len(due) == 0 {
		return
	}
	leaders := make([]int, 0, len(due))
	for id := range due {
		leaders = append(leaders, id)
	}
	sort.Ints(leaders)
	var outcomes []raidOutcome
	for _, leaderUserID := range leaders {
		roomID := due[leaderUserID]
		leader := m.userByID(leaderUserID)
		online := leader != nil && leader.Character != nil
		present := online && leader.Character.RoomId == roomID
		var watch archetypes.Specialist
		hasWatch := false
		if present && m.specialist != nil {
			watch, hasWatch = m.specialist(leaderUserID, archetypes.UtilityWatch, roomID)
		}
		spotted := hasWatch && m.rollPct() < archetypes.PctByLevel(watch.Level, cfg.WatchPctPerLevel, 100)

		m.mu.Lock()
		camp, ok := m.camps[leaderUserID]
		if !ok || camp.Rest == nil || !camp.Rest.RaidDue(now) || camp.RoomID != roomID {
			m.mu.Unlock()
			continue
		}
		rest := *camp.Rest
		raid := *rest.Raid
		raid.Fired = true
		raid.Spotted = present && spotted
		if !online || present {
			rest.Broken = !raid.Spotted
		}
		rest.Raid = &raid
		updated := camp
		updated.Rest = &rest
		m.camps[leaderUserID] = updated
		if err := m.saveLocked(); err != nil {
			m.camps[leaderUserID] = camp
			m.mu.Unlock()
			mudlog.Error("camping: raid save", "leader", leaderUserID, "error", err)
			continue
		}
		m.mu.Unlock()
		if present {
			outcomes = append(outcomes, raidOutcome{leader: leader, roomID: roomID, mobID: raid.MobID, spotted: raid.Spotted, watch: watch})
		}
	}
	for _, o := range outcomes {
		spawn := m.spawnRaid
		if spawn == nil {
			continue
		}
		first, err := spawn(o.roomID, o.mobID, o.leader.UserId)
		if err != nil {
			// No raiders after all: the rest is not spoiled.
			mudlog.Warn("camping: raid spawn", "leader", o.leader.UserId, "error", err)
			m.unbreakRest(o.leader.UserId)
			continue
		}
		m.trackRaiders(o.leader.UserId, o.roomID, first)
		if o.spotted {
			o.leader.SendText(fmt.Sprintf(`<ansi fg="yellow-bold">%s %s raiders creeping toward the fire and %s the company! Your rest can go on once they're dealt with.</ansi>`, o.watch.Subject(), o.watch.Verb("spot", "spots"), o.watch.Verb("rouse", "rouses")))
		} else {
			o.leader.SendText(`<ansi fg="red-bold">Raiders fall on your sleeping camp! Nobody saw them coming, and the rest is spoiled.</ansi>`)
		}
	}
}

// unbreakRest clears a raid's broken mark when its raiders never came.
func (m *CampingModule) unbreakRest(leaderUserID int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	camp, ok := m.camps[leaderUserID]
	if !ok || camp.Rest == nil || !camp.Rest.Broken {
		return
	}
	rest := *camp.Rest
	rest.Broken = false
	updated := camp
	updated.Rest = &rest
	m.camps[leaderUserID] = updated
	if err := m.saveLocked(); err != nil {
		m.camps[leaderUserID] = camp
		mudlog.Error("camping: raid unbreak save", "leader", leaderUserID, "error", err)
	}
}

// raiders are a leader's live raid group (runtime only: mobs never outlive
// a restart).
type raiders struct {
	roomID    int
	instances []int
}

// trackRaiders remembers the raid group that just spawned in roomID, led
// by first, so it can be sent off once its leader is gone.
func (m *CampingModule) trackRaiders(leaderUserID, roomID, first int) {
	ids := []int{first}
	if m.raidGroup != nil {
		ids = m.raidGroup(roomID, first)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.raiders == nil {
		m.raiders = map[int]raiders{}
	}
	m.raiders[leaderUserID] = raiders{roomID: roomID, instances: ids}
}

// clearRaiders (game loop, 33f3 review: no bystander may be drawn into
// another company's raid) sends off a raid group whose leader is offline,
// fallen, or gone from the camp room, so it never turns on other players
// there. A group whose members have all fallen is forgotten.
func (m *CampingModule) clearRaiders() {
	m.mu.Lock()
	tracked := map[int]raiders{}
	for leaderUserID, r := range m.raiders {
		tracked[leaderUserID] = r
	}
	m.mu.Unlock()
	for leaderUserID, r := range tracked {
		alive := m.liveRaiders(r)
		leader := m.userByID(leaderUserID)
		gone := leader == nil || leader.Character == nil || leader.Character.Health <= 0 || leader.Character.RoomId != r.roomID
		if len(alive) > 0 && gone && m.despawnRaider != nil {
			for _, id := range alive {
				m.despawnRaider(id)
			}
			alive = nil
		}
		if len(alive) == 0 {
			m.mu.Lock()
			delete(m.raiders, leaderUserID)
			m.mu.Unlock()
		}
	}
}

func (m *CampingModule) liveRaiders(r raiders) []int {
	var alive []int
	for _, id := range r.instances {
		if m.raiderAlive != nil && m.raiderAlive(id) {
			alive = append(alive, id)
		}
	}
	return alive
}

// --- rewards of a completed rest: Forage and Vigil ---

// campReward is a completed, unbroken rest's owed Forage and Vigil: the
// rest's operation ID and the camp's room.
type campReward struct {
	Op     string `yaml:"op"`
	RoomID int    `yaml:"room_id"`
}

// grantCampRewards applies each owed camp reward (game loop) once the
// leader is online, out of battle, and at the camp: Forage into cargo and
// Vigil on loyalty, each once by its operation ID, then clears the debt.
// A leader who left the camp's room before collecting forfeits them.
func (m *CampingModule) grantCampRewards() {
	m.mu.Lock()
	owed := map[int]campReward{}
	for leaderUserID, r := range m.campRewards {
		owed[leaderUserID] = r
	}
	m.mu.Unlock()
	leaders := make([]int, 0, len(owed))
	for id := range owed {
		leaders = append(leaders, id)
	}
	sort.Ints(leaders)
	for _, leaderUserID := range leaders {
		leader := m.userByID(leaderUserID)
		if leader == nil || leader.Character == nil {
			continue // owed until the leader is back online
		}
		if m.inBattle != nil && m.inBattle(leaderUserID) {
			continue // owed until the fight (a spotted raid's, say) is over
		}
		reward := owed[leaderUserID]
		var lines []string
		if leader.Character.RoomId == reward.RoomID {
			room := rooms.LoadRoom(reward.RoomID)
			text, err := m.forage(leader, room, reward.Op+":forage")
			if err != nil {
				mudlog.Warn("camping: forage", "leader", leaderUserID, "error", err)
				continue // retried next round; the op keeps it once
			}
			if text != "" {
				lines = append(lines, text)
			}
			text, err = m.vigil(leader, reward.RoomID, reward.Op+":vigil")
			if err != nil {
				mudlog.Warn("camping: vigil", "leader", leaderUserID, "error", err)
				continue
			}
			if text != "" {
				lines = append(lines, text)
			}
		}
		m.mu.Lock()
		if current, ok := m.campRewards[leaderUserID]; ok && current == reward {
			delete(m.campRewards, leaderUserID)
			if err := m.saveLocked(); err != nil {
				m.campRewards[leaderUserID] = reward
				mudlog.Error("camping: camp rewards save", "leader", leaderUserID, "error", err)
			}
		}
		m.mu.Unlock()
		for _, line := range lines {
			leader.SendText(line)
		}
	}
}

// forage: the company's best forager finds 1 + level/2 items from the
// zone's table, as many as the company can carry, into its cargo.
func (m *CampingModule) forage(leader *users.UserRecord, room *rooms.Room, op string) (string, error) {
	if room == nil || m.specialist == nil {
		return "", nil
	}
	cfg := m.campSettings()
	table := cfg.Forage[room.Zone]
	if len(table) == 0 {
		return "", nil
	}
	sp, ok := m.specialist(leader.UserId, archetypes.UtilityForage, room.RoomId)
	if !ok {
		return "", nil
	}
	count := cfg.ForageBase + sp.Level/cfg.ForageLevelsPerOne
	total := 0
	for _, f := range table {
		total += f.Weight
	}
	found := map[int]int{}
	var order []int
	grams, leftOver := 0, 0
	for i := 0; i < count; i++ {
		pick := m.rollPct() * total / 100
		itemID := table[len(table)-1].ItemID
		for _, f := range table {
			if pick < f.Weight {
				itemID = f.ItemID
				break
			}
			pick -= f.Weight
		}
		weight := 0
		if spec := items.GetItemSpec(itemID); spec != nil {
			weight = spec.Weight
		}
		if _, full := encumbrance.WouldExceed(leader.UserId, grams+weight); full {
			leftOver++
			continue
		}
		grams += weight
		if found[itemID] == 0 {
			order = append(order, itemID)
		}
		found[itemID]++
	}
	if len(order) == 0 {
		if leftOver > 0 {
			return fmt.Sprintf("%s %s food nearby, but the company can carry nothing more.", sp.Subject(), sp.Verb("find", "finds")), nil
		}
		return "", nil
	}
	deposits := make([]encumbrance.CargoStack, 0, len(order))
	names := make([]string, 0, len(order))
	for _, id := range order {
		deposits = append(deposits, encumbrance.CargoStack{ItemId: id, Count: found[id]})
		name := fmt.Sprintf("item %d", id)
		if spec := items.GetItemSpec(id); spec != nil {
			name = spec.Name
		}
		if found[id] > 1 {
			name = fmt.Sprintf("%d %s", found[id], name)
		}
		names = append(names, fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, name))
	}
	if err := encumbrance.DepositCargo(leader.UserId, op, deposits); err != nil {
		if errors.Is(err, encumbrance.ErrNoCargo) {
			return "", nil
		}
		return "", err
	}
	var text string
	if sp.IsLeader {
		text = fmt.Sprintf("While the company rested, you foraged %s, now in the company's cargo.", strings.Join(names, ", "))
	} else {
		text = fmt.Sprintf("While the company rested, %s foraged %s, now in the company's cargo.", sp.Name, strings.Join(names, ", "))
	}
	if leftOver > 0 {
		text += " There was more, but the company can carry no more."
	}
	return text, nil
}

// vigil: the company's best cleric keeps a vigil; each present living
// companion gains loyalty equal to the cleric's level, up to VigilCap.
func (m *CampingModule) vigil(leader *users.UserRecord, roomID int, op string) (string, error) {
	if m.specialist == nil {
		return "", nil
	}
	sp, ok := m.specialist(leader.UserId, archetypes.UtilityVigil, roomID)
	if !ok {
		return "", nil
	}
	live, _ := m.companions(leader.UserId)
	var present []int
	for id, c := range live {
		if c != nil && c.RoomId == roomID && c.Health > 0 {
			present = append(present, id)
		}
	}
	if len(present) == 0 {
		return "", nil
	}
	sort.Ints(present)
	raised, err := company.RaiseLoyaltyOnce(leader.UserId, op, present, sp.Level, m.campSettings().VigilCap)
	if err != nil || len(raised) == 0 {
		return "", err
	}
	return fmt.Sprintf("%s %s a vigil by the fire; %s %s steadier for it.", sp.Subject(), sp.Verb("keep", "keeps"), strings.Join(raised, ", "), map[bool]string{true: "is", false: "are"}[len(raised) == 1]), nil
}

// --- camp cooking ---

// cook is "camp cook": at the leader's own camp with its fire lit, out of
// battle, the leader cooks the first recipe their Cooking allows from
// ingredients in their pack and the company cargo (the pack first). The
// dish goes into the cargo (into the pack when there is no cargo).
func (m *CampingModule) cook(user *users.UserRecord, room *rooms.Room) string {
	if user == nil || room == nil {
		return "You can't cook here."
	}
	m.mu.Lock()
	camp, ok := m.camps[user.UserId]
	m.mu.Unlock()
	switch {
	case !ok || camp.RoomID != room.RoomId:
		return `You need your own camp here to cook. Use "camp" to make one.`
	case !camp.FireLit:
		return `You need a lit campfire to cook. Use "camp fire" first.`
	}
	if m.inBattle != nil && m.inBattle(user.UserId) {
		return "You can't cook in the middle of a fight."
	}
	recipes := m.campSettings().Recipes
	if len(recipes) == 0 {
		return "There is nothing to cook over a campfire."
	}
	// The ingredients to hand: pack items, then cargo stacks.
	have := map[int]int{}
	if !user.Character.CompanyCargo {
		for _, itm := range user.Character.Items {
			have[itm.ItemId]++
		}
	}
	cargo := map[int]int{}
	for _, s := range encumbrance.CargoContents(user.UserId) {
		cargo[s.ItemId] += s.Count
		have[s.ItemId] += s.Count
	}
	var chosen *campRecipe
	var blocked *campRecipe
	for i := range recipes {
		r := &recipes[i]
		need := map[int]int{}
		for _, id := range r.Inputs {
			need[id]++
		}
		ready := true
		for id, n := range need {
			if have[id] < n {
				ready = false
				break
			}
		}
		if !ready {
			continue
		}
		if r.Skill != "" && user.Character.GetSkillLevel(r.Skill) < r.MinLevel {
			if blocked == nil || r.MinLevel < blocked.MinLevel {
				blocked = r
			}
			continue
		}
		chosen = r
		break
	}
	if chosen == nil {
		if blocked != nil {
			return fmt.Sprintf("You have the makings of %s, but it needs %s %d.", itemName(blocked.Output), blocked.Skill, blocked.MinLevel)
		}
		return "You have nothing to cook: see help cooking for what each dish needs."
	}
	// Plan where each ingredient comes from: the pack first, then cargo.
	var fromPack []items.Item
	fromCargo := map[int]int{}
	taken := map[string]bool{}
	inputGrams := 0
	for _, id := range chosen.Inputs {
		if spec := items.GetItemSpec(id); spec != nil {
			inputGrams += spec.Weight
		}
		picked := false
		for _, itm := range user.Character.Items {
			if itm.ItemId == id && !taken[itm.UUID.String()] {
				taken[itm.UUID.String()] = true
				fromPack = append(fromPack, itm)
				picked = true
				break
			}
		}
		if !picked {
			fromCargo[id]++
		}
	}
	// The dish may weigh more than what went into it (33f3 review).
	dishGrams := 0
	if spec := items.GetItemSpec(chosen.Output); spec != nil {
		dishGrams = spec.Weight
	}
	if text, refuse := encumbrance.TooMuchToCarry(user.UserId, dishGrams-inputGrams); refuse {
		return text
	}
	if user.Character.CompanyCargo {
		if err := encumbrance.TransformCargo(user.UserId, fromPack, []items.Item{items.New(chosen.Output)}); err != nil {
			return "The ingredients couldn't be saved; nothing was cooked."
		}
		return fmt.Sprintf("You cook %s over the campfire; it goes into company cargo.", itemName(chosen.Output))
	}
	// Take from the cargo first, putting back what was taken if any of it
	// fails, so a failed save never eats ingredients (33f3 review).
	var withdrawn []encumbrance.CargoStack
	for _, id := range sortedKeys(fromCargo) {
		if err := encumbrance.WithdrawCargo(user.UserId, id, fromCargo[id]); err != nil {
			mudlog.Warn("camping: cook withdraw", "leader", user.UserId, "item", id, "error", err)
			if len(withdrawn) > 0 {
				if err := encumbrance.DepositCargo(user.UserId, "", withdrawn); err != nil {
					mudlog.Error("camping: cook restore", "leader", user.UserId, "error", err)
				}
			}
			return "The ingredients couldn't be gathered right now."
		}
		withdrawn = append(withdrawn, encumbrance.CargoStack{ItemId: id, Count: fromCargo[id]})
	}
	for _, itm := range fromPack {
		user.Character.RemoveItem(itm)
	}
	dish := itemName(chosen.Output)
	if err := encumbrance.DepositCargo(user.UserId, "", []encumbrance.CargoStack{{ItemId: chosen.Output, Count: 1}}); err != nil {
		if !user.Character.StoreItem(items.New(chosen.Output)) {
			room.AddItem(items.New(chosen.Output), false)
			return fmt.Sprintf(`You cook <ansi fg="itemname">%s</ansi> over the campfire and set it down by the fire.`, dish)
		}
		return fmt.Sprintf(`You cook <ansi fg="itemname">%s</ansi> over the campfire.`, dish)
	}
	return fmt.Sprintf(`You cook <ansi fg="itemname">%s</ansi> over the campfire; it goes into the company's cargo.`, dish)
}

func sortedKeys(m map[int]int) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func packItem(user *users.UserRecord, itemID int) (items.Item, bool) {
	for _, itm := range user.Character.Items {
		if itm.ItemId == itemID {
			return itm, true
		}
	}
	return items.Item{}, false
}

func itemName(itemID int) string {
	if spec := items.GetItemSpec(itemID); spec != nil {
		return spec.Name
	}
	return fmt.Sprintf("item %d", itemID)
}

// raidGroupOf lists the instances of the raid group led by first in roomID.
func raidGroupOf(roomID, first int) []int {
	ids := []int{first}
	room := rooms.LoadRoom(roomID)
	lead := mobs.GetInstance(first)
	if room == nil || lead == nil || lead.SpawnGroup == "" {
		return ids
	}
	for _, id := range room.GetMobs() {
		if id == first {
			continue
		}
		if mob := mobs.GetInstance(id); mob != nil && mob.SpawnGroup == lead.SpawnGroup {
			ids = append(ids, id)
		}
	}
	return ids
}
