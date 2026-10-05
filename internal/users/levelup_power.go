package users

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/strategy"
)

// Phase 35b: the level-up report names what each owned scaling spell and
// ability does now, from the same numbers the spells roll
// (internal/spellpower).

// powerEntry is one scaling spell's or ability's size, as the report shows
// it ("8-13", "+3 damage").
type powerEntry struct {
	name, size string
}

// powerSnapshot lists the character's scaling spells (those with a power
// block, by name) and damage abilities, in a stable order.
func powerSnapshot(c *characters.Character) []powerEntry {
	var out []powerEntry
	ids := make([]string, 0, len(c.SpellBook))
	for id := range c.SpellBook {
		if c.HasSpell(id) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		sp := spells.GetSpell(id)
		if sp == nil || sp.Power == nil {
			continue
		}
		lo, hi := sp.Power.Range(c.Level, c.Stats.Mysticism.ValueAdj)
		out = append(out, powerEntry{name: sp.Name, size: fmt.Sprintf("%d-%d", lo, hi)})
	}
	for _, id := range strategy.PlayerAbilities(c.GetSkillLevel) {
		switch id {
		case strategy.OpeningStrike:
			out = append(out, powerEntry{name: "Opening Strike", size: fmt.Sprintf("+%d damage", strategy.OpeningStrikeBonus(c.Level))})
		case strategy.AimedShot:
			out = append(out, powerEntry{name: "Aimed Shot", size: fmt.Sprintf("+%d damage", strategy.AimedShotBonus(c.Level))})
		}
	}
	return out
}

// powerLines are the report's lines for what grew: "Magic Missile 8-13 ->
// 9-14". A spell first owned at this level shows its size alone.
func powerLines(before, after []powerEntry) []string {
	was := map[string]string{}
	for _, e := range before {
		was[e.name] = e.size
	}
	var out []string
	for _, e := range after {
		old, had := was[e.name]
		switch {
		case !had:
			out = append(out, fmt.Sprintf("%s %s", e.name, e.size))
		case old != e.size:
			out = append(out, fmt.Sprintf("%s %s -> %s", e.name, old, e.size))
		}
	}
	return out
}
