package company

import (
	"fmt"
	"slices"
	"strings"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// chemistryWorld is what Phase 24 needs from the running game, behind a
// seam so unit tests need no world.
type chemistryWorld interface {
	// LeaderOnline reports whether the leader is signed in.
	LeaderOnline(leaderUserID int) bool
	// LeaderPresence is a signed-in, living leader's room.
	LeaderPresence(leaderUserID int) (roomID int, ok bool)
	// InstancePresence is a living mob's room.
	InstancePresence(instanceID int) (roomID int, ok bool)
	Tell(leaderUserID int, text string)
}

type nativeChemistryWorld struct{}

func (nativeChemistryWorld) LeaderOnline(leaderUserID int) bool {
	user := users.GetByUserId(leaderUserID)
	return user != nil && user.Character != nil
}

func (nativeChemistryWorld) LeaderPresence(leaderUserID int) (int, bool) {
	user := users.GetByUserId(leaderUserID)
	if user == nil || user.Character == nil || user.Character.Health < 1 {
		return 0, false
	}
	return user.Character.RoomId, true
}

func (nativeChemistryWorld) InstancePresence(instanceID int) (int, bool) {
	mob := mobs.GetInstance(instanceID)
	if mob == nil || mob.Character.Health < 1 {
		return 0, false
	}
	return mob.Character.RoomId, true
}

func (nativeChemistryWorld) Tell(leaderUserID int, text string) {
	if user := users.GetByUserId(leaderUserID); user != nil {
		user.SendText(text)
	}
}

func (m *CompanyModule) chemistryWorld() chemistryWorld {
	if m.chem != nil {
		return m.chem
	}
	return nativeChemistryWorld{}
}

// parseChemistryConfig reads the Phase 24 knobs. Values out of range, or a
// set whose thresholds don't strictly increase or whose bonuses decrease,
// fall back to the defaults; ok is false for such a set.
func parseChemistryConfig(get func(string) any) (rules domain.ChemistryRules, ok bool) {
	rules = domain.DefaultChemistryRules()
	read := func(key string, into *int, lo, hi int) {
		if v, ok := configInt(get(key)); ok && v >= lo && v <= hi {
			*into = v
		}
	}
	read("ChemistryFamiliarRounds", &rules.TierRounds[0], 1, 1000000)
	read("ChemistryTrustedRounds", &rules.TierRounds[1], 1, 1000000)
	read("ChemistrySwornRounds", &rules.TierRounds[2], 1, 1000000)
	read("ChemistryFamiliarBonus", &rules.TierBonus[0], 0, 10)
	read("ChemistryTrustedBonus", &rules.TierBonus[1], 0, 10)
	read("ChemistrySwornBonus", &rules.TierBonus[2], 0, 10)
	if !rules.Valid() {
		return domain.DefaultChemistryRules(), false
	}
	return rules, true
}

// refreshChemistryRules parses the knobs into the cache. It runs on load and
// once a round, not per strike: reading plugin config flattens the whole
// modules config. A bad set is warned about once, until it is fixed.
func (m *CompanyModule) refreshChemistryRules() {
	if m.plug == nil {
		return
	}
	rules, ok := parseChemistryConfig(m.plug.Config.Get)
	if !ok && !m.chemConfigWarned {
		mudlog.Warn("company: chemistry config out of order; using defaults")
	}
	m.chemConfigWarned = !ok
	m.chemRules = &rules
}

// chemistryRules are the cached rules (see refreshChemistryRules).
func (m *CompanyModule) chemistryRules() domain.ChemistryRules {
	if m.chemRulesForTest != nil {
		return *m.chemRulesForTest
	}
	if m.chemRules == nil {
		m.refreshChemistryRules()
	}
	if m.chemRules != nil {
		return *m.chemRules
	}
	return domain.DefaultChemistryRules()
}

// presentMembers maps each member of a signed-in leader's company who is
// alive and there to its room: the leader when living, and each companion
// whose tracked live mob still serves the leader. online is false when the
// leader is signed out, and then nothing is present.
func (m *CompanyModule) presentMembers(leaderUserID int, record domain.Record) (present map[domain.MemberKey]int, online bool) {
	world := m.chemistryWorld()
	if !world.LeaderOnline(leaderUserID) {
		return nil, false
	}
	present = map[domain.MemberKey]int{}
	if room, ok := world.LeaderPresence(leaderUserID); ok {
		present[domain.LeaderMemberKey] = room
	}
	for _, c := range record.Companions {
		instanceID, tracked := m.instance(leaderUserID, c.ID)
		if !tracked || !m.runtime.IsLive(instanceID) || !m.runtime.IsAttached(leaderUserID, instanceID) {
			continue
		}
		if room, ok := world.InstancePresence(instanceID); ok {
			present[domain.CompanionMemberKey(c.ID)] = room
		}
	}
	return present, true
}

type bondCrossing struct {
	leaderUserID int
	a, b         domain.MemberKey
	tier         int
}

// accrueChemistry charges round to every pair of present members sharing a
// room. A bond that reaches a new tier is saved at once; if the save fails
// it is held one round short of the tier, so no tier is used or shown before
// it is on disk. It never writes the round counter.
func (m *CompanyModule) accrueChemistry(round uint64) {
	rules := m.chemistryRules()
	var crossings []bondCrossing
	for leaderUserID := range m.instances {
		record, ok := m.registry.Companies[leaderUserID]
		if !ok {
			continue
		}
		present, _ := m.presentMembers(leaderUserID, record)
		if len(present) < 2 {
			continue
		}
		keys := make([]domain.MemberKey, 0, len(present))
		for key := range present {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		changed := false
		for i, a := range keys {
			for _, b := range keys[i+1:] {
				if present[a] != present[b] {
					continue
				}
				before, after, charged := record.ChargeBond(a, b, round)
				if !charged {
					continue
				}
				changed = true
				if tier := rules.Tier(after); tier > rules.Tier(before) {
					crossings = append(crossings, bondCrossing{leaderUserID, a, b, tier})
				}
			}
		}
		if changed {
			m.registry.Companies[leaderUserID] = record
		}
	}
	if len(crossings) == 0 {
		return
	}
	if err := m.save(); err != nil {
		mudlog.Error("company: save chemistry tier", "error", err)
		for _, c := range crossings {
			m.holdBondShort(c, rules)
		}
		return
	}
	for _, c := range crossings {
		m.chemistryWorld().Tell(c.leaderUserID, m.crossingText(c, rules))
	}
}

// holdBondShort sets a bond one round short of the tier it just reached.
func (m *CompanyModule) holdBondShort(c bondCrossing, rules domain.ChemistryRules) {
	record := m.registry.Companies[c.leaderUserID]
	a, b := domain.BondPair(c.a, c.b)
	for i := range record.Bonds {
		if record.Bonds[i].A == a && record.Bonds[i].B == b {
			record.Bonds[i].Rounds = rules.TierRounds[c.tier-1] - 1
		}
	}
	m.registry.Companies[c.leaderUserID] = record
}

func (m *CompanyModule) crossingText(c bondCrossing, rules domain.ChemistryRules) string {
	name := domain.TierName(c.tier)
	effect := ""
	if bonus := rules.Bonus(c.tier); bonus > 0 {
		effect = fmt.Sprintf(" (+%d%% to hit fighting side by side)", bonus)
	}
	if c.a == domain.LeaderMemberKey || c.b == domain.LeaderMemberKey {
		other := c.a
		if other == domain.LeaderMemberKey {
			other = c.b
		}
		return fmt.Sprintf("Your bond with %s deepens: %s%s.", m.bondName(c.leaderUserID, other), name, effect)
	}
	return fmt.Sprintf("%s and %s have grown %s%s.", m.bondName(c.leaderUserID, c.a), m.bondName(c.leaderUserID, c.b), name, effect)
}

// bondName is "you" for the leader and "<name> (#id)" for a companion.
func (m *CompanyModule) bondName(leaderUserID int, key domain.MemberKey) string {
	if id, ok := domain.CompanionIDFromMemberKey(key); ok {
		return m.companionLabel(leaderUserID, id)
	}
	return "you"
}

var _ domain.ChemistryProvider = (*CompanyModule)(nil)

// ChemistryHitBonus implements domain.ChemistryProvider: the member's
// highest bond tier's bonus among partners alive and in its room now. It
// reads the registry in place, without copying, and checks presence only
// for a member with a tier at all, since combat calls it for every strike.
func (m *CompanyModule) ChemistryHitBonus(leaderUserID int, key domain.MemberKey) int {
	if m.persistenceAvailable() != nil {
		return 0
	}
	record, ok := m.registry.Companies[leaderUserID]
	if !ok || len(record.Bonds) == 0 {
		return 0
	}
	rules := m.chemistryRules()
	if _, tier, ok := domain.BestBond(record.Bonds, key, nil, rules); !ok || tier == domain.TierNone {
		return 0
	}
	_, tier := m.activeBond(leaderUserID, record, key, rules)
	return rules.Bonus(tier)
}

// activeBond is key's highest-tier bond with a partner beside it now, and
// that tier; TierNone when there is none or key isn't present.
func (m *CompanyModule) activeBond(leaderUserID int, record domain.Record, key domain.MemberKey, rules domain.ChemistryRules) (domain.Bond, int) {
	present, _ := m.presentMembers(leaderUserID, record)
	room, here := present[key]
	if !here {
		return domain.Bond{}, domain.TierNone
	}
	bond, tier, _ := domain.BestBond(record.Bonds, key, func(partner domain.MemberKey) bool {
		r, ok := present[partner]
		return ok && r == room
	}, rules)
	return bond, tier
}

// ChemistryStanding implements domain.ChemistryProvider for displays: the
// bond giving the member its bonus now or, when none does, its strongest.
func (m *CompanyModule) ChemistryStanding(leaderUserID int, key domain.MemberKey) (domain.ChemistryStandingView, bool) {
	if m.persistenceAvailable() != nil {
		return domain.ChemistryStandingView{}, false
	}
	record, ok := m.registry.Companies[leaderUserID]
	if !ok {
		return domain.ChemistryStandingView{}, false
	}
	rules := m.chemistryRules()
	best, tier, ok := domain.BestBond(record.Bonds, key, nil, rules)
	if !ok {
		return domain.ChemistryStandingView{}, false
	}
	view := domain.ChemistryStandingView{Tier: tier}
	if active, activeTier := m.activeBond(leaderUserID, record, key, rules); activeTier > domain.TierNone && rules.Bonus(activeTier) > 0 {
		best, view.Tier, view.Bonus = active, activeTier, rules.Bonus(activeTier)
	}
	partner, _ := best.Partner(key)
	view.Partner = m.bondName(leaderUserID, partner)
	return view, true
}

// chemistryView is "company chemistry": each member's strongest bond, what
// it gives now, and progress to its next tier.
func (m *CompanyModule) chemistryView(leaderUserID int) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok || len(record.Companions) == 0 {
		return "No companions."
	}
	rules := m.chemistryRules()
	present, _ := m.presentMembers(leaderUserID, record)
	lines := []string{fmt.Sprintf("Company chemistry (Familiar +%d%%, Trusted +%d%%, Sworn +%d%% to hit):",
		rules.TierBonus[0], rules.TierBonus[1], rules.TierBonus[2])}
	keys := []domain.MemberKey{domain.LeaderMemberKey}
	for _, c := range record.Companions {
		keys = append(keys, domain.CompanionMemberKey(c.ID))
	}
	for _, key := range keys {
		label := "You"
		if key != domain.LeaderMemberKey {
			label = m.bondName(leaderUserID, key)
		}
		best, tier, ok := domain.BestBond(record.Bonds, key, nil, rules)
		if !ok {
			lines = append(lines, fmt.Sprintf("  %s: no shared service yet.", label))
			continue
		}
		partner, _ := best.Partner(key)
		next, pct := rules.Progress(best.Rounds)
		progress := "the strongest bond there is"
		if next != domain.TierNone {
			progress = fmt.Sprintf("%d%% of the way to %s", pct, domain.TierName(next))
		}
		if tier == domain.TierNone {
			lines = append(lines, fmt.Sprintf("  %s: no bond yet; closest with %s, %s.", label, m.bondName(leaderUserID, partner), progress))
			continue
		}
		now := "not at their side"
		if key == domain.LeaderMemberKey {
			now = "not at your side"
		}
		if active, activeTier := m.activeBond(leaderUserID, record, key, rules); rules.Bonus(activeTier) > 0 {
			now = fmt.Sprintf("+%d%% to hit now", rules.Bonus(activeTier))
			if activePartner, _ := active.Partner(key); activePartner != partner {
				now += fmt.Sprintf(", from %s with %s", domain.TierName(activeTier), m.bondName(leaderUserID, activePartner))
			}
		}
		if _, here := present[key]; !here {
			now = "not here"
		}
		lines = append(lines, fmt.Sprintf("  %s: %s with %s (%s); %s.", label, domain.TierName(tier), m.bondName(leaderUserID, partner), now, progress))
	}
	lines = append(lines, "Bonds grow while members are together and alive and you are signed in. Only the strongest bond with someone beside you counts.")
	return strings.Join(lines, "\n")
}
