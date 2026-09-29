package company

// Phase 32a2: per-player recruit rosters. A Generated recruiter posts, for
// each player, their own generated candidates, who come and go. The
// roster is brought up to date when read (the notice, "company recruit",
// "company inspect", "look"), from the world round, which is only read.
// Refreshes stay in memory and reach disk with the next company save; a
// hire saves at once. See modules/company/AGENTS.md.

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Roster defaults, in world rounds (900 a game day) and gold.
const (
	defaultRosterSize    = 3
	defaultStayMin       = 450
	defaultStayMax       = 1800
	defaultRefillMin     = 60
	defaultRefillMax     = 180
	defaultPriceBase     = 30
	defaultPricePerLevel = 30
	defaultAlignmentMin  = -80
	defaultAlignmentMax  = 80
	defaultBynamePercent = 60
	maxRosterSize        = 8
)

// parseRosterRules reads the roster knobs, falling back to the default for
// a missing or out-of-range value. A min above its max takes the max.
// RecruitTemplates' archetypes come from CompanionArchetypes.
func parseRosterRules(get func(string) any, archetypeOf map[int]string) domain.RosterRules {
	num := func(key string, def, lo, hi int) int {
		if n, ok := configInt(get(key)); ok && n >= lo && n <= hi {
			return n
		}
		return def
	}
	rules := domain.RosterRules{
		Size:          num("RosterSize", defaultRosterSize, 0, maxRosterSize),
		StayMin:       uint64(num("CandidateStayRoundsMin", defaultStayMin, 1, 1000000)),
		StayMax:       uint64(num("CandidateStayRoundsMax", defaultStayMax, 1, 1000000)),
		RefillMin:     uint64(num("RefillDelayRoundsMin", defaultRefillMin, 1, 1000000)),
		RefillMax:     uint64(num("RefillDelayRoundsMax", defaultRefillMax, 1, 1000000)),
		PriceBase:     num("RecruitPriceBase", defaultPriceBase, 0, 100000),
		PricePerLevel: num("RecruitPricePerLevel", defaultPricePerLevel, 0, 100000),
		AlignmentMin:  num("RecruitAlignmentMin", defaultAlignmentMin, -100, 100),
		AlignmentMax:  num("RecruitAlignmentMax", defaultAlignmentMax, -100, 100),
		BynamePercent: num("RecruitBynamePercent", defaultBynamePercent, 0, 100),
		GivenNames:    configStrings(get("RecruitGivenNames")),
		Bynames:       configStrings(get("RecruitBynames")),
		Traits:        configStrings(get("RecruitTraits")),
	}
	rules.StayMin = min(rules.StayMin, rules.StayMax)
	rules.RefillMin = min(rules.RefillMin, rules.RefillMax)
	rules.AlignmentMin = min(rules.AlignmentMin, rules.AlignmentMax)
	list, _ := get("RecruitTemplates").([]any)
	for _, entry := range list {
		fields := lowerKeys(entry)
		templateID, ok := configInt(fields["mobtemplateid"])
		if fields == nil || !ok || templateID <= 0 {
			mudlog.Warn("company: recruit template without a mob template; skipped")
			continue
		}
		archetype := archetypeOf[templateID]
		if archetype == "" {
			mudlog.Warn("company: recruit template has no CompanionArchetypes entry; skipped", "template", templateID)
			continue
		}
		weight := 1
		if raw, set := fields["weight"]; set {
			if weight, ok = configInt(raw); !ok || weight < 0 {
				weight = 0
			}
		}
		percent, _ := configInt(fields["pricepercent"])
		rules.Archetypes = append(rules.Archetypes, domain.RosterArchetype{
			Archetype: archetype, MobTemplateID: templateID, Weight: weight, PricePercent: max(percent, 0),
		})
	}
	return rules
}

// parseArchetypeWeights reads a recruiter's ArchetypeWeights ([{Archetype,
// Weight}]); nil when unset.
func parseArchetypeWeights(raw any, roomID int) map[string]int {
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		return nil
	}
	out := map[string]int{}
	for _, entry := range list {
		fields := lowerKeys(entry)
		name, _ := fields["archetype"].(string)
		name = strings.ToLower(strings.TrimSpace(name))
		weight, ok := configInt(fields["weight"])
		if name == "" || !ok || weight < 0 {
			mudlog.Warn("company: recruiter archetype weight malformed; skipped", "roomid", roomID)
			continue
		}
		out[name] = weight
	}
	return out
}

func configStrings(raw any) []string {
	var out []string
	switch v := raw.(type) {
	case []string:
		for _, s := range v {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
	case []any:
		for _, x := range v {
			if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
	}
	return out
}

func (m *CompanyModule) rosterRules() domain.RosterRules {
	if m.rosterRulesForTest != nil {
		return *m.rosterRulesForTest
	}
	if m.plug != nil {
		return parseRosterRules(m.plug.Config.Get, m.companionArchetypes())
	}
	return parseRosterRules(func(string) any { return nil }, nil)
}

// round is the world round, read and never advanced.
func (m *CompanyModule) round() uint64 {
	if m.roundNow != nil {
		return m.roundNow()
	}
	return util.GetRoundCount()
}

// random is the module's own source (game loop only).
func (m *CompanyModule) random() domain.Rand {
	if m.rng == nil {
		m.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return m.rng
}

func leaderLevel(userID int) int {
	if user := users.GetByUserId(userID); user != nil && user.Character != nil && user.Character.Level > 0 {
		return user.Character.Level
	}
	return 1
}

// givenKey is the word a name is typed by: its first, lower case.
func givenKey(name string) string {
	if fields := strings.Fields(strings.ToLower(name)); len(fields) > 0 {
		return fields[0]
	}
	return ""
}

// rosterFor is the leader's roster at a Generated recruiter, brought up to
// date now. The refresh is kept in memory; it reaches disk with the next
// company save. false when the recruiter posts none or the company can't
// be read.
func (m *CompanyModule) rosterFor(leaderUserID int, rec recruiter) (domain.Roster, bool) {
	if !rec.Generated || leaderUserID <= 0 || m.persistenceAvailable() != nil {
		return domain.Roster{}, false
	}
	record, _ := m.registry.Get(leaderUserID)
	current, _ := record.Roster(rec.RoomID)
	current.RoomID = rec.RoomID
	taken := map[string]bool{}
	for _, c := range record.Companions {
		taken[givenKey(nameOf(c, ""))] = true
	}
	// Unique across all the leader's rosters, so a face hired at one
	// recruiter can't also be waiting at another.
	for _, other := range record.Rosters {
		if other.RoomID != rec.RoomID {
			for _, c := range other.Candidates {
				taken[c.Key] = true
			}
		}
	}
	// Every recruiter's regulars, not only this one's, so a generated face
	// never shares a name with a well-known one.
	for _, r := range m.recruiters() {
		for _, c := range r.Candidates {
			taken[c.ID] = true
			taken[givenKey(templateName(c.MobTemplateID, ""))] = true
		}
	}
	ctx := domain.RosterContext{Now: m.round(), LeaderLevel: leaderLevel(leaderUserID), Taken: taken, Weights: rec.Weights}
	next, changed := domain.RefreshRoster(current, m.rosterRules(), ctx, m.random())
	if changed {
		if err := m.registry.PutRoster(leaderUserID, next); err != nil {
			mudlog.Warn("company: store roster", "leader", leaderUserID, "error", err)
		}
	}
	return next, true
}

// findGenerated matches a roster candidate: exact (key or name) only, or
// a unique partial name too.
func findGenerated(roster domain.Roster, selector string, partial bool) (domain.Candidate, bool) {
	sel := strings.ToLower(strings.TrimSpace(selector))
	if sel == "" {
		return domain.Candidate{}, false
	}
	c, ok := roster.Find(sel)
	if !ok || partial || c.Key == sel || strings.ToLower(c.Name) == sel {
		return c, ok
	}
	return domain.Candidate{}, false
}

// generatedListing is "company recruit"'s entry for a roster candidate.
func (m *CompanyModule) generatedListing(c domain.Candidate) []string {
	lines := []string{fmt.Sprintf("  %s (company recruit %s): %s, level %d, alignment %s",
		c.Name, c.Key, archetypeLabel(c.Archetype), c.Level, alignmentLabel(c.Alignment))}
	if c.Trait != "" {
		lines = append(lines, "    "+c.Trait)
	}
	if state, ok := m.runtime.TemplateState(c.MobTemplateID); ok {
		lines = append(lines, "    Gear: "+gearSummary(state))
	}
	return append(lines, fmt.Sprintf("    Price: %d gold", c.Price))
}

// hireGenerated takes a roster candidate on: the capacity, alignment, and
// gold checks, then enlist with the candidate's identity, level, and
// alignment and the roster without them, in one save.
func (m *CompanyModule) hireGenerated(user *users.UserRecord, roomID int, roster domain.Roster, c domain.Candidate) (string, error) {
	if _, ok := m.runtime.TemplateState(c.MobTemplateID); !ok {
		return fmt.Sprintf("%s isn't available right now.", c.Name), nil
	}
	record, _ := m.registry.Get(user.UserId)
	if len(record.Companions) >= m.maxCompanions() {
		return fmt.Sprintf("Your company is full (%d/%d companions). Dismiss someone first.", len(record.Companions), m.maxCompanions()), nil
	}
	if refusal := m.alignmentRefusal(user.UserId, c.Alignment, c.Name); refusal != "" {
		return refusal, nil
	}
	if user.Character.Gold < c.Price {
		return fmt.Sprintf("%s asks %d gold, and you have %d.", c.Name, c.Price, user.Character.Gold), nil
	}
	after, ok := domain.HireFromRoster(roster, c.Key, m.rosterRules(), m.round(), m.random())
	if !ok {
		return fmt.Sprintf(`No one called "%s" is hiring here.`, c.Key), nil
	}
	companion, err := m.enlist(user.UserId, roomID, c.MobTemplateID, map[int]struct{}{c.MobTemplateID: {}}, false, &generatedHire{candidate: c, rosterAfter: after})
	if err != nil {
		return "", err
	}
	text := fmt.Sprintf("%s joins your company (#%d).", c.Name, companion.ID)
	if c.Price > 0 {
		m.chargeGold(user, c.Price)
		text = fmt.Sprintf("You pay %d gold. %s", c.Price, text)
	}
	return text + ` Place them with "formation move".`, nil
}

// chargeGold takes a paid recruit's price after the company save, then
// saves the user at once. A failed user save leaves the deduction in
// memory for the next autosave, logout, or copyover save.
func (m *CompanyModule) chargeGold(user *users.UserRecord, price int) {
	user.Character.Gold -= price
	events.AddToQueue(events.EquipmentChange{UserId: user.UserId, GoldChange: -price})
	if m.saveUser != nil {
		if err := m.saveUser(user); err != nil {
			mudlog.Error("company: save after recruit", "user", user.UserId, "error", err)
		}
	}
}

// inspectGenerated weighs a roster candidate against the company.
func (m *CompanyModule) inspectGenerated(leaderUserID int, c domain.Candidate) string {
	lines := []string{fmt.Sprintf("%s, %s, level %d: alignment %s.", c.Name, archetypeLabel(c.Archetype), c.Level, alignmentLabel(c.Alignment))}
	if c.Trait != "" {
		lines = append(lines, c.Trait)
	}
	if average, ok := m.companyAverage(leaderUserID); ok {
		lines = append(lines, fmt.Sprintf("Your company: %s.", alignmentLabel(average)))
		switch {
		case m.alignmentRefusal(leaderUserID, c.Alignment, c.Name) != "":
			lines = append(lines, "They won't join a company so far from their ways.")
		case m.uneasy(c.Alignment, average):
			lines = append(lines, "Their ways are close enough to your company's, but far enough that they would lose loyalty.")
		default:
			lines = append(lines, "Their ways are close enough to your company's.")
		}
	}
	return strings.Join(append(lines, fmt.Sprintf("They ask %d gold.", c.Price)), "\n")
}

// lookGenerated is "look <name>" at a roster candidate on the notice.
func lookGenerated(noticeName string, c domain.Candidate) string {
	lines := []string{
		fmt.Sprintf(`You read about <ansi fg="mobname">%s</ansi> on %s: %s, level %d, asking %d gold.`, c.Name, noticeName, archetypeLabel(c.Archetype), c.Level, c.Price),
	}
	if c.Trait != "" {
		lines = append(lines, c.Trait)
	}
	lines = append(lines, fmt.Sprintf(`Type <ansi fg="command">company inspect %s</ansi> to weigh them against your company.`, c.Key))
	return strings.Join(lines, "\n")
}

// inspectAt is "company inspect": a candidate on the leader's own roster
// at this room's recruiter first, then whoever inspect finds.
func (m *CompanyModule) inspectAt(leaderUserID, roomID int, selector string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	rec, ok := m.recruiters()[rooms.GetOriginalRoom(roomID)]
	if !ok {
		return m.inspect(leaderUserID, selector)
	}
	roster, _ := m.rosterFor(leaderUserID, rec)
	if c, ok := findGenerated(roster, selector, false); ok {
		return m.inspectGenerated(leaderUserID, c)
	}
	// The same resolution "company recruit" uses, so inspect and recruit
	// always mean the same person.
	if !m.summonableName(selector) {
		if _, c := resolveCandidate(rec, roster, selector); c != nil {
			return m.inspectGenerated(leaderUserID, *c)
		}
	}
	return m.inspect(leaderUserID, selector)
}

// summonableName reports whether a selector names a template "company
// summon" allows (by number or name), which inspect weighs first.
func (m *CompanyModule) summonableName(selector string) bool {
	templateID, err := m.resolveTemplateID(strings.TrimSpace(selector))
	if err != nil {
		return false
	}
	_, allowed := m.allowedTemplates()[templateID]
	return allowed
}
