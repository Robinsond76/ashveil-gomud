package company

import (
	"fmt"
	"strconv"
	"strings"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// defaultDriftEveryRounds is the drift interval when the config has none
// (5 minutes at 4-second rounds).
const defaultDriftEveryRounds = 75

// alignmentWorld is what Phase 21a needs from the running game, behind a
// seam so unit tests need no world.
type alignmentWorld interface {
	// TemplateAlignment is a mob template's alignment, or its race's
	// default when the template's is 0 (as mobs.NewMobById does).
	TemplateAlignment(templateID int) int
	// LeaderAlignment is an online leader's alignment; false when offline.
	LeaderAlignment(leaderUserID int) (int, bool)
	LeaderInCombat(leaderUserID int) bool
	InstanceInCombat(instanceID int) bool
	SetInstanceAlignment(instanceID, alignment int)
	Tell(leaderUserID int, text string)
}

type nativeAlignmentWorld struct{}

func (nativeAlignmentWorld) TemplateAlignment(templateID int) int {
	spec := mobs.GetMobSpec(mobs.MobId(templateID))
	if spec == nil {
		return 0
	}
	alignment := int(spec.Character.Alignment)
	if alignment == 0 {
		if race := races.GetRace(spec.Character.GetRaceId()); race != nil {
			alignment = int(race.DefaultAlignment)
		}
	}
	return domain.ClampAlignment(alignment)
}

func (nativeAlignmentWorld) LeaderAlignment(leaderUserID int) (int, bool) {
	user := users.GetByUserId(leaderUserID)
	if user == nil || user.Character == nil {
		return 0, false
	}
	return int(user.Character.Alignment), true
}

func (nativeAlignmentWorld) LeaderInCombat(leaderUserID int) bool {
	user := users.GetByUserId(leaderUserID)
	return user != nil && user.Character != nil && user.Character.Aggro != nil
}

func (nativeAlignmentWorld) InstanceInCombat(instanceID int) bool {
	mob := mobs.GetInstance(instanceID)
	return mob != nil && mob.Character.Aggro != nil
}

func (nativeAlignmentWorld) SetInstanceAlignment(instanceID, alignment int) {
	if mob := mobs.GetInstance(instanceID); mob != nil {
		mob.Character.Alignment = int8(domain.ClampAlignment(alignment))
	}
}

func (nativeAlignmentWorld) Tell(leaderUserID int, text string) {
	if user := users.GetByUserId(leaderUserID); user != nil {
		user.SendText(text)
	}
}

func (m *CompanyModule) alignmentWorld() alignmentWorld {
	if m.world != nil {
		return m.world
	}
	return nativeAlignmentWorld{}
}

// parseAlignmentConfig reads the Phase 21a knobs; a missing, unparsable,
// or out-of-range value falls back to its default.
func parseAlignmentConfig(get func(string) any) (domain.AlignmentRules, int) {
	rules := domain.DefaultAlignmentRules()
	read := func(key string, into *int, lo, hi int) {
		if v, ok := configInt(get(key)); ok && v >= lo && v <= hi {
			*into = v
		}
	}
	every := defaultDriftEveryRounds
	read("DriftEveryRounds", &every, 1, 100000)
	read("DriftStep", &rules.DriftStep, 1, 50)
	read("LoyaltyToleranceGap", &rules.LoyaltyToleranceGap, 0, 200)
	read("LoyaltyLoss", &rules.LoyaltyLoss, 0, 100)
	read("LoyaltyGain", &rules.LoyaltyGain, 0, 100)
	read("StartLoyalty", &rules.StartLoyalty, 1, 100)
	read("LoyaltyWarnBelow", &rules.LoyaltyWarnBelow, 0, 100)
	read("RecruitMaxGap", &rules.RecruitMaxGap, 0, 200)
	return rules, every
}

func (m *CompanyModule) alignmentConfig() (domain.AlignmentRules, int) {
	if m.rulesForTest != nil {
		return *m.rulesForTest, defaultDriftEveryRounds
	}
	if m.plug != nil {
		return parseAlignmentConfig(m.plug.Config.Get)
	}
	return parseAlignmentConfig(func(string) any { return nil })
}

// alignmentLabel renders an engine alignment as players see it.
func alignmentLabel(alignment int) string {
	return fmt.Sprintf("%d (%s)", domain.DisplayAlignment(alignment), domain.AlignmentBand(alignment))
}

// companionAlignment is a companion's stored alignment, or its template's
// when it has not been seeded yet.
func (m *CompanyModule) companionAlignment(c domain.Companion) int {
	if c.Disposition != nil {
		return c.Disposition.Alignment
	}
	return m.alignmentWorld().TemplateAlignment(c.MobTemplateID)
}

// companyAverage is the average alignment of the leader (when online) and
// every companion; false when there is no one to average.
func (m *CompanyModule) companyAverage(leaderUserID int) (int, bool) {
	var values []int
	if leader, ok := m.alignmentWorld().LeaderAlignment(leaderUserID); ok {
		values = append(values, leader)
	}
	if record, ok := m.registry.Get(leaderUserID); ok {
		for _, c := range record.Companions {
			if !c.Dead() { // Phase 25b: the dead don't sway the company
				values = append(values, m.companionAlignment(c))
			}
		}
	}
	if len(values) == 0 {
		return 0, false
	}
	return domain.AverageAlignment(values), true
}

// recruitRefusal is the message refusing a candidate too far from the
// company's alignment, or "" when the candidate would join.
func (m *CompanyModule) recruitRefusal(leaderUserID, templateID int, name string) string {
	rules, _ := m.alignmentConfig()
	average, ok := m.companyAverage(leaderUserID)
	if !ok {
		return ""
	}
	candidate := m.alignmentWorld().TemplateAlignment(templateID)
	if domain.CanRecruit(candidate, average, rules) {
		return ""
	}
	return fmt.Sprintf("%s (alignment %d, %s) won't join a company of alignment %d, %s.", name,
		domain.DisplayAlignment(candidate), domain.AlignmentBand(candidate), domain.DisplayAlignment(average), domain.AlignmentBand(average))
}

// seedDisposition records a new recruit's starting alignment and loyalty.
func (m *CompanyModule) seedDisposition(leaderUserID int, companion domain.Companion) {
	rules, _ := m.alignmentConfig()
	d := domain.Disposition{Alignment: m.alignmentWorld().TemplateAlignment(companion.MobTemplateID), Loyalty: rules.StartLoyalty}
	if err := m.registry.SetDisposition(leaderUserID, companion.ID, d); err != nil {
		mudlog.Warn("company: seed disposition", "leader", leaderUserID, "companion", companion.ID, "error", err)
	}
}

// seedLegacyDispositions gives every companion saved before Phase 21a its
// template's alignment at full loyalty, and reports whether any changed.
func (m *CompanyModule) seedLegacyDispositions() bool {
	changed := false
	for leaderUserID, record := range m.registry.Companies {
		for _, c := range record.Companions {
			if c.Disposition != nil {
				continue
			}
			d := domain.Disposition{Alignment: m.alignmentWorld().TemplateAlignment(c.MobTemplateID), Loyalty: domain.MaxLoyalty}
			if err := m.registry.SetDisposition(leaderUserID, c.ID, d); err == nil {
				changed = true
			}
		}
	}
	return changed
}

// applyInstanceAlignment gives a live companion mob its stored alignment.
func (m *CompanyModule) applyInstanceAlignment(leaderUserID, companionID, instanceID int) {
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return
	}
	for _, c := range record.Companions {
		if c.ID == companionID {
			m.alignmentWorld().SetInstanceAlignment(instanceID, m.companionAlignment(c))
			return
		}
	}
}

// onNewRound counts the persisted drift countdown down and runs a drift
// tick when it reaches zero. The drift never reads the round number;
// chemistry reads it only to count each round once.
func (m *CompanyModule) onNewRound(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.NewRound)
	if !ok {
		return events.Continue
	}
	if m.persistenceAvailable() != nil {
		return events.Continue
	}
	// Phase 24: charge this round to every eligible chemistry bond first,
	// so a drift tick's rollback snapshot includes it.
	m.refreshChemistryRules()
	m.accrueChemistry(evt.RoundNumber)
	// Phase 25b: the dead are charged the leader's online time.
	m.chargeAllowances()
	_, every := m.alignmentConfig()
	if m.registry.DriftIn <= 0 || m.registry.DriftIn > every {
		m.registry.DriftIn = every
	}
	m.registry.DriftIn--
	if m.registry.DriftIn > 0 {
		return events.Continue
	}
	m.registry.DriftIn = every
	m.driftTick()
	return events.Continue
}

type deserter struct{ leaderUserID, companionID int }

// driftTick drifts every online leader's companions, saves once, then
// updates live mobs, warns leaders, and processes desertions. A failed
// save restores the pre-tick registry.
func (m *CompanyModule) driftTick() {
	rules, _ := m.alignmentConfig()
	world := m.alignmentWorld()
	before := m.registry.Clone()
	changed := false
	var warnings []deserter
	var deserters []deserter
	for leaderUserID, record := range before.Companies {
		leader, online := world.LeaderAlignment(leaderUserID)
		if !online {
			continue
		}
		// Phase 25b: the dead neither drift, sway the others, nor desert.
		var members []domain.MemberAlignment
		for _, c := range record.Companions {
			if c.Dead() {
				continue
			}
			loyalty := domain.MaxLoyalty
			if c.Disposition != nil {
				loyalty = c.Disposition.Loyalty
			}
			members = append(members, domain.MemberAlignment{ID: c.ID, Alignment: m.companionAlignment(c), Loyalty: loyalty})
		}
		if len(members) == 0 {
			continue
		}
		result := domain.TickAlignment(leader, members, rules)
		for _, member := range result.Members {
			if err := m.registry.SetDisposition(leaderUserID, member.ID, domain.Disposition{Alignment: member.Alignment, Loyalty: member.Loyalty}); err == nil {
				changed = true
			}
		}
		for _, id := range result.Warned {
			warnings = append(warnings, deserter{leaderUserID, id})
		}
		for _, id := range result.Deserters {
			deserters = append(deserters, deserter{leaderUserID, id})
		}
	}
	if !changed {
		return
	}
	if err := m.save(); err != nil {
		driftIn := m.registry.DriftIn
		m.registry = before
		m.registry.DriftIn = driftIn
		mudlog.Error("company: alignment drift", "error", err)
		return
	}
	for leaderUserID, byCompanion := range m.instances {
		for companionID, instanceID := range byCompanion {
			if m.runtime.IsLive(instanceID) {
				m.applyInstanceAlignment(leaderUserID, companionID, instanceID)
			}
		}
	}
	for _, w := range warnings {
		world.Tell(w.leaderUserID, fmt.Sprintf("%s grows uneasy with the company's ways.", m.companionLabel(w.leaderUserID, w.companionID)))
	}
	for _, d := range deserters {
		if m.inCombat(d.leaderUserID, d.companionID) {
			continue // re-evaluated on the next tick
		}
		label := m.companionLabel(d.leaderUserID, d.companionID)
		record, ok := m.registry.Get(d.leaderUserID)
		if !ok {
			continue
		}
		companion, ok := findCompanion(record, d.companionID)
		if !ok {
			continue
		}
		if err := m.removeCompanion(d.leaderUserID, record, companion); err != nil {
			mudlog.Error("company: desertion", "leader", d.leaderUserID, "companion", d.companionID, "error", err)
			continue
		}
		world.Tell(d.leaderUserID, fmt.Sprintf("%s has lost faith in your company and deserts.", label))
	}
}

// inCombat reports whether the leader or the companion's live mob is
// fighting; desertion never happens mid-fight.
func (m *CompanyModule) inCombat(leaderUserID, companionID int) bool {
	world := m.alignmentWorld()
	if world.LeaderInCombat(leaderUserID) {
		return true
	}
	instanceID, tracked := m.instance(leaderUserID, companionID)
	return tracked && m.runtime.IsLive(instanceID) && world.InstanceInCombat(instanceID)
}

func findCompanion(record domain.Record, companionID int) (domain.Companion, bool) {
	for _, c := range record.Companions {
		if c.ID == companionID {
			return c, true
		}
	}
	return domain.Companion{}, false
}

// companionLabel is "<name> (#id)" for messages.
func (m *CompanyModule) companionLabel(leaderUserID, companionID int) string {
	name := "A companion"
	if record, ok := m.registry.Get(leaderUserID); ok {
		if c, found := findCompanion(record, companionID); found {
			name = templateName(c.MobTemplateID, name)
		}
	}
	return fmt.Sprintf("%s (#%d)", name, companionID)
}

// inspect shows a recruit candidate's alignment against the company's.
func (m *CompanyModule) inspect(leaderUserID int, selector string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	selector = strings.TrimSpace(selector)
	templateID, isCandidate := m.candidateTemplate(selector)
	if !isCandidate {
		var err error
		if templateID, err = m.resolveTemplateID(selector); err != nil {
			return fmt.Sprintf("There's no one called %q to recruit.", selector)
		}
	}
	name := templateName(templateID, selector)
	if _, ok := m.allowedTemplates()[templateID]; !ok && !isCandidate {
		return fmt.Sprintf("%s isn't available to recruit.", name)
	}
	candidate := m.alignmentWorld().TemplateAlignment(templateID)
	lines := []string{fmt.Sprintf("%s: alignment %s.", name, alignmentLabel(candidate))}
	average, ok := m.companyAverage(leaderUserID)
	if !ok {
		return lines[0]
	}
	lines = append(lines, fmt.Sprintf("Your company: %s.", alignmentLabel(average)))
	if m.recruitRefusal(leaderUserID, templateID, name) == "" {
		rules, _ := m.alignmentConfig()
		gap := candidate - average
		if gap < 0 {
			gap = -gap
		}
		if gap > rules.LoyaltyToleranceGap {
			lines = append(lines, "They would join, but uneasily: far from the company's ways, they would lose loyalty.")
		} else {
			lines = append(lines, "They would join.")
		}
	} else {
		lines = append(lines, "They won't join a company so far from their ways.")
	}
	return strings.Join(lines, "\n")
}

// candidateTemplate finds a recruiter's candidate by id or name, at any
// recruiter (Phase 27d: the tutorial's Oath Stone weighs one it can't
// take). ok is false unless exactly one template matches.
func (m *CompanyModule) candidateTemplate(selector string) (int, bool) {
	found := map[int]bool{}
	for _, rec := range m.recruiters() {
		if c, ok := matchCandidate(rec, selector); ok {
			found[c.MobTemplateID] = true
		}
	}
	if len(found) != 1 {
		return 0, false
	}
	for id := range found {
		return id, true
	}
	return 0, false
}

// alignmentView shows the leader, every companion, and the company average.
func (m *CompanyModule) alignmentView(leaderUserID int) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	rules, _ := m.alignmentConfig()
	leader, online := m.alignmentWorld().LeaderAlignment(leaderUserID)
	record, _ := m.registry.Get(leaderUserID)
	if !online && len(record.Companions) == 0 {
		return "No companions."
	}
	average, _ := m.companyAverage(leaderUserID)
	lines := []string{fmt.Sprintf("Company alignment: %s (-100 is most evil, 100 most good)", alignmentLabel(average))}
	if online {
		lines = append(lines, fmt.Sprintf("  You: %s", alignmentLabel(leader)))
	}
	// Phase 25b: the dead are listed but sway no one.
	var members []domain.MemberAlignment
	for _, c := range record.Companions {
		if !c.Dead() {
			members = append(members, domain.MemberAlignment{ID: c.ID, Alignment: m.companionAlignment(c)})
		}
	}
	i := -1
	for _, c := range record.Companions {
		if c.Dead() {
			lines = append(lines, fmt.Sprintf("  #%d %s: fallen", c.ID, templateName(c.MobTemplateID, strconv.Itoa(c.MobTemplateID))))
			continue
		}
		i++
		loyalty := domain.MaxLoyalty
		if c.Disposition != nil {
			loyalty = c.Disposition.Loyalty
		}
		mood := "content"
		if online {
			gap := members[i].Alignment - domain.RestAverage(leader, members, i)
			if gap < 0 {
				gap = -gap
			}
			if gap > rules.LoyaltyToleranceGap {
				mood = "uneasy"
			}
		}
		lines = append(lines, fmt.Sprintf("  #%d %s: %s, loyalty %d, %s", c.ID, templateName(c.MobTemplateID, strconv.Itoa(c.MobTemplateID)), alignmentLabel(members[i].Alignment), loyalty, mood))
	}
	lines = append(lines, "Companions drift toward the rest of the company. One far from it loses loyalty, and at none, deserts.")
	return strings.Join(lines, "\n")
}

var _ domain.AlignmentProvider = (*CompanyModule)(nil)

// CompanyAlignment implements domain.AlignmentProvider (Phase 21b): the
// company average the recruit gate uses. Unavailable while company data
// failed to load, so nothing is judged against a partial company.
func (m *CompanyModule) CompanyAlignment(leaderUserID int) (int, bool) {
	if m.persistenceAvailable() != nil {
		return 0, false
	}
	return m.companyAverage(leaderUserID)
}
