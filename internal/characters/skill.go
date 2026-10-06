package characters

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/creatures"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// Phase 35a2 (skill over hit points): Attack and Evasion are derived from
// level, class and an enemy template's offset every time they are read, so
// nothing is saved and death, copyover and replays can't drift them.

// MaxSkillOffset bounds an enemy template's attackskill and evasion.
const MaxSkillOffset = 5

// ArchetypeID is the character's class: a player's chosen archetype, a
// companion's runtime class, or "" for enemies and the unchosen.
func (c *Character) ArchetypeID() string {
	if c == nil {
		return ""
	}
	if c.userId > 0 {
		id, _ := archetypes.PlayerArchetype(c.userId)
		return id
	}
	return c.HPArchetype
}

func (c *Character) combatProfile() (archetypes.Profile, bool) {
	return archetypes.CombatProfile(c.ArchetypeID())
}

// HPStart is the class's head start in health (0 without a class).
func (c *Character) HPStart() int {
	if p, ok := c.combatProfile(); ok {
		return p.HPStart
	}
	return 0
}

// skillRates is the Attack and Evasion gained a level: the class's, else
// the combat defaults.
func (c *Character) skillRates() (attack, evasion float64) {
	cfg := configs.GetCombatConfig()
	attack, evasion = float64(cfg.DefaultAttackRate), float64(cfg.DefaultEvasionRate)
	if p, ok := c.combatProfile(); ok {
		if p.AttackRate > 0 {
			attack = p.AttackRate
		}
		if p.EvasionRate > 0 {
			evasion = p.EvasionRate
		}
	}
	return attack, evasion
}

// skillPenalty is what untrained armor costs Attack and Evasion.
func (c *Character) skillPenalty() int {
	if c.UntrainedArmor() {
		return int(configs.GetCombatConfig().UntrainedSkillLoss)
	}
	return 0
}

// AttackSkill is the character's Attack: floor(level × rate) plus an enemy
// template's offset, less the untrained-armor loss.
func (c *Character) AttackSkill() int {
	rate, _ := c.skillRates()
	return levelRating(c.Level, rate) + c.AttackOffset - c.skillPenalty() + c.ClassEffects().Int(classes.Attack) + c.blessPoints()
}

// Evasion is the character's Evasion: floor(level × rate) plus an enemy
// template's offset, less the untrained-armor loss.
func (c *Character) Evasion() int {
	_, rate := c.skillRates()
	bonus := c.ClassEffects().Int(classes.Evasion) + c.Aura.Evasion + c.blessPoints()
	if c.RT != nil {
		bonus += c.RT.PackSense   // Phase 39e: Pack Sense, while the beast stands
		bonus += c.RT.VanishEvade // Phase 38c2: a Pathfinder's Vanish
	}
	if c.Aggro != nil && c.Aggro.Type == SpellCast {
		bonus += c.ClassEffects().Int(classes.ChantEvade) // Phase 38b: Sanctuary
	}
	return levelRating(c.Level, rate) + c.EvasionOffset - c.skillPenalty() + bonus
}

func levelRating(level int, rate float64) int {
	return int(math.Floor(float64(max(level, 0))*rate + 1e-9))
}

// BaseSkills are Attack and Evasion at a level without armor or offset,
// for the level-up report and help.
func (c *Character) BaseSkills(level int) (attack, evasion int) {
	a, e := c.skillRates()
	return levelRating(level, a), levelRating(level, e)
}

// SkillEdge turns a rating gap into an edge: (attack − evasion) over the
// configured SkillEdgeSpan, held to −1..1.
func SkillEdge(attack, evasion int) float64 {
	span := float64(configs.GetCombatConfig().SkillEdgeSpan)
	if span <= 0 || math.IsNaN(span) || math.IsInf(span, 0) {
		span = 16
	}
	return max(-1, min(1, float64(attack-evasion)/span))
}

// ArmorBulk is the bulk of the heaviest armor worn (light when none).
func (c *Character) ArmorBulk() string {
	best := items.BulkLight
	for _, slot := range items.ArmorSlots() {
		itm := c.Equipment.Get(slot)
		if itm == nil || itm.ItemId == 0 {
			continue
		}
		spec := itm.GetSpec()
		if !spec.IsArmor() {
			continue
		}
		if archetypes.BulkRank(spec.Bulk) > archetypes.BulkRank(best) {
			best = spec.Bulk
		}
	}
	return best
}

// ArmorTraining is the heaviest bulk the character wears without penalty:
// the class's training, heavy without a class.
func (c *Character) ArmorTraining() string {
	if p, ok := c.combatProfile(); ok && p.ArmorTraining != "" {
		return p.ArmorTraining
	}
	return items.BulkHeavy
}

// UntrainedArmor reports whether the character wears armor heavier than
// it is trained for.
func (c *Character) UntrainedArmor() bool {
	return archetypes.BulkRank(c.ArmorBulk()) > archetypes.BulkRank(c.ArmorTraining())
}

// ArmorChantDelay is the extra chant rounds untrained armor costs, sampled
// when a cast starts, beside ColdDelay.
func (c *Character) ArmorChantDelay() int {
	if c == nil || !c.UntrainedArmor() {
		return 0
	}
	return int(configs.GetCombatConfig().UntrainedChantRounds)
}

// bulkLoss is the configured tempo or dodge share lost for a bulk,
// doubled when untrained and held to 0..1.
func (c *Character) bulkLoss(medium, heavy float64) float64 {
	loss := 0.0
	switch c.ArmorBulk() {
	case items.BulkMedium:
		loss = medium
	case items.BulkHeavy:
		loss = heavy
	}
	if c.UntrainedArmor() {
		loss *= 2
	}
	return max(0, min(1, loss))
}

// BulkTempoFactor is the share of tempo armor bulk leaves.
func (c *Character) BulkTempoFactor() float64 {
	cfg := configs.GetCombatConfig()
	return 1 - c.bulkLoss(float64(cfg.BulkTempoMedium), float64(cfg.BulkTempoHeavy))
}

// BulkDodgeFactor is the share of dodge armor bulk leaves.
func (c *Character) BulkDodgeFactor() float64 {
	cfg := configs.GetCombatConfig()
	return 1 - c.bulkLoss(float64(cfg.BulkDodgeMedium), float64(cfg.BulkDodgeHeavy))
}

// GearOf is what the class rules need to know about an item.
func GearOf(spec items.ItemSpec) archetypes.Gear {
	return archetypes.Gear{
		Shield:      spec.IsShield(),
		ShieldSize:  spec.ShieldSize,
		Weapon:      spec.Type == items.Weapon,
		WeaponClass: spec.WeaponClass,
	}
}

// CanWield applies the character's class rules on shields and weapons
// (Phase 35a2); reason is player-facing when refused.
func (c *Character) CanWield(itm items.Item) (bool, string) {
	spec := itm.GetSpec()
	// Phase 38e: a creature wears only gear cut for its species, and nobody
	// else wears that gear.
	if ok, reason := creatures.CanWear(c.ArchetypeID(), spec.WornBy); !ok {
		return false, reason
	}
	return archetypes.CanWield(c.ArchetypeID(), GearOf(spec))
}

// WouldBeUntrained reports whether wearing the item would put the
// character in armor heavier than their training (equip's warning).
func (c *Character) WouldBeUntrained(itm items.Item) bool {
	spec := itm.GetSpec()
	return spec.IsArmor() && archetypes.BulkRank(spec.Bulk) > archetypes.BulkRank(c.ArmorTraining())
}

// HealingBonusPct is the percent a healer's gear adds to its heals (the
// holy symbol's +5).
func (c *Character) HealingBonusPct() int {
	return max(0, c.StatMod("healing")) + c.ClassEffects().Int(classes.HealPct)
}

// UnequipDisallowed moves any held shield or weapon the class may not use
// from the hands to carried items (Phase 35a2, existing characters). It
// returns what was moved; nothing when everything is allowed, so it is
// safe to run on every load.
func (c *Character) UnequipDisallowed() []items.Item {
	if c == nil || c.ArchetypeID() == "" {
		return nil
	}
	var moved []items.Item
	for _, slot := range items.WeaponSlots() {
		itm := c.Equipment.Get(slot)
		if itm == nil || itm.ItemId == 0 {
			continue
		}
		if ok, _ := c.CanWield(*itm); ok {
			continue
		}
		held := *itm
		*itm = items.Item{}
		c.StoreItem(held)
		moved = append(moved, held)
	}
	if len(moved) > 0 {
		c.reapplyPermabuffs(moved...)
		c.Validate(true)
	}
	return moved
}

// ManaRates is the character's mana pool base and gain a level (Phase
// 35b): its class's (a player's archetype, else a companion's), else an
// enemy template's override, else the progression defaults.
func (c *Character) ManaRates() (base int, perLevel float64) {
	cfg := configs.GetProgressionConfig()
	base, perLevel = int(cfg.ManaBase), float64(cfg.ManaPerLevel)
	if p, ok := c.combatProfile(); ok {
		if p.ManaBase > 0 {
			base = p.ManaBase
		}
		if p.ManaPerLevel > 0 {
			perLevel = p.ManaPerLevel
		}
		return base, perLevel
	}
	if c.userId == 0 {
		if c.ManaBaseOverride > 0 {
			base = c.ManaBaseOverride
		}
		if c.ManaPerLevelOverride > 0 {
			perLevel = c.ManaPerLevelOverride
		}
	}
	return base, perLevel
}

// ClassTitle is the class a player sees beside a name: the chosen
// archetype ("Warrior"), else the skill-derived profession title. The
// profession title alone read "scrub paladin" for every Ashveil warrior.
func (c *Character) ClassTitle() string {
	if id := c.ArchetypeID(); id != "" {
		if name, ok := archetypes.Name(id); ok && name != "" {
			return name
		}
	}
	return skills.GetProfession(c.GetAllSkillRanks())
}
