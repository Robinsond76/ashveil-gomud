package company

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/prompt"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spellpower"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/wounds"
)

// Phase 30b: `heal` lists the company's wounds; `heal wounds`, after a
// fight, has whoever and whatever can help tend the company: the clerics'
// tend and heal (their mana, resolved at once), then splints and bandages
// from the cargo and packs, then a physician for gold where there is one.
// Everything runs on the game loop and never advances the clock.

// Treatment items (_datafiles/world/default/items/other-0).
const (
	bandageItemID = 36
	splintItemID  = 37
)

const healUsage = `Usage: <ansi fg="command">heal</ansi> (who is hurt), <ansi fg="command">heal wounds</ansi> (tend the company). See <ansi fg="command">help wounds</ansi>.`

const healInFight = `You can't tend wounds in the middle of a fight.`

// physician is a room where wounds are healed for gold, configured like a
// recruiter: Physicians: [{RoomId, Name, Description, PricePerWound}].
type physician struct {
	RoomID        int
	Name          string
	Description   string
	PricePerWound int
}

func parsePhysicians(raw any) map[int]physician {
	out := map[int]physician{}
	list, ok := raw.([]any)
	if !ok {
		return out
	}
	for _, entry := range list {
		fields := lowerKeys(entry)
		if fields == nil {
			mudlog.Warn("company: physician entry is not a map; skipped")
			continue
		}
		roomID, ok := configInt(fields["roomid"])
		if !ok || roomID <= 0 {
			mudlog.Warn("company: physician without a room; skipped", "roomid", fields["roomid"])
			continue
		}
		price, ok := configInt(fields["priceperwound"])
		if !ok || price < 0 {
			mudlog.Warn("company: physician with a bad price; skipped", "roomid", roomID)
			continue
		}
		name, _ := fields["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			name = "the physician"
		}
		desc, _ := fields["description"].(string)
		out[roomID] = physician{RoomID: roomID, Name: name, Description: strings.TrimSpace(desc), PricePerWound: price}
	}
	return out
}

func (m *CompanyModule) physicians() map[int]physician {
	if m.physiciansForTest != nil {
		return m.physiciansForTest
	}
	if m.plug != nil {
		return parsePhysicians(m.plug.Config.Get("Physicians"))
	}
	return nil
}

// physicianIn is the physician of room (matched by its template room, as
// recruiters are), if any.
func (m *CompanyModule) physicianIn(room *rooms.Room) (physician, bool) {
	if room == nil {
		return physician{}, false
	}
	p, ok := m.physicians()[rooms.GetOriginalRoom(room.RoomId)]
	return p, ok
}

// woundMember is a living member present: the leader, or a companion out
// and walking with them.
type woundMember struct {
	key         string
	name        string // as a line names them
	char        *characters.Character
	companionID int // 0 for the leader
	knows       func(spellID string) bool
}

func (w woundMember) leader() bool { return w.companionID == 0 }

// possessive is the member's name as a wound's owner: "your", "Tamsin
// Reed's".
func (w woundMember) possessive() string {
	if w.leader() {
		return "your"
	}
	return w.name + "'s"
}

// woundMembers lists the leader and the living companions with them.
func (m *CompanyModule) woundMembers(user *users.UserRecord) []woundMember {
	out := []woundMember{{
		key: "leader", name: "you", char: user.Character,
		knows: func(id string) bool {
			return user.Character.GetSkillLevel(`cast`) > 0 && user.Character.HasSpell(id)
		},
	}}
	record, _ := m.registry.Get(user.UserId)
	for _, id := range m.CompanionsWithLeader(user.UserId) {
		instanceID, ok := m.instance(user.UserId, id)
		if !ok {
			continue
		}
		mob := mobs.GetInstance(instanceID)
		if mob == nil || mob.Character.Health < 1 {
			continue
		}
		name := mob.Character.Name
		arch := ""
		for _, c := range record.Companions {
			if c.ID == id {
				name = nameOf(c, name)
				arch = c.Archetype
			}
		}
		known := map[string]bool{}
		for _, s := range archetypes.CompanionSpells(arch, mob.Character.Level) {
			known[s] = true
		}
		out = append(out, woundMember{
			key: "companion:" + strconv.Itoa(id), name: name, char: &mob.Character, companionID: id,
			knows: func(s string) bool { return known[s] },
		})
	}
	return out
}

// companyFighting reports whether the leader or anyone with them is in a
// fight.
func companyFighting(user *users.UserRecord, members []woundMember) bool {
	if usercommands.InBattle(user) {
		return true
	}
	for _, w := range members {
		if w.char.Aggro != nil {
			return true
		}
	}
	return false
}

func (m *CompanyModule) healCommand(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(rest)) {
	case "":
		user.SendText(m.woundsView(user, room))
	case "wounds", "wound":
		if text := m.healWounds(user, room); text != "" {
			user.SendText(text)
		}
	default:
		user.SendText(healUsage)
	}
	return true, nil
}

// healthText is "13/16 (wound limit 13)" or "12/14".
func healthText(c *characters.Character) string {
	out := fmt.Sprintf("%d/%d", c.Health, c.HealthMax.Value)
	if c.HealthLimit() < c.HealthMax.Value {
		out += fmt.Sprintf(" (wound limit %d)", c.HealthLimit())
	}
	return out
}

// woundsView is `heal`: each member present who is hurt, and what would
// help. It changes nothing.
func (m *CompanyModule) woundsView(user *users.UserRecord, room *rooms.Room) string {
	lines := []string{}
	for _, w := range m.woundMembers(user) {
		lasting := wounds.Lasting(w.char.Wounds)
		if len(lasting) == 0 && w.char.Health >= w.char.HealthLimit() {
			continue
		}
		name := util.CapitalizeFirst(w.name)
		if !w.leader() {
			name = `<ansi fg="username">` + w.name + `</ansi>`
		}
		line := fmt.Sprintf("%s: %s", name, healthText(w.char))
		if len(lasting) > 0 {
			var what []string
			for _, wd := range lasting {
				what = append(what, fmt.Sprintf("%s (holds back %d)", wounds.Describe(wd), wd.Points))
			}
			line += "; " + strings.Join(what, ", ")
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return "No one in your company is hurt."
	}
	out := "Hurt:\n  " + strings.Join(lines, "\n  ")
	hint := "  <ansi fg=\"command\">heal wounds</ansi> tends them; an inn stay heals wounds fully, a camp rest one per splint or bandage."
	if p, ok := m.physicianIn(room); ok {
		hint += fmt.Sprintf(" %s here closes wounds for %d gold each.", util.CapitalizeFirst(p.Name), p.PricePerWound)
	}
	return out + "\n" + hint
}

// spellRules reads the spells' costs from their files, falling back to
// the defaults.
func spellRules() wounds.Rules {
	r := wounds.DefaultRules
	if sp := spells.GetSpell("tend"); sp != nil && sp.Cost > 0 {
		r.TendCost = sp.Cost
	}
	if sp := spells.GetSpell("heal"); sp != nil && sp.Cost > 0 {
		r.HealCost = sp.Cost
	}
	// Phase 35b: the heal's dice come from its power block, the one source
	// heal.js reads too.
	if q, sd := healPower().DiceQS(); q > 0 {
		r.HealDice = [2]int{q, sd}
	}
	return r
}

// healPower is Minor Heal's size (heal.yaml's power block), or its shipped
// numbers when the spell file has none.
func healPower() spellpower.Power {
	if sp := spells.GetSpell("heal"); sp != nil && sp.Power != nil {
		return *sp.Power
	}
	return spellpower.Power{Base: 8, Dice: "2d4", LevelDiv: 6}
}

// healBonus is a healer's flat heal bonus: Minor Heal's base plus its level
// and Mysticism bonuses.
func healBonus(c *characters.Character) int {
	return healPower().Flat(c.Level, c.Stats.Mysticism.ValueAdj)
}

// supply is one treatment item the company can reach: the cargo, a
// companion's own pack, or the leader's.
type supply struct {
	source larderSource
	owner  int // companion id for fromOwnPack
	item   items.Item
}

// supplies lists the reachable items of itemID, in the order they are
// used: cargo, then the patient's own pack, then the leader's pack, then
// the other companions' packs.
func (m *CompanyModule) supplies(user *users.UserRecord, members []woundMember, itemID int, patient int) []supply {
	var out []supply
	for _, stack := range encumbrance.CargoContents(user.UserId) {
		if stack.ItemId == itemID {
			for i := 0; i < stack.Count; i++ {
				out = append(out, supply{source: fromCargo})
			}
		}
	}
	pack := func(id int) {
		carried, ok := m.carriedBy(user.UserId, id)
		if !ok {
			return
		}
		for _, itm := range carried {
			if itm.ItemId == itemID {
				out = append(out, supply{source: fromOwnPack, owner: id, item: itm})
			}
		}
	}
	if patient > 0 {
		pack(patient)
	}
	for _, itm := range user.Character.Items {
		if !user.Character.CompanyCargo && itm.ItemId == itemID {
			out = append(out, supply{source: fromLeaderPack, item: itm})
		}
	}
	for _, w := range members {
		if !w.leader() && w.companionID != patient {
			pack(w.companionID)
		}
	}
	return out
}

// SpendSupply uses one bandage or splint the company can reach, for a
// camp rest (company.SupplyProvider).
func (m *CompanyModule) SpendSupply(leaderUserID int, item wounds.Item) bool {
	user := users.GetByUserId(leaderUserID)
	if user == nil || user.Character == nil || m.persistenceAvailable() != nil {
		return false
	}
	itemID := bandageItemID
	if item == wounds.Splint {
		itemID = splintItemID
	}
	return m.spendItem(user, m.woundMembers(user), itemID, 0)
}

var _ domain.SupplyProvider = (*CompanyModule)(nil)

// useSupply spends one item. It reports whether it was spent.
func (m *CompanyModule) useSupply(user *users.UserRecord, s supply, itemID int) bool {
	switch s.source {
	case fromCargo:
		if err := encumbrance.ConsumeCargoUse(user.UserId, itemID); err != nil {
			mudlog.Error("company: heal wounds", "error", err)
			return false
		}
		return true
	case fromOwnPack:
		return m.useCompanionItem(user.UserId, s.owner, s.item)
	}
	for i := range user.Character.Items {
		if user.Character.Items[i].Equals(s.item) {
			current := user.Character.Items[i]
			if user.Character.UseItem(current) < 1 {
				events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: current, Gained: false})
			}
			return true
		}
	}
	return false
}

// spendItem spends one of itemID for patient, from the first source that
// has one.
func (m *CompanyModule) spendItem(user *users.UserRecord, members []woundMember, itemID, patient int) bool {
	for _, s := range m.supplies(user, members, itemID, patient) {
		if m.useSupply(user, s, itemID) {
			return true
		}
	}
	return false
}

func patientsOf(members []woundMember) []wounds.Patient {
	out := make([]wounds.Patient, 0, len(members))
	for _, w := range members {
		out = append(out, wounds.Patient{Key: w.key, Health: w.char.Health, Max: w.char.HealthMax.Value, Wounds: w.char.Wounds})
	}
	return out
}

func byKey(members []woundMember) map[string]woundMember {
	out := map[string]woundMember{}
	for _, w := range members {
		out[w.key] = w
	}
	return out
}

// healWounds is `heal wounds`. The physician's question is asked through
// the prompt; its answer comes back here with the prompt no longer new.
func (m *CompanyModule) healWounds(user *users.UserRecord, room *rooms.Room) string {
	members := m.woundMembers(user)
	if companyFighting(user, members) {
		user.ClearPrompt()
		return healInFight
	}
	if user.Character.Health < 1 {
		user.ClearPrompt()
		return "You're in no state to tend anyone."
	}

	cmdPrompt, isNew := user.StartPrompt("heal", "wounds")
	if !isNew {
		return m.physicianAnswer(user, room, members, cmdPrompt)
	}

	lines := m.treat(user, members)
	// Review fix: the leader's health and mana may have changed.
	events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})

	// The physician, for what is still wounded.
	if p, ok := m.physicianIn(room); ok {
		count := 0
		for _, w := range members {
			count += len(wounds.Lasting(w.char.Wounds))
		}
		if count > 0 {
			price := count * p.PricePerWound
			intro := util.CapitalizeFirst(p.Name)
			if p.Description != "" {
				intro += ", " + p.Description + ","
			}
			lines = append(lines, fmt.Sprintf(`%s looks the company's wounds over. "%d gold, and they'll all be closed by morning."`, intro, price))
			if user.Character.Gold < price {
				// Review fix: no question the leader can't afford to answer.
				user.ClearPrompt()
				lines = append(lines, fmt.Sprintf("You have only %d gold.", user.Character.Gold))
				lines = append(lines, m.stillHurt(members)...)
				return strings.Join(lines, "\n")
			}
			cmdPrompt.Store("price", price)
			cmdPrompt.Ask(fmt.Sprintf("Pay %d gold?", price), []string{"yes", "no"}, "no")
			user.SendText(strings.Join(lines, "\n"))
			return ""
		}
	}
	user.ClearPrompt()
	lines = append(lines, m.stillHurt(members)...)
	return strings.Join(lines, "\n")
}

// treat runs the clerics, then the items, and returns the lines.
func (m *CompanyModule) treat(user *users.UserRecord, members []woundMember) []string {
	who := byKey(members)
	rules := spellRules()
	lines := []string{}

	// 1–3: the clerics.
	var healers []wounds.Healer
	knowers := 0
	for _, w := range members {
		tend, heal := w.knows("tend"), w.knows("heal")
		if tend || heal {
			knowers++
		}
		if (tend || heal) && w.char.Mana > 0 {
			healers = append(healers, wounds.Healer{Key: w.key, Mana: w.char.Mana, Tend: tend, Heal: heal, HealBonus: healBonus(w.char), HealPct: w.char.HealingBonusPct()})
		}
	}
	anyHurt := false
	for _, p := range patientsOf(members) {
		anyHurt = anyHurt || p.Hurt()
	}
	if !anyHurt {
		return []string{"No one in your company is hurt."}
	}
	switch {
	case knowers == 0:
		lines = append(lines, "There's no one in the company who can heal. You open the packs instead.")
	case len(healers) == 0:
		lines = append(lines, "Your healers have no mana left. You open the packs instead.")
	}
	res := wounds.Plan(patientsOf(members), healers, wounds.Stock{}, rules, util.Rand)
	m.applyPlan(who, res)
	lines = append(lines, stepLines(who, res.Steps)...)
	for _, h := range res.Healers {
		w := who[h.Key]
		cheapest := rules.TendCost
		if h.Heal && (!h.Tend || rules.HealCost < cheapest) {
			cheapest = rules.HealCost
		}
		if h.Mana < cheapest && h.Mana < w.char.ManaMax.Value && usedHealer(res.Steps, h.Key) {
			if w.leader() {
				lines = append(lines, fmt.Sprintf("You sway, spent. (mana %d of %d)", h.Mana, w.char.ManaMax.Value))
			} else {
				lines = append(lines, fmt.Sprintf(`<ansi fg="username">%s</ansi> sways, pale and spent. (mana %d of %d)`, w.name, h.Mana, w.char.ManaMax.Value))
			}
		}
	}

	// 4: splints and bandages. Each is spent before it is applied.
	stock := wounds.Stock{
		Bandages: len(m.supplies(user, members, bandageItemID, 0)),
		Splints:  len(m.supplies(user, members, splintItemID, 0)),
	}
	if stock.Bandages+stock.Splints == 0 {
		return lines
	}
	items := wounds.Plan(patientsOf(members), nil, stock, rules, util.Rand)
	var done []wounds.Step
	spent := wounds.Stock{}
	for _, step := range items.Steps {
		itemID := bandageItemID
		if step.Kind == wounds.StepSplint {
			itemID = splintItemID
		}
		if !m.spendItem(user, members, itemID, who[step.Patient].companionID) {
			mudlog.Warn("company: heal wounds item not spent", "leader", user.UserId, "item", itemID)
			break
		}
		if itemID == splintItemID {
			spent.Splints++
		} else {
			spent.Bandages++
		}
		done = append(done, step)
	}
	if len(done) < len(items.Steps) {
		// Something could not be spent: treat with what was.
		items = wounds.Plan(patientsOf(members), nil, spent, rules, util.Rand)
		done = items.Steps
	}
	m.applyPlan(who, items)
	left := wounds.Stock{Bandages: stock.Bandages - spent.Bandages, Splints: stock.Splints - spent.Splints}
	lines = append(lines, itemLines(who, done, left)...)
	return lines
}

func usedHealer(steps []wounds.Step, key string) bool {
	for _, s := range steps {
		if s.Healer == key {
			return true
		}
	}
	return false
}

// applyPlan puts a plan's outcome on the live characters.
func (m *CompanyModule) applyPlan(who map[string]woundMember, res wounds.Result) {
	for _, p := range res.Patients {
		w, ok := who[p.Key]
		if !ok {
			continue
		}
		w.char.Wounds = p.Wounds
		if p.Health > w.char.Health {
			w.char.Health = w.char.CapHealing(w.char.Health, p.Health)
		}
	}
	for _, h := range res.Healers {
		if w, ok := who[h.Key]; ok {
			w.char.Mana = h.Mana
		}
	}
}

func limitNote(s wounds.Step) string {
	return fmt.Sprintf("limit %d of %d", s.Limit, s.Max)
}

// stepLines narrates the clerics' steps.
func stepLines(who map[string]woundMember, steps []wounds.Step) []string {
	var lines []string
	first := map[string]bool{}
	for _, s := range steps {
		h, p := who[s.Healer], who[s.Patient]
		healer := `<ansi fg="username">` + h.name + `</ansi>`
		if h.leader() {
			healer = "You"
		}
		patient := `<ansi fg="username">` + p.name + `</ansi>`
		if p.leader() {
			patient = "you"
		}
		isFirst := !first[s.Healer]
		first[s.Healer] = true
		if isFirst && h.key != p.key {
			if h.leader() {
				lines = append(lines, fmt.Sprintf("You kneel beside %s first, the worst hurt.", patient))
			} else {
				lines = append(lines, fmt.Sprintf("%s kneels beside %s first, the worst hurt.", healer, patient))
			}
		}
		owner := p.possessive()
		if h.key == p.key {
			owner = "your own"
			if !h.leader() {
				owner = h.char.CombatPronouns().Possessive + " own"
			}
		} else if !p.leader() {
			owner = `<ansi fg="username">` + p.name + `</ansi>'s`
		}
		verb := "tends"
		if h.leader() {
			verb = "tend"
		}
		switch s.Kind {
		case wounds.StepTend:
			lines = append(lines, fmt.Sprintf("%s %s %s, and it draws closed. (wound treated, %s)", healer, verb, wounds.Possessive(owner, s.Wound), limitNote(s)))
		case wounds.StepHeal:
			hand := "lays glowing hands on"
			if h.leader() {
				hand = "lay glowing hands on"
			}
			target := patient
			if h.key == p.key {
				target = "yourself"
				if !h.leader() {
					target = h.char.CombatPronouns().Object + "self"
				}
			}
			note := ""
			if s.Health >= s.Limit && s.Limit < s.Max {
				note = ", wound limit " + strconv.Itoa(s.Limit) + " of " + strconv.Itoa(s.Max)
			}
			lines = append(lines, fmt.Sprintf("%s %s %s. (%d healed%s)", healer, hand, target, s.Healed, note))
		}
	}
	return lines
}

// itemLines narrates the splints and bandages, with what is left.
func itemLines(who map[string]woundMember, steps []wounds.Step, left wounds.Stock) []string {
	var lines []string
	usedB, usedS := 0, 0
	for _, s := range steps {
		p := who[s.Patient]
		owner := p.possessive()
		if !p.leader() {
			owner = `<ansi fg="username">` + p.name + `</ansi>'s`
		}
		switch s.Kind {
		case wounds.StepSplint:
			usedS++
			lines = append(lines, fmt.Sprintf("You set %s against a splint and bind it with linen. (wound treated, %s; %s left)", wounds.Possessive(owner, s.Wound), limitNote(s), count(left.Splints+countAfter(steps, s, wounds.StepSplint), "splint")))
		case wounds.StepBandage:
			usedB++
			lines = append(lines, fmt.Sprintf("You bandage %s. (wound treated, %s; %s left)", wounds.Possessive(owner, s.Wound), limitNote(s), count(left.Bandages+countAfter(steps, s, wounds.StepBandage, wounds.StepBandageHeal), "bandage")))
		case wounds.StepBandageHeal:
			usedB++
			whom := `<ansi fg="username">` + p.name + `</ansi>'s hurts`
			if p.leader() {
				whom = "your hurts"
			}
			lines = append(lines, fmt.Sprintf("You wrap %s. (%d healed; %s left)", whom, s.Healed, count(left.Bandages+countAfter(steps, s, wounds.StepBandage, wounds.StepBandageHeal), "bandage")))
		}
	}
	return lines
}

// countAfter is how many steps of kinds come after s: they were already
// spent, so they count toward what was left at s.
func countAfter(steps []wounds.Step, s wounds.Step, kinds ...wounds.StepKind) int {
	n, seen := 0, false
	for _, t := range steps {
		if seen {
			for _, k := range kinds {
				if t.Kind == k {
					n++
				}
			}
		}
		if t == s {
			seen = true
		}
	}
	return n
}

func count(n int, what string) string {
	if n == 1 {
		return "1 " + what
	}
	return strconv.Itoa(n) + " " + what + "s"
}

// stillHurt is the closing report: who is still hurt, and what would help.
func (m *CompanyModule) stillHurt(members []woundMember) []string {
	var hurt []string
	wounded := false
	for _, w := range members {
		if w.char.Health >= w.char.HealthLimit() && len(wounds.Lasting(w.char.Wounds)) == 0 {
			continue
		}
		name := `<ansi fg="username">` + w.name + `</ansi>`
		if w.leader() {
			name = "you"
		}
		text := fmt.Sprintf("%s %d/%d", name, w.char.Health, w.char.HealthMax.Value)
		if len(wounds.Lasting(w.char.Wounds)) > 0 {
			wounded = true
			text += fmt.Sprintf(" (wounded, limit %d)", w.char.HealthLimit())
		}
		hurt = append(hurt, text)
	}
	if len(hurt) == 0 {
		return nil
	}
	out := []string{"    Still hurt: " + strings.Join(hurt, ", ") + "."}
	if wounded {
		out = append(out, "    An inn will do the rest, or a camp rest with a splint or bandage for each wound.")
	}
	return out
}

// physicianAnswer takes the answer to the physician's question: on yes,
// the price is checked again, the gold taken, and every lasting wound
// closed; the user is saved at once, as a paid recruit is.
func (m *CompanyModule) physicianAnswer(user *users.UserRecord, room *rooms.Room, members []woundMember, cmdPrompt *prompt.Prompt) string {
	if len(cmdPrompt.Questions) == 0 {
		user.ClearPrompt()
		return ""
	}
	q := cmdPrompt.Questions[len(cmdPrompt.Questions)-1]
	if !q.Done {
		return "" // still waiting for a yes or a no
	}
	user.ClearPrompt()
	if q.Response != "yes" {
		return "You thank the physician and keep your gold."
	}
	p, ok := m.physicianIn(room)
	if !ok {
		return "There's no physician here."
	}
	count := 0
	for _, w := range members {
		count += len(wounds.Lasting(w.char.Wounds))
	}
	if count == 0 {
		return "There's nothing left for the physician to do."
	}
	price := count * p.PricePerWound
	if stored, _ := cmdPrompt.Recall("price"); stored != price {
		return "The company's wounds have changed since the price was named. Ask again: <ansi fg=\"command\">heal wounds</ansi>."
	}
	if user.Character.Gold < price {
		return fmt.Sprintf("You don't have %d gold.", price)
	}

	user.Character.Gold -= price
	for _, w := range members {
		w.char.Wounds = nil
		if !w.leader() {
			m.refreshSnapshot(user.UserId, w.companionID)
		}
	}
	if err := m.save(); err != nil {
		mudlog.Error("company: physician", "leader", user.UserId, "error", err)
	}
	if m.saveUser != nil {
		if err := m.saveUser(user); err != nil {
			mudlog.Error("company: physician: save user", "leader", user.UserId, "error", err)
		}
	}
	events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})
	noun := "wounds"
	if count == 1 {
		noun = "wound"
	}
	lines := []string{fmt.Sprintf("You pay %d gold. %s sets and stitches each wound properly, and packs it in a poultice that reeks of comfrey. (%d %s healed)", price, util.CapitalizeFirst(p.Name), count, noun)}
	return strings.Join(append(lines, m.stillHurt(members)...), "\n")
}
