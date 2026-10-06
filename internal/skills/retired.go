package skills

// Ashveil 33f1 and 33f2 (search, trading): skills retired from the game
// (scribe was retired too, and returns in 36a: see ScribeReset). A
// character still holding one is refunded the training points it cost and the entry is removed
// (characters.Character.RetireSkills, run at login).
var retired = []string{`changeform`, `peep`, `portal`, `search`, `tame`, `trading`}

// ScribeReset (Ashveil 36a): `scribe` was retired in 33f1 and returns as a
// new caster skill. A character that has not been reset yet gives up any old
// rank for a refund, once (characters.Character.ScribeReset marks it), so
// nobody keeps a rank they did not train for the new skill.
const ScribeReset = `scribe`

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
