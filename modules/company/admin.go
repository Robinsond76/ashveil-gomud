package company

// The admin test area's company tools (modules/testarea): recruit a
// companion of any class at any level and change a companion's class or
// level. They bypass the recruit rules on purpose and are reached only by an
// admin command; nothing here is a player path.

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/classes"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/creatures"
	"github.com/GoMudEngine/GoMud/internal/users"
)

var _ domain.AdminProvider = (*CompanyModule)(nil)

// maxAdminLevel caps the levels the admin tools set.
const maxAdminLevel = 100

// adminClass resolves a base archetype id or a class id to the archetype a
// companion of it has and the class record (blank for a base archetype).
func adminClass(id string) (archetype, class string, ok bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	if c, found := classes.Get(id); found {
		return c.Lineage, c.ID, true
	}
	if archetypes.Exists(id) {
		return id, "", true
	}
	return "", "", false
}

// templateFor is the companion template configured for an archetype: the
// generated-recruit templates (80 and up) first, lowest id first.
func (m *CompanyModule) templateFor(archetype string) (int, bool) {
	var ids []int
	for id, a := range m.companionArchetypes() {
		if a == archetype {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return 0, false
	}
	slices.Sort(ids)
	for _, id := range ids {
		if id >= 80 {
			return id, true
		}
	}
	return ids[0], true
}

// AdminRecruit implements domain.AdminProvider.
func (m *CompanyModule) AdminRecruit(leaderUserID, roomID int, class string, level int) (string, error) {
	if err := m.persistenceAvailable(); err != nil {
		return "", err
	}
	archetype, classID, ok := adminClass(class)
	if !ok {
		return "", fmt.Errorf("there is no archetype or class called %q", class)
	}
	templateID, ok := m.templateFor(archetype)
	if !ok {
		return "", fmt.Errorf("no companion template is configured for %s", archetype)
	}
	if record, has := m.registry.Get(leaderUserID); has && len(record.Companions) >= m.maxCompanions() {
		return "", domain.ErrCompanyFull
	}
	name := ""
	if !creatures.Is(archetype) {
		name = m.adminRecruitName(leaderUserID)
	}
	companion, err := m.enlistNamed(leaderUserID, roomID, templateID, map[int]struct{}{templateID: {}}, false, nil, name)
	if err != nil {
		return "", err
	}
	if classID == "" && level <= 0 {
		return fmt.Sprintf("Recruited %s (#%d), a %s.", nameOf(companion, templateName(templateID, "a companion")), companion.ID, archetype), nil
	}
	text, err := m.AdminSetMember(leaderUserID, roomID, fmt.Sprintf("#%d", companion.ID), classID, level)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Recruited %s (#%d), a %s. %s", nameOf(companion, templateName(templateID, "a companion")), companion.ID, archetype, text), nil
}

// AdminSetMember implements domain.AdminProvider. A base archetype id sets
// the archetype and clears the class; a class id sets both. Gear is kept.
func (m *CompanyModule) AdminSetMember(leaderUserID, roomID int, selector, class string, level int) (string, error) {
	if err := m.persistenceAvailable(); err != nil {
		return "", err
	}
	if class == "" && level <= 0 {
		return "", errors.New("give a class, a level, or both")
	}
	if level > maxAdminLevel {
		return "", fmt.Errorf("level is at most %d", maxAdminLevel)
	}
	var archetype, classID string
	if class != "" {
		var ok bool
		if archetype, classID, ok = adminClass(class); !ok {
			return "", fmt.Errorf("there is no archetype or class called %q", class)
		}
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return "", domain.ErrUnknownMember
	}
	companion, ok := resolveCompanion(record, selector)
	if !ok {
		return "", fmt.Errorf("no companion matches %q", selector)
	}
	// Phase 38e review: a creature's body is its template (a canine's bites,
	// a golem's stone), so a person can't become one and a creature can't
	// become a person; recruit the other kind instead.
	if archetype != "" && archetype != companion.Archetype && (creatures.Is(archetype) || creatures.Is(companion.Archetype)) {
		return "", fmt.Errorf("a creature keeps its species and a person can't become one (they can't trade places): recruit one with testarea companion add %s", archetype)
	}
	// What the live mob carries is the truth: record it before the respawn.
	m.refreshSnapshot(leaderUserID, companion.ID)
	record, _ = m.registry.Get(leaderUserID)
	companion, _ = resolveCompanion(record, fmt.Sprintf("#%d", companion.ID))
	state, err := m.ensureState(leaderUserID, companion)
	if err != nil {
		return "", err
	}
	if state == nil {
		return "", fmt.Errorf("companion #%d has no template to build from", companion.ID)
	}
	before := record
	for i := range record.Companions {
		if record.Companions[i].ID != companion.ID {
			continue
		}
		if archetype != "" {
			record.Companions[i].Archetype = archetype
			record.Companions[i].Class = classID
			record.Companions[i].Talents = nil
		}
		if level > 0 {
			s := state.Clone()
			s.Level, s.Experience, s.Vitals = level, 0, nil
			record.Companions[i].State = &s
		}
	}
	m.registry.Put(record)
	if err := m.save(); err != nil {
		m.registry.Put(before)
		return "", err
	}
	if instanceID, tracked := m.instance(leaderUserID, companion.ID); tracked {
		if m.runtime.IsLive(instanceID) {
			m.runtime.Detach(leaderUserID, instanceID)
		}
		m.clearInstance(leaderUserID, companion.ID)
	}
	if err := m.restoreForLeader(leaderUserID, roomID); err != nil {
		return "", err
	}
	var parts []string
	if archetype != "" {
		label := archetype
		if classID != "" {
			label = classID
		}
		parts = append(parts, "is now a "+label)
	}
	if level > 0 {
		parts = append(parts, fmt.Sprintf("is now level %d", level))
	}
	return fmt.Sprintf("%s (#%d) %s.", nameOf(companion, templateName(companion.MobTemplateID, "The companion")), companion.ID, strings.Join(parts, " and ")), nil
}

// adminRecruitName is a random name for a test-area recruit, from the
// generated-recruit name lists: its given name is free in the company, at
// the recruiters and among online players, and is never the leader's.
// Blank when the lists are empty or every name is taken (the template's
// name stays).
func (m *CompanyModule) adminRecruitName(leaderUserID int) string {
	rules := m.rosterRules()
	taken := map[string]bool{}
	if record, ok := m.registry.Get(leaderUserID); ok {
		for _, c := range record.Companions {
			taken[givenKey(nameOf(c, ""))] = true
		}
	}
	for _, r := range m.recruiters() {
		for _, c := range r.Candidates {
			taken[c.ID] = true
			taken[givenKey(templateName(c.MobTemplateID, ""))] = true
		}
	}
	for _, u := range users.GetAllActiveUsers() {
		if u != nil && u.Character != nil {
			taken[givenKey(u.Character.Name)] = true
		}
	}
	if leader := users.GetByUserId(leaderUserID); leader != nil && leader.Character != nil {
		taken[givenKey(leader.Character.Name)] = true
	}
	var free []string
	for _, n := range rules.GivenNames {
		if n = strings.TrimSpace(n); n != "" && !taken[givenKey(n)] {
			free = append(free, n)
		}
	}
	if len(free) == 0 {
		return ""
	}
	rng := m.random()
	name := free[rng.Intn(len(free))]
	if len(rules.Bynames) > 0 && rng.Intn(100) < rules.BynamePercent {
		if by := strings.TrimSpace(rules.Bynames[rng.Intn(len(rules.Bynames))]); by != "" {
			name += " " + by
		}
	}
	return name
}
