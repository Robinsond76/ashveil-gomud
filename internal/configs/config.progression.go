package configs

import "math"

type ProgressionConfig struct {
	// StatStepLevels sets the racial growth interval (levels 5, 10, ... by default).
	StatStepLevels ConfigInt `yaml:"StatStepLevels"`
	// DefaultHPPerLevel applies when no archetype or enemy override is available.
	DefaultHPPerLevel ConfigFloat `yaml:"DefaultHPPerLevel"`
	HPFullLevels      ConfigInt   `yaml:"HPFullLevels"`
	HPAfterFull       ConfigFloat `yaml:"HPAfterFull"`
	XPKneeLevel       ConfigInt   `yaml:"XPKneeLevel"`
	XPKneeGrowth      ConfigFloat `yaml:"XPKneeGrowth"`
	// Stat gain formula: GainsForLevel
	//   racial_value = floor(base * BaseModFactor * (step-1)^BaseModExponent)
	//                + floor(NaturalGainsModFactor * step^NaturalGainsExponent)
	//
	// BaseModFactor controls how much a racial base stat matters at high levels.
	// Higher values cause races with strong bases to diverge more from weak-base races.
	BaseModFactor ConfigFloat `yaml:"BaseModFactor"`
	// BaseModExponent controls the shape of the base-scaled component.
	// 1.0 = linear growth, <1.0 = diminishing returns, >1.0 = accelerating returns.
	BaseModExponent ConfigFloat `yaml:"BaseModExponent"`
	// NaturalGainsModFactor controls the universal flat gains every character receives
	// per stat step, regardless of race. Higher values raise the floor for weak-base races.
	NaturalGainsModFactor ConfigFloat `yaml:"NaturalGainsModFactor"`
	// NaturalGainsExponent controls the shape of the flat gains component.
	// 1.0 = linear, <1.0 = diminishing returns, >1.0 = accelerating.
	NaturalGainsExponent ConfigFloat `yaml:"NaturalGainsExponent"`

	// HP formula: base + archetype gains through HPFullLevels + HPAfterFull thereafter + Vitality and bonuses
	HPBase        ConfigInt   `yaml:"HPBase"`
	HPPerVitality ConfigFloat `yaml:"HPPerVitality"`

	// Mana formula: ManaMax = ManaBase + level*ManaPerLevel + Mysticism_adj*ManaPerMysticism + mods
	ManaBase         ConfigInt   `yaml:"ManaBase"`
	ManaPerLevel     ConfigFloat `yaml:"ManaPerLevel"`
	ManaPerMysticism ConfigFloat `yaml:"ManaPerMysticism"`

	// Points awarded to the player on each level-up.
	TrainingPointsPerLevel     ConfigInt `yaml:"TrainingPointsPerLevel"`
	TrainingPointsEveryNLevels ConfigInt `yaml:"TrainingPointsEveryNLevels"`
	StatPointsPerLevel         ConfigInt `yaml:"StatPointsPerLevel"`
	StatPointsEveryNLevels     ConfigInt `yaml:"StatPointsEveryNLevels"`

	// XP curve: XP_to_level(L) = (XPBase + L^XPLevelPower * XPLevelFactor * XPBase) * TNLScale
	// XPBase is the flat XP cost at level 1.
	XPBase ConfigInt `yaml:"XPBase"`
	// XPLevelFactor scales how fast the curve rises. Higher = more XP per level.
	XPLevelFactor ConfigFloat `yaml:"XPLevelFactor"`
	// XPLevelPower is the exponent. 2.0 = quadratic, 1.5 = gentler, 3.0 = steeper.
	XPLevelPower ConfigFloat `yaml:"XPLevelPower"`

	// MaxLevel is the admin chart display range; it never caps character levels.
	MaxLevel ConfigInt `yaml:"MaxLevel"`

	// Stat value compression (applied in Recalculate).
	// Once a stat's Value reaches StatCapThreshold, further gains are compressed:
	//   ValueAdj = StatCapAnchor + round((Value-StatCapAnchor)^StatCapExponent * StatCapScale)
	// StatCapThreshold: the Value at which compression begins. Default 105.
	StatCapThreshold ConfigInt `yaml:"StatCapThreshold"`
	// StatCapAnchor: the value the compression formula is anchored to. Default 100.
	// Overage is measured from this point: overage = Value - StatCapAnchor.
	StatCapAnchor ConfigInt `yaml:"StatCapAnchor"`
	// StatCapExponent: controls how aggressively gains are compressed above the threshold.
	// 0.5 = sqrt (default, strong compression), 1.0 = linear pass-through (no compression),
	// 0.25 = very aggressive. Valid range: 0.01 to 1.0.
	StatCapExponent ConfigFloat `yaml:"StatCapExponent"`
	// StatCapScale: multiplier applied after the exponent. Default 2.0.
	StatCapScale ConfigFloat `yaml:"StatCapScale"`
	// StatCapExemptBonus: when true, only the racial portion of a stat is compressed.
	// Training points and equipment/buff mods are added on top of the compressed racial
	// value without being subject to the cap. This ensures deliberate investment always
	// pays off fully. Default false (original behaviour: everything compressed together).
	StatCapExemptBonus ConfigBool `yaml:"StatCapExemptBonus"`
}

func (p *ProgressionConfig) Validate() {
	if p.StatStepLevels < 1 {
		p.StatStepLevels = 5
	}
	if p.DefaultHPPerLevel <= 0 || math.IsNaN(float64(p.DefaultHPPerLevel)) || math.IsInf(float64(p.DefaultHPPerLevel), 0) {
		p.DefaultHPPerLevel = 5
	}
	if p.HPFullLevels < 1 {
		p.HPFullLevels = 20
	}
	if p.HPAfterFull <= 0 || math.IsNaN(float64(p.HPAfterFull)) || math.IsInf(float64(p.HPAfterFull), 0) {
		p.HPAfterFull = 1
	}
	if p.XPKneeLevel < 2 {
		p.XPKneeLevel = 60
	}
	if p.XPKneeGrowth <= 1 || math.IsNaN(float64(p.XPKneeGrowth)) || math.IsInf(float64(p.XPKneeGrowth), 0) {
		p.XPKneeGrowth = 1.1
	}

	if p.BaseModFactor <= 0 {
		p.BaseModFactor = 0.3333333334
	}
	if p.BaseModExponent < 0.1 || p.BaseModExponent > 5.0 {
		p.BaseModExponent = 1.0
	}
	if p.NaturalGainsModFactor <= 0 {
		p.NaturalGainsModFactor = 0.5
	}
	if p.NaturalGainsExponent < 0.1 || p.NaturalGainsExponent > 5.0 {
		p.NaturalGainsExponent = 1.0
	}

	if p.HPBase < 0 {
		p.HPBase = 5
	}
	if p.HPPerVitality <= 0 {
		p.HPPerVitality = 1.0
	}

	if p.ManaBase < 0 {
		p.ManaBase = 4
	}
	if p.ManaPerLevel <= 0 {
		p.ManaPerLevel = 1.0
	}
	if p.ManaPerMysticism <= 0 {
		p.ManaPerMysticism = 3.0
	}

	if p.TrainingPointsPerLevel < 0 {
		p.TrainingPointsPerLevel = 1
	}
	if p.TrainingPointsEveryNLevels < 1 {
		p.TrainingPointsEveryNLevels = 1
	}
	if p.StatPointsPerLevel < 0 {
		p.StatPointsPerLevel = 1
	}
	if p.StatPointsEveryNLevels < 1 {
		p.StatPointsEveryNLevels = 5
	}

	if p.XPBase < 1 {
		p.XPBase = 1000
	}
	if p.XPLevelFactor <= 0 {
		p.XPLevelFactor = 0.75
	}
	if p.XPLevelPower < 0.1 || p.XPLevelPower > 5.0 {
		p.XPLevelPower = 2.0
	}

	if p.MaxLevel < 10 {
		p.MaxLevel = 100
	}

	if p.StatCapThreshold < 1 {
		p.StatCapThreshold = 105
	}
	if p.StatCapAnchor < 0 {
		p.StatCapAnchor = 100
	}
	if p.StatCapAnchor >= p.StatCapThreshold {
		p.StatCapAnchor = p.StatCapThreshold - 1
	}
	if p.StatCapExponent <= 0 || p.StatCapExponent > 1.0 {
		p.StatCapExponent = 0.5
	}
	if p.StatCapScale <= 0 {
		p.StatCapScale = 2.0
	}
}

func GetProgressionConfig() ProgressionConfig {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	if !configData.validated {
		configData.Validate()
	}
	return configData.GamePlay.Progression
}

// StatStep returns the progression input, retaining the original formula for
// an interval of one. Other intervals advance at their level multiples.
func (p ProgressionConfig) StatStep(level int) int {
	level = max(level, 1)
	every := max(int(p.StatStepLevels), 1)
	if every == 1 {
		return level
	}
	return 1 + level/every
}

// NextStatStep saturates at MaxInt rather than wrapping for extreme levels.
func (p ProgressionConfig) NextStatStep(level int) int {
	level = max(level, 1)
	every := max(int(p.StatStepLevels), 1)
	delta := every - level%every
	if level > math.MaxInt-delta {
		return math.MaxInt
	}
	return level + delta
}

// HealthAtLevel is the HP formula without equipment, buffs or explicit training.
func (p ProgressionConfig) HealthAtLevel(level, vitality int, perLevel float64) int {
	level = max(level, 1)
	if perLevel <= 0 {
		perLevel = float64(p.DefaultHPPerLevel)
	}
	full := min(level, int(p.HPFullLevels))
	return boundedProgressionInt(float64(p.HPBase) + math.Trunc(float64(full)*perLevel) +
		math.Trunc(float64(level-full)*p.HealthAfterFull(perLevel)) + math.Trunc(float64(vitality)*float64(p.HPPerVitality)))
}

// HealthAfterFull is an archetype's HP per level after HPFullLevels (30g6
// amendment E): HPAfterFull for the default (middle) rate, scaled by the
// archetype's own rate, so a warrior keeps out-gaining a wizard.
func (p ProgressionConfig) HealthAfterFull(perLevel float64) float64 {
	if perLevel <= 0 || p.DefaultHPPerLevel <= 0 {
		return float64(p.HPAfterFull)
	}
	return float64(p.HPAfterFull) * perLevel / float64(p.DefaultHPPerLevel)
}

// XPThreshold preserves cumulative thresholds through the knee. After it,
// incremental costs grow geometrically. A closed form avoids level-sized loops.
func (p ProgressionConfig) XPThreshold(level int, scale float64) int {
	level = max(level, 1)
	polynomial := func(l int) float64 {
		return float64(p.XPBase) + math.Pow(float64(l), float64(p.XPLevelPower))*float64(p.XPLevelFactor)*float64(p.XPBase)
	}
	knee := max(int(p.XPKneeLevel), 2)
	xp := polynomial(min(level, knee))
	if level > knee {
		growth := float64(p.XPKneeGrowth)
		cost := polynomial(knee) - polynomial(knee-1)
		// expm1 avoids cancellation when an administrator sets growth near 1.
		xp += cost * growth * math.Expm1(float64(level-knee)*math.Log(growth)) / (growth - 1)
	}
	xp *= scale
	if math.IsInf(xp, 1) || xp >= float64(math.MaxInt) {
		return math.MaxInt
	}
	if math.IsNaN(xp) || xp < 0 {
		return 0
	}
	return int(xp)
}

// RacialForLevel is shared with the editor so its step previews cannot drift.
func (p ProgressionConfig) RacialForLevel(level, base int) int {
	step := p.StatStep(level)
	racial := math.Trunc(math.Pow(float64(step-1), float64(p.BaseModExponent)) * float64(p.BaseModFactor) * float64(base))
	free := math.Trunc(math.Pow(float64(step), float64(p.NaturalGainsExponent)) * float64(p.NaturalGainsModFactor))
	return boundedProgressionInt(racial + free)
}

func boundedProgressionInt(value float64) int {
	if value >= float64(math.MaxInt) {
		return math.MaxInt
	}
	if value <= float64(math.MinInt) {
		return math.MinInt
	}
	if math.IsNaN(value) {
		return 0
	}
	return int(value)
}

// CompressStat applies the configured soft cap to a racial or total stat value.
func (p ProgressionConfig) CompressStat(value int) int {
	if value < int(p.StatCapThreshold) {
		return value
	}
	overage := max(value-int(p.StatCapAnchor), 0)
	return boundedProgressionInt(float64(p.StatCapAnchor) + math.Round(math.Pow(float64(overage), float64(p.StatCapExponent))*float64(p.StatCapScale)))
}
