package dolls

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestGiftsFromRoutes(t *testing.T) {
	base := GiftsFor(classes.EffectsForLineage("dollmaster", "", 1, nil))
	assert.Equal(t, 1, base.Count)
	assert.Equal(t, BaseHPPct, base.HPPct)
	two := GiftsFor(classes.EffectsForLineage("dollmaster", "puppeteer", 10, nil))
	assert.Equal(t, 2, two.Count)
	assert.Equal(t, 35, two.HPPct, "each of two dolls has half the doll health")
	golem := GiftsFor(classes.EffectsForLineage("dollmaster", "golemancer", 10, nil))
	assert.Equal(t, 1, golem.Count)
	assert.Equal(t, 91, golem.HPPct, "130% of 70%")
	assert.Equal(t, 15, golem.Armor)
	assert.True(t, golem.NoWear)
	hardwood := GiftsFor(classes.EffectsForLineage("dollmaster", "", 5, []string{"hardwood"}))
	assert.Equal(t, 77, hardwood.HPPct, "70% x 110%")
}

func TestMendPartsMendBrokenDollsFirst(t *testing.T) {
	c := &characters.Character{Level: 4}
	c.EnsureDolls(2)
	c.Dolls[0].Damage = 5
	c.Dolls[1].Broken, c.Dolls[1].Damage = true, 30
	per := MendPerPart(4)
	assert.Equal(t, 22, per)
	used, _ := Mend(c, 1)
	assert.Equal(t, 1, used)
	assert.False(t, c.Dolls[1].Broken, "the first part stands a broken doll up")
	assert.Equal(t, 8, c.Dolls[1].Damage)
	assert.Equal(t, 5, c.Dolls[0].Damage)
	used, _ = Mend(c, 5)
	assert.Equal(t, 2, used, "only what needs mending is used")
	assert.False(t, NeedsMending(c))
	used, _ = Mend(c, 3)
	assert.Zero(t, used)
}

func TestPartsAreTakenFromThePack(t *testing.T) {
	c := &characters.Character{}
	for i := 0; i < 3; i++ {
		c.Items = append(c.Items, items.Item{ItemId: PartsItemID})
	}
	c.Items = append(c.Items, items.Item{ItemId: 30})
	assert.Equal(t, 3, Parts(c))
	taken := TakeParts(c, 2)
	assert.Len(t, taken, 2)
	assert.Equal(t, 1, Parts(c))
	assert.Len(t, c.Items, 2)
}

func TestMendCompanyHonorsOrderAndParts(t *testing.T) {
	leader := &characters.Character{Level: 3}
	leader.EnsureDolls(1)
	leader.Dolls[0].Damage = 100
	other := &characters.Character{Level: 3}
	other.EnsureDolls(1)
	other.Dolls[0].Damage = 100
	leader.Items = append(leader.Items, items.Item{ItemId: PartsItemID}, items.Item{ItemId: PartsItemID})
	taken, used := MendCompany(leader, []*characters.Character{leader, other})
	assert.Len(t, taken, 2)
	assert.Equal(t, []int{2, 0}, used, "the leader's dolls are mended first, until the parts run out")
	assert.Zero(t, Parts(leader))
}

func TestConditionWords(t *testing.T) {
	assert.Equal(t, "whole", Condition(characters.DollState{}, 40))
	assert.Equal(t, "worn", Condition(characters.DollState{Damage: 5}, 40))
	assert.Equal(t, "badly worn", Condition(characters.DollState{Damage: 20}, 40))
	assert.Equal(t, "broken", Condition(characters.DollState{Broken: true}, 40))
}

// A doll's record is part of its Master's saved character.
func TestDollRecordsSurviveYAML(t *testing.T) {
	c := characters.Character{Name: "Tamsin"}
	c.EnsureDolls(2)
	c.Dolls[1].Name, c.Dolls[1].Broken, c.Dolls[1].Damage = "Pim", true, 12
	c.Dolls[0].Equipment.Body = items.Item{ItemId: 20008}
	c.Dolls[0].Equipment.Weapon = items.Item{ItemId: characters.DollStarterWeapon}
	data, err := yaml.Marshal(c)
	assert.NoError(t, err)
	var back characters.Character
	assert.NoError(t, yaml.Unmarshal(data, &back))
	assert.Len(t, back.Dolls, 2)
	assert.Equal(t, "Pim", back.Dolls[1].Name)
	assert.True(t, back.Dolls[1].Broken)
	assert.Equal(t, 12, back.Dolls[1].Damage)
	assert.Equal(t, 20008, back.Dolls[0].Equipment.Body.ItemId)
	assert.Equal(t, characters.DollStarterWeapon, back.Dolls[0].Equipment.Weapon.ItemId)
}
