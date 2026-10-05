package company

import "strings"

// Phase 35c companion training. A companion's training points are derived,
// never banked: what its level has earned less what its trained ranks cost.
// Ranks it arrived with (GrantedSkills) cost nothing. Nothing but the ranks
// is saved, so death, regain, re-summon, resurrection and copyover can't
// mint points; a level lost to death leaves the ranks and a shortfall that
// later levels repay first.

// SkillRankCost is what one rank costs: the rank itself (1+2+3+4), as a
// player pays at a trainer.
func SkillRankCost(rank int) int { return max(rank, 0) }

// RanksCost is what training a skill from rank `from` up to rank `to` costs.
func RanksCost(from, to int) int {
	total := 0
	for r := max(from, 0) + 1; r <= to; r++ {
		total += SkillRankCost(r)
	}
	return total
}

// SpentTrainingPoints is what the companion's trained ranks above the
// granted ones have cost.
func (c Companion) SpentTrainingPoints() int {
	total := 0
	for skill, rank := range c.Skills {
		total += RanksCost(c.GrantedSkills[skill], rank)
	}
	return total
}

// TrainingPoints is the points left from what its level earned; it is
// negative after a lost level, until later levels repay it.
func (c Companion) TrainingPoints(earned int) int {
	return earned - c.SpentTrainingPoints()
}

// SkillRank is the companion's rank in an optional skill.
func (c Companion) SkillRank(skill string) int {
	return c.Skills[strings.ToLower(strings.TrimSpace(skill))]
}

// SetSkillRank records a companion's rank in a skill (never lower than a
// granted rank). It returns ErrUnknownMember for a leader or companion not
// in the registry.
func (r *Registry) SetSkillRank(leaderUserID, companionID int, skill string, rank int) error {
	skill = strings.ToLower(strings.TrimSpace(skill))
	record, ok := r.Get(leaderUserID)
	if !ok {
		return ErrUnknownMember
	}
	for i, c := range record.Companions {
		if c.ID != companionID {
			continue
		}
		rank = max(rank, c.GrantedSkills[skill])
		if record.Companions[i].Skills == nil {
			record.Companions[i].Skills = map[string]int{}
		}
		if rank <= 0 {
			delete(record.Companions[i].Skills, skill)
		} else {
			record.Companions[i].Skills[skill] = rank
		}
		if len(record.Companions[i].Skills) == 0 {
			record.Companions[i].Skills = nil
		}
		r.Put(record)
		return nil
	}
	return ErrUnknownMember
}

// GrantSkills gives a new companion the ranks it arrived with: they go in
// both Skills and GrantedSkills, so they cost no points.
func (r *Registry) GrantSkills(leaderUserID, companionID int, ranks map[string]int) error {
	record, ok := r.Get(leaderUserID)
	if !ok {
		return ErrUnknownMember
	}
	for i, c := range record.Companions {
		if c.ID != companionID {
			continue
		}
		for skill, rank := range ranks {
			skill = strings.ToLower(strings.TrimSpace(skill))
			if skill == "" || rank <= 0 {
				continue
			}
			if record.Companions[i].Skills == nil {
				record.Companions[i].Skills = map[string]int{}
			}
			if record.Companions[i].GrantedSkills == nil {
				record.Companions[i].GrantedSkills = map[string]int{}
			}
			record.Companions[i].Skills[skill] = max(record.Companions[i].Skills[skill], rank)
			record.Companions[i].GrantedSkills[skill] = max(record.Companions[i].GrantedSkills[skill], rank)
		}
		r.Put(record)
		return nil
	}
	return ErrUnknownMember
}

func cloneRanks(in map[string]int) map[string]int {
	if in == nil {
		return nil
	}
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
