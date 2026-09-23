// Package archetypes contains the GoMud-free archetype table: exclusive
// character archetypes (warrior, rogue, wizard, ...) that claim skills and
// spell schools, the gating decisions derived from those claims, and the
// read-only seam the engine consults. A skill or school no archetype claims
// is open to everyone (a "trade skill").
//
// The table itself is module config (modules/archetype); this package only
// validates and answers questions about it.
package archetypes

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// SkillLevels is how many companion levels an archetype table carries: one
// character-level threshold per skill level 1..4.
const SkillLevels = 4

var (
	ErrInvalidArchetype = errors.New("archetypes: invalid archetype")
	idRegex             = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
)

// Archetype is one configured, exclusive archetype.
type Archetype struct {
	ID          string
	Name        string
	Description string
	// Skills this archetype claims. Only archetypes listing a skill may
	// train it; several archetypes may list the same skill.
	Skills []string
	// Schools of spells this archetype claims for learning.
	Schools []string
	// GrantSkills and GrantSpells are applied once, when a player chooses
	// this archetype.
	GrantSkills map[string]int
	GrantSpells []string
	// Utility actions (e.g. "light", "traps") this archetype performs.
	Utility []string
	// CompanionLevels are the character levels at which a companion of this
	// archetype reaches utility skill levels 1..4.
	CompanionLevels []int
}

func normalizeList(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func contains(list []string, v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

// Validate normalizes ids to lowercase and rejects a malformed archetype
// rather than guessing at it.
func (a *Archetype) Validate() error {
	a.ID = strings.ToLower(strings.TrimSpace(a.ID))
	if !idRegex.MatchString(a.ID) {
		return fmt.Errorf("%w: id %q", ErrInvalidArchetype, a.ID)
	}
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("%w: %q has no name", ErrInvalidArchetype, a.ID)
	}
	a.Skills = normalizeList(a.Skills)
	a.Schools = normalizeList(a.Schools)
	a.Utility = normalizeList(a.Utility)
	a.GrantSpells = normalizeList(a.GrantSpells)
	if len(a.Skills) == 0 {
		return fmt.Errorf("%w: %q claims no skills", ErrInvalidArchetype, a.ID)
	}
	grants := make(map[string]int, len(a.GrantSkills))
	for skill, level := range a.GrantSkills {
		skill = strings.ToLower(strings.TrimSpace(skill))
		if !contains(a.Skills, skill) {
			return fmt.Errorf("%w: %q grants unlisted skill %q", ErrInvalidArchetype, a.ID, skill)
		}
		if level < 1 {
			return fmt.Errorf("%w: %q grants %q at level %d", ErrInvalidArchetype, a.ID, skill, level)
		}
		grants[skill] = level
	}
	a.GrantSkills = grants
	if len(a.CompanionLevels) != SkillLevels {
		return fmt.Errorf("%w: %q needs %d companion levels", ErrInvalidArchetype, a.ID, SkillLevels)
	}
	for i, level := range a.CompanionLevels {
		if level < 1 || (i > 0 && level <= a.CompanionLevels[i-1]) {
			return fmt.Errorf("%w: %q companion levels must ascend from 1", ErrInvalidArchetype, a.ID)
		}
	}
	return nil
}

// ValidateGrantedSpells checks every granted spell exists and is learnable
// by this archetype: its school is empty or one this archetype claims.
func (a Archetype) ValidateGrantedSpells(schoolOf func(spellID string) (school string, ok bool)) error {
	for _, spell := range a.GrantSpells {
		school, ok := schoolOf(spell)
		if !ok {
			return fmt.Errorf("%w: %q grants unknown spell %q", ErrInvalidArchetype, a.ID, spell)
		}
		school = strings.ToLower(strings.TrimSpace(school))
		if school != "" && !contains(a.Schools, school) {
			return fmt.Errorf("%w: %q grants %q from unclaimed school %q", ErrInvalidArchetype, a.ID, spell, school)
		}
	}
	return nil
}

// ListsSkill reports whether this archetype claims the skill.
func (a Archetype) ListsSkill(skillID string) bool { return contains(a.Skills, skillID) }

// HasUtility reports whether this archetype performs the utility action.
func (a Archetype) HasUtility(utility string) bool { return contains(a.Utility, utility) }

// CompanionSkillLevel is the utility skill level (0..4) a companion of this
// archetype has at the given character level.
func (a Archetype) CompanionSkillLevel(characterLevel int) int {
	level := 0
	for i, threshold := range a.CompanionLevels {
		if characterLevel >= threshold {
			level = i + 1
		}
	}
	return level
}

// Table is a validated, immutable archetype table with claim indexes.
type Table struct {
	byID    map[string]Archetype
	skills  map[string][]string
	schools map[string][]string
}

// NewTable validates each archetype, skipping (and reporting) invalid or
// duplicate entries, and indexes skill and school claims.
func NewTable(list []Archetype) (Table, []error) {
	t := Table{byID: map[string]Archetype{}, skills: map[string][]string{}, schools: map[string][]string{}}
	var errs []error
	for _, a := range list {
		if err := a.Validate(); err != nil {
			errs = append(errs, err)
			continue
		}
		if _, dup := t.byID[a.ID]; dup {
			errs = append(errs, fmt.Errorf("%w: duplicate %q", ErrInvalidArchetype, a.ID))
			continue
		}
		t.byID[a.ID] = a
		for _, s := range a.Skills {
			t.skills[s] = append(t.skills[s], a.ID)
		}
		for _, s := range a.Schools {
			t.schools[s] = append(t.schools[s], a.ID)
		}
	}
	for _, m := range []map[string][]string{t.skills, t.schools} {
		for k := range m {
			sort.Strings(m[k])
		}
	}
	return t, errs
}

// Get returns an archetype by id.
func (t Table) Get(id string) (Archetype, bool) {
	a, ok := t.byID[strings.ToLower(strings.TrimSpace(id))]
	return a, ok
}

// List returns every archetype sorted by id.
func (t Table) List() []Archetype {
	out := make([]Archetype, 0, len(t.byID))
	for _, a := range t.byID {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Len is the number of valid archetypes.
func (t Table) Len() int { return len(t.byID) }

// SkillClaimants lists the archetypes claiming a skill, sorted.
func (t Table) SkillClaimants(skillID string) []string {
	return append([]string(nil), t.skills[strings.ToLower(strings.TrimSpace(skillID))]...)
}

// SkillClaimed reports whether any archetype claims the skill.
func (t Table) SkillClaimed(skillID string) bool { return len(t.SkillClaimants(skillID)) > 0 }

// SchoolClaimants lists the archetypes claiming a spell school, sorted.
func (t Table) SchoolClaimants(school string) []string {
	return append([]string(nil), t.schools[strings.ToLower(strings.TrimSpace(school))]...)
}

func (t Table) namesOf(ids []string) string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if a, ok := t.byID[id]; ok {
			names = append(names, a.Name)
		}
	}
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " or " + names[len(names)-1]
}

// SkillClaimantNames is the display names of a skill's claimants, joined
// for prose ("Cleric or Wizard"); "" for an unclaimed skill.
func (t Table) SkillClaimantNames(skillID string) string {
	return t.namesOf(t.SkillClaimants(skillID))
}

// claimDecision is the shared rule for skills and schools: open when
// unclaimed; otherwise the character's (known) archetype must be a claimant.
func (t Table) claimDecision(archetypeID string, claimants []string, what string) (bool, string) {
	if len(claimants) == 0 {
		return true, ""
	}
	a, chosen := t.Get(archetypeID)
	if !chosen {
		return false, fmt.Sprintf(`Only a %s may learn %s. Choose an archetype first (see "archetype").`, t.namesOf(claimants), what)
	}
	for _, id := range claimants {
		if id == a.ID {
			return true, ""
		}
	}
	return false, fmt.Sprintf("Only a %s may learn %s; you are a %s.", t.namesOf(claimants), what, a.Name)
}

// CanTrain decides whether a character of archetypeID ("" = unchosen) may
// train a skill. The reason is player-facing.
func (t Table) CanTrain(archetypeID, skillID string) (bool, string) {
	return t.claimDecision(archetypeID, t.SkillClaimants(skillID), strings.ToLower(strings.TrimSpace(skillID)))
}

// CanLearnSchool decides whether a character of archetypeID may learn a
// spell of the given school ("" = no school, always open).
func (t Table) CanLearnSchool(archetypeID, school string) (bool, string) {
	school = strings.ToLower(strings.TrimSpace(school))
	return t.claimDecision(archetypeID, t.SchoolClaimants(school), school+" magic")
}
