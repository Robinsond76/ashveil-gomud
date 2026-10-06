package items

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const (
	testSwordID = 986001
	testMailID  = 986002
)

func rollTestSpecs(t *testing.T) {
	t.Helper()
	sword := &ItemSpec{
		ItemId: testSwordID, Name: "short sword", NameSimple: "sword", Type: Weapon, Subtype: Slashing, Hands: 1,
		Damage: Damage{Attacks: 1, DiceCount: 1, SideCount: 6}, Value: 100, Weight: 1000,
		StatMods: statmods.StatMods{"speed": 1},
	}
	mail := &ItemSpec{
		ItemId: testMailID, Name: "chain shirt", NameSimple: "shirt", Type: Body, Subtype: Wearable,
		DamageReduction: 10, Value: 200, Weight: 5000,
	}
	SetTestItemSpec(sword)
	SetTestItemSpec(mail)
	t.Cleanup(func() { RemoveTestItemSpec(testSwordID); RemoveTestItemSpec(testMailID) })
}

func testRoll(q Quality, r Rarity, identified bool) Rolled {
	return Rolled{
		Version: RollVersion, Tier: 2, ILvl: 20, Quality: q, Rarity: r, Identified: identified, LevelReq: 15, BaseValue: 100,
		Name: "Gloomfang",
		Affixes: []RolledAffix{
			{ID: "strength", Label: "Mighty", Mechanic: "statmod:strength", Value: 3, Tier: 2, MinValue: 2, MaxValue: 3},
			{ID: "protection", Label: "of Warding", Suffix: true, Mechanic: "protection", Value: 2},
			{ID: "featherweight", Label: "Light", Mechanic: "weightpct", Value: 10},
		},
	}
}

func TestQualityScalesBaseDamageProtectionAndValue(t *testing.T) {
	rollTestSpecs(t)
	cases := []struct {
		q          Quality
		bonus, dr  int
		wantValue  int
		wantSwords string
	}{
		{QualityCrude, -1, 8, 40, "crude short sword"},
		{QualityStandard, 0, 10, 100, "short sword"},
		{QualityFine, 1, 11, 160, "fine short sword"},
		{QualityExquisite, 1, 13, 400, "exquisite short sword"},
	}
	for _, c := range cases {
		t.Run(string(c.q), func(t *testing.T) {
			sword := New(testSwordID)
			sword.ApplyRoll(testRoll(c.q, RarityCommon, true))
			sword.Loot.Affixes = nil
			sword.rebuildRolledSpec()
			spec := sword.GetSpec()
			assert.Equal(t, c.bonus, spec.Damage.BonusDamage, "weapon damage bonus")
			assert.Equal(t, c.wantValue, spec.Value, "value")
			assert.Equal(t, c.wantSwords, stripAnsi(sword.DisplayName()))

			mail := New(testMailID)
			mail.ApplyRoll(Rolled{Version: RollVersion, Quality: c.q, Rarity: RarityCommon, Identified: true, BaseValue: 200})
			assert.Equal(t, c.dr, mail.GetSpec().DamageReduction, "armor protection")
		})
	}
}

func TestExquisiteStaysAtLeastAsStrongAsFine(t *testing.T) {
	rollTestSpecs(t)
	last := -100
	for _, q := range Qualities() {
		itm := New(testSwordID)
		itm.ApplyRoll(Rolled{Version: RollVersion, Quality: q, Rarity: RarityCommon, Identified: true})
		bonus := itm.GetSpec().Damage.BonusDamage
		assert.GreaterOrEqual(t, bonus, last, "%s must not be worse than the quality below it", q)
		last = bonus
	}
}

func TestUnidentifiedKeepsAffixesHiddenAndOutOfTheNumbers(t *testing.T) {
	rollTestSpecs(t)
	itm := New(testSwordID)
	itm.ApplyRoll(testRoll(QualityFine, RarityRare, false))

	assert.False(t, itm.IsIdentified())
	spec := itm.GetSpec()
	assert.Equal(t, 1, spec.StatMods.Get("speed"), "base stats stay")
	assert.Zero(t, spec.StatMods.Get("strength"), "an unread affix gives nothing yet")
	assert.Equal(t, 160, spec.Value, "unidentified items sell for their base value only")
	name := itm.DisplayName()
	assert.Contains(t, name, "fine short sword")
	assert.Contains(t, name, "unidentified")
	assert.Contains(t, name, `fg="rarity-rare"`)
	assert.NotContains(t, name, "Gloomfang")
	assert.NotContains(t, itm.GetLongDescription(), "+3 strength")
	assert.Contains(t, itm.GetLongDescription(), "not yet known")
}

func TestIdentifyAppliesAffixesNamesAndValueOnce(t *testing.T) {
	rollTestSpecs(t)
	itm := New(testSwordID)
	itm.ApplyRoll(testRoll(QualityStandard, RarityRare, false))

	require.True(t, itm.Identify())
	assert.False(t, itm.Identify(), "identifying twice changes nothing")
	spec := itm.GetSpec()
	assert.Equal(t, 3, spec.StatMods.Get("strength"))
	assert.Equal(t, 1, spec.StatMods.Get("speed"))
	assert.Equal(t, 2, spec.DamageReduction, "protection adds to the base (a sword has none)")
	assert.Equal(t, 900, spec.Weight, "10% lighter")
	assert.Greater(t, spec.Value, 100, "affixes add value")
	assert.Contains(t, itm.DisplayName(), "Gloomfang, a short sword")
	assert.NotContains(t, itm.DisplayName(), "unidentified")

	base := GetItemSpec(testSwordID)
	assert.Zero(t, base.StatMods.Get("strength"), "the shared base spec is never touched")
	assert.Equal(t, 1000, base.Weight)
}

func TestUncommonNameCarriesItsAffixLabel(t *testing.T) {
	rollTestSpecs(t)
	itm := New(testSwordID)
	r := testRoll(QualityFine, RarityUncommon, true)
	r.Affixes = r.Affixes[:1]
	itm.ApplyRoll(r)
	assert.Equal(t, "Mighty fine short sword (uncommon)", stripAnsi(itm.DisplayName()))

	r.Affixes = []RolledAffix{{ID: "x", Label: "of the Bear", Suffix: true, Mechanic: "statmod:strength", Value: 1}}
	itm.ApplyRoll(r)
	assert.Equal(t, "fine short sword of the Bear (uncommon)", stripAnsi(itm.DisplayName()))
}

func stripAnsi(s string) string {
	for {
		i := strings.Index(s, "<ansi")
		if i < 0 {
			return strings.TrimSpace(s)
		}
		j := strings.Index(s[i:], ">")
		s = s[:i] + s[i+j+1:]
		s = strings.ReplaceAll(s, "</ansi>", "")
	}
}

func TestLevelRequirementRefusesLowLevelWearers(t *testing.T) {
	rollTestSpecs(t)
	itm := New(testSwordID)
	itm.ApplyRoll(testRoll(QualityStandard, RarityRare, false))
	assert.Equal(t, 15, itm.LevelRequirement())
	assert.Contains(t, itm.WearRefusal(14), "requires level 15")
	assert.Empty(t, itm.WearRefusal(15))

	plain := New(testSwordID)
	assert.Zero(t, plain.LevelRequirement(), "legacy items have none")
	assert.Empty(t, plain.WearRefusal(1))
	assert.True(t, plain.IsIdentified())
	assert.Equal(t, RarityCommon, plain.RollRarity())
}

func TestRolledNameMatchesTheGeneratedNameOnlyOnceIdentified(t *testing.T) {
	rollTestSpecs(t)
	itm := New(testSwordID)
	itm.ApplyRoll(testRoll(QualityStandard, RarityRare, false))
	part, _ := itm.NameMatch("gloom", true)
	assert.False(t, part, "an unread item doesn't answer to a name it hasn't earned")
	part, _ = itm.NameMatch("short", true)
	assert.True(t, part)

	itm.Identify()
	part, full := itm.NameMatch("gloomfang", true)
	assert.True(t, part && full)
	part, _ = itm.NameMatch("short", true)
	assert.True(t, part, "its base type still works")
}

func TestUnEnchantKeepsARolledItemsNumbers(t *testing.T) {
	rollTestSpecs(t)
	itm := New(testSwordID)
	itm.ApplyRoll(testRoll(QualityFine, RarityRare, true))
	strength := itm.GetSpec().StatMods.Get("strength")
	require.Equal(t, 3, strength)

	itm.Enchant(2, 0, map[string]int{"speed": 1}, false)
	assert.Equal(t, 3, itm.GetSpec().Damage.BonusDamage, "the enchantment (+2) stacks on the fine roll (+1)")
	itm.UnEnchant()
	assert.Equal(t, 3, itm.GetSpec().StatMods.Get("strength"), "the roll survives unenchanting")
	assert.Equal(t, 1, itm.GetSpec().Damage.BonusDamage)
}

func TestRolledItemSurvivesSaveAndLoadWithIdenticalNumbers(t *testing.T) {
	rollTestSpecs(t)
	itm := New(testSwordID)
	itm.ApplyRoll(testRoll(QualityFine, RarityEpic, false))

	raw, err := yaml.Marshal(itm)
	require.NoError(t, err)
	var loaded Item
	require.NoError(t, yaml.Unmarshal(raw, &loaded))
	assert.Equal(t, itm.Loot, loaded.Loot)
	assert.Equal(t, itm.GetSpec().Value, loaded.GetSpec().Value)
	assert.Equal(t, itm.GetSpec().Damage, loaded.GetSpec().Damage)
	assert.Equal(t, itm.DisplayName(), loaded.DisplayName())
	assert.False(t, loaded.IsIdentified())

	loaded.Identify()
	assert.Equal(t, 3, loaded.GetSpec().StatMods.Get("strength"))
}

func TestLegacyItemSavesWithoutRollData(t *testing.T) {
	rollTestSpecs(t)
	raw, err := yaml.Marshal(New(testSwordID))
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "loot")
	var loaded Item
	require.NoError(t, yaml.Unmarshal([]byte("itemid: 986001\n"), &loaded))
	assert.False(t, loaded.IsRolled())
	assert.Equal(t, "short sword", loaded.DisplayName())
}

func TestRolledDescriptionShowsLayersAndRankFourDetail(t *testing.T) {
	rollTestSpecs(t)
	itm := New(testSwordID)
	r := testRoll(QualityFine, RarityRare, true)
	r.Source = "the Gloamwood Hollow"
	itm.ApplyRoll(r)
	plain := itm.GetLongDescriptionFor(0)
	assert.Contains(t, plain, "Rare")
	assert.Contains(t, plain, "Item level: 20")
	assert.Contains(t, plain, "Requires level 15")
	assert.Contains(t, plain, "+3 strength")
	assert.NotContains(t, plain, "tier 2, 2 to 3")
	assert.NotContains(t, plain, "Gloamwood")
	rank4 := itm.GetLongDescriptionFor(4)
	assert.Contains(t, rank4, "(tier 2, 2 to 3)")
	assert.Contains(t, rank4, "Found from: the Gloamwood Hollow.")
}

func TestCopyingARolledItemDoesNotAliasItsAffixes(t *testing.T) {
	rollTestSpecs(t)
	a := New(testSwordID)
	a.ApplyRoll(testRoll(QualityStandard, RarityRare, false))
	b := a
	b.Identify()
	assert.False(t, a.IsIdentified(), "identifying a copy leaves the original unread")
	assert.False(t, a.Loot.Identified)
	assert.NotEqual(t, a.GetSpec().StatMods.Get("strength"), b.GetSpec().StatMods.Get("strength"))
}

// Identifying an item that was enchanted, renamed or given a worn buff since
// it rolled adds its affixes to the item as it is, instead of rebuilding it
// from the base type.
func TestIdentifyKeepsLaterEnchantmentsRenamesAndBuffs(t *testing.T) {
	rollTestSpecs(t)
	itm := New(testSwordID)
	itm.ApplyRoll(testRoll(QualityFine, RarityRare, false))
	itm.Enchant(2, 0, map[string]int{"speed": 2}, false)
	itm.Rename("sunblade")
	itm.AddWornBuff(77)
	before := itm.GetSpec()

	require.True(t, itm.Identify())
	after := itm.GetSpec()
	assert.Equal(t, uint8(1), itm.Enchantments)
	assert.Equal(t, before.Damage.BonusDamage, after.Damage.BonusDamage, "the enchantment's damage stays")
	assert.Equal(t, 3, after.StatMods.Get("speed"), "base speed 1 plus the enchantment's 2")
	assert.Equal(t, 3, after.StatMods.Get("strength"), "and the affix now applies")
	assert.Equal(t, "sunblade", after.Name)
	assert.Equal(t, []int{77}, after.WornBuffIds)
	assert.Equal(t, 900, after.Weight)
}

// A multi-attack weapon scales per hit: Exquisite adds the same bonus to each
// hit that it would to a single-attack weapon of the same dice.
func TestQualityScalesPerHitOnMultiAttackWeapons(t *testing.T) {
	rollTestSpecs(t)
	single := RolledSpec(*GetItemSpec(testSwordID), testRoll(QualityExquisite, RarityCommon, true))
	multi := *GetItemSpec(testSwordID)
	multi.Damage.Attacks = 3
	got := RolledSpec(multi, testRoll(QualityExquisite, RarityCommon, true))
	assert.Equal(t, single.Damage.BonusDamage, got.Damage.BonusDamage)
	assert.Positive(t, got.Damage.BonusDamage)
}

// Review fix (36a): Weight() reads base data, so an identified weight affix
// must be applied there too, or it never reaches encumbrance.
func TestWeightAffixLightensTheCarriedWeightOnceRead(t *testing.T) {
	rollTestSpecs(t)
	itm := New(testSwordID)
	itm.ApplyRoll(testRoll(QualityStandard, RarityRare, false))
	assert.Equal(t, 1000, itm.Weight(), "an unread affix weighs nothing off")
	require.True(t, itm.Identify())
	assert.Equal(t, 900, itm.Weight(), "10% lighter for load")
	plain := New(testSwordID)
	assert.Equal(t, 1000, plain.Weight(), "plain items keep base weight")
}

// Review fix (36a): warmth is a bonus on top of the item's resolved warmth
// (0 is the slot default, negative is none), never a change to Warmth.
func TestWarmthAffixAddsABonusNotARawWarmth(t *testing.T) {
	rollTestSpecs(t)
	itm := New(testMailID)
	itm.ApplyRoll(Rolled{Version: RollVersion, Quality: QualityStandard, Rarity: RarityUncommon, Identified: true,
		Affixes: []RolledAffix{{ID: "furlined", Label: "Fur-lined", Mechanic: "warmth", Value: 2}}})
	spec := itm.GetSpec()
	assert.Zero(t, spec.Warmth, "still the slot default")
	assert.Equal(t, 2, spec.WarmthBonus)
}

func TestRareNameUsesAnBeforeAVowel(t *testing.T) {
	rollTestSpecs(t)
	itm := New(testSwordID)
	itm.ApplyRoll(testRoll(QualityExquisite, RarityRare, true))
	assert.Contains(t, stripAnsi(itm.DisplayName()), "Gloomfang, an exquisite short sword")
}

// Review fix (36a): players can name rolled gear by the words they see.
func TestRolledItemAnswersToItsShownWords(t *testing.T) {
	rollTestSpecs(t)
	itm := New(testSwordID)
	itm.ApplyRoll(Rolled{Version: RollVersion, Quality: QualityFine, Rarity: RarityUncommon, Identified: true,
		Affixes: []RolledAffix{{ID: "strength", Label: "Mighty", Mechanic: "statmod:strength", Value: 2}}})
	for _, word := range []string{"mighty", "fine", "mighty fine short sword", "sword"} {
		part, _ := itm.NameMatch(word, true)
		assert.True(t, part, word)
	}
	_, full := itm.NameMatch("mighty fine short sword", true)
	assert.True(t, full)

	hidden := New(testSwordID)
	hidden.ApplyRoll(testRoll(QualityFine, RarityRare, false))
	part, _ := hidden.NameMatch("mighty", true)
	assert.False(t, part, "an unread affix word stays hidden")
	part, _ = hidden.NameMatch("gloom", true)
	assert.False(t, part)
	part, _ = hidden.NameMatch("fine", true)
	assert.True(t, part, "quality is visible")
}

// Review fix (36a): enchanting keeps the quality-scaled value.
func TestEnchantKeepsARolledItemsQualityValue(t *testing.T) {
	rollTestSpecs(t)
	itm := New(testSwordID)
	itm.ApplyRoll(Rolled{Version: RollVersion, Quality: QualityExquisite, Rarity: RarityCommon, Identified: true, BaseValue: 100})
	before := itm.GetSpec().Value
	require.Equal(t, 400, before)
	itm.Enchant(2, 0, map[string]int{"strength": 1}, false)
	assert.Greater(t, itm.GetSpec().Value, before, "the enchantment adds worth on top")
	assert.Zero(t, GetItemSpec(testSwordID).StatMods.Get("strength"), "base spec untouched")
}
