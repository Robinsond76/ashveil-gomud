package camping

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 43b weapon poisons. `coat` puts one dose of a bought vial on one
// equipped blade (main or off hand), outside a fight. `camp poison` keeps a
// per-company preparation list (member, hand, poison) and applies it whole
// or not at all. Doses come from the company's supplies (cargo first, then
// packs) through the itemCount/spendItem seams. Everything runs on the game
// loop in one step, so the doses and the coatings change together.

const coatUsage = "Usage: coat <poison> [self|member] [main|off] | coat status | coat clear [self|member] [main|off]  (poisons: bitterleaf, leechbane, leadroot, mirethorn)"

const poisonUsage = "Usage: camp poison | camp poison assign <member|self> <main|off> <poison> | camp poison unassign <member|self> <main|off> | camp poison preview | camp poison apply"

// PoisonAssign is one row of a company's camp preparation list: a blade
// (member, hand) and the poison it should carry. A preference, never a
// reservation or an active coating.
type PoisonAssign struct {
	Member int    `yaml:"member"` // 0 is the leader, else the companion ID
	Hand   string `yaml:"hand"`   // "main" or "off"
	Poison string `yaml:"poison"`
}

func clonePlans(in map[int][]PoisonAssign) map[int][]PoisonAssign {
	out := make(map[int][]PoisonAssign, len(in))
	for k, v := range in {
		out[k] = append([]PoisonAssign(nil), v...)
	}
	return out
}

// coatBlade is one equipped blade of one member.
type coatBlade struct {
	Member  int
	Name    string
	Hand    string
	Item    *items.Item
	Present bool
}

func (m *CampingModule) now() time.Time {
	if m.clock != nil {
		return m.clock()
	}
	return time.Now()
}

func (m *CampingModule) doseCount(leaderUserID, itemID int) int {
	count := m.itemCount
	if count == nil {
		count = company.CompanyItemCount
	}
	return count(leaderUserID, itemID)
}

func (m *CampingModule) spendDose(leaderUserID, itemID int) bool {
	spend := m.spendItem
	if spend == nil {
		spend = company.SpendCompanyItem
	}
	return spend(leaderUserID, itemID)
}

// coatBlades lists every rostered member's equipped bladed weapons, leader
// first, then companions by ID; a member not here has no blades listed (a
// companion without a live mob has its gear on the company record).
func (m *CampingModule) coatBlades(user *users.UserRecord) (rows []coatBlade, names map[int]string, fighting bool) {
	names = map[int]string{camping.LeaderID: user.Character.Name}
	add := func(id int, name string, c *characters.Character, present bool) {
		names[id] = name
		if c == nil || !present {
			return
		}
		for _, h := range []struct {
			hand string
			itm  *items.Item
		}{{"main", &c.Equipment.Weapon}, {"off", &c.Equipment.Offhand}} {
			if h.itm.ItemId <= 0 {
				continue
			}
			spec := h.itm.GetSpec()
			if spec.Type == items.Weapon && camping.Bladed(string(spec.Subtype)) {
				rows = append(rows, coatBlade{Member: id, Name: name, Hand: h.hand, Item: h.itm, Present: true})
			}
		}
	}
	fighting = user.Character.Aggro != nil
	add(camping.LeaderID, user.Character.Name, user.Character, true)
	live, roster := m.companions(user.UserId)
	rn := rosterNames(user.UserId)
	for _, id := range sortedIDs(live) {
		c := live[id]
		if c.Aggro != nil {
			fighting = true
		}
		add(id, c.Name, c, c.RoomId == user.Character.RoomId)
	}
	for _, id := range roster {
		if _, ok := live[id]; ok {
			continue
		}
		name := rn[id]
		if name == "" {
			name = fmt.Sprintf("companion #%d", id)
		}
		names[id] = name
	}
	return rows, names, fighting
}

// memberByName resolves "self", or a member's name (a unique prefix).
func memberByName(names map[int]string, who string) (int, string) {
	who = strings.ToLower(who)
	if who == "" || who == "self" || who == "me" {
		return camping.LeaderID, ""
	}
	match, found := -1, 0
	for id, n := range names {
		ln := strings.ToLower(n)
		if ln == who {
			return id, ""
		}
		if id != camping.LeaderID && strings.HasPrefix(ln, who) {
			match, found = id, found+1
		}
	}
	switch {
	case found == 1:
		return match, ""
	case found > 1:
		return 0, fmt.Sprintf("More than one member matches %q.", who)
	}
	return 0, fmt.Sprintf("Nobody in your company is called %q.", who)
}

// coatCommand serves "coat ...".
func (m *CampingModule) coatCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	user.SendText(m.coatArgs(user, strings.Fields(strings.ToLower(strings.TrimSpace(rest)))))
	return true, nil
}

// coatArgs serves both "coat ..." and "camp coat ...".
func (m *CampingModule) coatArgs(user *users.UserRecord, args []string) string {
	if len(args) == 0 {
		return coatUsage + "\n" + m.coatStatus(user)
	}
	switch args[0] {
	case "status":
		return m.coatStatus(user)
	case "clear":
		return m.coatClear(user, args[1:])
	}
	poison, ok := items.PoisonByID(args[0])
	if !ok {
		return fmt.Sprintf("There is no poison called %q.\n%s", args[0], coatUsage)
	}
	return m.coatApply(user, poison, args[1:])
}

// handArgs splits [member] [main|off] arguments.
func handArgs(args []string) (who, hand string, bad bool) {
	hand = "main"
	rest := []string{}
	for _, a := range args {
		if a == "main" || a == "off" {
			hand = a
			continue
		}
		rest = append(rest, a)
	}
	if len(rest) > 1 {
		return "", "", true
	}
	if len(rest) == 1 {
		who = rest[0]
	}
	return who, hand, false
}

// stationary says why the company can't handle poisons now, or "".
func (m *CampingModule) stationary(user *users.UserRecord, fighting bool) string {
	switch {
	case fighting:
		return "You can't coat blades in the middle of a fight."
	case m.isTravelling(user.UserId):
		return "You can't coat blades while travelling; stop first."
	}
	m.mu.Lock()
	camp, ok := m.camps[user.UserId]
	resting := ok && camp.Rest != nil && camp.Rest.State == camping.Resting
	stay, stayOK := m.stays[user.UserId]
	m.mu.Unlock()
	if resting || (stayOK && stay.Resting()) {
		return "You can't coat blades while resting."
	}
	return ""
}

func findBlade(rows []coatBlade, member int, hand string) *coatBlade {
	for i := range rows {
		if rows[i].Member == member && rows[i].Hand == hand {
			return &rows[i]
		}
	}
	return nil
}

// bladeTarget finds the one blade a coat or clear names, or says why not.
func (m *CampingModule) bladeTarget(user *users.UserRecord, args []string) (*coatBlade, string) {
	who, hand, bad := handArgs(args)
	if bad {
		return nil, coatUsage
	}
	rows, names, fighting := m.coatBlades(user)
	if msg := m.stationary(user, fighting); msg != "" {
		return nil, msg
	}
	id, msg := memberByName(names, who)
	if msg != "" {
		return nil, msg
	}
	b := findBlade(rows, id, hand)
	if b == nil {
		if _, known := names[id]; known && !memberHere(rows, id) && id != camping.LeaderID {
			return nil, fmt.Sprintf("%s is not here, or carries no blade.", names[id])
		}
		return nil, fmt.Sprintf("%s has no bladed weapon in the %s hand. Poison goes on blades only.", names[id], hand)
	}
	return b, ""
}

func memberHere(rows []coatBlade, id int) bool {
	for _, r := range rows {
		if r.Member == id {
			return true
		}
	}
	return false
}

func (m *CampingModule) coatApply(user *users.UserRecord, poison items.Poison, args []string) string {
	b, msg := m.bladeTarget(user, args)
	if msg != "" {
		return msg
	}
	now := m.now()
	if b.Item.Coated(now) {
		return fmt.Sprintf("%s's %s already carries a coating (%s). Clear it first: coat clear.", b.Name, b.Item.DisplayName(), b.Item.CoatLabel(now))
	}
	if m.doseCount(user.UserId, poison.VialID) < 1 {
		return fmt.Sprintf("You have no %s vial.", poison.Name)
	}
	if !m.spendDose(user.UserId, poison.VialID) {
		return fmt.Sprintf("You have no %s vial.", poison.Name)
	}
	b.Item.ClearCoat() // drop a lapsed coating's leftovers
	b.Item.Coat(poison.ID, items.CoatExpiry(now), items.CoatContacts, now)
	return fmt.Sprintf("You coat %s's %s with %s: it lasts %d minutes or %d wounding blows. Each blow that wounds has a %d%% chance to leave it in the wound.",
		possessive(b.Name, user), b.Item.DisplayName(), poison.Name, items.CoatMinutes, items.CoatContacts, items.DeliveryPct)
}

func possessive(name string, user *users.UserRecord) string {
	if name == user.Character.Name {
		return "your"
	}
	return name
}

func (m *CampingModule) coatClear(user *users.UserRecord, args []string) string {
	b, msg := m.bladeTarget(user, args)
	if msg != "" {
		return msg
	}
	if !b.Item.Coated(m.now()) && b.Item.CoatKind == "" {
		return fmt.Sprintf("%s's %s carries no coating.", b.Name, b.Item.DisplayName())
	}
	b.Item.ClearCoat()
	return fmt.Sprintf("You wipe %s's %s clean. The dose is lost.", b.Name, b.Item.DisplayName())
}

func (m *CampingModule) vialLine(leaderUserID int) string {
	parts := []string{}
	for _, p := range items.Poisons {
		parts = append(parts, fmt.Sprintf("%s %d", p.Name, m.doseCount(leaderUserID, p.VialID)))
	}
	return "Vials: " + strings.Join(parts, ", ") + "."
}

// coatStatus shows every blade's coating and the vials on hand.
func (m *CampingModule) coatStatus(user *users.UserRecord) string {
	rows, _, _ := m.coatBlades(user)
	now := m.now()
	lines := []string{m.vialLine(user.UserId)}
	if len(rows) == 0 {
		lines = append(lines, "No blade in your company can take a coating.")
	}
	for _, r := range rows {
		label := r.Item.CoatLabel(now)
		if label == "" {
			label = "no coating"
		}
		lines = append(lines, fmt.Sprintf("  %s %s: %s %s", r.Name, r.Hand, r.Item.DisplayName(), label))
	}
	return strings.Join(lines, "\n")
}

// --- camp poison: the preparation list ---

func (m *CampingModule) poisonPlan(leaderUserID int) []PoisonAssign {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]PoisonAssign(nil), m.poisonPlans[leaderUserID]...)
}

// poisonArgs serves "camp poison ...".
func (m *CampingModule) poisonArgs(user *users.UserRecord, args []string) string {
	if !m.hasCamp(user.UserId) {
		return "You have no camp. Make one with \"camp\"; poisons are prepared there."
	}
	if len(args) == 0 {
		return m.poisonShow(user)
	}
	switch args[0] {
	case "preview":
		text, _, _ := m.poisonCheck(user)
		return text
	case "apply":
		return m.poisonApply(user)
	case "assign":
		if len(args) != 4 {
			return poisonUsage
		}
		return m.poisonAssign(user, args[1], args[2], args[3])
	case "unassign":
		if len(args) != 3 {
			return poisonUsage
		}
		return m.poisonUnassign(user, args[1], args[2])
	}
	return poisonUsage
}

func (m *CampingModule) poisonShow(user *users.UserRecord) string {
	rows, names, _ := m.coatBlades(user)
	now := m.now()
	plan := m.poisonPlan(user.UserId)
	planned := map[string]string{}
	for _, a := range plan {
		planned[fmt.Sprintf("%d/%s", a.Member, a.Hand)] = a.Poison
	}
	lines := []string{m.vialLine(user.UserId)}
	for _, r := range rows {
		coat := r.Item.CoatLabel(now)
		if coat == "" {
			coat = "no coating"
		}
		want := "-"
		if p, ok := planned[fmt.Sprintf("%d/%s", r.Member, r.Hand)]; ok {
			want = p
		}
		lines = append(lines, fmt.Sprintf("  %s %s: %s, %s; assigned: %s", r.Name, r.Hand, r.Item.DisplayName(), coat, want))
	}
	for _, a := range plan {
		if _, ok := names[a.Member]; !ok {
			continue
		}
		if findBlade(rows, a.Member, a.Hand) == nil {
			lines = append(lines, fmt.Sprintf("  %s %s: assigned %s, but no usable blade there now", names[a.Member], a.Hand, a.Poison))
		}
	}
	lines = append(lines, "camp poison assign <member|self> <main|off> <poison>, then camp poison preview and camp poison apply.")
	return strings.Join(lines, "\n")
}

func (m *CampingModule) poisonAssign(user *users.UserRecord, who, hand, poisonID string) string {
	if hand != "main" && hand != "off" {
		return poisonUsage
	}
	if _, ok := items.PoisonByID(poisonID); !ok {
		return fmt.Sprintf("There is no poison called %q.", poisonID)
	}
	rows, names, _ := m.coatBlades(user)
	id, msg := memberByName(names, who)
	if msg != "" {
		return msg
	}
	if findBlade(rows, id, hand) == nil {
		return fmt.Sprintf("%s has no bladed weapon in the %s hand here, so nothing can be assigned.", names[id], hand)
	}
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	old := m.poisonPlans[user.UserId]
	next := make([]PoisonAssign, 0, len(old)+1)
	for _, a := range old {
		if a.Member != id || a.Hand != hand {
			next = append(next, a)
		}
	}
	next = append(next, PoisonAssign{Member: id, Hand: hand, Poison: poisonID})
	m.poisonPlans[user.UserId] = next
	if err := m.saveLocked(); err != nil {
		m.poisonPlans[user.UserId] = old
		return err.Error()
	}
	return fmt.Sprintf("Assigned %s to %s's %s hand. Nothing is spent until camp poison apply.", poisonID, names[id], hand)
}

func (m *CampingModule) poisonUnassign(user *users.UserRecord, who, hand string) string {
	if hand != "main" && hand != "off" {
		return poisonUsage
	}
	_, names, _ := m.coatBlades(user)
	id, msg := memberByName(names, who)
	if msg != "" {
		return msg
	}
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	old := m.poisonPlans[user.UserId]
	next := make([]PoisonAssign, 0, len(old))
	for _, a := range old {
		if a.Member != id || a.Hand != hand {
			next = append(next, a)
		}
	}
	if len(next) == len(old) {
		return fmt.Sprintf("%s's %s hand has no assignment.", names[id], hand)
	}
	if len(next) == 0 {
		delete(m.poisonPlans, user.UserId)
	} else {
		m.poisonPlans[user.UserId] = next
	}
	if err := m.saveLocked(); err != nil {
		m.poisonPlans[user.UserId] = old
		return err.Error()
	}
	return fmt.Sprintf("Cleared the assignment on %s's %s hand.", names[id], hand)
}

// poisonRow is one validated line of the preparation list.
type poisonRow struct {
	blade  *coatBlade
	poison items.Poison
	skip   bool // already carries this poison
}

// poisonCheck validates the whole list against the company as it stands:
// the preview text, the rows to apply and whether it may proceed. Any
// blocker stops everything; nothing is spent here.
func (m *CampingModule) poisonCheck(user *users.UserRecord) (string, []poisonRow, bool) {
	plan := m.poisonPlan(user.UserId)
	if len(plan) == 0 {
		return "No poison is assigned. Use camp poison assign <member|self> <main|off> <poison>.", nil, false
	}
	rows, names, fighting := m.coatBlades(user)
	now := m.now()
	sort.SliceStable(plan, func(i, j int) bool {
		if plan[i].Member != plan[j].Member {
			return plan[i].Member < plan[j].Member
		}
		return plan[i].Hand == "main" && plan[j].Hand != "main"
	})
	var lines, blockers []string
	var out []poisonRow
	need := map[string]int{}
	if msg := m.stationary(user, fighting); msg != "" {
		blockers = append(blockers, msg)
	}
	for _, a := range plan {
		name, known := names[a.Member]
		if !known {
			continue // a member who left the roster: the row is dead
		}
		poison, ok := items.PoisonByID(a.Poison)
		b := findBlade(rows, a.Member, a.Hand)
		switch {
		case !ok:
			blockers = append(blockers, fmt.Sprintf("%s %s: unknown poison %q.", name, a.Hand, a.Poison))
		case b == nil:
			blockers = append(blockers, fmt.Sprintf("%s %s: no usable blade here.", name, a.Hand))
		case b.Item.Coated(now) && b.Item.CoatKind == poison.ID:
			out = append(out, poisonRow{blade: b, poison: poison, skip: true})
			lines = append(lines, fmt.Sprintf("  %-8s %-4s %-14s %-10s already coated (skipped)", name, a.Hand, b.Item.DisplayName(), poison.Name))
		case b.Item.Coated(now):
			blockers = append(blockers, fmt.Sprintf("%s %s: %s already carries another coating; clear it first (coat clear).", name, a.Hand, b.Item.DisplayName()))
		default:
			out = append(out, poisonRow{blade: b, poison: poison})
			need[poison.ID]++
			lines = append(lines, fmt.Sprintf("  %-8s %-4s %-14s %-10s 1 dose", name, a.Hand, b.Item.DisplayName(), poison.Name))
		}
	}
	var needed []string
	for _, p := range items.Poisons {
		if n := need[p.ID]; n > 0 {
			needed = append(needed, fmt.Sprintf("%s %d", p.Name, n))
			if have := m.doseCount(user.UserId, p.VialID); have < n {
				blockers = append(blockers, fmt.Sprintf("Short of %s: need %d, have %d.", p.Name, n, have))
			}
		}
	}
	if len(blockers) > 0 {
		lines = append(lines, "Blocked, nothing spent:")
		for _, b := range blockers {
			lines = append(lines, "  "+b)
		}
		return strings.Join(lines, "\n"), out, false
	}
	switch {
	case len(needed) == 0 && len(out) == 0:
		return "No assigned member is on the roster; nothing to apply.", nil, false
	case len(needed) == 0:
		lines = append(lines, "Every assigned blade already carries its poison; no application is needed.")
		return strings.Join(lines, "\n"), out, false
	}
	n := 0
	for _, r := range out {
		if !r.skip {
			n++
		}
	}
	lines = append(lines, "Needed: "+strings.Join(needed, ", ")+".", fmt.Sprintf("Ready to apply to %d blades.", n))
	return strings.Join(lines, "\n"), out, true
}

// poisonApply coats every assigned blade, or none of them.
func (m *CampingModule) poisonApply(user *users.UserRecord) string {
	text, rows, ok := m.poisonCheck(user)
	if !ok {
		return text
	}
	now := m.now()
	coated := 0
	for _, r := range rows {
		if r.skip {
			continue
		}
		if !m.spendDose(user.UserId, r.poison.VialID) {
			return fmt.Sprintf("A %s vial went missing partway; %d blades are coated. Check camp poison.", r.poison.Name, coated)
		}
		r.blade.Item.ClearCoat()
		r.blade.Item.Coat(r.poison.ID, items.CoatExpiry(now), items.CoatContacts, now)
		coated++
	}
	return fmt.Sprintf("You coat %d blades. Each lasts %d minutes or %d wounding blows.", coated, items.CoatMinutes, items.CoatContacts)
}
