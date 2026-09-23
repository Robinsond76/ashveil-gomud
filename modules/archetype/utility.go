package archetype

import (
	"strings"
)

// utilityConfig holds Phase 17b's utility-skill balance. All values are
// module config (files/data-overlays/config.yaml).
type utilityConfig struct {
	// UtilitySkills maps a utility (e.g. "light") to the skill whose level a
	// player uses for it (e.g. "cast").
	UtilitySkills map[string]string

	SensePerLevel          int
	SenseDifficultyFactor  int
	SenseCooldownRounds    int
	AutoSensePenalty       int
	DisarmPerLevel         int
	DisarmDifficultyFactor int
	DisarmBackfireMargin   int
	DisarmRounds           int

	AutoLightSpell         string
	AutoLightBelow         int
	AutoLightCooldown      int
	CompanionLightBuffID   int
	CompanionLightManaCost int
}

func defaultUtilityConfig() utilityConfig {
	return utilityConfig{
		UtilitySkills:          map[string]string{"light": "cast", "traps": "skulduggery"},
		SensePerLevel:          20,
		SenseDifficultyFactor:  5,
		SenseCooldownRounds:    2,
		AutoSensePenalty:       20,
		DisarmPerLevel:         20,
		DisarmDifficultyFactor: 5,
		DisarmBackfireMargin:   25,
		DisarmRounds:           900,
		AutoLightSpell:         "floatinglight",
		AutoLightBelow:         1,
		AutoLightCooldown:      10,
		CompanionLightBuffID:   1000,
		CompanionLightManaCost: 10,
	}
}

// parseUtilityConfig overlays configured values on the defaults. A missing
// or non-positive number keeps its default (AutoSensePenalty may be 0).
func parseUtilityConfig(get func(string) any) utilityConfig {
	cfg := defaultUtilityConfig()
	if get == nil {
		return cfg
	}
	positive := func(key string, into *int) {
		if v := configInt(get(key)); v > 0 {
			*into = v
		}
	}
	positive("SensePerLevel", &cfg.SensePerLevel)
	positive("SenseDifficultyFactor", &cfg.SenseDifficultyFactor)
	positive("SenseCooldownRounds", &cfg.SenseCooldownRounds)
	positive("DisarmPerLevel", &cfg.DisarmPerLevel)
	positive("DisarmDifficultyFactor", &cfg.DisarmDifficultyFactor)
	positive("DisarmBackfireMargin", &cfg.DisarmBackfireMargin)
	positive("DisarmRounds", &cfg.DisarmRounds)
	positive("AutoLightCooldown", &cfg.AutoLightCooldown)
	positive("CompanionLightBuffID", &cfg.CompanionLightBuffID)
	positive("CompanionLightManaCost", &cfg.CompanionLightManaCost)
	if raw := get("AutoSensePenalty"); raw != nil {
		if v := configInt(raw); v >= 0 {
			cfg.AutoSensePenalty = v
		}
	}
	if raw := get("AutoLightBelow"); raw != nil {
		if v := configInt(raw); v >= 1 && v <= 3 {
			cfg.AutoLightBelow = v
		}
	}
	if s := configString(get("AutoLightSpell")); s != "" {
		cfg.AutoLightSpell = strings.ToLower(s)
	}
	return cfg
}

// parseUtilitySkills reads the Utilities list ([{Utility, Skill}]); an
// empty or missing list keeps the defaults.
func parseUtilitySkills(raw any, defaults map[string]string) map[string]string {
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		return defaults
	}
	out := map[string]string{}
	for _, entry := range list {
		fields := stringMap(entry)
		if fields == nil {
			continue
		}
		utility := strings.ToLower(configString(fields["utility"]))
		skill := strings.ToLower(configString(fields["skill"]))
		if utility == "" || skill == "" {
			continue
		}
		out[utility] = skill
	}
	if len(out) == 0 {
		return defaults
	}
	return out
}

// registerUtility wires Phase 17b's commands and listeners.
func (m *ArchetypeModule) registerUtility() {}
