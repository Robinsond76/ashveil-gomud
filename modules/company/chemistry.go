package company

import (
	"fmt"
	"slices"
	"strings"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/gametime"
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

// memberOrder sorts the leader first, then companions by number.
func memberOrder(a, b domain.MemberKey) int {
	rank := func(k domain.MemberKey) int {
		if id, ok := domain.CompanionIDFromMemberKey(k); ok {
			return id
		}
		return 0
	}
	return rank(a) - rank(b)
}

// band is the members present in one room, leader first.
func band(present map[domain.MemberKey]int, room int) []domain.MemberKey {
	var members []domain.MemberKey
	for key, r := range present {
		if r == room {
			members = append(members, key)
		}
	}
	slices.SortFunc(members, memberOrder)
	return members
}

// bands groups the present members by room; every band has two or more.
func bands(present map[domain.MemberKey]int) [][]domain.MemberKey {
	rooms := map[int]bool{}
	for _, room := range present {
		rooms[room] = true
	}
	var out [][]domain.MemberKey
	for room := range rooms {
		if members := band(present, room); len(members) >= 2 {
			out = append(out, members)
		}
	}
	slices.SortFunc(out, func(a, b []domain.MemberKey) int { return memberOrder(a[0], b[0]) })
	return out
}

type bandCrossing struct {
	leaderUserID int
	members      []domain.MemberKey
	tier         int
}

// accrueChemistry charges round to every member who is present with at
// least one other. Tiers come from saved service only, so a band whose
// current service would lift its tier is only noted: the tier takes effect,
// and the leader is told, at the next successful company save (autosave,
// drift tick, logout, a command). Chemistry adds no save of its own, so the
// company file still changes only on the existing save seams. It never
// writes the round counter.
func (m *CompanyModule) accrueChemistry(round uint64) {
	rules := m.chemistryRules()
	top := rules.TierRounds[len(rules.TierRounds)-1]
	for leaderUserID := range m.instances {
		record, ok := m.registry.Companies[leaderUserID]
		if !ok {
			continue
		}
		present, _ := m.presentMembers(leaderUserID, record)
		changed := false
		for _, members := range bands(present) {
			for _, member := range members {
				if record.ChargeService(member, round) {
					changed = true
				}
			}
			current, _ := record.BandAverage(members, false, top)
			if tier := rules.Tier(current); tier > record.BandTier(members, rules) {
				m.notePendingTierUp(bandCrossing{leaderUserID, members, tier})
			}
		}
		if changed {
			m.registry.Companies[leaderUserID] = record
		}
	}
}

// notePendingTierUp remembers a band due to rise a tier at the next save.
func (m *CompanyModule) notePendingTierUp(c bandCrossing) {
	if m.pendingTierUps == nil {
		m.pendingTierUps = map[string]bandCrossing{}
	}
	key := fmt.Sprint(c.leaderUserID, c.members)
	if prev, ok := m.pendingTierUps[key]; !ok || c.tier > prev.tier {
		m.pendingTierUps[key] = c
	}
}

// announceTierUps tells leaders of bands whose saved tier has now risen. It
// runs after each successful save; a band that no longer reaches the tier
// (a member dismissed meanwhile) is dropped silently.
func (m *CompanyModule) announceTierUps() {
	if len(m.pendingTierUps) == 0 {
		return
	}
	rules := m.chemistryRules()
	pending := m.pendingTierUps
	m.pendingTierUps = nil
	keys := make([]string, 0, len(pending))
	for key := range pending {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		c := pending[key]
		record, ok := m.registry.Companies[c.leaderUserID]
		if !ok {
			continue
		}
		if tier := record.BandTier(c.members, rules); tier >= c.tier {
			c.tier = tier
			m.chemistryWorld().Tell(c.leaderUserID, m.crossingText(c, rules))
		}
	}
}

// markServiceSaved records that every member's service is on disk.
func (m *CompanyModule) markServiceSaved() {
	for leaderUserID, record := range m.registry.Companies {
		if len(record.Service) > 0 {
			record.MarkServiceSaved()
			m.registry.Companies[leaderUserID] = record
		}
	}
}

func (m *CompanyModule) crossingText(c bandCrossing, rules domain.ChemistryRules) string {
	effect := ""
	if bonus := rules.Bonus(c.tier); bonus > 0 {
		effect = fmt.Sprintf(" (+%d%% to hit fighting together)", bonus)
	}
	if slices.Contains(c.members, domain.LeaderMemberKey) {
		return fmt.Sprintf("Your band grows closer: %s%s.", domain.TierName(c.tier), effect)
	}
	return fmt.Sprintf("Your companions %s grow closer: %s%s.", m.bandNames(c.leaderUserID, c.members), domain.TierName(c.tier), effect)
}

// bandNames lists members by name: "you", "<name> (#id)".
func (m *CompanyModule) bandNames(leaderUserID int, members []domain.MemberKey) string {
	names := make([]string, len(members))
	for i, member := range members {
		names[i] = m.bandMemberName(leaderUserID, member)
	}
	if len(names) <= 2 {
		return strings.Join(names, " and ")
	}
	return strings.Join(names[:len(names)-1], ", ") + ", and " + names[len(names)-1]
}

// bandMemberName is "you" for the leader and "<name> (#id)" for a companion.
func (m *CompanyModule) bandMemberName(leaderUserID int, key domain.MemberKey) string {
	if id, ok := domain.CompanionIDFromMemberKey(key); ok {
		return m.companionLabel(leaderUserID, id)
	}
	return "you"
}

var _ domain.ChemistryProvider = (*CompanyModule)(nil)

// ChemistryHitBonus implements domain.ChemistryProvider: the bonus of the
// band together in the member's room now. It reads the registry in place,
// without copying, and looks for the band only when some member's saved
// service could reach a tier, since combat calls it for every strike.
func (m *CompanyModule) ChemistryHitBonus(leaderUserID int, key domain.MemberKey) int {
	if m.persistenceAvailable() != nil {
		return 0
	}
	record, ok := m.registry.Companies[leaderUserID]
	if !ok {
		return 0
	}
	rules := m.chemistryRules()
	longest := 0
	for _, s := range record.Service {
		longest = max(longest, s.Saved)
	}
	if rules.Tier(longest) == domain.TierNone {
		return 0 // an average never exceeds its largest member
	}
	return rules.Bonus(record.BandTier(m.bandOf(leaderUserID, record, key), rules))
}

// bandOf is the members present in key's room, key included; nil when key
// isn't present.
func (m *CompanyModule) bandOf(leaderUserID int, record domain.Record, key domain.MemberKey) []domain.MemberKey {
	present, _ := m.presentMembers(leaderUserID, record)
	room, here := present[key]
	if !here {
		return nil
	}
	return band(present, room)
}

// ChemistryStanding implements domain.ChemistryProvider for displays.
func (m *CompanyModule) ChemistryStanding(leaderUserID int, key domain.MemberKey) (domain.ChemistryStandingView, bool) {
	if m.persistenceAvailable() != nil {
		return domain.ChemistryStandingView{}, false
	}
	record, ok := m.registry.Companies[leaderUserID]
	if !ok {
		return domain.ChemistryStandingView{}, false
	}
	members := m.bandOf(leaderUserID, record, key)
	if members == nil {
		return domain.ChemistryStandingView{}, false
	}
	rules := m.chemistryRules()
	tier := record.BandTier(members, rules)
	return domain.ChemistryStandingView{Together: len(members), Tier: tier, Bonus: rules.Bonus(tier)}, true
}

// daysServed renders saved rounds as game days.
func daysServed(rounds int) string {
	perDay := gametime.GetDate().RoundsPerDay
	if perDay <= 0 {
		perDay = 900
	}
	return fmt.Sprintf("%.1f days", float64(rounds)/float64(perDay))
}

// bandLine describes one band: its size, tier, bonus, and progress.
func bandLine(record domain.Record, members []domain.MemberKey, rules domain.ChemistryRules) string {
	average, _ := record.BandAverage(members, true, rules.TierRounds[len(rules.TierRounds)-1])
	tier := rules.Tier(average)
	text := fmt.Sprintf("%d together, %s", len(members), domain.TierName(tier))
	if bonus := rules.Bonus(tier); bonus > 0 {
		text += fmt.Sprintf(" (+%d%% to hit)", bonus)
	}
	if next, pct := rules.Progress(average); next != domain.TierNone {
		text += fmt.Sprintf("; %d%% of the way to %s", pct, domain.TierName(next))
	}
	return text
}

// chemistryView is "company chemistry": the band with the leader, any
// other band apart from them, and each member's saved service.
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
	lines := []string{fmt.Sprintf("Company chemistry (Familiar +%d%%, Trusted +%d%%, Sworn +%d%% to hit for everyone in the band):",
		rules.TierBonus[0], rules.TierBonus[1], rules.TierBonus[2])}
	withLeader := "  With you: no one from the band."
	if _, here := present[domain.LeaderMemberKey]; !here {
		withLeader = "  With you: you are not with the band."
	}
	var apart []string
	for _, members := range bands(present) {
		if slices.Contains(members, domain.LeaderMemberKey) {
			withLeader = "  With you: " + bandLine(record, members, rules) + "."
			continue
		}
		apart = append(apart, fmt.Sprintf("  Apart: %s, %s.", m.bandNames(leaderUserID, members), bandLine(record, members, rules)))
	}
	lines = append(lines, withLeader)
	lines = append(lines, apart...)
	lines = append(lines, "Service with the band:")
	keys := []domain.MemberKey{domain.LeaderMemberKey}
	for _, c := range record.Companions {
		keys = append(keys, domain.CompanionMemberKey(c.ID))
	}
	for _, key := range keys {
		label := "You"
		if key != domain.LeaderMemberKey {
			label = m.bandMemberName(leaderUserID, key)
		}
		served := 0
		if s, found := record.FindService(key); found {
			served = s.Saved
		}
		line := fmt.Sprintf("  %s: %s", label, daysServed(served))
		if _, here := present[key]; !here {
			line += " (not here)"
		}
		lines = append(lines, line)
	}
	lines = append(lines, "Service grows while members are alive and together and you are signed in. A band's tier comes from the average service of those together, so new members pull it down until they settle in.")
	return strings.Join(lines, "\n")
}
