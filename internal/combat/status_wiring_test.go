package combat

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/status"
)

// forceCrits makes every strike land and critically hit, undodged.
func forceCrits(t *testing.T) {
	t.Helper()
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.ToHitMin, gameplay.Combat.ToHitMax = 100, 100
	gameplay.Combat.CritChanceMin, gameplay.Combat.CritChanceMax = 100, 100
	gameplay.Combat.DodgeChanceMin, gameplay.Combat.DodgeChanceMax = 0, 0
	gameplay.Combat.CritMultMin, gameplay.Combat.CritMultMax = 1, 1
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
}

// TestCritLeavesItsWeaponsStatus drives the real resolver with a forced crit
// for each weapon subtype: the target gets the subtype's status, and the
// hit line names it in its parentheses.
func TestCritLeavesItsWeaponsStatus(t *testing.T) {
	edgeSpecs(t)
	buffs.LoadFlagDataFiles()
	buffs.LoadDataFiles()
	forceCrits(t)

	cases := []struct {
		sub  items.ItemSubType
		want []int
		word string
	}{
		{items.Slashing, []int{status.Bleeding}, "bleeding"},
		{items.Claws, []int{status.Bleeding}, "bleeding"},
		{items.Stabbing, []int{status.Bleeding, status.Bleeding}, "bleeding"},
		{items.Bludgeoning, []int{status.Staggered}, "staggered"},
		{items.Shooting, []int{status.Exposed}, "exposed"},
		{items.Whipping, []int{status.Hobbled}, "hobbled"},
	}
	for i, c := range cases {
		id := 99300 + i
		items.SetTestItemSpec(&items.ItemSpec{ItemId: id, Name: "test weapon", Type: items.Weapon, Subtype: c.sub, Hands: 1,
			Damage: items.Damage{DiceRoll: "1d4", Attacks: 1, DiceCount: 1, SideCount: 4}})
		t.Cleanup(func() { items.RemoveTestItemSpec(id) })

		source := edgeFighter(90231)
		source.Equipment.Weapon = items.New(id)
		target := edgeFighter(90231)

		result := calculateCombat(*source, *target, User, User, 0, 0)
		if !result.Hit || !result.Crit {
			t.Fatalf("%s: forced crit expected, got %+v", c.sub, result)
		}
		if len(result.BuffTarget) != len(c.want) {
			t.Fatalf("%s: buffs %v, want %v", c.sub, result.BuffTarget, c.want)
		}
		for j := range c.want {
			if result.BuffTarget[j] != c.want[j] {
				t.Fatalf("%s: buffs %v, want %v", c.sub, result.BuffTarget, c.want)
			}
		}
		line := strings.Join(result.MessagesToSource, "\n")
		if !strings.Contains(line, "critical hit") || !strings.Contains(line, ", "+c.word+", wounded)") {
			t.Fatalf("%s: hit line does not name %q: %q", c.sub, c.word, line)
		}
	}
}

func TestPlainHitLeavesNoStatus(t *testing.T) {
	edgeSpecs(t)
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.ToHitMin, gameplay.Combat.ToHitMax = 100, 100
	gameplay.Combat.CritChanceMin, gameplay.Combat.CritChanceMax = 0, 0
	gameplay.Combat.DodgeChanceMin, gameplay.Combat.DodgeChanceMax = 0, 0
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))

	source := edgeFighter(90231)
	source.Equipment.Weapon = items.New(edgeSwordID)
	result := calculateCombat(*source, *edgeFighter(90231), User, User, 0, 0)
	if !result.Hit || result.Crit || len(result.BuffTarget) != 0 {
		t.Fatalf("a plain hit must leave nothing: %+v", result)
	}
}

// A crit the armor takes entirely reads as a miss and leaves no status.
func TestFullyBlockedCritLeavesNoStatus(t *testing.T) {
	edgeSpecs(t)
	forceCrits(t)
	const plateID = 99310
	items.SetTestItemSpec(&items.ItemSpec{ItemId: plateID, Name: "test plate", Type: items.Body, DamageReduction: 100000})
	t.Cleanup(func() { items.RemoveTestItemSpec(plateID) })

	seen := 0
	for i := 0; i < 4000 && seen < 3; i++ {
		source := edgeFighter(90231)
		source.Equipment.Weapon = items.New(edgeSwordID)
		target := edgeFighter(90231)
		target.Equipment.Body = items.New(plateID)
		result := calculateCombat(*source, *target, User, User, 0, 0)
		if !result.Crit || result.DamageToTarget > 0 {
			continue
		}
		seen++
		if len(result.BuffTarget) != 0 {
			t.Fatalf("a fully blocked crit left %v", result.BuffTarget)
		}
	}
	if seen == 0 {
		t.Fatal("no crit was fully blocked")
	}
}

func TestWeaponsOwnCritBuffsOverrideTheTable(t *testing.T) {
	edgeSpecs(t)
	forceCrits(t)
	const flamingID = 99311
	items.SetTestItemSpec(&items.ItemSpec{ItemId: flamingID, Name: "test flamebrand", Type: items.Weapon, Subtype: items.Slashing, Hands: 1,
		Damage: items.Damage{DiceRoll: "1d4", Attacks: 1, DiceCount: 1, SideCount: 4, CritBuffIds: []int{status.Burning}}})
	t.Cleanup(func() { items.RemoveTestItemSpec(flamingID) })

	source := edgeFighter(90231)
	source.Equipment.Weapon = items.New(flamingID)
	result := calculateCombat(*source, *edgeFighter(90231), User, User, 0, 0)
	if len(result.BuffTarget) != 1 || result.BuffTarget[0] != status.Burning {
		t.Fatalf("override should win, got %v", result.BuffTarget)
	}
	if !strings.Contains(strings.Join(result.MessagesToSource, "\n"), ", burning, wounded)") {
		t.Fatalf("the override's status is named: %q", result.MessagesToSource)
	}
}

func TestArmorBrokenHalvesDefenseAndExposedRaisesCritChance(t *testing.T) {
	edgeSpecs(t)
	buffs.LoadFlagDataFiles()
	buffs.LoadDataFiles()
	const plateID = 99312
	items.SetTestItemSpec(&items.ItemSpec{ItemId: plateID, Name: "test plate", Type: items.Body, DamageReduction: 40})
	t.Cleanup(func() { items.RemoveTestItemSpec(plateID) })

	c := edgeFighter(90231)
	c.Equipment.Body = items.New(plateID)
	whole := c.GetDefense()
	if whole == 0 {
		t.Skip("the test plate rolled no defense")
	}
	c.AddBuff(status.ArmorBroken, false)
	if broken := c.GetDefense(); broken != whole/2 {
		t.Fatalf("broken defense %d, want half of %d", broken, whole)
	}

	// Exposed: a crit floor of 0 becomes the bonus.
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.CritChanceMin, gameplay.Combat.CritChanceMax = 0, 0
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	src, tgt := edgeFighter(90231), edgeFighter(90231)
	plain := 0
	for i := 0; i < 400; i++ {
		if Crits(*src, *tgt) {
			plain++
		}
	}
	tgt.AddBuff(status.Exposed, false)
	exposed := 0
	for i := 0; i < 400; i++ {
		if Crits(*src, *tgt) {
			exposed++
		}
	}
	if plain != 0 || exposed < 60 || exposed > 140 {
		t.Fatalf("crits over 400 rolls: plain %d (want 0), exposed %d (want ~100)", plain, exposed)
	}
}

// Phase 30a review: every critical strike in a round leaves its status; a
// later crit never replaces an earlier one's.
func TestEveryCritStrikeInARoundLeavesItsStatus(t *testing.T) {
	edgeSpecs(t)
	forceCrits(t)
	const twinID = 99313
	items.SetTestItemSpec(&items.ItemSpec{ItemId: twinID, Name: "test twin blade", Type: items.Weapon, Subtype: items.Slashing, Hands: 1,
		Damage: items.Damage{DiceRoll: "2@1d4", Attacks: 2, DiceCount: 1, SideCount: 4}})
	t.Cleanup(func() { items.RemoveTestItemSpec(twinID) })

	source := edgeFighter(90231)
	source.Equipment.Weapon = items.New(twinID)
	result := calculateCombat(*source, *edgeFighter(90231), User, User, 0, 0)
	if len(result.BuffTarget) != 2 || result.BuffTarget[0] != status.Bleeding || result.BuffTarget[1] != status.Bleeding {
		t.Fatalf("two critical cuts leave two bleeds, got %v", result.BuffTarget)
	}
}

// Owner, 2026-09-30: a stunned fighter can't dodge, and can't block with a
// shield (its armor still takes its share).
func TestStunnedCantDodgeOrBlock(t *testing.T) {
	edgeSpecs(t)
	buffs.LoadFlagDataFiles()
	buffs.LoadDataFiles()
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.ToHitMin, gameplay.Combat.ToHitMax = 100, 100
	gameplay.Combat.DodgeChanceMin, gameplay.Combat.DodgeChanceMax = 100, 100
	gameplay.Combat.CritChanceMin, gameplay.Combat.CritChanceMax = 0, 0
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))

	source := edgeFighter(90231)
	source.Equipment.Weapon = items.New(edgeSwordID)
	target := edgeFighter(90231)
	if r := calculateCombat(*source, *target, User, Mob, 0, 0); !strings.Contains(strings.Join(r.MessagesToTarget, "\n"), "twist aside") {
		t.Fatalf("unstunned, a certain dodge: %+v", r.MessagesToTarget)
	}
	target.AddBuff(status.Stunned, false)
	if r := calculateCombat(*source, *target, User, Mob, 0, 0); !r.Hit || strings.Contains(strings.Join(r.MessagesToTarget, "\n"), "twist aside") {
		t.Fatalf("stunned, no dodge: hit %v, %q", r.Hit, r.MessagesToTarget)
	}

	const plateID, shieldID = 99313, 99314
	items.SetTestItemSpec(&items.ItemSpec{ItemId: plateID, Name: "test plate", Type: items.Body, DamageReduction: 40})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: shieldID, Name: "test shield", Type: items.Offhand, DamageReduction: 20})
	t.Cleanup(func() { items.RemoveTestItemSpec(plateID); items.RemoveTestItemSpec(shieldID) })
	c := edgeFighter(90231)
	c.Equipment.Body = items.New(plateID)
	bare := c.GetDefense()
	c.Equipment.Offhand = items.New(shieldID)
	shielded := c.GetDefense()
	if shielded <= bare {
		t.Fatalf("the shield adds no defense: bare %d, shielded %d", bare, shielded)
	}
	c.AddBuff(status.Stunned, false)
	armorOnly := bare + c.Equipment.Offhand.GetDefense() // the shield's own armor, no +50%
	if got := c.GetDefense(); got != armorOnly {
		t.Fatalf("stunned defense %d, want %d (armor without the shield's block; shielded %d)", got, armorOnly, shielded)
	}
}
