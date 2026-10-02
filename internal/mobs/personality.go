package mobs

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/races"
)

// Personality is the mob's personality as an enemy (Ashveil Phase 30c):
// the target rule it re-aims by and the percent of re-aims that take a
// random foe. Its template's targeting wins over its race's. ok is false
// when neither sets one: it aims at the weakest, as before.
func (m *Mob) Personality() (rule string, noise int, ok bool) {
	rule, noise = m.Targeting, m.TargetingNoise
	if strings.TrimSpace(rule) == "" {
		r := races.GetRace(m.Character.RaceId)
		if r == nil {
			return "", 0, false
		}
		rule, noise = r.Targeting, r.TargetingNoise
	}
	rule = strings.ToLower(strings.TrimSpace(rule))
	if rule == "" {
		return "", 0, false
	}
	return rule, min(max(noise, 0), 100), true
}

// EnemyRole is the mob's role as an enemy (Ashveil Phase 33i2): its
// template's role, lower case, or "fighter" when it gives none.
func (m *Mob) EnemyRole() string {
	r := strings.ToLower(strings.TrimSpace(m.Role))
	if r == "" {
		return "fighter"
	}
	return r
}

// TakesWounds reports whether the mob takes wounds as an enemy (Ashveil
// Phase 33i2): all do, light ones only, unless its template says
// `wounds: none`.
func (m *Mob) TakesWounds() bool {
	return !strings.EqualFold(strings.TrimSpace(m.WoundsRule), "none")
}
