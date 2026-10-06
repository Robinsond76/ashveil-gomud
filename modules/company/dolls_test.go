package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/dolls"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

// Phase 39d: the doll command, through the real user command path.

func dollCommandBrawl(t *testing.T) *brawl {
	t.Helper()
	b := newBrawl(t)
	b.withArchetypesFor("dollmaster", map[int]string{1: "dollmaster", 2: "cleric", 3: "warrior", 4: "ranger"})
	b.companion(1).Character.HPArchetype = "dollmaster"
	return b
}

func TestDollCommandShowsNamesAndDressesTheMastersDoll(t *testing.T) {
	b := dollCommandBrawl(t)
	c := b.aria.Character
	out := b.cmd("doll", "")
	assert.Contains(t, out, "You drive 1 doll:")
	assert.Contains(t, out, "1. Pip, whole")
	assert.Contains(t, out, "Weapon cudgel")
	assert.Contains(t, out, "Tamsin Reed drives 1 doll:", "companion Masters are listed")

	// Dress it from the pack.
	c.Items = append(c.Items, items.New(10004), items.New(20008))
	out = b.cmd("doll", "wield dagger")
	assert.Contains(t, out, "Pip takes")
	assert.Contains(t, out, "cudgel back in your pack")
	assert.Equal(t, 10004, c.Dolls[0].Equipment.Weapon.ItemId)
	out = b.cmd("doll", "wear shirt")
	assert.Contains(t, out, "Pip takes")
	assert.Equal(t, 20008, c.Dolls[0].Equipment.Body.ItemId)
	out = b.cmd("doll", "remove body")
	assert.Contains(t, out, "You take")
	assert.Zero(t, c.Dolls[0].Equipment.Body.ItemId)
	out = b.cmd("doll", "wear nothing-like-it")
	assert.Contains(t, out, "You carry no")

	out = b.cmd("doll", "name Splinter")
	assert.Contains(t, out, "You name the doll Splinter.")
	assert.Equal(t, "Splinter", c.Dolls[0].Name)
	assert.Contains(t, b.cmd("doll", "name 2 Bob"), "You have 1 doll; pick 1 to 1.")
}

func TestDollMendUsesPartsAndWontRunInBattle(t *testing.T) {
	b := dollCommandBrawl(t)
	c := b.aria.Character
	c.EnsureDolls(1)
	c.Dolls[0].Broken, c.Dolls[0].Damage = true, 40
	assert.Contains(t, b.cmd("doll", "mend"), "You have no doll parts.")
	for i := 0; i < 5; i++ {
		c.Items = append(c.Items, items.Item{ItemId: dolls.PartsItemID})
	}
	out := b.cmd("doll", "mend")
	assert.Contains(t, out, "mend the dolls with 3 doll parts")
	assert.False(t, c.Dolls[0].Broken)
	assert.Equal(t, 2, dolls.Parts(c), "three parts are spent, two are kept")
	assert.Zero(t, c.Dolls[0].Damage)
	assert.Contains(t, b.cmd("doll", "mend"), "need no mending")

	// Not in the middle of a battle.
	c.Dolls[0].Damage = 5
	captain, _, _, _, _ := b.shapeBandits()
	b.cmd("attack", "#"+itoa(captain))
	b.hardenBandits()
	b.toughen()
	b.fight() // the round opens the battle
	assert.Contains(t, b.cmd("doll", "mend"), "Not in the middle of a battle")
	assert.Contains(t, b.cmd("doll", "wield cudgel"), "Not in the middle of a battle")
}

func TestOnlyDollMastersHaveDolls(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypesFor("warrior", map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"})
	assert.Contains(t, b.cmd("doll", ""), "No one in your company drives a doll")
}

func TestAGolemsViewSaysItsBodyIsItsArmor(t *testing.T) {
	b := dollCommandBrawl(t)
	tamsin := b.companion(1)
	tamsin.Character.Level = 10
	tamsin.Character.SetClassState("golemancer", nil)
	out := b.cmd("doll", "tamsin")
	assert.Contains(t, out, "Tamsin Reed drives 1 doll:")
	assert.Contains(t, out, "its own body is its armor")
}
