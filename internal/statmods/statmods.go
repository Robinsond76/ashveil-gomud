package statmods

// This contains centralized structs and constants regarding statmods
// Statmods are found in buffs, items, etc.
// They are used to augment in-game stats, calculations, etc.

// Statmods are a simple map of "name" to "modifier"
type StatMods map[string]int
type StatName string

var (
	// specific skills
	Picklock StatName = `picklock`

	// Not an exhaustive list, but ideally keep track of
	RacialBonusPrefix StatName = `racial-bonus-`

	// any statnames/prefixes here
	Casting        StatName = `casting`        // also used for `casting-` prefix followed by spell School
	CastingPrefix  StatName = `casting-`       // followed by spell School
	XPScale        StatName = `xpscale`        // Used for scaling xp after kills by this %
	HealthRecovery StatName = `healthrecovery` // When recovering HP naturally, recover this much extra
	ManaRecovery   StatName = `manarecovery`   // When recovering MP naturally, recover this much extra

	// Combat
	Attacks StatName = `attacks` // Additional attacks per combat round
	Damage  StatName = `damage`  // Flat bonus damage added to every hit

	// Stat based
	Strength   StatName = `strength`
	Speed      StatName = `speed`
	Smarts     StatName = `smarts`
	Vitality   StatName = `vitality`
	Mysticism  StatName = `mysticism`
	Perception StatName = `perception`
	HealthMax  StatName = `healthmax`
	ManaMax    StatName = `manamax`
)

var extraStatMods = map[StatName]string{}

func RegisterStatMod(name StatName, description string) {
	extraStatMods[name] = description
}

func GetStatMods() map[StatName]string {
	result := map[StatName]string{
		Picklock:          "Reduces the difficulty of a lock-picking attempt by this many pins.",
		RacialBonusPrefix: "Flat bonus damage against a specific race in combat. Format: `racial-bonus-giant spider`.",
		Casting:           "Increases spell casting success chance by this percentage.",
		CastingPrefix:     "Increases casting success chance for a specific school of magic. Format: `casting-restoration`.",
		XPScale:           "Scales experience gained from kills by this percentage (stacks additively with the server XPScale setting).",
		HealthRecovery:    "Extra HP recovered each round during natural regeneration.",
		ManaRecovery:      "Extra MP recovered each round during natural regeneration.",
		Attacks:           "Additional attacks granted per combat round.",
		Damage:            "Flat bonus damage added to every successful hit.",
		Strength:          "Increases the Strength stat, affecting melee damage and carrying capacity.",
		Speed:             "Increases the Speed stat, affecting hit chance, dodge, and attack frequency.",
		Smarts:            "Increases the Smarts stat, affecting spell power and skill learning.",
		Vitality:          "Increases the Vitality stat, affecting maximum HP and physical endurance.",
		Mysticism:         "Increases the Mysticism stat, affecting maximum MP and spell effectiveness.",
		Perception:        "Increases the Perception stat, affecting detection, awareness, and ranged accuracy.",
		HealthMax:         "Directly increases maximum HP (added after stat-based calculation).",
		ManaMax:           "Directly increases maximum MP (added after stat-based calculation).",
	}
	for k, v := range extraStatMods {
		result[k] = v
	}
	return result
}

func (s StatMods) Get(statName ...string) int {

	if len(s) == 0 {
		return 0
	}

	retAmt := 0

	for _, sn := range statName {
		if modAmt, ok := s[sn]; ok {
			retAmt += modAmt
		}
	}

	return retAmt
}

func (s StatMods) Add(statName string, statVal int) {
	if s == nil {
		s = make(StatMods)
	}

	if oldVal, ok := s[statName]; ok {
		s[statName] = oldVal + statVal
	} else {
		s[statName] = statVal
	}
}
