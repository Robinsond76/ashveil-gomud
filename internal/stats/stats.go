package stats

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
)

type Statistics struct {
	Strength   StatInfo `yaml:"strength,omitempty"`   // Muscular strength (damage?)
	Speed      StatInfo `yaml:"speed,omitempty"`      // Speed and agility (dodging)
	Smarts     StatInfo `yaml:"smarts,omitempty"`     // Intelligence and wisdom (magic power, memory, deduction, etc)
	Vitality   StatInfo `yaml:"vitality,omitempty"`   // Health and stamina (health capacity)
	Mysticism  StatInfo `yaml:"mysticism,omitempty"`  // Magic and mana (magic capacity)
	Perception StatInfo `yaml:"perception,omitempty"` // How well you notice things
}

// When saving to a file, we don't need to write all the properties that we calculate.
// Just keep track of "Training" because that's not calculated.
type StatInfo struct {
	Training int  `yaml:"training,omitempty"` // How much it's been trained with Training Points spending
	Value    int  `yaml:"-"`                  // Final calculated value
	ValueAdj int  `yaml:"-"`                  // Final calculated value (Adjusted)
	Racial   int  `yaml:"-"`                  // Value provided by racial benefits
	Base     int  `yaml:"base,omitempty"`     // Base stat value
	Mods     int  `yaml:"-"`                  // How much it's modded by equipment, spells, etc.
	NoCap    bool `yaml:"-"`                  // When true, skip the stat cap compression in Recalculate
}

func (si *StatInfo) SetMod(mod ...int) {
	if len(mod) == 0 {
		si.Mods = 0
		return
	}
	si.Mods = 0
	for _, m := range mod {
		si.Mods += m
	}
}

// GainsForLevel returns the racial stat value at the given level, using the
// configured progression formula:
//
//	racial = floor(base * BaseModFactor * (step-1)^BaseModExponent)
//	       + floor(NaturalGainsModFactor * step^NaturalGainsExponent)
func (si *StatInfo) GainsForLevel(level int) int {
	return configs.GetProgressionConfig().RacialForLevel(level, si.Base)
}

func (si *StatInfo) Recalculate(level int) {
	si.Racial = si.GainsForLevel(level)
	si.Value = SaturatingSum(si.Racial, si.Training, si.Mods)
	si.ValueAdj = si.Value
	if si.NoCap {
		return
	}
	cfg := configs.GetProgressionConfig()
	if bool(cfg.StatCapExemptBonus) {
		si.ValueAdj = SaturatingSum(cfg.CompressStat(si.Racial), si.Training, si.Mods)
	} else {
		si.ValueAdj = cfg.CompressStat(si.Value)
	}
}
