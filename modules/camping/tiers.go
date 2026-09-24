package camping

import (
	"sort"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/survival"
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
		if !ok {
			continue
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
		// Buffs are granted outside m.mu: character state belongs to the
		// game loop, and nothing below calls back into camping.
		granted := m.applyTier(user.Character, tier, rounds, settings)
		live, roster := m.companions(leaderUserID)
		for _, companionID := range sortedIDs(live) {
			if m.applyTier(live[companionID], tier, rounds, settings) {
				granted = true
			}
		}
		owed := map[int]camping.OwedGrant{}
		for _, companionID := range roster {
			owed[companionID] = camping.OwedGrant{BuffID: settings.tierBuff(tier), Tier: tier, ExpiresAtUTC: now.Add(duration)}
		}
		// A failed save retries next round, so only announce a saved grant,
		// and only one that gave anybody anything.
		if !m.finishGrant(leaderUserID, tier, owed, now) || !granted {
			continue
		}
		if tier == camping.TierWellRested {
			user.SendText("Your company feels well rested.")
		} else {
			user.SendText("Your company is Rested: the road will feel a little lighter for a while.")
		}
	}
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
// only refreshes).
func (m *CampingModule) finishGrant(leaderUserID int, granted camping.Tier, owed map[int]camping.OwedGrant, now time.Time) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	wellPending := granted == camping.TierWellRested && m.wellRestedPending[leaderUserID]
	restedPending := m.restedPending[leaderUserID]
	if !wellPending && !restedPending {
		return false
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
		return false
	}
	return true
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
