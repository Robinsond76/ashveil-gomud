package company

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/creatures"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 38e wiring: creature recruits (a hound and a stone golem) through the
// real company commands, the shipped config, and the shipped mobs.

// creatureBrawl replaces Garrick and Ysolde with Brindle (a hound) and Cairn
// (a stone golem); they are companions #5 and #6.
func creatureBrawl(t *testing.T) *brawl {
	t.Helper()
	b := newBrawl(t)
	b.aria.Character.CompanyCargo = true
	b.aria.Character.CargoMigrated = true
	b.withArchetypesFor("", nil)
	require.Contains(t, b.cmd("company", "dismiss #3"), "")
	require.Contains(t, b.cmd("company", "dismiss #4"), "")
	require.Contains(t, b.cmd("company", "summon brindle"), "Companion summoned")
	require.Contains(t, b.cmd("company", "summon cairn"), "Companion summoned")
	return b
}

func TestCreatureRecruitsEnlistWithTheirSpecies(t *testing.T) {
	b := creatureBrawl(t)
	record, _ := module.registry.Get(7)
	var archetypes []string
	for _, c := range record.Companions {
		archetypes = append(archetypes, c.Archetype)
	}
	assert.Contains(t, archetypes, creatures.Hound)
	assert.Contains(t, archetypes, creatures.StoneGolem)
	assert.Contains(t, b.cmd("class", "#5"), "Hound (creature)")
	assert.Contains(t, b.cmd("class", "#6"), "Stone Golem (construct)")
	status := b.cmd("company", "status")
	assert.Contains(t, status, "Hound")
	assert.Contains(t, status, "Stone Golem")
}

func TestCreatureInspectNamesTheFamily(t *testing.T) {
	b := creatureBrawl(t)
	hound := b.cmd("company", "inspect #5")
	assert.Contains(t, hound, "biological creature")
	golem := b.cmd("company", "inspect #6")
	assert.Contains(t, golem, "construct")
	assert.Contains(t, golem, "company repair")
	assert.Contains(t, b.cmd("company", "train #6"), "creature")
}

func TestCreaturesCarryNothingAndTakeNoStarterPack(t *testing.T) {
	b := creatureBrawl(t)
	for _, id := range []int{5, 6} {
		mob := b.companion(id)
		assert.Empty(t, mob.Character.Items, "creature %d carries nothing", id)
	}
	// Aria's own pack plus two person companions' packs; none for creatures.
	assert.Len(t, module.CompanionCarry(7), 2)
}

func TestCreaturesRefuseOrdinaryGearAndPeopleRefuseCreatureGear(t *testing.T) {
	b := creatureBrawl(t)
	boots := items.New(20003)
	collar := items.New(20026)
	harness := items.New(20500)
	b.aria.Character.Items = []items.Item{boots, collar, harness}
	assert.NotContains(t, b.cmd("company", "equip #5 "+boots.ShorthandId()), "equipment updated")
	assert.NotContains(t, b.cmd("company", "equip #6 "+collar.ShorthandId()), "equipment updated")
	assert.Contains(t, b.cmd("company", "equip #5 "+collar.ShorthandId()), "equipment updated")
	assert.Contains(t, b.cmd("company", "equip #5 "+harness.ShorthandId()), "equipment updated")
	assert.NotContains(t, b.cmd("equip", collar.ShorthandId()), "equipment updated")
	assert.Zero(t, b.aria.Character.Equipment.Neck.ItemId)
}

func TestCreaturesCannotChangeArchetypeOrBeChosen(t *testing.T) {
	b := creatureBrawl(t)
	out := b.cmd("company", "archetype #1 hound")
	assert.NotContains(t, strings.ToLower(out), "set to")
	recordArch := func() string { c, _ := module.CompanionArchetype(7, 5); return c }
	before := recordArch()
	b.cmd("company", "archetype #5 warrior")
	assert.Equal(t, before, recordArch())
}

func TestGolemRepairSpendsMortarAndMendsHalf(t *testing.T) {
	b := creatureBrawl(t)
	cairn := b.companion(6)
	limit := cairn.Character.HealthLimit()
	cairn.Character.Health = 1
	assert.Contains(t, b.cmd("company", "repair"), "needs repair")
	assert.Contains(t, b.cmd("company", "repair cairn"), "no stone mortar")
	b.aria.Character.Items = []items.Item{items.New(creatures.RepairItemID), items.New(creatures.RepairItemID), items.New(creatures.RepairItemID)}
	out := b.cmd("company", "repair cairn")
	assert.Contains(t, out, "mended")
	assert.Equal(t, limit, cairn.Character.Health, out)
	assert.Contains(t, b.cmd("company", "repair cairn"), "whole")
	left := 0
	for _, it := range b.aria.Character.Items {
		if it.ItemId == creatures.RepairItemID {
			left++
		}
	}
	assert.GreaterOrEqual(t, left, 1, "only what was needed is spent")
}

func TestRepairRefusesLivingMembersAndFights(t *testing.T) {
	b := creatureBrawl(t)
	b.aria.Character.Items = []items.Item{items.New(creatures.RepairItemID)}
	assert.Contains(t, b.cmd("company", "repair brindle"), "isn't a construct")
	b.companion(6).Character.Health = 1
	b.aimAt("bandit captain")
	assert.Contains(t, b.cmd("company", "repair cairn"), "battle")
}

func TestConstructsNeedNothingAndNeverLoseHeart(t *testing.T) {
	b := creatureBrawl(t)
	cairn := b.companion(6)
	// The company status tells the player why the golem has no needs.
	status := b.cmd("company", "status")
	assert.Contains(t, status, "Stone Golem (construct), bound")
	assert.Contains(t, status, "Hound (creature), alignment")
	// Morale and drift pass the golem by and count the hound.
	var ids []int
	for _, m := range module.MoraleMembers(7) {
		ids = append(ids, m.ID)
	}
	assert.Contains(t, ids, 5, "a hound can lose heart")
	assert.NotContains(t, ids, 6, "a golem is bound")
	// Neither food nor rest is spent on the golem.
	assert.Contains(t, b.cmd("company", "eat"), "")
	assert.Equal(t, cairn.Character.Health, b.companion(6).Character.Health)
}

func TestHoundAndGolemBothFightUnderTheirOwnRules(t *testing.T) {
	b := creatureBrawl(t)
	for _, id := range []int{5, 6} {
		b.cmd("formation", "clear #"+string(rune('0'+id)))
	}
	b.cmd("attack", "bandit captain")
	b.hardenBandits()
	hound, golem := b.companion(5), b.companion(6)
	assert.Equal(t, 30, hound.Character.ClassEffects().Int(classes.Pounce), "the hound's rank is live in the real mob")
	assert.Equal(t, 25, golem.Character.ClassEffects().Int(classes.Slow), "the golem's rank is live in the real mob")
	assert.Equal(t, 25, golem.Character.ClassEffects().Int(classes.SpellWeak))
	for range 4 {
		b.fight()
	}
}
