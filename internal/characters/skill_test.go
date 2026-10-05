package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skillClasses serves the shipped 35a2 combat profiles; user 7 is a
// cleric, every other player has no class.
type skillClasses struct{}

var skillProfiles = map[string]archetypes.Profile{
	"warrior": {Name: "Warrior", AttackRate: 1, EvasionRate: 1, HPStart: 10, ArmorTraining: items.BulkHeavy},
	"rogue":   {Name: "Rogue", AttackRate: 0.9, EvasionRate: 1.1, HPStart: 4, ArmorTraining: items.BulkLight, ShieldSizes: []string{"none"}},
	"ranger":  {Name: "Ranger", AttackRate: 1, EvasionRate: 0.9, HPStart: 6, ArmorTraining: items.BulkMedium, ShieldSizes: []string{items.ShieldBuckler}},
	"cleric":  {Name: "Cleric", AttackRate: 0.7, EvasionRate: 0.75, HPStart: 2, ArmorTraining: items.BulkLight, ShieldSizes: []string{"none"}, WeaponClasses: []string{"staff", "rod", "mace"}},
	"wizard":  {Name: "Wizard", AttackRate: 0.7, EvasionRate: 0.75, ArmorTraining: items.BulkLight, ShieldSizes: []string{"none"}},
}

func (skillClasses) CanTrain(int, string) (bool, string)      { return true, "" }
func (skillClasses) CanLearnSpell(int, string) (bool, string) { return true, "" }
func (skillClasses) Exists(id string) bool                    { _, ok := skillProfiles[id]; return ok }
func (skillClasses) ArchetypeName(id string) (string, bool) {
	p, ok := skillProfiles[id]
	return p.Name, ok
}
func (skillClasses) PlayerArchetype(userID int) (string, bool) {
	if userID == 7 {
		return "cleric", true
	}
	return "", false
}
func (skillClasses) CombatProfile(id string) (archetypes.Profile, bool) {
	p, ok := skillProfiles[id]
	return p, ok
}

const (
	skillRobeID    = 99601 // light
	skillLeatherID = 99602 // medium by weight (3 kg)
	skillPlateID   = 99603 // heavy
	skillBucklerID = 99604
	skillTowerID   = 99605
	skillMaceID    = 99606
	skillClubID    = 99607
	skillSymbolID  = 99608
)

func skillSetup(t *testing.T) {
	t.Helper()
	archetypes.SetProvider(skillClasses{})
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	g := configs.GetGamePlayConfig()
	g.Combat.SkillEdgeSpan = 20
	g.Combat.DefaultAttackRate, g.Combat.DefaultEvasionRate = 1, 1
	g.Combat.BulkTempoMedium, g.Combat.BulkTempoHeavy = 0.08, 0.20
	g.Combat.BulkDodgeMedium, g.Combat.BulkDodgeHeavy = 0.2, 0.5
	g.Combat.UntrainedSkillLoss, g.Combat.UntrainedChantRounds = 10, 1
	t.Cleanup(configs.SetTestGamePlayConfig(g))
	specs := []*items.ItemSpec{
		{ItemId: skillRobeID, Name: "test robe", Type: items.Body, Subtype: items.Wearable, Weight: 1000, DamageReduction: 1},
		{ItemId: skillLeatherID, Name: "test leather", Type: items.Body, Subtype: items.Wearable, Weight: 3000, DamageReduction: 4},
		{ItemId: skillPlateID, Name: "test plate", Type: items.Body, Subtype: items.Wearable, Weight: 12000, DamageReduction: 10},
		{ItemId: skillBucklerID, Name: "test buckler", Type: items.Offhand, Subtype: items.Wearable, Weight: 1500, DamageReduction: 3, ShieldSize: items.ShieldBuckler},
		{ItemId: skillTowerID, Name: "test tower", Type: items.Offhand, Subtype: items.Wearable, Weight: 9000, DamageReduction: 14, ShieldSize: items.ShieldTower, Bulk: items.BulkHeavy},
		{ItemId: skillMaceID, Name: "test mace", Type: items.Weapon, Subtype: items.Bludgeoning, Weight: 1400, WeaponClass: "mace", Damage: items.Damage{DiceRoll: "1d6"}},
		{ItemId: skillClubID, Name: "test club", Type: items.Weapon, Subtype: items.Bludgeoning, Weight: 1400, WeaponClass: "club", Damage: items.Damage{DiceRoll: "1d6"}},
		{ItemId: skillSymbolID, Name: "test symbol", Type: items.Offhand, Subtype: items.Wearable, Weight: 300, StatMods: map[string]int{"healing": 5}},
	}
	for _, s := range specs {
		require.NoError(t, s.Validate(), s.Name)
		items.SetTestItemSpec(s)
	}
	t.Cleanup(func() {
		for _, s := range specs {
			items.RemoveTestItemSpec(s.ItemId)
		}
	})
}

func classed(class string, level int) *Character {
	c := New()
	c.RaceId = 1 // a human, when another test has loaded races: both hands free
	c.HPArchetype = class
	c.Level = level
	return c
}

// Attack and Evasion are floor(level × the class rate) plus a template
// offset; enemies take the default rates, and a lost level lowers both.
func TestSkillRatings(t *testing.T) {
	skillSetup(t)
	for _, tc := range []struct {
		class           string
		level           int
		attack, evasion int
	}{
		{"warrior", 30, 30, 30},
		{"rogue", 30, 27, 33},
		{"ranger", 30, 30, 27},
		{"cleric", 30, 21, 22},
		{"wizard", 30, 21, 22},
		{"wizard", 1, 0, 0},
		{"rogue", 10, 9, 11},
		{"", 30, 30, 30},
	} {
		c := classed(tc.class, tc.level)
		assert.Equal(t, tc.attack, c.AttackSkill(), "%s %d attack", tc.class, tc.level)
		assert.Equal(t, tc.evasion, c.Evasion(), "%s %d evasion", tc.class, tc.level)
	}

	brute := classed("", 20)
	brute.AttackOffset, brute.EvasionOffset = 3, -2
	assert.Equal(t, 23, brute.AttackSkill())
	assert.Equal(t, 18, brute.Evasion())

	w := classed("warrior", 12)
	w.Level--
	assert.Equal(t, 11, w.AttackSkill(), "a lost level lowers Attack")
	assert.Equal(t, 11, w.Evasion(), "and Evasion")

	player := New()
	player.SetUserId(7)
	player.Level = 30
	assert.Equal(t, "cleric", player.ArchetypeID(), "a player's class comes from the provider")
	assert.Equal(t, 21, player.AttackSkill())
	assert.Equal(t, 2, player.HPStart())
	a, e := player.BaseSkills(31)
	assert.Equal(t, [2]int{21, 23}, [2]int{a, e}, "the level-up report's next ratings")
}

func TestSkillEdgeBounds(t *testing.T) {
	skillSetup(t)
	assert.Equal(t, 0.0, SkillEdge(10, 10))
	assert.Equal(t, 0.5, SkillEdge(20, 10))
	assert.Equal(t, -0.5, SkillEdge(10, 20))
	assert.Equal(t, 1.0, SkillEdge(60, 0), "held to a full edge")
	assert.Equal(t, -1.0, SkillEdge(0, 60))
}

// Bulk is the heaviest armor worn; untrained armor doubles its costs,
// takes 10 Attack and Evasion, and adds a chant round through SetCast.
func TestArmorBulkAndTraining(t *testing.T) {
	skillSetup(t)
	assert.Equal(t, items.BulkLight, items.GetItemSpec(skillRobeID).Bulk, "light by weight")
	assert.Equal(t, items.BulkMedium, items.GetItemSpec(skillLeatherID).Bulk, "medium by weight")
	assert.Equal(t, items.BulkHeavy, items.GetItemSpec(skillPlateID).Bulk, "heavy by weight")
	assert.Equal(t, items.ShieldBuckler, items.GetItemSpec(skillBucklerID).ShieldSize)

	for _, tc := range []struct {
		class     string
		body      int
		bulk      string
		untrained bool
		tempo     float64
		dodge     float64
	}{
		{"warrior", skillPlateID, items.BulkHeavy, false, 0.80, 0.5},
		{"warrior", skillLeatherID, items.BulkMedium, false, 0.92, 0.8},
		{"ranger", skillLeatherID, items.BulkMedium, false, 0.92, 0.8},
		{"ranger", skillPlateID, items.BulkHeavy, true, 0.60, 0},
		{"rogue", skillLeatherID, items.BulkMedium, true, 0.84, 0.6},
		{"wizard", skillRobeID, items.BulkLight, false, 1, 1},
		{"", skillPlateID, items.BulkHeavy, false, 0.80, 0.5},
	} {
		c := classed(tc.class, 20)
		c.Equipment.Body = items.New(tc.body)
		assert.Equal(t, tc.bulk, c.ArmorBulk(), tc.class)
		assert.Equal(t, tc.untrained, c.UntrainedArmor(), tc.class)
		assert.InDelta(t, tc.tempo, c.BulkTempoFactor(), 1e-9, "%s tempo", tc.class)
		assert.InDelta(t, tc.dodge, c.BulkDodgeFactor(), 1e-9, "%s dodge", tc.class)
		base := classed(tc.class, 20)
		loss := 0
		if tc.untrained {
			loss = 10
		}
		assert.Equal(t, base.AttackSkill()-loss, c.AttackSkill(), "%s attack", tc.class)
		assert.Equal(t, base.Evasion()-loss, c.Evasion(), "%s evasion", tc.class)
	}

	wizard := classed("wizard", 10)
	wizard.SetCast(1, SpellAggroInfo{SpellId: "mm"})
	assert.Equal(t, 1, wizard.Aggro.RoundsWaiting, "a trained caster chants as listed")
	wizard.Equipment.Body = items.New(skillPlateID)
	wizard.SetCast(1, SpellAggroInfo{SpellId: "mm"})
	assert.Equal(t, 2, wizard.Aggro.RoundsWaiting, "untrained armor adds a chant round")

	tower := classed("warrior", 10)
	tower.Equipment.Offhand = items.New(skillTowerID)
	assert.Equal(t, items.BulkHeavy, tower.ArmorBulk(), "a tower shield is heavy bulk")
	ranger := classed("ranger", 10)
	assert.True(t, ranger.WouldBeUntrained(items.New(skillPlateID)))
	assert.False(t, ranger.WouldBeUntrained(items.New(skillLeatherID)))
}

// Class gear rules hold in Wear itself, and UnequipDisallowed moves what
// a class may not hold to carried items, once.
func TestClassGearRules(t *testing.T) {
	skillSetup(t)
	for _, tc := range []struct {
		class string
		item  int
		ok    bool
		why   string
	}{
		{"warrior", skillTowerID, true, ""},
		{"warrior", skillBucklerID, true, ""},
		{"ranger", skillBucklerID, true, ""},
		{"ranger", skillTowerID, false, "Rangers carry only bucklers."},
		{"rogue", skillBucklerID, false, "Rogues don't carry shields."},
		{"cleric", skillMaceID, true, ""},
		{"cleric", skillClubID, false, "Clerics fight with staffs, rods and maces."},
		{"cleric", skillSymbolID, true, ""},
		{"wizard", skillClubID, true, ""},
		{"", skillTowerID, true, ""},
	} {
		c := classed(tc.class, 5)
		ok, why := c.CanWield(items.New(tc.item))
		assert.Equal(t, tc.ok, ok, "%s %d", tc.class, tc.item)
		assert.Equal(t, tc.why, why, "%s %d", tc.class, tc.item)
		if !tc.ok { // Wear refuses before it looks at the race's hands
			_, worn, reason := c.Wear(items.New(tc.item))
			assert.False(t, worn, "%s %d through Wear", tc.class, tc.item)
			assert.Equal(t, tc.why, reason)
		}
	}

	cleric := classed("cleric", 5)
	cleric.Equipment.Weapon = items.New(skillClubID)
	cleric.Equipment.Offhand = items.New(skillBucklerID)
	moved := cleric.UnequipDisallowed()
	assert.Len(t, moved, 2)
	assert.Zero(t, cleric.Equipment.Weapon.ItemId)
	assert.Zero(t, cleric.Equipment.Offhand.ItemId)
	carried := map[int]bool{}
	for _, it := range cleric.Items {
		carried[it.ItemId] = true
	}
	assert.True(t, carried[skillClubID] && carried[skillBucklerID], "moved to carried items")
	assert.Empty(t, cleric.UnequipDisallowed(), "a second load moves nothing")

	symbol := classed("cleric", 5)
	symbol.Equipment.Offhand = items.New(skillSymbolID)
	symbol.Validate(true)
	assert.Empty(t, symbol.UnequipDisallowed(), "a holy symbol stays in hand")
	assert.Equal(t, 5, symbol.HealingBonusPct())
}
