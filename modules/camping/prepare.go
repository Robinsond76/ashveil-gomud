package camping

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/cookbook"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 43a camp supplies (help campsupplies). Fortifying broth, a warming
// draught, a cooling salve and watch incense are ordinary items the company
// carries, prepared at an established camp with `camp prepare`:
//
//   - A warming draught or cooling salve starts at once, as a buff on its
//     member that lasts fifteen real minutes. The exposure module cuts the
//     exposure that member takes on by a quarter, in that one direction.
//   - Broth and incense are queued on the camp (Camp.Prepared) and spent
//     when the next rest begins. Incense is locked on the rest and adds ten
//     points to the watch's chance to spot raiders; broth is locked per
//     member and fortifies them when the rest is done, so a spoiled rest
//     gives none.
//   - A member holds one personal benefit (broth, draught or salve) at a
//     time, and nothing is replaced silently.
//
// The company's stock is counted before m.mu (it calls the company
// module), as camp gear is.

// brothBuffs and the two draught buffs are the personal benefit buffs.
var personalBuffs = []int{camping.WarmingBuffID, camping.CoolingBuffID}

const prepareUsage = "Usage: camp supplies | camp prepare status | camp prepare broth|warming|cooling [member|all] | camp prepare incense | camp prepare remedy [member|all] | camp prepare clear [member|all|incense]"

// prepTarget is one company member a supply can be prepared for: the leader
// or a live companion in the leader's room.
type prepTarget struct {
	key  string
	name string
	char *characters.Character
}

func (m *CampingModule) prepTargets(user *users.UserRecord) (targets []prepTarget, fighting bool) {
	targets = []prepTarget{{key: string(survival.LeaderMemberKey), name: user.Character.Name, char: user.Character}}
	fighting = user.Character.Aggro != nil || (m.inBattle != nil && m.inBattle(user.UserId))
	live, _ := m.companions(user.UserId)
	for _, id := range sortedIDs(live) {
		c := live[id]
		if c.Aggro != nil {
			fighting = true
		}
		if c.RoomId != user.Character.RoomId {
			continue // a companion elsewhere is not at the camp
		}
		targets = append(targets, prepTarget{key: string(survival.CompanionMemberKey(id)), name: c.Name, char: c})
	}
	return targets, fighting
}

// personalBuff reports which personal benefit a member holds, if any: the
// buff id, and a label for the status line.
func (m *CampingModule) personalBuff(c *characters.Character) (int, bool) {
	for _, id := range personalBuffs {
		if m.holdsBuff(c, id) {
			return id, true
		}
	}
	for _, tier := range camping.BrothBuffs {
		if m.holdsBuff(c, tier.BuffID) {
			return tier.BuffID, true
		}
	}
	return 0, false
}

func personalBuffName(buffID int) string {
	switch {
	case buffID == camping.WarmingBuffID:
		return "a warming draught"
	case buffID == camping.CoolingBuffID:
		return "a cooling salve"
	case camping.IsBrothBuff(buffID):
		return "fortifying broth"
	}
	return "a supply"
}

// resolveTargets picks the members a prepare names: no word is the leader,
// "all" or "company" everyone at the camp, anything else a member by name
// (a prefix is enough).
func resolveTargets(word string, all []prepTarget) ([]prepTarget, string) {
	word = strings.ToLower(strings.TrimSpace(word))
	switch word {
	case "", "me", "self":
		return all[:1], ""
	case "all", "company":
		return all, ""
	}
	// A companion's #id tells apart two members of one name.
	if id, err := strconv.Atoi(strings.TrimPrefix(word, "#")); err == nil {
		key := string(survival.CompanionMemberKey(id))
		for _, t := range all[1:] {
			if t.key == key {
				return []prepTarget{t}, ""
			}
		}
		return nil, fmt.Sprintf("There is nobody numbered %q at your camp.", word)
	}
	var exact, found []prepTarget
	for _, t := range all {
		if strings.EqualFold(t.name, word) {
			exact = append(exact, t)
		} else if strings.HasPrefix(strings.ToLower(t.name), word) {
			found = append(found, t)
		}
	}
	if len(exact) == 1 {
		return exact, ""
	}
	if len(exact) > 1 {
		found = exact
	}
	switch len(found) {
	case 1:
		return found, ""
	case 0:
		return nil, fmt.Sprintf("There is nobody called %q at your camp.", word)
	}
	if len(exact) > 1 {
		return nil, fmt.Sprintf("More than one member is called %q; use a number: %s.", word, numbered(found))
	}
	return nil, fmt.Sprintf("More than one member matches %q; use the whole name.", word)
}

// numbered lists members with the #id that picks each one.
func numbered(list []prepTarget) string {
	out := make([]string, len(list))
	for i, t := range list {
		out[i] = t.name
		if id, ok := survival.CompanionIDFromMemberKey(survival.MemberKey(t.key)); ok {
			out[i] += " (#" + strconv.Itoa(id) + ")"
		}
	}
	return strings.Join(out, ", ")
}

func (m *CampingModule) spendOne(leaderUserID, itemID int) bool {
	spend := m.spendItem
	if spend == nil {
		spend = company.SpendCompanyItem
	}
	return spend(leaderUserID, itemID)
}

// suppliesHeld is one line per supply the company carries or has queued.
func (m *CampingModule) suppliesText(user *users.UserRecord, prepared *camping.Prepared) string {
	lines := []string{"Camp supplies (help campsupplies):"}
	any := false
	for _, s := range camping.Supplies {
		n := m.gearCount(user.UserId, s.ItemID)
		if n == 0 {
			continue
		}
		any = true
		lines = append(lines, fmt.Sprintf("  %s x%d (camp prepare %s)", s.Name, n, s.Key))
	}
	if !any {
		lines = append(lines, "  None. Provisioners and the road post sell broth, draughts, salves and incense.")
	}
	if !prepared.Empty() {
		lines = append(lines, "Queued for the next rest: "+m.queuedText(user, prepared)+".")
	}
	lines = append(lines, m.remediesText(user))
	return strings.Join(lines, "\n")
}

func (m *CampingModule) queuedText(user *users.UserRecord, prepared *camping.Prepared) string {
	return m.queuedTextLocked(user.UserId, prepared)
}

// queuedTextLocked names what is queued, members by their roster names.
func (m *CampingModule) queuedTextLocked(leaderUserID int, prepared *camping.Prepared) string {
	names := map[string]string{}
	for _, member := range m.survival.CompanyNeeds(leaderUserID) {
		names[string(member.Key)] = member.Name
	}
	var parts []string
	for _, key := range prepared.Broth {
		name := names[key]
		if name == "" {
			name = "a member"
		}
		parts = append(parts, "fortifying broth for "+name)
	}
	if prepared.Incense {
		parts = append(parts, "watch incense")
	}
	return strings.Join(parts, ", ")
}

// prepareStatus lists each member's benefit, with the time it has left, and
// what is queued.
func (m *CampingModule) prepareStatus(user *users.UserRecord, prepared *camping.Prepared) string {
	targets, _ := m.prepTargets(user)
	lines := []string{"Prepared supplies:"}
	for _, t := range targets {
		line := fmt.Sprintf("  %s: ", t.name)
		if id, ok := m.personalBuff(t.char); ok {
			left := time.Duration(m.buffRoundsLeft(t.char, id)*m.roundLength()) * time.Second
			line += fmt.Sprintf("%s (%s left)", personalBuffName(id), left.Round(time.Minute))
			if camping.IsBrothBuff(id) {
				for _, tier := range camping.BrothBuffs {
					if tier.BuffID == id {
						line += fmt.Sprintf(", health limit +%d", tier.Bonus)
					}
				}
			}
		} else if prepared.HasBroth(t.key) {
			line += "fortifying broth queued for the next rest"
		} else {
			line += "nothing"
		}
		lines = append(lines, line)
	}
	if prepared != nil && prepared.Incense {
		lines = append(lines, "  Camp: watch incense queued for the next rest.")
	}
	for _, n := range m.survival.CompanyNeeds(user.UserId) {
		for _, a := range survival.ActiveAilments(n.Needs) {
			lines = append(lines, fmt.Sprintf("  %s has a %s (%s left): camp prepare remedy.", n.Name, strings.ToLower(a.Name), survival.BattlesLeft(survival.AilmentBattles(n.Needs, a.Kind))))
		}
	}
	return strings.Join(lines, "\n")
}

// preparedOf is the leader's queue, read under the lock.
func (m *CampingModule) preparedOf(leaderUserID int) *camping.Prepared {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.camps[leaderUserID].Prepared
}

// suppliesCommand is `camp supplies`.
func (m *CampingModule) suppliesCommand(user *users.UserRecord) string {
	return m.suppliesText(user, m.preparedOf(user.UserId))
}

// prepareCommand is `camp prepare ...`.
func (m *CampingModule) prepareCommand(user *users.UserRecord, room *rooms.Room, args []string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	m.mu.Lock()
	camp, has := m.camps[user.UserId]
	m.mu.Unlock()
	if !has {
		return "You have no camp. Use \"camp\" to make one, then prepare supplies."
	}
	if len(args) == 0 || args[0] == "status" {
		return m.prepareStatus(user, camp.Prepared) + "\n" + prepareUsage
	}
	if camp.RoomID != room.RoomId {
		return "Your camp is not here."
	}
	if camp.Rest != nil && camp.Rest.State == camping.Resting {
		return "You can't prepare supplies while the company rests. Wait for the rest to end."
	}
	targets, fighting := m.prepTargets(user)
	if fighting {
		return "You can't prepare supplies in the middle of a fight."
	}
	if args[0] == "clear" {
		return m.clearPrepared(user, camp, targets, args[1:])
	}
	if args[0] == "remedy" || args[0] == "remedies" {
		if len(args) > 1 && args[1] == "with" { // Phase 56: try a mix of herbs
			return m.prepareRemedyMix(user, targets, args[2:])
		}
		if len(args) > 2 {
			return prepareUsage
		}
		rest := ""
		if len(args) > 1 {
			rest = args[1]
		}
		chosen, refusal := resolveTargets(rest, targets)
		if refusal != "" {
			return refusal
		}
		return m.prepareRemedy(user, chosen)
	}
	supply, ok := camping.FindSupply(args[0])
	if !ok {
		return "Prepare what? " + prepareUsage
	}
	rest := ""
	if len(args) > 1 {
		rest = args[1]
	}
	if len(args) > 2 {
		return prepareUsage
	}
	if supply.Key == "incense" {
		return m.queueIncense(user, room, camp, supply)
	}
	chosen, refusal := resolveTargets(rest, targets)
	if refusal != "" {
		return refusal
	}
	return m.prepareFor(user, camp, supply, chosen)
}

// prepareFor applies or queues a personal supply for members. Every member
// is checked, and the stock counted, before anything is spent; a refusal
// consumes nothing.
func (m *CampingModule) prepareFor(user *users.UserRecord, camp camping.Camp, supply camping.Supply, chosen []prepTarget) string {
	var lines []string
	var apply []prepTarget
	for _, t := range chosen {
		if id, ok := m.personalBuff(t.char); ok {
			lines = append(lines, fmt.Sprintf("%s already has %s; \"camp prepare clear\" to drop it first.", t.name, personalBuffName(id)))
			continue
		}
		if camp.Prepared.HasBroth(t.key) {
			lines = append(lines, fmt.Sprintf("%s already has fortifying broth queued for the next rest.", t.name))
			continue
		}
		apply = append(apply, t)
	}
	if len(apply) == 0 {
		return strings.Join(lines, "\n")
	}
	queued := 0
	if supply.Key == "broth" && camp.Prepared != nil {
		queued = len(camp.Prepared.Broth) // already counted against stock
	}
	if have := m.gearCount(user.UserId, supply.ItemID); have < len(apply)+queued {
		noun := supply.Name
		if len(apply)+queued > 1 {
			return fmt.Sprintf("You need %d %s for that, and carry %d. Nothing was used.", len(apply)+queued, noun, have)
		}
		return fmt.Sprintf("You carry no %s. Nothing was used.", noun)
	}
	switch supply.Key {
	case "broth":
		return strings.Join(append(lines, m.queueBroth(user, camp, apply)), "\n")
	default:
		return strings.Join(append(lines, m.startDraught(user, supply, apply)...), "\n")
	}
}

// queueBroth queues broth for members, saved. The broth is spent when the
// rest begins.
func (m *CampingModule) queueBroth(user *users.UserRecord, camp camping.Camp, apply []prepTarget) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.camps[user.UserId]
	if !ok || current.RoomID != camp.RoomID {
		return "Your camp is gone."
	}
	updated := current
	var names []string
	for _, t := range apply {
		updated.Prepared = updated.Prepared.WithBroth(t.key)
		names = append(names, t.name)
	}
	m.camps[user.UserId] = updated
	if err := m.saveLocked(); err != nil {
		m.camps[user.UserId] = current
		return err.Error()
	}
	return fmt.Sprintf("Fortifying broth is set by for %s. It is spent when the company next rests, and takes hold when the rest is done.", strings.Join(names, ", "))
}

// startDraught spends a dose for each member and gives them the buff.
func (m *CampingModule) startDraught(user *users.UserRecord, supply camping.Supply, apply []prepTarget) []string {
	buffID := camping.WarmingBuffID
	verb := "drinks the warming draught"
	if supply.Key == "cooling" {
		buffID = camping.CoolingBuffID
		verb = "rubs on the cooling salve"
	}
	var lines []string
	for _, t := range apply {
		if !m.spendOne(user.UserId, supply.ItemID) {
			lines = append(lines, fmt.Sprintf("%s: the %s ran out.", t.name, supply.Name))
			break
		}
		if err := m.addBuff(t.char, buffID, 0); err != nil {
			mudlog.Warn("camping: prepare supply", "supply", supply.Key, "member", t.name, "error", err)
			lines = append(lines, fmt.Sprintf("%s couldn't use the %s.", t.name, supply.Name))
			continue
		}
		who := t.name + " " + verb
		if t.key == string(survival.LeaderMemberKey) {
			who = "You " + strings.Replace(verb, "drinks", "drink", 1)
			who = strings.Replace(who, "rubs", "rub", 1)
		}
		lines = append(lines, fmt.Sprintf("%s. It lasts %d minutes.", who, camping.SupplyMinutes))
	}
	return lines
}

// queueIncense queues a bundle of incense. It needs a posted watch, which it
// does not invent, and is spent when the rest begins.
func (m *CampingModule) queueIncense(user *users.UserRecord, room *rooms.Room, camp camping.Camp, supply camping.Supply) string {
	if camp.Prepared != nil && camp.Prepared.Incense {
		return "Watch incense is already set for the next rest."
	}
	if m.gearCount(user.UserId, supply.ItemID) < 1 {
		return "You carry no watch incense. Nothing was used."
	}
	if !m.hasWatch(user.UserId, room.RoomId) {
		return "Watch incense needs a Camp Watch: a company member with the watch skill, here at the camp, to be the one who looks out. Nothing was used."
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.camps[user.UserId]
	if !ok || current.RoomID != camp.RoomID {
		return "Your camp is gone."
	}
	updated := current
	updated.Prepared = updated.Prepared.WithIncense()
	m.camps[user.UserId] = updated
	if err := m.saveLocked(); err != nil {
		m.camps[user.UserId] = current
		return err.Error()
	}
	return "A bundle of watchsage is set by. It is lit when the company next rests, and sharpens the watch's eye for raiders by ten points."
}

func (m *CampingModule) hasWatch(leaderUserID, roomID int) bool {
	if m.specialist == nil {
		return false
	}
	_, ok := m.specialist(leaderUserID, archetypes.UtilityWatch, roomID)
	return ok
}

// clearPrepared drops what is queued, and an active personal benefit,
// refunding nothing.
func (m *CampingModule) clearPrepared(user *users.UserRecord, camp camping.Camp, targets []prepTarget, args []string) string {
	word := "all"
	if len(args) > 0 {
		word = strings.ToLower(args[0])
	}
	if len(args) > 1 {
		return prepareUsage
	}
	var chosen []prepTarget
	incense := false
	switch word {
	case "incense":
		incense = true
	case "all", "company":
		chosen, incense = targets, true
	default:
		var refusal string
		if chosen, refusal = resolveTargets(word, targets); refusal != "" {
			return refusal
		}
	}
	var lines []string
	for _, t := range chosen {
		if id, ok := m.personalBuff(t.char); ok {
			m.dropBuff(t.char, id)
			lines = append(lines, fmt.Sprintf("%s's %s is dropped.", t.name, strings.TrimPrefix(personalBuffName(id), "a ")))
		}
		if camp.Prepared.HasBroth(t.key) {
			lines = append(lines, fmt.Sprintf("%s's queued broth is set aside.", t.name))
		}
	}
	if incense && camp.Prepared != nil && camp.Prepared.Incense {
		lines = append(lines, "The watch incense is set aside.")
	}
	m.mu.Lock()
	current, ok := m.camps[user.UserId]
	if ok {
		updated := current
		for _, t := range chosen {
			updated.Prepared = updated.Prepared.Cleared(t.key)
		}
		if incense && updated.Prepared != nil && updated.Prepared.Incense {
			rest := *updated.Prepared
			rest.Incense = false
			if rest.Empty() {
				updated.Prepared = nil
			} else {
				updated.Prepared = &rest
			}
		}
		m.camps[user.UserId] = updated
		if err := m.saveLocked(); err != nil {
			m.camps[user.UserId] = current
			m.mu.Unlock()
			return err.Error()
		}
	}
	m.mu.Unlock()
	if len(lines) == 0 {
		return "Nothing to clear."
	}
	lines = append(lines, "Nothing is refunded.")
	return strings.Join(lines, "\n")
}

// restPrep is the queue funded for a rest: the members whose broth the
// company can supply, and whether incense can burn.
type restPrep struct {
	Broth   []string
	Incense bool
	// notes are lines for the rest-start report about what was dropped.
	notes []string
	// drop are the queued broth keys the rest takes off the queue (funded
	// or gone from the pack), dropIncense likewise for the incense.
	drop        []string
	dropIncense bool
	// present are the member keys at the camp when the rest begins
	// (Phase 51): only they take their rest duties.
	present map[string]bool
	// song (camp music) is the song planned for the rest.
	song camping.Song
}

// clearQueue is the queue with what the rest settled removed.
func (p restPrep) clearQueue(prepared *camping.Prepared) *camping.Prepared {
	if prepared == nil {
		return nil
	}
	out := camping.Prepared{Incense: prepared.Incense && !p.dropIncense}
	for _, key := range prepared.Broth {
		dropped := false
		for _, d := range p.drop {
			dropped = dropped || d == key
		}
		if !dropped {
			out.Broth = append(out.Broth, key)
		}
	}
	if out.Empty() {
		return nil
	}
	return &out
}

// fundPrepared counts the stock against the queue before the rest starts
// (it calls the company module, so it runs before m.mu). Broth the company
// no longer carries is dropped; incense is kept queued when no watch is
// posted, and burned only with one.
func (m *CampingModule) fundPrepared(user *users.UserRecord, room *rooms.Room) restPrep {
	prepared := m.preparedOf(user.UserId)
	var out restPrep
	if prepared.Empty() {
		return out
	}
	targets, _ := m.prepTargets(user)
	name := func(key string) string {
		for _, t := range targets {
			if t.key == key {
				return t.name
			}
		}
		return "a member"
	}
	stock := m.gearCount(user.UserId, camping.BrothItemID)
	for _, key := range prepared.Broth {
		present := false
		for _, t := range targets {
			present = present || t.key == key
		}
		switch {
		case !present:
			out.notes = append(out.notes, fmt.Sprintf("%s is not at the camp, so the broth is saved.", name(key)))
		case stock < 1:
			out.notes = append(out.notes, fmt.Sprintf("There is no broth left for %s.", name(key)))
			out.drop = append(out.drop, key)
		default:
			stock--
			out.Broth = append(out.Broth, key)
			out.drop = append(out.drop, key)
		}
	}
	if prepared.Incense {
		switch {
		case m.gearCount(user.UserId, camping.IncenseItemID) < 1:
			out.notes = append(out.notes, "The incense is gone from your pack.")
			out.dropIncense = true
		case !m.hasWatch(user.UserId, room.RoomId):
			out.notes = append(out.notes, "No watch is posted, so the incense stays unlit for another rest.")
		default:
			out.Incense = true
			out.dropIncense = true
		}
	}
	return out
}

// settlePrepared spends what the rest locked and clears it from the queue.
// It runs outside m.mu, after the rest is saved.
func (m *CampingModule) settlePrepared(leaderUserID int, funded restPrep) {
	for range funded.Broth {
		m.spendOne(leaderUserID, camping.BrothItemID)
	}
	if funded.Incense {
		m.spendOne(leaderUserID, camping.IncenseItemID)
	}
}

// restPrepText is the rest start's report of what was lit and set by.
func restPrepText(funded restPrep, names func(key string) string) string {
	var parts []string
	if len(funded.Broth) > 0 {
		var who []string
		for _, key := range funded.Broth {
			who = append(who, names(key))
		}
		parts = append(parts, "broth simmering for "+strings.Join(who, ", "))
	}
	if funded.Incense {
		parts = append(parts, "watchsage burning for the watch")
	}
	var text string
	if len(parts) > 0 {
		text = "Supplies: " + strings.Join(parts, "; ") + "."
	}
	for _, note := range funded.notes {
		if text != "" {
			text += "\n"
		}
		text += note
	}
	return text
}

// grantBroth gives the rest's broth drinkers their buff when the rest is
// done. It runs once, with the save that clears the rest's marker, on the
// game loop. Members who are dead, separated or not spawned miss it.
func (m *CampingModule) grantBroth(user *users.UserRecord, live map[int]*characters.Character) {
	m.mu.Lock()
	camp, ok := m.camps[user.UserId]
	var members []string
	if ok && camp.Rest != nil && !camp.Rest.Broken {
		members = append(members, camp.Rest.Broth...)
	}
	m.mu.Unlock()
	if len(members) == 0 {
		return
	}
	var fed []string
	for _, key := range members {
		var c *characters.Character
		if key == string(survival.LeaderMemberKey) {
			c = user.Character
		} else if id, isCompanion := company.CompanionIDFromMemberKey(survival.MemberKey(key)); isCompanion {
			c = live[id]
		}
		if c == nil || c.Health < 1 {
			continue
		}
		if _, held := m.personalBuff(c); held {
			continue // never replaced silently; the broth is lost with the rest
		}
		tier := camping.BrothTierFor(c.HealthMax.Value)
		if err := m.addBuff(c, tier.BuffID, 0); err != nil {
			mudlog.Warn("camping: grant broth", "member", c.Name, "error", err)
			continue
		}
		fed = append(fed, fmt.Sprintf("%s (+%d)", c.Name, tier.Bonus))
	}
	if len(fed) > 0 {
		user.SendText(fmt.Sprintf("The broth takes hold: health limit up for %d minutes for %s.", camping.SupplyMinutes, strings.Join(fed, ", ")))
	}
}

// prepName names a member by key for the rest report.
func (m *CampingModule) prepName(user *users.UserRecord) func(key string) string {
	targets, _ := m.prepTargets(user)
	return func(key string) string {
		for _, t := range targets {
			if t.key == key {
				return t.name
			}
		}
		return "a member"
	}
}

// supplyLabels is one short label per camp supply the company carries, for
// the web Camp tab. It counts through the company module, so it runs before
// m.mu.
func (m *CampingModule) supplyLabels(leaderUserID int) []string {
	var out []string
	for _, s := range camping.Supplies {
		if n := m.gearCount(leaderUserID, s.ItemID); n > 0 {
			out = append(out, fmt.Sprintf("%s x%d", s.Name, n))
		}
	}
	return out
}

// remedyNeed is one member's ailment, to be cured with its herbs.
type remedyNeed struct {
	target  prepTarget
	ailment survival.AilmentSpec
}

// remedyHerbsText names a remedy's herbs: "2 wild thyme".
func remedyHerbsText(a survival.AilmentSpec) string {
	var parts []string
	for _, ing := range a.Remedy {
		parts = append(parts, fmt.Sprintf("%d %s", ing.Count, ing.Name))
	}
	return strings.Join(parts, " and ")
}

// remediesText lists what each ailment's remedy takes, and how much of it
// the company carries (for `camp supplies`).
func (m *CampingModule) remediesText(user *users.UserRecord) string {
	leaderUserID := user.UserId
	lines := []string{"Remedies you know (camp prepare remedy), made from gathered herbs; to find another, camp prepare remedy with [herb] [herb]:"}
	for _, a := range survival.Ailments() {
		if !remedyKnown(user, a) {
			continue
		}
		have := make([]string, 0, len(a.Remedy))
		for _, ing := range a.Remedy {
			have = append(have, fmt.Sprintf("%s %d/%d", ing.Name, m.gearCount(leaderUserID, ing.ItemID), ing.Count))
		}
		lines = append(lines, fmt.Sprintf("  %s for %s: %s (you carry %s)", a.RemedyName, strings.ToLower(a.Name), remedyHerbsText(a), strings.Join(have, ", ")))
	}
	return strings.Join(lines, "\n")
}

// prepareRemedy cures the ailments of the chosen members, making each
// remedy from herbs the company carries. Every member is checked and the
// herbs counted before anything is spent; a refusal consumes nothing. A
// remedy is used up as it is made, never an item, so it can't be sold.
func (m *CampingModule) prepareRemedy(user *users.UserRecord, chosen []prepTarget) string {
	text, _ := m.cureWith(user, chosen, nil)
	return text
}

// remedyKnown reports whether the leader can make an ailment's remedy.
func remedyKnown(user *users.UserRecord, a survival.AilmentSpec) bool {
	return cookbook.KnowsRemedy(user.Character, a.Kind, a.Common)
}

// remedyRecipe is an ailment's remedy as a herb mix, one entry per herb.
func remedyRecipe(a survival.AilmentSpec) []int {
	var out []int
	for _, ing := range a.Remedy {
		for i := 0; i < ing.Count; i++ {
			out = append(out, ing.ItemID)
		}
	}
	return out
}

// prepareRemedyMix is `camp prepare remedy with [herb]...` (Phase 56): it
// tries exactly that mix of herbs. A mix that is the remedy for an ailment
// someone in the company has cures it, and teaches the remedy the first
// time; any other mix is a useless brew that uses up its herbs, so a guess
// always costs.
func (m *CampingModule) prepareRemedyMix(user *users.UserRecord, targets []prepTarget, words []string) string {
	herb := func(id int) bool {
		for _, a := range survival.Ailments() {
			for _, ing := range a.Remedy {
				if ing.ItemID == id {
					return true
				}
			}
		}
		return false
	}
	// The herbs the company carries (the seam `camp prepare remedy` spends).
	var stockOfHerbs []cookbook.Stack
	seen := map[int]bool{}
	for _, a := range survival.Ailments() {
		for _, ing := range a.Remedy {
			if !seen[ing.ItemID] {
				seen[ing.ItemID] = true
				stockOfHerbs = append(stockOfHerbs, cookbook.Stack{ItemID: ing.ItemID, Count: m.gearCount(user.UserId, ing.ItemID)})
			}
		}
	}
	mix, problem := cookbook.ResolveWith(words, stockOfHerbs, herb)
	if problem != "" {
		return problem
	}
	var matched *survival.AilmentSpec
	for _, a := range survival.Ailments() {
		a := a
		if _, ok := cookbook.Match(mix, []cookbook.Recipe{{Inputs: remedyRecipe(a)}}); ok {
			matched = &a
			break
		}
	}
	// 56 review: one mix is one dose, for the first member it helps, so
	// the herbs spent are exactly the herbs named.
	var patient []prepTarget
	if matched != nil {
		needs := map[string]survival.Needs{}
		for _, n := range m.survival.CompanyNeeds(user.UserId) {
			needs[string(n.Key)] = n.Needs
		}
		for _, t := range targets {
			for _, a := range survival.ActiveAilments(needs[t.key]) {
				if a.Kind == matched.Kind && patient == nil {
					patient = []prepTarget{t}
				}
			}
		}
	}
	if matched == nil || patient == nil {
		// A guess that helps nobody: the herbs are spent.
		for _, id := range mix {
			if !m.spendOne(user.UserId, id) {
				return "The herbs were not there when you reached for them."
			}
		}
		return "You steep " + cookbook.Describe(mix) + ", but the brew does nothing for anyone here. The herbs are used up."
	}
	fresh := !cookbook.KnowsRemedy(user.Character, matched.Kind, matched.Common)
	only := *matched
	text, cured := m.cureWith(user, patient, &only)
	if fresh && cured > 0 {
		cookbook.LearnRemedy(user.Character, only.Kind)
		text += fmt.Sprintf("\nYou have worked out a new remedy: %s (%s). It is in your recipe book (recipes).", only.RemedyName, remedyHerbsText(only))
	}
	return text
}

// cureWith cures the ailments of the chosen members. only (a mix being
// tried) limits it to one ailment, whether or not it is known; otherwise
// only the remedies the leader knows are made. It returns how many
// ailments broke.
func (m *CampingModule) cureWith(user *users.UserRecord, chosen []prepTarget, only *survival.AilmentSpec) (string, int) {
	needs := map[string]survival.Needs{}
	for _, n := range m.survival.CompanyNeeds(user.UserId) {
		needs[string(n.Key)] = n.Needs
	}
	var lines []string
	var work []remedyNeed
	for _, t := range chosen {
		active := survival.ActiveAilments(needs[t.key])
		if len(active) == 0 {
			lines = append(lines, fmt.Sprintf("%s is not ill.", t.name))
			continue
		}
		for _, a := range active {
			switch {
			case only != nil && a.Kind != only.Kind:
			case only == nil && !remedyKnown(user, a):
				lines = append(lines, fmt.Sprintf("%s has a %s, and you know no remedy for it. Try a mix of herbs: camp prepare remedy with [herb] [herb] (help recipes).", t.name, strings.ToLower(a.Name)))
			default:
				work = append(work, remedyNeed{target: t, ailment: a})
			}
		}
	}
	if len(work) == 0 {
		return strings.Join(lines, "\n"), 0
	}
	curing, ok := m.survival.(curingSurvival)
	if !ok {
		return "Remedies are not available right now.", 0
	}
	want := map[int]int{}
	names := map[int]string{}
	var order []int
	for _, w := range work {
		for _, ing := range w.ailment.Remedy {
			if _, seen := want[ing.ItemID]; !seen {
				order = append(order, ing.ItemID)
				names[ing.ItemID] = ing.Name
			}
			want[ing.ItemID] += ing.Count
		}
	}
	var short []string
	for _, id := range order {
		if have := m.gearCount(user.UserId, id); have < want[id] {
			short = append(short, fmt.Sprintf("%d %s (you carry %d)", want[id], names[id], have))
		}
	}
	if len(short) > 0 {
		return strings.Join(append(lines, "You need "+strings.Join(short, ", ")+" for that. Nothing was used. \"camp supplies\" lists the herbs."), "\n"), 0
	}
	for _, id := range order {
		for i := 0; i < want[id]; i++ {
			if !m.spendOne(user.UserId, id) {
				return strings.Join(append(lines, "The herbs were not there when you reached for them."), "\n"), 0
			}
		}
	}
	broke := 0
	for _, w := range work {
		cured, err := curing.CureAilment(user.UserId, survival.MemberKey(w.target.key), w.ailment.Kind)
		if err != nil {
			mudlog.Warn("camping: cure ailment", "leader", user.UserId, "member", w.target.key, "error", err)
			lines = append(lines, fmt.Sprintf("%s could not be treated: %s", w.target.name, err))
			continue
		}
		if cured {
			broke++
			lines = append(lines, fmt.Sprintf("You make %s for %s, and the %s breaks.", w.ailment.RemedyName, w.target.name, strings.ToLower(w.ailment.Name)))
		}
	}
	return strings.Join(lines, "\n"), broke
}
