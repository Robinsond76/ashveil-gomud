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
	"math"
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
	HPPerLevel  float64 // HP per level through the configured HPFullLevels. Zero uses the progression default.
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
	// Kit is the item ids granted once, as a starter kit, to a player who
	// chooses this archetype (Phase 22a). Repeat an id to grant it twice.
	Kit []int
	// CompanionSpells are the spells a companion of this archetype knows,
	// each from a character level (Phase 32d). Nothing is written to the
	// companion: the list is read when it acts in a battle.
	CompanionSpells []LevelSpell
	// Growth weights a companion's stat points by stat name (Phase 33h1):
	// strength, speed, smarts, vitality, mysticism, perception. Empty
	// deals them evenly.
	Growth map[string]int
	// Phase 35a2: Attack and Evasion gained a level (zero uses the combat
	// defaults), the head start in health, the heaviest armor bulk worn
	// without penalty (empty: heavy), the shield sizes it may carry (empty:
	// any; "none": no shield) and the weapon classes it may wield (empty:
	// any).
	AttackRate    float64
	EvasionRate   float64
	HPStart       int
	ArmorTraining string
	ShieldSizes   []string
	WeaponClasses []string
	// Phase 35b: the mana pool's base and gain a level (zero uses the
	// progression defaults), and the spells a player of this archetype
	// learns from a character level.
	ManaBase     int
	ManaPerLevel float64
	LevelSpells  []LevelSpell
}

// GrowthStatNames are the stats a Growth weight may name.
var GrowthStatNames = []string{"strength", "speed", "smarts", "vitality", "mysticism", "perception"}

// LevelSpell is a spell known from a character level.
type LevelSpell struct {
	Spell string
	Level int
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
	if a.HPPerLevel < 0 || math.IsNaN(a.HPPerLevel) || math.IsInf(a.HPPerLevel, 0) {
		return fmt.Errorf("%w: invalid HPPerLevel", ErrInvalidArchetype)
	}
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
	kit := make([]int, 0, len(a.Kit))
	for _, id := range a.Kit {
		if id > 0 {
			kit = append(kit, id)
		}
	}
	a.Kit = kit
	spells := make([]LevelSpell, 0, len(a.CompanionSpells))
	for _, ls := range a.CompanionSpells {
		ls.Spell = strings.ToLower(strings.TrimSpace(ls.Spell))
		if ls.Spell == "" || ls.Level < 1 {
			return fmt.Errorf("%w: %q has a companion spell %q at level %d", ErrInvalidArchetype, a.ID, ls.Spell, ls.Level)
		}
		spells = append(spells, ls)
	}
	a.CompanionSpells = spells
	levelSpells := make([]LevelSpell, 0, len(a.LevelSpells))
	for _, ls := range a.LevelSpells {
		ls.Spell = strings.ToLower(strings.TrimSpace(ls.Spell))
		if ls.Spell == "" || ls.Level < 1 {
			return fmt.Errorf("%w: %q has a level spell %q at level %d", ErrInvalidArchetype, a.ID, ls.Spell, ls.Level)
		}
		levelSpells = append(levelSpells, ls)
	}
	a.LevelSpells = levelSpells
	growth := make(map[string]int, len(a.Growth))
	for stat, weight := range a.Growth {
		stat = strings.ToLower(strings.TrimSpace(stat))
		if !contains(GrowthStatNames, stat) || weight < 1 {
			return fmt.Errorf("%w: %q has growth %q weight %d", ErrInvalidArchetype, a.ID, stat, weight)
		}
		growth[stat] += weight
	}
	a.Growth = growth
	if err := a.validateCombat(); err != nil {
		return err
	}
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

// FilterCompanionSpells keeps the companion spells that exist and that
// this archetype may learn (their school is empty or claimed), and reports
// each one dropped.
func (a *Archetype) FilterCompanionSpells(schoolOf func(spellID string) (school string, ok bool)) []error {
	var errs []error
	kept := a.CompanionSpells[:0:0]
	for _, ls := range a.CompanionSpells {
		school, ok := schoolOf(ls.Spell)
		if !ok {
			errs = append(errs, fmt.Errorf("%w: %q lists unknown companion spell %q", ErrInvalidArchetype, a.ID, ls.Spell))
			continue
		}
		school = strings.ToLower(strings.TrimSpace(school))
		if school != "" && !contains(a.Schools, school) {
			errs = append(errs, fmt.Errorf("%w: %q lists companion spell %q from unclaimed school %q", ErrInvalidArchetype, a.ID, ls.Spell, school))
			continue
		}
		kept = append(kept, ls)
	}
	a.CompanionSpells = kept
	return errs
}

// FilterLevelSpells keeps the level spells that exist and that this
// archetype may learn (Phase 35b), and reports each one dropped.
func (a *Archetype) FilterLevelSpells(schoolOf func(spellID string) (school string, ok bool)) []error {
	var errs []error
	kept := a.LevelSpells[:0:0]
	for _, ls := range a.LevelSpells {
		school, ok := schoolOf(ls.Spell)
		if !ok {
			errs = append(errs, fmt.Errorf("%w: %q lists unknown level spell %q", ErrInvalidArchetype, a.ID, ls.Spell))
			continue
		}
		school = strings.ToLower(strings.TrimSpace(school))
		if school != "" && !contains(a.Schools, school) {
			errs = append(errs, fmt.Errorf("%w: %q lists level spell %q from unclaimed school %q", ErrInvalidArchetype, a.ID, ls.Spell, school))
			continue
		}
		kept = append(kept, ls)
	}
	a.LevelSpells = kept
	return errs
}

// PlayerSpellsAtLevel is the level spells a player of this archetype owns
// at a character level (Phase 35b), in the order configured.
func (a Archetype) PlayerSpellsAtLevel(level int) []string {
	var out []string
	for _, ls := range a.LevelSpells {
		if level >= ls.Level {
			out = append(out, ls.Spell)
		}
	}
	return out
}

// SpellsAtLevel is the companion spells known at a character level, in
// the order configured.
func (a Archetype) SpellsAtLevel(level int) []string {
	var out []string
	for _, ls := range a.CompanionSpells {
		if level >= ls.Level {
			out = append(out, ls.Spell)
		}
	}
	return out
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
