package camping

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/beasts"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dolls"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/flasks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/wounds"
)

// Phase 23a rest tiers. A completed camp rest owes the company Rested, a
// completed inn stay Well Rested. Both are granted here, on the game loop
// (NewRound), never from a rest timer. Tier exclusivity is
// camping.Decide. Every rostered companion's grant is also kept durably
// until it would have expired: a companion mob's buffs die with the mob,
// so one absent at the grant, or respawned after a relog, restart, or
// copyover, is given the tier for the time left.

// --- native seams ---

func (m *CampingModule) companions(leaderUserID int) (map[int]*characters.Character, []int) {
	if m.companionsOf != nil {
		return m.companionsOf(leaderUserID)
	}
	live := map[int]*characters.Character{}
	var roster []int
	for _, ref := range survival.CurrentRoster(leaderUserID) {
		companionID, ok := company.CompanionIDFromMemberKey(ref.Key)
		if !ok || ref.Dead || ref.Away || ref.Needless {
			continue // a dead (Phase 25b), separated (33h3) or construct (38e) companion earns no rest
		}
		roster = append(roster, companionID)
		instanceID, ok := company.InstanceFor(leaderUserID, companionID)
		if !ok {
			continue
		}
		if mob := mobs.GetInstance(instanceID); mob != nil {
			live[companionID] = &mob.Character
		}
	}
	return live, roster
}

func (m *CampingModule) addBuff(c *characters.Character, buffID, rounds int) error {
	if m.grantBuff != nil {
		return m.grantBuff(c, buffID, rounds)
	}
	if rounds <= 0 { // Phase 43a: the buff's own length (a supply's fifteen minutes)
		return c.AddBuff(buffID, false)
	}
	return c.AddBuff(buffID, false, rounds)
}

func (m *CampingModule) dropBuff(c *characters.Character, buffID int) {
	if m.removeBuff != nil {
		m.removeBuff(c, buffID)
		return
	}
	c.RemoveBuff(buffID)
}

func (m *CampingModule) holdsBuff(c *characters.Character, buffID int) bool {
	if m.hasBuff != nil {
		return m.hasBuff(c, buffID)
	}
	// HasBuff also counts a removed or expired buff until the engine prunes
	// it, so only a buff with triggers left is held.
	return c.Buffs.TriggersLeft(buffID) > 0
}

func (m *CampingModule) roundLength() int {
	if m.roundSeconds != nil {
		return m.roundSeconds()
	}
	return int(configs.GetTimingConfig().RoundSeconds)
}

// --- tier helpers ---

func (s innSettings) tierBuff(tier camping.Tier) int {
	if tier == camping.TierWellRested {
		return s.WellRestedBuffId
	}
	return s.RestedBuffId
}

func (s innSettings) tierDuration(tier camping.Tier) time.Duration {
	if tier == camping.TierWellRested {
		return s.WellRestedDuration
	}
	return s.RestedDuration
}

// heldTier is the best rest tier a member holds.
func (m *CampingModule) heldTier(c *characters.Character, s innSettings) camping.Tier {
	switch {
	case m.holdsBuff(c, s.WellRestedBuffId):
		return camping.TierWellRested
	case m.holdsBuff(c, s.RestedBuffId):
		return camping.TierRested
	}
	return camping.TierNone
}

// applyTier gives one member a tier for rounds, removing any lower tier
// and never downgrading a higher one. It reports whether it granted.
func (m *CampingModule) applyTier(c *characters.Character, tier camping.Tier, rounds int, s innSettings) bool {
	grant, _ := camping.Decide(m.heldTier(c, s), tier)
	if !grant {
		return false
	}
	// Remove every lower tier held, not only the best one: a script or
	// admin grant can leave a member holding two.
	for lower := camping.TierRested; lower < tier; lower++ {
		if m.holdsBuff(c, s.tierBuff(lower)) {
			m.dropBuff(c, s.tierBuff(lower))
		}
	}
	if err := m.addBuff(c, s.tierBuff(tier), rounds); err != nil {
		mudlog.Warn("camping: grant rest tier", "tier", tier.String(), "error", err)
		return false
	}
	return true
}

// --- the grant pass ---

// onNewRound grants owed rest tiers and restores owed companion grants.
func (m *CampingModule) onNewRound(e events.Event) events.ListenerReturn {
	if _, ok := e.(events.NewRound); !ok {
		return events.Continue
	}
	m.grantPendingTiers()
	m.restoreOwedTiers()
	m.fireDueRaids()     // Phase 33f3
	m.clearRaiders()     // Phase 33f3
	m.grantCampRewards() // Phase 33f3
	m.resolveCampTheft() // Phase 40a4
	return events.Continue
}

// grantPendingTiers grants each pending tier to the online leader and
// every live companion, records the rest as owed to absent companions, and
// clears the pending marker. Well Rested wins when both are pending.
func (m *CampingModule) grantPendingTiers() {
	m.mu.Lock()
	pending := map[int]camping.Tier{}
	for leaderUserID, owed := range m.restedPending {
		if owed {
			pending[leaderUserID] = camping.TierRested
		}
	}
	for leaderUserID, owed := range m.wellRestedPending {
		if owed {
			pending[leaderUserID] = camping.TierWellRested
		}
	}
	settings := m.innSettings()
	m.mu.Unlock()
	leaders := make([]int, 0, len(pending))
	for leaderUserID := range pending {
		leaders = append(leaders, leaderUserID)
	}
	sort.Ints(leaders)

	for _, leaderUserID := range leaders {
		user := m.userByID(leaderUserID)
		if user == nil || user.Character == nil {
			continue // owed until the leader is back online
		}
		tier := pending[leaderUserID]
		now := m.clock().UTC()
		duration := settings.tierDuration(tier)
		rounds := camping.RoundsFor(duration, m.roundLength())
		// Phase 51: a camp rest's Rested buff skips the members on a duty.
		var onDuty map[string]bool
		duties := m.pendingDuties(leaderUserID, tier)
		if tier == camping.TierRested {
			onDuty = restedExcluded(duties)
		}
		// Buffs are granted outside m.mu: character state belongs to the
		// game loop, and nothing below calls back into camping.
		granted := false
		if !onDuty[string(survival.LeaderMemberKey)] {
			granted = m.applyTier(user.Character, tier, rounds, settings)
		}
		live, roster := m.companions(leaderUserID)
		for _, companionID := range sortedIDs(live) {
			if onDuty[string(survival.CompanionMemberKey(companionID))] {
				continue
			}
			if m.applyTier(live[companionID], tier, rounds, settings) {
				granted = true
			}
		}
		owed := map[int]camping.OwedGrant{}
		for _, companionID := range roster {
			if onDuty[string(survival.CompanionMemberKey(companionID))] {
				continue
			}
			owed[companionID] = camping.OwedGrant{BuffID: settings.tierBuff(tier), Tier: tier, ExpiresAtUTC: now.Add(duration)}
		}
		// A failed save retries next round, so only announce a saved grant,
		// and only one that gave anybody anything.
		saved, campRest := m.finishGrant(leaderUserID, tier, owed, now)
		if !saved {
			continue
		}
		if granted && tier == camping.TierWellRested {
			user.SendText("Your company feels well rested.")
		} else if granted {
			user.SendText("Your company is Rested: the road will feel a little lighter for a while.")
		}
		// Phase 30b: an inn stay knits every wound of the members present;
		// a camp rest only those it has a splint (a broken bone) or a
		// bandage (a cut or a puncture) for (owner, 2026-09-30).
		// Phase 40a3: a field surgeon's kit packed for the rest has a
		// healer with mana treat the worst lasting wound first, before
		// the bandages and splints.
		if campRest && tier == camping.TierRested && m.restKit(leaderUserID) {
			if lines, treated := m.fieldSurgery(leaderUserID); treated {
				for _, line := range lines {
					user.SendText(line)
				}
				user.SendText("The field surgeon's kit wears a little with the work.")
			}
		}
		spend := m.supply(leaderUserID)
		if tier == camping.TierWellRested {
			spend = nil
		}
		for _, line := range healRestWounds(user.Character, live, spend) {
			user.SendText(line)
		}
		// Phase 33h2: a night at the inn restores everyone with the leader
		// to their (now raised) wound limit and full mana. Phase 35b: so
		// does a finished camp rest, the only other way mana comes back
		// besides a draught. campRest is true only for the save that
		// cleared the rest's marker, so a rest refills once, even across
		// a crash or copyover.
		// Phase 43a: the rest's fortifying broth takes hold now that it is
		// done, before the vitals are restored, so the drinkers wake to the
		// raised limit (43a review: granted after, it filled nobody).
		if campRest {
			m.grantBroth(user, live)
		}
		if tier == camping.TierWellRested || campRest {
			restoreVitals(user.Character, live)
			// Phase 39e: the rest also closes a bonded beast's wounds and
			// brings its health back.
			for _, line := range restBeasts(user, live) {
				user.SendText(line)
			}
		}
		// Review fix: the leader's limit may have risen.
		events.AddToQueue(events.CharacterVitalsChanged{UserId: leaderUserID})
		// Phase 23b: a camp rest's end also sharpens the company for a
		// leader with auto-sharpen on. campRest is whether this save
		// cleared a camp rest's marker, read under the lock at the clear,
		// so a camp rest that finished after the snapshot still counts.
		// Once per camp rest; a pass is idempotent anyway, since sharp
		// blades cost nothing.
		// Phase 36a: the rest also lets the company's Scribe read any
		// unidentified gear (and a worn item reveals itself).
		if campRest {
			for _, line := range archetypes.CampIdentify(leaderUserID) {
				user.SendText(line)
			}
		}
		// Phase 39d: the rest also lets a Doll Master mend its dolls with the
		// leader's doll parts, the leader's own first, then companions' by
		// number.
		if campRest {
			for _, line := range mendRestDolls(user, live) {
				user.SendText(line)
			}
			// Phase 39g: and refills an Alchemist's flask satchel from the
			// leader's reagents.
			for _, line := range brewRestFlasks(user, live, m.restBrewers(user, duties)) {
				user.SendText(line)
			}
			// Phase 51: and the members on a work duty do their work.
			for _, line := range m.settleDuties(user, live, duties) {
				user.SendText(line)
			}
		}
		if campRest && m.autoSharpenOn(leaderUserID) {
			if text := m.sharpen(user, true); text != "" {
				user.SendText(text)
			}
		}
	}
}

// restKit reports whether the leader's camp rest packed a field surgeon's
// kit (locked when the rest began).
func (m *CampingModule) restKit(leaderUserID int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	camp, ok := m.camps[leaderUserID]
	return ok && camp.Rest != nil && camp.Rest.Kit && !camp.Rest.Broken
}

// fieldSurgery is the seam the kit's treatment goes through.
func (m *CampingModule) fieldSurgery(leaderUserID int) ([]string, bool) {
	if m.surgery != nil {
		return m.surgery(leaderUserID)
	}
	return company.FieldSurgery(leaderUserID, surgeonKitItemID)
}

// restoreVitals brings the leader and the live companions to their wound
// limit and full mana (Phase 33h2, an inn stay; Phase 35b, a camp rest).
// It never lowers anyone.
func restoreVitals(leader *characters.Character, live map[int]*characters.Character) {
	restore := func(c *characters.Character) {
		if c == nil || c.Health < 1 {
			return
		}
		c.Health = max(c.Health, c.HealthLimit())
		c.Mana = max(c.Mana, c.ManaMax.Value)
		c.RestClass() // Phase 38b: Lay on Hands comes back with rest
	}
	restore(leader)
	for _, id := range sortedIDs(live) {
		restore(live[id])
	}
}

// supply is the seam a camp rest spends the company's bandages and
// splints through.
func (m *CampingModule) supply(leaderUserID int) func(wounds.Item) bool {
	if m.spendSupply != nil {
		return func(item wounds.Item) bool { return m.spendSupply(leaderUserID, item) }
	}
	return func(item wounds.Item) bool { return company.SpendSupply(leaderUserID, item) }
}

// restItem is what a camp rest needs to close a wound: a splint for a
// broken bone, a bandage for anything else.
func restItem(w wounds.Wound) wounds.Item {
	if w.Kind == wounds.Fracture {
		return wounds.Splint
	}
	return wounds.Bandage
}

// healRestWounds closes the wounds of the leader and the live companions
// at a rest's end (Phase 30b), with a line for each lasting wound it
// closes. With spend nil (an inn stay) every wound closes. Otherwise (a
// camp rest) each lasting wound closes only for the item it needs (a
// splint for a broken bone, a bandage for a cut or a puncture) that spend
// can use up; the rest stay open, and a line says what would help.
func healRestWounds(leader *characters.Character, live map[int]*characters.Character, spend func(wounds.Item) bool) []string {
	var lines []string
	open := 0
	outOf := map[wounds.Item]bool{}
	heal := func(c *characters.Character, owner string) {
		var kept []wounds.Wound
		for _, w := range wounds.Lasting(c.Wounds) {
			used := ""
			if spend != nil {
				item := restItem(w)
				if outOf[item] || !spend(item) {
					outOf[item] = true
					kept = append(kept, w)
					open++
					continue
				}
				used = "; a " + string(item) + " used"
			}
			verb := " has knit."
			if w.Place == "ribs" {
				verb = " have knit."
			}
			lines = append(lines, util.CapitalizeFirst(wounds.Possessive(owner, w))+verb+" (wound healed"+used+")")
		}
		c.Wounds = kept
	}
	heal(leader, "your")
	for _, id := range sortedIDs(live) {
		heal(live[id], `<ansi fg="username">`+live[id].Name+`</ansi>'s`)
	}
	if open > 0 {
		var missing []string
		if outOf[wounds.Splint] {
			missing = append(missing, "splints")
		}
		if outOf[wounds.Bandage] {
			missing = append(missing, "bandages")
		}
		what := strings.Join(missing, " or ")
		noun, them := "wounds stay", "them"
		if open == 1 {
			noun, them = "wound stays", "it"
		}
		lines = append(lines, fmt.Sprintf("With no %s left, %d %s open. %s, an inn, or a physician will close %s (<ansi fg=\"command\">help wounds</ansi>).",
			what, open, noun, util.CapitalizeFirst(strings.Join(missing, ", ")), them))
	}
	return lines
}

func sortedIDs(live map[int]*characters.Character) []int {
	ids := make([]int, 0, len(live))
	for id := range live {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// finishGrant clears the pending markers the granted tier covers (Well
// Rested covers both; Rested only its own, so a Well Rested an inn timer
// marked after the snapshot is still granted next round), removes a
// finished inn stay, and merges the new owed entries, in one save. A
// failed save restores everything, so the next round grants again (which
// only refreshes). It also reports whether it cleared a camp rest's
// marker (Phase 23b auto-sharpen).
func (m *CampingModule) finishGrant(leaderUserID int, granted camping.Tier, owed map[int]camping.OwedGrant, now time.Time) (bool, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	wellPending := granted == camping.TierWellRested && m.wellRestedPending[leaderUserID]
	restedPending := m.restedPending[leaderUserID]
	if !wellPending && !restedPending {
		return false, false
	}
	stay, hadStay := m.stays[leaderUserID]
	innApplied, hadInnApplied := m.innRecoveryApplied[leaderUserID]
	oldOwed, hadOwed := m.owed[leaderUserID]
	snapshot := make(map[int]camping.OwedGrant, len(oldOwed))
	for id, g := range oldOwed {
		snapshot[id] = g
	}

	if wellPending {
		delete(m.wellRestedPending, leaderUserID)
	}
	delete(m.restedPending, leaderUserID)
	duties, hadDuties := m.restedDuties[leaderUserID]
	if restedPending {
		delete(m.restedDuties, leaderUserID)
	}
	if wellPending && hadStay && !stay.Resting() {
		delete(m.stays, leaderUserID)
		delete(m.innRecoveryApplied, leaderUserID)
	}
	if len(owed) > 0 {
		merged := make(map[int]camping.OwedGrant, len(snapshot)+len(owed))
		for id, g := range snapshot {
			merged[id] = g
		}
		for id, g := range owed {
			old, had := snapshot[id]
			merged[id] = camping.MergeOwed(old, had, g, now)
		}
		m.owed[leaderUserID] = merged
	}
	if err := m.saveLocked(); err != nil {
		if wellPending {
			m.wellRestedPending[leaderUserID] = true
		}
		if restedPending {
			m.restedPending[leaderUserID] = true
		}
		if hadDuties {
			m.restedDuties[leaderUserID] = duties
		}
		if hadStay {
			m.stays[leaderUserID] = stay
		}
		if hadInnApplied {
			m.innRecoveryApplied[leaderUserID] = innApplied
		}
		if hadOwed {
			m.owed[leaderUserID] = snapshot
		} else {
			delete(m.owed, leaderUserID)
		}
		mudlog.Warn("camping: finish rest tier grant", "leader", leaderUserID, "error", err)
		return false, false
	}
	return true, restedPending
}

// --- owed grants ---

// restoreOwedTiers gives each companion with a durable grant, whose live
// mob holds less than that tier (it was absent at the grant, or has
// respawned since), the tier for the time remaining; it never extends the
// grant. Entries are dropped once expired.
func (m *CampingModule) restoreOwedTiers() {
	m.mu.Lock()
	if len(m.owed) == 0 {
		m.mu.Unlock()
		return
	}
	snapshot := cloneOwed(m.owed)
	settings := m.innSettings()
	m.mu.Unlock()
	leaders := make([]int, 0, len(snapshot))
	for leaderUserID := range snapshot {
		leaders = append(leaders, leaderUserID)
	}
	sort.Ints(leaders)

	now := m.clock().UTC()
	for _, leaderUserID := range leaders {
		entries := snapshot[leaderUserID]
		done := map[int]camping.OwedGrant{}
		var live map[int]*characters.Character
		for _, companionID := range sortedOwedIDs(entries) {
			grant := entries[companionID]
			rounds := camping.RemainingRounds(grant.ExpiresAtUTC, now, m.roundLength())
			if grant.Expired(now) || rounds <= 0 {
				done[companionID] = grant
				continue
			}
			if live == nil {
				live, _ = m.companions(leaderUserID)
			}
			c, ok := live[companionID]
			if !ok || m.heldTier(c, settings) >= grant.Tier {
				continue
			}
			m.applyTier(c, grant.Tier, rounds, settings)
		}
		if len(done) > 0 {
			m.clearOwed(leaderUserID, done)
		}
	}
}

func sortedOwedIDs(entries map[int]camping.OwedGrant) []int {
	ids := make([]int, 0, len(entries))
	for id := range entries {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// clearOwed removes expired entries still unchanged since the snapshot and
// saves; a failed save puts them back so the next round retries.
func (m *CampingModule) clearOwed(leaderUserID int, done map[int]camping.OwedGrant) {
	m.mu.Lock()
	defer m.mu.Unlock()
	current := m.owed[leaderUserID]
	removed := map[int]camping.OwedGrant{}
	for companionID, grant := range done {
		if got, ok := current[companionID]; ok && got == grant {
			removed[companionID] = got
			delete(current, companionID)
		}
	}
	if len(removed) == 0 {
		return
	}
	if len(current) == 0 {
		delete(m.owed, leaderUserID)
	}
	if err := m.saveLocked(); err != nil {
		if m.owed[leaderUserID] == nil {
			m.owed[leaderUserID] = map[int]camping.OwedGrant{}
		}
		for companionID, grant := range removed {
			m.owed[leaderUserID][companionID] = grant
		}
		mudlog.Warn("camping: clear owed rest tier", "leader", leaderUserID, "error", err)
	}
}

// --- normalization ---

// onPlayerSpawn leaves a returning player with at most one rest tier: a
// saved Rested under Well Rested is removed.
func (m *CampingModule) onPlayerSpawn(e events.Event) events.ListenerReturn {
	spawn, ok := e.(events.PlayerSpawn)
	if !ok {
		return events.Continue
	}
	user := m.userByID(spawn.UserId)
	if user == nil || user.Character == nil {
		return events.Continue
	}
	m.mu.Lock()
	settings := m.innSettings()
	m.mu.Unlock()
	if m.holdsBuff(user.Character, settings.WellRestedBuffId) && m.holdsBuff(user.Character, settings.RestedBuffId) {
		m.dropBuff(user.Character, settings.RestedBuffId)
	}
	return events.Continue
}

// mendRestDolls mends the company's dolls at the end of a camp rest (Phase
// 39d) with the leader's doll parts, the leader's dolls first and then each
// live companion's by number.
func mendRestDolls(user *users.UserRecord, live map[int]*characters.Character) []string {
	type master struct {
		name string
		char *characters.Character
	}
	var masters []master
	if dolls.IsMaster(user.Character) {
		masters = append(masters, master{"You", user.Character})
	}
	ids := make([]int, 0, len(live))
	for id := range live {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		if c := live[id]; c != nil && dolls.IsMaster(c) {
			masters = append(masters, master{c.Name, c})
		}
	}
	if len(masters) == 0 {
		return nil
	}
	chars := make([]*characters.Character, len(masters))
	for i, m := range masters {
		chars[i] = m.char
	}
	needs := false
	for _, c := range chars {
		needs = needs || dolls.NeedsMending(c)
	}
	if !needs {
		return nil
	}
	if dolls.Parts(user.Character) == 0 {
		return []string{"Your dolls stay worn: you have no doll parts."}
	}
	taken, used := dolls.MendCompany(user.Character, chars)
	for _, itm := range taken {
		events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: itm, Gained: false})
	}
	var lines []string
	for i, n := range used {
		if n > 0 {
			verb := "mend"
			if masters[i].name != "You" {
				verb = "mends"
			}
			lines = append(lines, fmt.Sprintf("%s %s the dolls through the night with %d doll part(s).", masters[i].name, verb, n))
		}
	}
	return lines
}

// brewRestFlasks refills the company's Alchemists' satchels at the end of a
// camp rest (Phase 39g) from the leader's reagents, one a flask, the leader's
// own first and then each live companion's by number.
func brewRestFlasks(user *users.UserRecord, live map[int]*characters.Character, brewers map[*characters.Character]bool) []string {
	type alchemist struct {
		name string
		char *characters.Character
	}
	var as []alchemist
	if flasks.IsAlchemist(user.Character) {
		as = append(as, alchemist{"You", user.Character})
	}
	ids := make([]int, 0, len(live))
	for id := range live {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		if c := live[id]; c != nil && flasks.IsAlchemist(c) {
			as = append(as, alchemist{c.Name, c})
		}
	}
	if len(as) == 0 {
		return nil
	}
	// Phase 51: the Alchemists on the brew duty are served first, so a
	// short supply of reagents goes to them.
	sort.SliceStable(as, func(i, j int) bool { return brewers[as[i].char] && !brewers[as[j].char] })
	chars := make([]*characters.Character, len(as))
	for i, a := range as {
		chars[i] = a.char
	}
	if !flasks.NeedsBrewing(chars...) {
		return nil
	}
	if flasks.Reagents(user.Character) == 0 {
		return []string{"Your flask satchels stay low: you have no reagents."}
	}
	taken, brewed := flasks.Brew(user.Character, chars)
	for _, itm := range taken {
		events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: itm, Gained: false})
	}
	var lines []string
	for i, b := range brewed {
		if b.Brewed > 0 {
			verb := "brew"
			if as[i].name != "You" {
				verb = "brews"
			}
			lines = append(lines, fmt.Sprintf("%s %s through the night: %d flask(s) from %d reagent(s).", as[i].name, verb, b.Brewed, b.Brewed))
		}
	}
	return lines
}

// restBeasts brings the company's bonded beasts back to health at the end of
// a rest (Phase 39e): the leader's own first, then each live companion's by
// number. Nothing is spent; a rest is the cure.
func restBeasts(user *users.UserRecord, live map[int]*characters.Character) []string {
	var lines []string
	if beasts.Recover(user.Character) {
		lines = append(lines, fmt.Sprintf("%s is rested and whole again.", user.Character.Beast.Name))
	}
	ids := make([]int, 0, len(live))
	for id := range live {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		if c := live[id]; c != nil && beasts.Recover(c) {
			lines = append(lines, fmt.Sprintf("%s's %s is rested and whole again.", c.Name, c.Beast.Name))
		}
	}
	return lines
}
