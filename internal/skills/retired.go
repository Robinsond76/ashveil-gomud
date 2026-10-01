package skills

// Ashveil 33f1: skills retired from the game. A character still holding one
// is refunded the training points it cost and the entry is removed
// (characters.Character.RetireSkills, run at login).
var retired = []string{`changeform`, `peep`, `portal`, `scribe`, `tame`}

// Retired lists the retired skill ids.
func Retired() []string {
	return append([]string{}, retired...)
}

// TrainingCost is the training points a skill at level costs in total:
// 1 + 2 + ... + level (each level n costs n, as train charges).
func TrainingCost(level int) int {
	if level <= 0 {
		return 0
	}
	return level * (level + 1) / 2
}
