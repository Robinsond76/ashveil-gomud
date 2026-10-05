package archetypes

import (
	"fmt"
	"sort"
	"strings"
)

// Phase 35c: optional skills. Companions learn these with training points
// the leader spends ("company train"); class skills and specialists stay
// derived from the companion's archetype and level.

// AnyArchetype in an optional skill's Archetypes lets every companion,
// including one with no archetype yet, learn it.
const AnyArchetype = "*"

// OptionalSkill is one skill a companion may be trained in.
type OptionalSkill struct {
	Skill string
	// Archetypes may learn it; AnyArchetype means everyone.
	Archetypes []string
	// MaxRank is the highest rank a companion may train (1..4).
	MaxRank int
}

// Validate normalizes the ids and rejects a malformed entry.
func (o *OptionalSkill) Validate() error {
	o.Skill = strings.ToLower(strings.TrimSpace(o.Skill))
	if !idRegex.MatchString(o.Skill) {
		return fmt.Errorf("%w: optional skill %q", ErrInvalidArchetype, o.Skill)
	}
	o.Archetypes = normalizeList(o.Archetypes)
	if len(o.Archetypes) == 0 {
		return fmt.Errorf("%w: optional skill %q names no archetypes", ErrInvalidArchetype, o.Skill)
	}
	for _, a := range o.Archetypes {
		if a != AnyArchetype && !idRegex.MatchString(a) {
			return fmt.Errorf("%w: optional skill %q names archetype %q", ErrInvalidArchetype, o.Skill, a)
		}
	}
	if o.MaxRank < 1 || o.MaxRank > SkillLevels {
		return fmt.Errorf("%w: optional skill %q has max rank %d", ErrInvalidArchetype, o.Skill, o.MaxRank)
	}
	return nil
}

// Allows reports whether a companion of the archetype ("" = none yet) may
// learn the skill.
func (o OptionalSkill) Allows(archetypeID string) bool {
	archetypeID = strings.ToLower(strings.TrimSpace(archetypeID))
	for _, a := range o.Archetypes {
		if a == AnyArchetype || (archetypeID != "" && a == archetypeID) {
			return true
		}
	}
	return false
}

// NewOptionalSkills validates each entry, skipping (and reporting) invalid
// and duplicate ones, sorted by skill.
func NewOptionalSkills(list []OptionalSkill) ([]OptionalSkill, []error) {
	var errs []error
	seen := map[string]bool{}
	out := make([]OptionalSkill, 0, len(list))
	for _, o := range list {
		if err := o.Validate(); err != nil {
			errs = append(errs, err)
			continue
		}
		if seen[o.Skill] {
			errs = append(errs, fmt.Errorf("%w: duplicate optional skill %q", ErrInvalidArchetype, o.Skill))
			continue
		}
		seen[o.Skill] = true
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Skill < out[j].Skill })
	return out, errs
}

// OptionalSkillProvider is optionally implemented by the provider (35c).
type OptionalSkillProvider interface {
	// OptionalSkills is the configured list, sorted by skill. Callers may
	// keep it: it is a copy.
	OptionalSkills() []OptionalSkill
}

// OptionalSkills is the configured optional skills; nil without a provider.
func OptionalSkills() []OptionalSkill {
	if op, ok := current().(OptionalSkillProvider); ok {
		return op.OptionalSkills()
	}
	return nil
}

// Optional looks up one optional skill by id.
func Optional(skillID string) (OptionalSkill, bool) {
	skillID = strings.ToLower(strings.TrimSpace(skillID))
	for _, o := range OptionalSkills() {
		if o.Skill == skillID {
			return o, true
		}
	}
	return OptionalSkill{}, false
}

// CanLearn reports whether a companion of the archetype may train the
// optional skill. A skill not on the list is never trainable.
func CanLearn(archetypeID, skillID string) bool {
	o, ok := Optional(skillID)
	return ok && o.Allows(archetypeID)
}

// LearnableSkills is the optional skills a companion of the archetype may
// train, sorted.
func LearnableSkills(archetypeID string) []string {
	var out []string
	for _, o := range OptionalSkills() {
		if o.Allows(archetypeID) {
			out = append(out, o.Skill)
		}
	}
	return out
}
