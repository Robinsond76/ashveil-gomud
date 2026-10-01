package skills

// Ashveil 33f1 and 33f2 (search): skills retired from the game. A
// character still holding one is refunded the training points it cost and the entry is removed
// (characters.Character.RetireSkills, run at login).
var retired = []string{`changeform`, `peep`, `portal`, `scribe`, `search`, `tame`}

// cappedLevels are skills whose top levels were retired with the ability
// they granted (protection 4 granted pray): a higher level is refunded down
// to the cap.
var cappedLevels = map[string]int{`protection`: 3}

// Retired lists the retired skill ids.
func Retired() []string {
	return append([]string{}, retired...)
}

// LevelCap is the highest level a skill keeps after a retirement, and
// whether it has one.
func LevelCap(skillID string) (int, bool) {
	cap, ok := cappedLevels[skillID]
	return cap, ok
}

// CappedSkills lists the skills with a retirement level cap, sorted.
func CappedSkills() []string {
	return []string{`protection`}
}

// TrainingCost is the training points a skill at level costs in total:
// 1 + 2 + ... + level (each level n costs n, as train charges).
func TrainingCost(level int) int {
	if level <= 0 {
		return 0
	}
	return level * (level + 1) / 2
}
