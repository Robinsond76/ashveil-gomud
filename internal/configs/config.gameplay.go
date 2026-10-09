package configs

import (
	"math"
	"strconv"
	"strings"
)

type GamePlay struct {
	AllowItemBuffRemoval ConfigBool `yaml:"AllowItemBuffRemoval"`
	// Death related settings
	Death GameplayDeath `yaml:"Death"`
	// Party settings
	Party GameplayParty `yaml:"Party"`
	// Progression settings
	Progression ProgressionConfig `yaml:"Progression"`

	LivesStart     ConfigInt `yaml:"LivesStart"`     // Starting permadeath lives
	LivesMax       ConfigInt `yaml:"LivesMax"`       // Maximum permadeath lives
	LivesOnLevelUp ConfigInt `yaml:"LivesOnLevelUp"` // # lives gained on level up
	PricePerLife   ConfigInt `yaml:"PricePerLife"`   // Price in gold to buy new lives
	// Shops/Containers
	ShopRestockRate       ConfigString `yaml:"ShopRestockRate"`       // Default time it takes to restock 1 quantity in shops
	MercHirePricePerLevel ConfigInt    `yaml:"MercHirePricePerLevel"` // Gold cost per mob level when auto-calculating mercenary hire price
	ContainerSizeMax      ConfigInt    `yaml:"ContainerSizeMax"`      // How many objects containers can hold before overflowing
	FloorItemCountMax     ConfigInt    `yaml:"FloorItemCountMax"`     // Maximum items allowed on the floor at once (0 = no limit)
	Combat                CombatConfig `yaml:"Combat"`

	// PVP Restrictions
	PVP GameplayPVP `yaml:"PVP"`
	// XpScale (difficulty)
	XPScale           ConfigFloat `yaml:"XPScale"`
	MobConverseChance ConfigInt   `yaml:"MobConverseChance"` // Chance 1-100 of attempting to converse when idle
	// Idle chatter limits (internal/mobs/chatter.go)
	MobChatterCooldownRounds ConfigInt `yaml:"MobChatterCooldownRounds"` // Rounds between an idle mob's says and emotes (0 = no limit)
	MobChatterMemoryRounds   ConfigInt `yaml:"MobChatterMemoryRounds"`   // Rounds before a player may hear the same idle line again (0 = no limit)
	AlignmentDecayRounds     ConfigInt `yaml:"AlignmentDecayRounds"`     // Rounds between each alignment decay step toward neutral (0 = disabled)
	// Elite mob settings
	EliteLevelBonus ConfigInt `yaml:"EliteLevelBonus"` // Percent level increase for elite mob spawns (e.g. 20 = 20% higher level)
	EliteXPBonus    ConfigInt `yaml:"EliteXPBonus"`    // Percent XP bonus for killing an elite mob (e.g. 10 = 10% more XP)
}

// CombatConfig holds configurable min/max bounds for every combat calculation.
type CombatConfig struct {
	ConsistentAttackMessages ConfigBool `yaml:"ConsistentAttackMessages"` // Whether each weapon has consistent attack messages

	// Damage bonus: absolute Strength growth plus the Strength advantage.
	DamageBonusMin    ConfigInt   `yaml:"DamageBonusMin"`    // Minimum flat damage bonus
	DamageBonusMax    ConfigInt   `yaml:"DamageBonusMax"`    // Maximum flat damage bonus
	DamagePerStrength ConfigFloat `yaml:"DamagePerStrength"` // Flat damage per effective Strength (0 disables absolute Strength growth)
	DamageEdgeMax     ConfigInt   `yaml:"DamageEdgeMax"`     // Extra damage at a full Strength edge over the defender (30g6)

	// Stat edge (30g6 amendment): a stat difference of StatEdgeSpan moves an
	// opposed chance all the way to a bound; one point moves 1/span of it.
	StatEdgeSpan ConfigFloat `yaml:"StatEdgeSpan"`

	// Skill edge (Phase 35a2): the attacker's Attack less the defender's
	// Evasion over SkillEdgeSpan, added to every opposed chance's stat edge
	// but crits. Characters without a class gain the default ratings a level.
	SkillEdgeSpan      ConfigFloat `yaml:"SkillEdgeSpan"`
	DefaultAttackRate  ConfigFloat `yaml:"DefaultAttackRate"`
	DefaultEvasionRate ConfigFloat `yaml:"DefaultEvasionRate"`

	// Chance to hit (Speed edge drives this)
	ToHitMin  ConfigInt `yaml:"ToHitMin"`  // Minimum hit chance (percent, 0-100)
	ToHitMax  ConfigInt `yaml:"ToHitMax"`  // Maximum hit chance (percent, 0-100)
	ToHitEven ConfigInt `yaml:"ToHitEven"` // Hit chance when even (percent, within the bounds)

	// Blow quality (Phase 35d): a blow that lands is glancing, solid or
	// telling. Each share moves linearly from its even value to its full-edge
	// value: "Full" is a full edge against the attacker for glancing and for
	// the attacker for telling; "Least" is the opposite end.
	GlanceEven    ConfigInt   `yaml:"GlanceEven"`    // Glancing share at an even edge (percent)
	GlanceFull    ConfigInt   `yaml:"GlanceFull"`    // Glancing share at a full edge against the attacker (percent)
	GlanceLeast   ConfigInt   `yaml:"GlanceLeast"`   // Glancing share at a full edge for the attacker (percent)
	TellingEven   ConfigInt   `yaml:"TellingEven"`   // Telling share at an even edge (percent)
	TellingLeast  ConfigInt   `yaml:"TellingLeast"`  // Telling share at a full edge against the attacker (percent)
	TellingFull   ConfigInt   `yaml:"TellingFull"`   // Telling share at a full edge for the attacker (percent)
	GlanceFactor  ConfigFloat `yaml:"GlanceFactor"`  // Damage multiplier of a glancing blow
	TellingFactor ConfigFloat `yaml:"TellingFactor"` // Damage multiplier of a telling blow

	// Legacy extra-attack keys: readable for old overrides, unused since 30g5.
	ExtraAttacksMin ConfigInt `yaml:"ExtraAttacksMin"` // Minimum extra attacks
	ExtraAttacksMax ConfigInt `yaml:"ExtraAttacksMax"` // Maximum extra attacks

	// Chance to crit (Smarts delta drives this)
	CritChanceMin  ConfigInt `yaml:"CritChanceMin"`  // Minimum crit chance (percent, 0-100)
	CritChanceMax  ConfigInt `yaml:"CritChanceMax"`  // Maximum crit chance (percent, 0-100)
	CritChanceEven ConfigInt `yaml:"CritChanceEven"` // Crit chance at equal Smarts (percent, within the bounds)

	// Crit damage multiplier (Perception delta drives this)
	CritMultMin ConfigFloat `yaml:"CritMultMin"` // Minimum crit damage multiplier
	CritMultMax ConfigFloat `yaml:"CritMultMax"` // Maximum crit damage multiplier

	// Chance to dodge (Perception delta drives this)
	DodgeChanceMin  ConfigInt `yaml:"DodgeChanceMin"`  // Minimum dodge chance (percent, 0-100)
	DodgeChanceMax  ConfigInt `yaml:"DodgeChanceMax"`  // Maximum dodge chance (percent, 0-100)
	DodgeChanceEven ConfigInt `yaml:"DodgeChanceEven"` // Dodge chance when even (percent, within the bounds)

	// Chance to block with a shield (Strength delta + shield armor drives this)
	BlockChanceMin  ConfigInt `yaml:"BlockChanceMin"`  // Minimum block chance (percent, 0-100)
	BlockChanceMax  ConfigInt `yaml:"BlockChanceMax"`  // Maximum block chance (percent, 0-100)
	BlockChanceEven ConfigInt `yaml:"BlockChanceEven"` // Block chance when even, before the shield's armor (percent)

	// Chance to parry (Speed delta drives this)
	ParryChanceMin  ConfigInt `yaml:"ParryChanceMin"`  // Minimum parry chance (percent, 0-100)
	ParryChanceMax  ConfigInt `yaml:"ParryChanceMax"`  // Maximum parry chance (percent, 0-100)
	ParryChanceEven ConfigInt `yaml:"ParryChanceEven"` // Parry chance when even, before the weapon's modifier (percent)

	// Chance to bash when blocking (Strength delta drives this)
	BashChanceMin ConfigInt `yaml:"BashChanceMin"` // Minimum bash chance (percent, 0-100)
	BashChanceMax ConfigInt `yaml:"BashChanceMax"` // Maximum bash chance (percent, 0-100)

	// Personal load and agility (Phase 30g3): what a character wears and
	// carries against AgilityBaseKg + AgilityStrengthKg per Strength; the
	// first AgilityFreeLoad of that is free, the rest burdens their dodge.
	AgilityBaseKg     ConfigFloat `yaml:"AgilityBaseKg"`     // Agility capacity before Strength (kg)
	AgilityStrengthKg ConfigFloat `yaml:"AgilityStrengthKg"` // Agility capacity per point of Strength (kg)
	AgilityFreeLoad   ConfigFloat `yaml:"AgilityFreeLoad"`   // Share of capacity carried with no burden (above 0, at most 0.95)
	// Armor bulk (Phase 35a2): the share of tempo and of dodge lost in the
	// heaviest piece worn; doubled, with UntrainedSkillLoss Attack and
	// Evasion and UntrainedChantRounds a chant, when the class isn't trained.
	BulkTempoMedium      ConfigFloat `yaml:"BulkTempoMedium"`
	BulkTempoHeavy       ConfigFloat `yaml:"BulkTempoHeavy"`
	BulkDodgeMedium      ConfigFloat `yaml:"BulkDodgeMedium"`
	BulkDodgeHeavy       ConfigFloat `yaml:"BulkDodgeHeavy"`
	UntrainedSkillLoss   ConfigInt   `yaml:"UntrainedSkillLoss"`
	UntrainedChantRounds ConfigInt   `yaml:"UntrainedChantRounds"`
	// Action meter (Phase 30g5); no fighter exceeds two turns a round.
	TempoMin         ConfigFloat `yaml:"TempoMin"`
	TempoMax         ConfigFloat `yaml:"TempoMax"`
	TempoSpeedRef    ConfigFloat `yaml:"TempoSpeedRef"`
	TempoSpeedSpan   ConfigFloat `yaml:"TempoSpeedSpan"`
	MaxTurnsPerRound ConfigInt   `yaml:"MaxTurnsPerRound"`
	// Battle clock (Phase 82c): a round in a player's fight lasts as long
	// as its actions' beats need, and never less than MinRoundMs.
	MinRoundMs ConfigInt `yaml:"MinRoundMs"`
}

type GameplayParty struct {
	MaxPlayerCount ConfigInt  `yaml:"MaxPlayerCount"` // Maximum number of players allowed in a party (0 = unlimited)
	SameRoomOnly   ConfigBool `yaml:"SameRoomOnly"`   // Whether players must be in the same room to create/invite/join parties
}

type GameplayPVP struct {
	Enabled      ConfigString `yaml:"Enabled"`      // Possible values: enabled, disabled, limited
	MinimumLevel ConfigInt    `yaml:"MinimumLevel"` // Minimum level required to participate in PVP
}

type GameplayDeath struct {
	EquipmentDropChance ConfigFloat  `yaml:"EquipmentDropChance"` // Chance a player will drop a given piece of equipment on death
	AlwaysDropBackpack  ConfigBool   `yaml:"AlwaysDropBackpack"`  // If true, players will always drop their backpack items on death
	XPPenalty           ConfigString `yaml:"XPPenalty"`           // Possible values are: none, level, 10%, 25%, 50%, 75%, 90%, 100%
	ProtectionLevels    ConfigInt    `yaml:"ProtectionLevels"`    // How many levels is the user protected from death penalties for?
	PermaDeath          ConfigBool   `yaml:"PermaDeath"`          // Is permadeath enabled?
	CorpsesEnabled      ConfigBool   `yaml:"CorpsesEnabled"`      // Whether corpses are left behind after mob/player deaths
	CorpseDecayTime     ConfigString `yaml:"CorpseDecayTime"`     // How long until corpses decay to dust (go away)
	CorpseItems         ConfigBool   `yaml:"CorpseItems"`         // If true, items/gold go onto the corpse instead of the floor
}

func (g *GamePlay) Validate() {

	if g.Party.MaxPlayerCount < 0 {
		g.Party.MaxPlayerCount = 0
	}

	// Ignore AllowItemBuffRemoval
	// Ignore OnDeathAlwaysDropBackpack
	// Ignore CorpsesEnabled
	// Ignore CorpseItems

	if g.Death.EquipmentDropChance < 0.0 || g.Death.EquipmentDropChance > 1.0 {
		g.Death.EquipmentDropChance = 0.0 // default
	}

	g.Death.XPPenalty.Set(strings.ToLower(string(g.Death.XPPenalty)))

	if g.Death.XPPenalty != `none` && g.Death.XPPenalty != `level` {
		// If not a valid percent, set to default
		if !strings.HasSuffix(string(g.Death.XPPenalty), `%`) {
			g.Death.XPPenalty = `none` // default
		} else {
			// If not a valid percent, set to default
			percent, err := strconv.ParseInt(string(g.Death.XPPenalty)[0:len(g.Death.XPPenalty)-1], 10, 64)
			if err != nil || percent < 0 || percent > 100 {
				g.Death.XPPenalty = `none` // default
			}
		}
	}

	if g.Death.ProtectionLevels < 0 {
		g.Death.ProtectionLevels = 0 // default
	}

	if g.LivesStart < 0 {
		g.LivesStart = 0
	}

	if g.LivesMax < 0 {
		g.LivesMax = 0
	}

	if g.LivesOnLevelUp < 0 {
		g.LivesOnLevelUp = 0
	}

	if g.PricePerLife < 1 {
		g.PricePerLife = 1
	}

	if g.ShopRestockRate == `` {
		g.ShopRestockRate = `6 hours`
	}

	if g.MercHirePricePerLevel < 1 {
		g.MercHirePricePerLevel = 250
	}

	if g.ContainerSizeMax < 1 {
		g.ContainerSizeMax = 1
	}

	if g.FloorItemCountMax < 0 {
		g.FloorItemCountMax = 0
	}

	if g.Death.CorpseDecayTime == `` {
		g.Death.CorpseDecayTime = `1 hour`
	}

	if g.PVP.Enabled != PVPEnabled && g.PVP.Enabled != PVPDisabled && g.PVP.Enabled != PVPLimited {
		if g.PVP.Enabled == PVPOff {
			g.PVP.Enabled = PVPDisabled
		} else {
			g.PVP.Enabled = PVPEnabled
		}
	}

	if int(g.PVP.MinimumLevel) < 0 {
		g.PVP.MinimumLevel = 0
	}

	if g.XPScale <= 0 {
		g.XPScale = 100
	}

	if g.MobConverseChance < 0 {
		g.MobConverseChance = 0
	} else if g.MobConverseChance > 100 {
		g.MobConverseChance = 100
	}

	if g.MobChatterCooldownRounds < 0 {
		g.MobChatterCooldownRounds = 60
	}

	if g.MobChatterMemoryRounds < 0 {
		g.MobChatterMemoryRounds = 900
	}

	if g.AlignmentDecayRounds < 0 {
		g.AlignmentDecayRounds = 100
	}

	if g.EliteLevelBonus < 0 {
		g.EliteLevelBonus = 20
	}

	if g.EliteXPBonus < 0 {
		g.EliteXPBonus = 10
	}

	g.Combat.validate()

	g.Progression.Validate()

}

func (c *CombatConfig) validate() {
	// Damage bonus
	if c.DamagePerStrength < 0 || math.IsNaN(float64(c.DamagePerStrength)) || math.IsInf(float64(c.DamagePerStrength), 0) {
		c.DamagePerStrength = 0
	}
	if c.DamageBonusMax < 1 {
		c.DamageBonusMax = 10
	}
	if c.DamageEdgeMax < 0 {
		c.DamageEdgeMax = 0
	}
	if c.DamageBonusMin < 0 {
		c.DamageBonusMin = 0
	}
	if c.DamageBonusMin > c.DamageBonusMax {
		c.DamageBonusMin = 0
	}

	// To-hit
	if c.ToHitMax < 1 || c.ToHitMax > 100 {
		c.ToHitMax = 100
	}
	if c.ToHitMin < 1 {
		c.ToHitMin = 10
	}
	if c.ToHitMin > c.ToHitMax {
		c.ToHitMin = 10
	}
	if c.ToHitEven == 0 {
		c.ToHitEven = 60
	}
	c.ToHitEven = max(c.ToHitMin, min(c.ToHitMax, c.ToHitEven))

	// Blow quality (Phase 35d). Zero values give the shipped defaults; a
	// factor of 1 for both turns the quality roll off.
	if c.GlanceEven == 0 && c.GlanceFull == 0 && c.GlanceLeast == 0 && c.TellingEven == 0 && c.TellingFull == 0 && c.TellingLeast == 0 {
		c.GlanceEven, c.GlanceFull, c.GlanceLeast = 25, 50, 5
		c.TellingEven, c.TellingFull, c.TellingLeast = 20, 50, 5
	}
	if c.GlanceFactor <= 0 {
		c.GlanceFactor = 0.5
	}
	if c.TellingFactor <= 0 {
		c.TellingFactor = 1.4
	}
	for _, v := range []*ConfigInt{&c.GlanceEven, &c.GlanceFull, &c.GlanceLeast, &c.TellingEven, &c.TellingFull, &c.TellingLeast} {
		*v = max(0, min(100, *v))
	}

	// Stat edge
	if math.IsNaN(float64(c.StatEdgeSpan)) || math.IsInf(float64(c.StatEdgeSpan), 0) || c.StatEdgeSpan <= 0 {
		c.StatEdgeSpan = 10
	}

	// Skill edge (Phase 35a2)
	if !finitePositive(float64(c.SkillEdgeSpan)) {
		c.SkillEdgeSpan = 16
	}
	if !finitePositive(float64(c.DefaultAttackRate)) {
		c.DefaultAttackRate = 1
	}
	if !finitePositive(float64(c.DefaultEvasionRate)) {
		c.DefaultEvasionRate = 1
	}

	// Extra attacks (weaponless/claws)
	if c.ExtraAttacksMax == 0 {
		c.ExtraAttacksMax = 3
	} else if c.ExtraAttacksMax < 0 {
		c.ExtraAttacksMax = 0
	}
	if c.ExtraAttacksMin < 0 {
		c.ExtraAttacksMin = 0
	}
	if c.ExtraAttacksMin > c.ExtraAttacksMax {
		c.ExtraAttacksMin = 0
	}

	// Crit chance
	if c.CritChanceMax < 1 || c.CritChanceMax > 100 {
		c.CritChanceMax = 30
	}
	if c.CritChanceMin < 1 {
		c.CritChanceMin = 5
	}
	if c.CritChanceMin > c.CritChanceMax {
		c.CritChanceMin = 5
	}
	if c.CritChanceEven == 0 {
		c.CritChanceEven = 15
	}
	c.CritChanceEven = max(c.CritChanceMin, min(c.CritChanceMax, c.CritChanceEven))

	// Crit multiplier
	if c.CritMultMin < 1.0 {
		c.CritMultMin = 1.5
	}
	if c.CritMultMax < c.CritMultMin {
		c.CritMultMax = 3.0
	}

	// Dodge chance
	if c.DodgeChanceMax < 1 || c.DodgeChanceMax > 100 {
		c.DodgeChanceMax = 40
	}
	if c.DodgeChanceMin < 1 {
		c.DodgeChanceMin = 3
	}
	if c.DodgeChanceMin > c.DodgeChanceMax {
		c.DodgeChanceMin = 3
	}
	if c.DodgeChanceEven == 0 {
		c.DodgeChanceEven = 12
	}
	c.DodgeChanceEven = max(c.DodgeChanceMin, min(c.DodgeChanceMax, c.DodgeChanceEven))

	// Block chance (Phase 30g2)
	if c.BlockChanceMax < 1 || c.BlockChanceMax > 100 {
		c.BlockChanceMax = 55
	}
	if c.BlockChanceMin < 1 {
		c.BlockChanceMin = 8
	}
	if c.BlockChanceMin > c.BlockChanceMax {
		c.BlockChanceMin = 8
	}
	if c.BlockChanceEven == 0 {
		c.BlockChanceEven = 20
	}
	c.BlockChanceEven = max(c.BlockChanceMin, min(c.BlockChanceMax, c.BlockChanceEven))

	// Parry chance (Phase 30g2)
	if c.ParryChanceMax < 1 || c.ParryChanceMax > 100 {
		c.ParryChanceMax = 40
	}
	if c.ParryChanceMin < 1 {
		c.ParryChanceMin = 3
	}
	if c.ParryChanceMin > c.ParryChanceMax {
		c.ParryChanceMin = 3
	}
	if c.ParryChanceEven == 0 {
		c.ParryChanceEven = 12
	}
	c.ParryChanceEven = max(c.ParryChanceMin, min(c.ParryChanceMax, c.ParryChanceEven))

	// Bash chance (Phase 30g2)
	if c.BashChanceMax < 1 || c.BashChanceMax > 100 {
		c.BashChanceMax = 20
	}
	if c.BashChanceMin < 1 {
		c.BashChanceMin = 5
	}
	if c.BashChanceMin > c.BashChanceMax {
		c.BashChanceMin = 5
	}

	// Invalid/missing tempo values fall back; enforce the gameplay rate cap.
	if math.IsNaN(float64(c.TempoMin)) || math.IsInf(float64(c.TempoMin), 0) || c.TempoMin <= 0 || c.TempoMin > 1 {
		c.TempoMin = 0.6
	}
	if math.IsNaN(float64(c.TempoMax)) || math.IsInf(float64(c.TempoMax), 0) || c.TempoMax < 1 || c.TempoMax > 1.5 {
		c.TempoMax = 1.5
	}
	if math.IsNaN(float64(c.TempoSpeedRef)) || math.IsInf(float64(c.TempoSpeedRef), 0) || c.TempoSpeedRef <= 0 {
		c.TempoSpeedRef = 10
	}
	if math.IsNaN(float64(c.TempoSpeedSpan)) || math.IsInf(float64(c.TempoSpeedSpan), 0) || c.TempoSpeedSpan <= 0 {
		c.TempoSpeedSpan = 40
	}
	if c.MaxTurnsPerRound < 1 || c.MaxTurnsPerRound > 2 {
		c.MaxTurnsPerRound = 2
	}
	if c.MinRoundMs < 500 || c.MinRoundMs > 30000 {
		c.MinRoundMs = 3000
	}

	// Agility (Phase 30g3)
	if c.AgilityBaseKg <= 0 {
		c.AgilityBaseKg = 15
	}
	if c.AgilityStrengthKg <= 0 {
		c.AgilityStrengthKg = 0.5
	}
	if c.AgilityFreeLoad <= 0 || c.AgilityFreeLoad > 0.95 {
		c.AgilityFreeLoad = 0.35
	}

	// Armor bulk (Phase 35a2): shares held to 0..1 (0 turns a cost off);
	// a doubled share past 1 leaves nothing, never a negative tempo or dodge.
	for _, b := range []*ConfigFloat{&c.BulkTempoMedium, &c.BulkTempoHeavy, &c.BulkDodgeMedium, &c.BulkDodgeHeavy} {
		switch {
		case math.IsNaN(float64(*b)) || *b < 0:
			*b = 0
		case *b > 1:
			*b = 1
		}
	}
	if c.UntrainedSkillLoss < 0 {
		c.UntrainedSkillLoss = 0
	}
	if c.UntrainedChantRounds < 0 {
		c.UntrainedChantRounds = 0
	}
}

func finitePositive(v float64) bool {
	return v > 0 && !math.IsNaN(v) && !math.IsInf(v, 0)
}

func GetGamePlayConfig() GamePlay {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	if !configData.validated {
		configData.Validate()
	}
	return configData.GamePlay
}

func GetPVPConfig() GameplayPVP {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	if !configData.validated {
		configData.Validate()
	}
	return configData.GamePlay.PVP
}

func GetCombatConfig() CombatConfig {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	if !configData.validated {
		configData.Validate()
	}
	return configData.GamePlay.Combat
}
