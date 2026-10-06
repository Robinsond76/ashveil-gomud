package classes

// Base ranks (Phase 39b): what every character of a neutral lineage gains by
// level before it promotes, such as the Samurai's Iaijutsu. The faith
// lineages gain theirs from spells and abilities; a neutral lineage's signature
// is a class effect, so it lives here. Like a route's ranks they are derived
// from the level and never saved.

var baseRanks = map[string][]Rank{}

// registerBase adds a lineage's base ranks, in level order.
func registerBase(lineageID string, ranks ...Rank) {
	lineageID = normalize(lineageID)
	baseRanks[lineageID] = append(baseRanks[lineageID], ranks...)
}

// HasBase reports whether a lineage has base ranks.
func HasBase(lineageID string) bool { return len(baseRanks[normalize(lineageID)]) > 0 }

// BaseRanks are a lineage's base ranks, in level order.
func BaseRanks(lineageID string) []Rank {
	return append([]Rank(nil), baseRanks[normalize(lineageID)]...)
}

// BaseRanksReached are the base ranks a character of a lineage has earned at
// a level.
func BaseRanksReached(lineageID string, level int) []Rank {
	var out []Rank
	for _, r := range baseRanks[normalize(lineageID)] {
		if level >= r.Level {
			out = append(out, r)
		}
	}
	return out
}

// EffectsForLineage is EffectsFor with the lineage's base ranks applied
// first: base ranks, then the class route's ranks over them, then talents.
func EffectsForLineage(lineageID, classID string, level int, talentIDs []string) Effects {
	out := Effects{}
	for _, r := range BaseRanksReached(lineageID, level) {
		for k, v := range r.Set {
			out[k] = v
		}
	}
	for _, c := range Path(classID) {
		for _, r := range c.Ranks {
			if level >= r.Level {
				for k, v := range r.Set {
					out[k] = v
				}
			}
		}
	}
	for _, id := range ActiveTalents(level, talentIDs) {
		if t, ok := TalentByID(id); ok {
			for k, v := range t.Add {
				out[k] += v
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
