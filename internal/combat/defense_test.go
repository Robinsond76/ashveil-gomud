package combat

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 30g2 test items. Each weapon rolls a fixed 1d1.
const (
	defAxeID    = 99401
	defClubID   = 99402
	defStaffID  = 99403
	defPikeID   = 99404
	defWhipID   = 99405
	defClawID   = 99406
	defSlingID  = 99407
	defShieldID = 99408
	defTowerID  = 99409
)

func defenseSpecs(t *testing.T) {
	t.Helper()
	edgeSpecs(t)
	buffs.LoadFlagDataFiles()
	buffs.LoadDataFiles()
	weapon := func(id int, name string, sub items.ItemSubType, mod func(*items.ItemSpec)) {
		spec := &items.ItemSpec{ItemId: id, Name: name, Type: items.Weapon, Subtype: sub, Hands: 1,
			Damage: items.Damage{DiceRoll: "1d1", Attacks: 1, DiceCount: 1, SideCount: 1}}
		if mod != nil {
			mod(spec)
		}
		items.SetTestItemSpec(spec)
	}
	weapon(defAxeID, "test axe", items.Cleaving, nil)
	weapon(defClubID, "test club", items.Bludgeoning, nil)
	weapon(defStaffID, "test staff", items.Bludgeoning, func(s *items.ItemSpec) { s.Parry = 5 })
	weapon(defPikeID, "test pike", items.Stabbing, func(s *items.ItemSpec) { s.Reach = true })
	weapon(defWhipID, "test whip", items.Whipping, nil)
	weapon(defClawID, "test claw", items.Claws, nil)
	weapon(defSlingID, "test sling", items.Shooting, nil)
	items.SetTestItemSpec(&items.ItemSpec{ItemId: defShieldID, Name: "test buckler", Type: items.Offhand, DamageReduction: 5})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: defTowerID, Name: "test tower shield", Type: items.Offhand, DamageReduction: 10})
	t.Cleanup(func() {
		for _, id := range []int{defAxeID, defClubID, defStaffID, defPikeID, defWhipID, defClawID, defSlingID, defShieldID, defTowerID} {
			items.RemoveTestItemSpec(id)
		}
	})
}

// defenseOdds pins every strike to hit and never crit, with the given
// block, parry, and dodge chances.
func defenseOdds(t *testing.T, block, parry, dodge int) {
	t.Helper()
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.ToHitMin, gameplay.Combat.ToHitMax = 100, 100
	gameplay.Combat.CritChanceMin, gameplay.Combat.CritChanceMax = 0, 0
	gameplay.Combat.BlockChanceMin, gameplay.Combat.BlockChanceMax = configs.ConfigInt(block), configs.ConfigInt(block)
	gameplay.Combat.ParryChanceMin, gameplay.Combat.ParryChanceMax = configs.ConfigInt(parry), configs.ConfigInt(parry)
	gameplay.Combat.DodgeChanceMin, gameplay.Combat.DodgeChanceMax = configs.ConfigInt(dodge), configs.ConfigInt(dodge)
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
}

func TestBlockChance(t *testing.T) {
	cases := []struct {
		name                  string
		armor, defStr, atkStr int
		want                  int
	}{
		{"even Strength, buckler", 5, 50, 50, 20},
		{"even Strength, tower shield", 10, 50, 50, 25},
		{"twice the Strength", 5, 100, 50, 25},
		{"half the Strength", 5, 50, 100, 15},
		{"held to the maximum", 40, 100, 0, 45},
		{"held to the minimum", 0, 0, 100, 15},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, blockChance(c.armor, c.defStr, c.atkStr), c.name)
	}
	assert.Greater(t, blockChance(10, 50, 50), blockChance(5, 50, 50), "a heavier shield blocks more")
}

func TestParryModifier(t *testing.T) {
	defenseSpecs(t)
	cases := []struct {
		name   string
		id     int
		mod    int
		parrys bool
	}{
		{"sword", edgeSwordID, 5, true},
		{"dagger", edgeDaggerID, -5, true},
		{"axe", defAxeID, 0, true},
		{"mace or club", defClubID, 0, true},
		{"staff (its own parry)", defStaffID, 5, true},
		{"polearm (reach)", defPikeID, 5, true},
		{"whip", defWhipID, 0, true},
		{"claws", defClawID, 0, false},
		{"ranged weapon", defSlingID, 0, false},
		{"no weapon", 0, 0, false},
		{"a shield is not a weapon", defShieldID, 0, false},
	}
	for _, c := range cases {
		itm := items.Item{}
		if c.id != 0 {
			itm = items.New(c.id)
		}
		mod, ok := parryModifier(itm)
		assert.Equal(t, c.parrys, ok, c.name)
		assert.Equal(t, c.mod, mod, c.name)
	}
}

func TestParryChance(t *testing.T) {
	assert.Equal(t, 10, parryChance(50, 50, 5), "even Speed, a sword: the 5% floor plus 5")
	assert.Equal(t, 0, parryChance(50, 50, -5), "even Speed, a dagger: the floor less 5")
	assert.Equal(t, 5, parryChance(50, 50, 0), "even Speed, an axe")
	assert.Equal(t, 35, parryChance(150, 50, 5), "the 30% cap plus a sword's 5")
	assert.Greater(t, parryChance(90, 50, 0), parryChance(50, 50, 0), "more Speed parries more")
}

func TestBashChance(t *testing.T) {
	assert.Equal(t, 5, BashChance(50, 50), "even Strength: the minimum")
	assert.Equal(t, 20, BashChance(150, 50), "a great Strength edge: the maximum")
	assert.Equal(t, 5, BashChance(20, 90), "a weaker bearer: the minimum")
	assert.Greater(t, BashChance(100, 50), BashChance(50, 50))
}

// strikeAt resolves one round of source on target through the real strike
// loop: a user attacking a mob.
func strikeAt(source, target *characters.Character) AttackResult {
	return calculateCombat(*source, *target, User, Mob, 0, 0)
}

func armed(weaponID int) *characters.Character {
	c := edgeFighter(90231)
	if weaponID != 0 {
		c.Equipment.Weapon = items.New(weaponID)
	}
	return c
}

// A shield-bearer blocks any strike, melee or ranged: no damage, the
// block's lines, and no dodge even when a dodge is certain.
func TestShieldBlocksInTheStrikeLoop(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 100, 0, 0)
	bearer := armed(edgeSwordID)
	bearer.Equipment.Offhand = items.New(defShieldID)

	for _, weaponID := range []int{edgeSwordID, defSlingID} {
		r := strikeAt(armed(weaponID), bearer)
		assert.False(t, r.Hit)
		assert.Zero(t, r.DamageToTarget)
		assert.Equal(t, []string{DefenseBlocked}, r.Defenses)
		assert.True(t, r.Blocked())
		assert.Contains(t, strings.Join(r.MessagesToSource, "\n"), "catches your blow on their shield")
		assert.Contains(t, strings.Join(r.MessagesToTarget, "\n"), "on your shield")
		assert.Contains(t, strings.Join(r.MessagesToSourceRoom, "\n"), "'s blow on their shield", "those watching see it")
	}

	defenseOdds(t, 0, 0, 100)
	r := strikeAt(armed(edgeSwordID), bearer)
	assert.True(t, r.Hit, "a shield-bearer whose block fails has no dodge to fall back on")
	assert.Empty(t, r.Defenses)
}

// A weapon parries a melee strike when its chance is at least the dodge's,
// and the line names the weapon; a lower parry leaves the dodge.
func TestParryOrDodgeInTheStrikeLoop(t *testing.T) {
	defenseSpecs(t)

	defenseOdds(t, 0, 100, 100)
	r := strikeAt(armed(edgeSwordID), armed(defAxeID))
	assert.Equal(t, []string{DefenseParried}, r.Defenses)
	assert.Zero(t, r.DamageToTarget)
	assert.Contains(t, strings.Join(r.MessagesToSource, "\n"), "turns your blow aside with their")
	assert.Contains(t, strings.Join(r.MessagesToSource, "\n"), "test axe")
	assert.Contains(t, strings.Join(r.MessagesToTarget, "\n"), "aside with your <ansi fg=\"item\">test axe")

	// A dagger's −5 puts its parry (95%) under the dodge (100%): the
	// dodge is the one rolled.
	r = strikeAt(armed(edgeSwordID), armed(edgeDaggerID))
	assert.Equal(t, []string{DefenseDodged}, r.Defenses)

	defenseOdds(t, 0, 0, 100)
	r = strikeAt(armed(edgeSwordID), armed(defAxeID))
	assert.Equal(t, []string{DefenseDodged}, r.Defenses, "a dodge above the parry is rolled")
}

// A ranged strike can't be parried; claws and bare hands can't parry.
func TestWhatCantBeParried(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 0, 100, 0)

	r := strikeAt(armed(defSlingID), armed(edgeSwordID))
	assert.True(t, r.Hit, "a shot is not parried")
	assert.Empty(t, r.Defenses)
	for _, weaponID := range []int{defClawID, 0} {
		r = strikeAt(armed(edgeSwordID), armed(weaponID))
		assert.True(t, r.Hit, "claws and bare hands don't parry (weapon %d)", weaponID)
	}

	defenseOdds(t, 0, 100, 100)
	r = strikeAt(armed(defSlingID), armed(edgeSwordID))
	assert.Equal(t, []string{DefenseDodged}, r.Defenses, "a shot can still be dodged")
	assert.Contains(t, strings.Join(r.MessagesToTarget, "\n"), "twist aside")
}

// A stunned fighter makes no active defense: not a block, a parry, or a
// dodge.
func TestStunnedMakesNoDefense(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 100, 100, 100)
	for _, setup := range []func(*characters.Character){
		func(c *characters.Character) { c.Equipment.Offhand = items.New(defShieldID) },
		func(c *characters.Character) {},
	} {
		target := armed(edgeSwordID)
		setup(target)
		target.AddBuff(status.Stunned, false)
		require.True(t, target.HasBuffFlag(status.FlagNoDodge))
		r := strikeAt(armed(edgeSwordID), target)
		assert.True(t, r.Hit)
		assert.Empty(t, r.Defenses)
	}
}

// Each defended strike of a round is recorded.
func TestEachDefendedStrikeIsCounted(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 0, 0, 100)
	const twoStrikes = 99410
	items.SetTestItemSpec(&items.ItemSpec{ItemId: twoStrikes, Name: "test flail", Type: items.Weapon, Subtype: items.Bludgeoning, Hands: 1,
		Damage: items.Damage{DiceRoll: "2@1d1", Attacks: 2, DiceCount: 1, SideCount: 1}})
	t.Cleanup(func() { items.RemoveTestItemSpec(twoStrikes) })
	r := strikeAt(armed(twoStrikes), armed(0))
	assert.Equal(t, []string{DefenseDodged, DefenseDodged}, r.Defenses)
	assert.False(t, r.Blocked())
}
