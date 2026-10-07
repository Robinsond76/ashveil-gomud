package items

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yamlv3 "gopkg.in/yaml.v3"
)

const (
	trophyHeartID = 99821
	trophyHideID  = 99822
	trophyBladeID = 99823
	trophyMailID  = 99824
)

func trophySpec(id int, name, part string, fx map[string]int) *ItemSpec {
	return &ItemSpec{ItemId: id, Name: name, Type: Commodity, Value: 12, Weight: 150,
		Trophy: &TrophySpec{Part: part, Races: []string{"ogre"}, Chance: 20, Effects: fx}}
}

func useTrophyItems(t *testing.T) {
	t.Helper()
	SetTestItemSpec(trophySpec(trophyHeartID, "test heart", TrophyHeart, map[string]int{classes.Damage: 1}))
	SetTestItemSpec(trophySpec(trophyHideID, "test hide", TrophyHide, map[string]int{classes.Armor: 3}))
	SetTestItemSpec(&ItemSpec{ItemId: trophyBladeID, Name: "plain blade", Type: Weapon, Subtype: Slashing, Hands: 1, Tier: 2, Value: 100,
		Damage: Damage{Attacks: 1, DiceCount: 1, SideCount: 6}})
	SetTestItemSpec(&ItemSpec{ItemId: trophyMailID, Name: "plain mail", Type: Body, Subtype: Wearable, Tier: 3, Value: 200})
	t.Cleanup(func() {
		for _, id := range []int{trophyHeartID, trophyHideID, trophyBladeID, trophyMailID} {
			RemoveTestItemSpec(id)
		}
	})
}

func TestTrophySpecsAreHeldToTheGearCaps(t *testing.T) {
	good := func() *ItemSpec { return trophySpec(1, "t", TrophyHeart, map[string]int{classes.Damage: 1}) }
	require.NoError(t, good().Trophy.validate(good()))

	cases := map[string]func(s *ItemSpec){
		"unknown part":         func(s *ItemSpec) { s.Trophy.Part = "tooth" },
		"not a commodity":      func(s *ItemSpec) { s.Type = Weapon },
		"no races":             func(s *ItemSpec) { s.Trophy.Races = nil },
		"uppercase race":       func(s *ItemSpec) { s.Trophy.Races = []string{"Ogre"} },
		"chance 0":             func(s *ItemSpec) { s.Trophy.Chance = 0 },
		"chance over 100":      func(s *ItemSpec) { s.Trophy.Chance = 101 },
		"no effects":           func(s *ItemSpec) { s.Trophy.Effects = nil },
		"unknown effect":       func(s *ItemSpec) { s.Trophy.Effects = map[string]int{"luck": 1} },
		"over a third of cap":  func(s *ItemSpec) { s.Trophy.Effects = map[string]int{classes.Damage: 2} }, // cap 4
		"a one-shot effect":    func(s *ItemSpec) { s.Trophy.Effects = map[string]int{classes.Bargain: 1} },
		"armor over its third": func(s *ItemSpec) { s.Trophy.Effects = map[string]int{classes.Armor: 4} }, // cap 10
	}
	for name, mutate := range cases {
		s := good()
		mutate(s)
		assert.Error(t, s.Trophy.validate(s), name)
	}
}

func TestAnEnchantAddsItsEffectsAndTouchesNothingElse(t *testing.T) {
	useTrophyItems(t)
	blade := New(trophyBladeID)
	sale := blade.SaleBaseValue()
	special := blade.IsSpecialForSale()

	require.NoError(t, blade.EnchantWithTrophy(trophyHeartID))
	assert.Equal(t, map[string]int{classes.Damage: 1}, blade.TrophyEffects())
	assert.Equal(t, sale, blade.SaleBaseValue(), "an enchant adds nothing to what a merchant pays")
	assert.Equal(t, special, blade.IsSpecialForSale(), "and does not make the piece unsellable (no spec override)")
	assert.Nil(t, blade.Spec, "the item's spec is untouched")
	assert.Equal(t, []string{"Enchanted with test heart (while worn): +1 damage on every landed blow."}, blade.TrophyLines())
	assert.Equal(t, blade.TrophyLines(), blade.RelicLines(), "a plain item's relic lines are its enchant, for the web client")

	assert.Contains(t, blade.EdgeLabel(), "(enchanted: test heart)", "lists tell it from a plain copy")
	plainBlade := New(trophyBladeID)
	assert.Empty(t, plainBlade.EdgeLabel())
	assert.Error(t, blade.EnchantWithTrophy(trophyHideID), "one trophy to a piece")
	mail := New(trophyMailID)
	assert.Error(t, mail.EnchantWithTrophy(trophyBladeID), "a non-trophy is refused")
	assert.Equal(t, 0, mail.Trophy)
	pack := New(trophyMailID)
	spec := pack.GetSpec()
	assert.True(t, Enchantable(&spec))
	trophy := New(trophyHeartID)
	trophySpecValue := trophy.GetSpec()
	assert.False(t, Enchantable(&trophySpecValue), "a trophy is not enchantable")
	assert.False(t, Enchantable(&ItemSpec{Type: Pack, Subtype: Wearable}), "a pack is not")
	assert.False(t, Enchantable(&ItemSpec{Type: Weapon, QuestToken: "q"}), "nor a quest item")
}

func TestTheEnchantFeeIsPerTier(t *testing.T) {
	useTrophyItems(t)
	blade, mail := New(trophyBladeID), New(trophyMailID)
	assert.Equal(t, 2*EnchantFeePerTier, blade.EnchantFee())
	assert.Equal(t, 3*EnchantFeePerTier, mail.EnchantFee())
	assert.Equal(t, EnchantFeePerTier, (&Item{}).EnchantFee(), "an unknown tier costs the least")
}

func TestEnchantedPiecesAddToGearEffectsHeldToTheAggregateCap(t *testing.T) {
	useTrophyItems(t)
	var worn []Item
	for range 4 {
		it := New(trophyMailID)
		require.NoError(t, it.EnchantWithTrophy(trophyHideID)) // armor 3 each
		worn = append(worn, it)
	}
	fx, sets := GearEffects(worn[:1])
	assert.Empty(t, sets)
	assert.Equal(t, 3, fx[classes.Armor])
	fx, _ = GearEffects(worn)
	armor, _ := classes.GearEffectFor(classes.Armor)
	assert.Equal(t, TrophyAggregateCap(armor), fx[classes.Armor], "four enchants stop at half the cap")
	assert.Equal(t, 5, TrophyAggregateCap(armor))
	fx, _ = GearEffects([]Item{New(trophyMailID)})
	assert.Empty(t, fx, "no enchant, no effect")
}

// An enchant on a relic never lifts one effect past its cap with the
// signature and awakenings.
func TestAnEnchantOnARelicStaysInsideItsCap(t *testing.T) {
	useTrophyItems(t)
	spec := testRelic(awakeBladeID, "Test Edge", Weapon, RelicSpec{Signature: "Edge", Effects: map[string]int{classes.Armor: 9}, ILvl: 20, Mob: 7, Chance: 5})
	SetTestItemSpec(spec)
	t.Cleanup(func() { RemoveTestItemSpec(awakeBladeID) })
	relic := New(awakeBladeID)
	require.NoError(t, relic.EnchantWithTrophy(trophyHideID))
	assert.Equal(t, map[string]int{classes.Armor: 1}, relic.TrophyEffects(), "armor 9 of 10 leaves 1 of the enchant's 3")
	fx, _ := GearEffects([]Item{relic})
	assert.Equal(t, 10, fx[classes.Armor])

	full := New(awakeBladeID)
	full.Trophy = trophyHideID
	spec.Relic.Effects = map[string]int{classes.Armor: 10}
	assert.Nil(t, full.TrophyEffects())
	assert.Contains(t, full.TrophyLines()[0], "adds nothing more")
}

func TestAnEnchantSurvivesSaving(t *testing.T) {
	useTrophyItems(t)
	blade := New(trophyBladeID)
	require.NoError(t, blade.EnchantWithTrophy(trophyHeartID))
	out, err := yamlv3.Marshal(blade)
	require.NoError(t, err)
	var back Item
	require.NoError(t, yamlv3.Unmarshal(out, &back))
	assert.Equal(t, trophyHeartID, back.Trophy)
	assert.Equal(t, blade.TrophyEffects(), back.TrophyEffects())
	assert.Zero(t, New(trophyBladeID).Trophy, "a plain item saves nothing extra")
}

func TestATrophyIsNeverAutoJunk(t *testing.T) {
	useTrophyItems(t)
	spec := trophySpec(trophyHeartID, "test heart", TrophyHeart, map[string]int{classes.Damage: 1})
	spec.Type = Junk
	SetTestItemSpec(spec)
	it := New(trophyHeartID)
	assert.False(t, it.IsAutoJunk(), "`sell junk` must not take a trophy the player may want")
}
