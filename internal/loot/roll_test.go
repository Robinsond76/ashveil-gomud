package loot

import (
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seq is a scripted random source: each Intn takes the next value mod n.
type seq struct {
	vals []int
	i    int
}

func (s *seq) Intn(n int) int {
	v := s.vals[s.i%len(s.vals)]
	s.i++
	return v % n
}

type seeded struct{ r *rand.Rand }

func (s seeded) Intn(n int) int { return s.r.Intn(n) }

const shippedAffixDir = "../../_datafiles/world/default/lootaffixes"

func shippedSet(t *testing.T) AffixSet {
	t.Helper()
	set, err := LoadAffixSet(shippedAffixDir)
	require.NoError(t, err)
	require.NotEmpty(t, set.Affixes)
	return set
}

func swordSpec() items.ItemSpec {
	return items.ItemSpec{
		ItemId: 985001, Name: "short sword", Type: items.Weapon, Subtype: items.Slashing, Hands: 1,
		Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 6}, Value: 100, Weight: 1200, Tier: 2,
	}
}

func mailSpec() items.ItemSpec {
	return items.ItemSpec{ItemId: 985002, Name: "chain shirt", Type: items.Body, Subtype: items.Wearable, DamageReduction: 8, Value: 150, Weight: 5000}
}

func TestShippedAffixDataLoadsAndIsValid(t *testing.T) {
	set := shippedSet(t)
	groups := map[string]bool{}
	majors := 0
	for _, a := range set.Affixes {
		require.NoError(t, a.Validate(), a.ID)
		groups[a.Group] = true
		if a.Major {
			majors++
		}
	}
	assert.GreaterOrEqual(t, majors, 3, "Epic needs majors to draw from")
	assert.Contains(t, groups, "stat-strength")
	assert.NotEmpty(t, set.Names.Prefixes)
	assert.NotEmpty(t, set.Names.Suffixes)
}

func TestLoadAffixSetRejectsBadData(t *testing.T) {
	cases := map[string]string{
		"unknown mechanic": "affixes:\n  - {id: x, group: g, prefix: A, mechanic: mystery, slots: [weapon], weight: 1, tiers: [{minilvl: 1, min: 1, max: 2}]}\n",
		"unknown slot":     "affixes:\n  - {id: x, group: g, prefix: A, mechanic: protection, slots: [boots], weight: 1, tiers: [{minilvl: 1, min: 1, max: 2}]}\n",
		"no label":         "affixes:\n  - {id: x, group: g, mechanic: protection, slots: [armor], weight: 1, tiers: [{minilvl: 1, min: 1, max: 2}]}\n",
		"range inverted":   "affixes:\n  - {id: x, group: g, prefix: A, mechanic: protection, slots: [armor], weight: 1, tiers: [{minilvl: 1, min: 3, max: 2}]}\n",
		"tiers fall":       "affixes:\n  - {id: x, group: g, prefix: A, mechanic: protection, slots: [armor], weight: 1, tiers: [{minilvl: 5, min: 1, max: 2}, {minilvl: 5, min: 2, max: 3}]}\n",
		"duplicate id":     "affixes:\n  - {id: x, group: g, prefix: A, mechanic: protection, slots: [armor], weight: 1, tiers: [{minilvl: 1, min: 1, max: 2}]}\n  - {id: x, group: h, prefix: A, mechanic: protection, slots: [armor], weight: 1, tiers: [{minilvl: 1, min: 1, max: 2}]}\n",
		"unknown field":    "affixes:\n  - {id: x, group: g, prefix: A, mechanic: protection, slots: [armor], weight: 1, tiers: [{minilvl: 1, min: 1, max: 2}], bogus: 1}\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "a.yaml"), []byte(body), 0600))
			_, err := LoadAffixSet(dir)
			assert.Error(t, err)
		})
	}
	set, err := LoadAffixSet(filepath.Join(t.TempDir(), "missing"))
	require.NoError(t, err, "a world with no affix directory still starts")
	assert.Empty(t, set.Affixes)
}

func TestGenerateRarityDecidesAffixCountsAndIdentification(t *testing.T) {
	set := shippedSet(t)
	cases := []struct {
		rarity         items.Rarity
		min, max       int
		identified     bool
		major          int
		wantName       bool
		levelReq50ilvl int
	}{
		{items.RarityCommon, 0, 0, true, 0, false, 0},
		{items.RarityUncommon, 1, 1, true, 0, false, 40},
		{items.RarityRare, 2, 3, false, 0, true, 45},
		{items.RarityEpic, 3, 4, false, 1, true, 45},
		{items.RarityLegendary, 2, 2, false, 0, true, 45},
		{items.RaritySet, 2, 2, false, 0, true, 45},
	}
	rng := seeded{rand.New(rand.NewSource(7))}
	for _, c := range cases {
		t.Run(string(c.rarity), func(t *testing.T) {
			for i := 0; i < 200; i++ {
				r, err := set.Generate(swordSpec(), Options{ILvl: 50, Rarity: c.rarity}, rng)
				require.NoError(t, err)
				assert.GreaterOrEqual(t, len(r.Affixes), c.min)
				assert.LessOrEqual(t, len(r.Affixes), c.max)
				assert.Equal(t, c.identified, r.Identified)
				assert.Equal(t, c.wantName, r.Name != "")
				assert.Equal(t, c.levelReq50ilvl, r.LevelReq)
				majors, groups := 0, map[string]bool{}
				for _, a := range r.Affixes {
					if a.Major {
						majors++
					}
					assert.True(t, items.ValidMechanic(a.Mechanic))
					assert.GreaterOrEqual(t, a.Value, a.MinValue)
					assert.LessOrEqual(t, a.Value, a.MaxValue)
					for _, def := range set.Affixes {
						if def.ID == a.ID {
							assert.False(t, groups[def.Group], "one affix per group: %s twice", def.Group)
							groups[def.Group] = true
						}
					}
				}
				assert.Equal(t, c.major, majors)
			}
		})
	}
}

func TestGenerateItemLevelGatesAffixTiers(t *testing.T) {
	set := shippedSet(t)
	rng := seeded{rand.New(rand.NewSource(11))}
	for ilvl, wantTier := range map[int]int{1: 1, 14: 1, 15: 2, 29: 2, 30: 3, 45: 4, 60: 5, 70: 5} {
		for i := 0; i < 50; i++ {
			r, err := set.Generate(mailSpec(), Options{ILvl: ilvl, Rarity: items.RarityRare}, rng)
			require.NoError(t, err)
			for _, a := range r.Affixes {
				assert.Equal(t, wantTier, a.Tier, "ilvl %d affix %s", ilvl, a.ID)
			}
		}
	}
	r, err := set.Generate(mailSpec(), Options{ILvl: 500, Rarity: items.RarityUncommon}, rng)
	require.NoError(t, err)
	assert.Equal(t, MaxILvl, r.ILvl, "item level is held to 70")
	r, err = set.Generate(mailSpec(), Options{ILvl: -3, Rarity: items.RarityUncommon}, rng)
	require.NoError(t, err)
	assert.Equal(t, 1, r.ILvl)
}

func TestGenerateRespectsSlotEligibilityAndBaseRequirements(t *testing.T) {
	set := shippedSet(t)
	rng := seeded{rand.New(rand.NewSource(3))}
	unarmored := mailSpec()
	unarmored.DamageReduction = 0
	for i := 0; i < 300; i++ {
		r, _ := set.Generate(swordSpec(), Options{ILvl: 70, Rarity: items.RarityEpic}, rng)
		for _, a := range r.Affixes {
			assert.NotEqual(t, "protection", a.Mechanic, "a sword does not roll protection")
		}
		r, _ = set.Generate(mailSpec(), Options{ILvl: 70, Rarity: items.RarityEpic}, rng)
		for _, a := range r.Affixes {
			assert.NotEqual(t, "statmod:damage", a.Mechanic, "armor does not roll weapon damage")
			assert.NotEqual(t, "parry", a.Mechanic)
		}
		r, _ = set.Generate(unarmored, Options{ILvl: 70, Rarity: items.RarityEpic}, rng)
		for _, a := range r.Affixes {
			assert.NotEqual(t, "protection", a.Mechanic, "protection builds on gear that already protects")
		}
	}
}

func TestGenerateDeterministicForTheSameScript(t *testing.T) {
	set := shippedSet(t)
	a, _ := set.Generate(swordSpec(), Options{ILvl: 33}, &seq{vals: []int{950, 5, 3, 1, 2, 4, 7}})
	b, _ := set.Generate(swordSpec(), Options{ILvl: 33}, &seq{vals: []int{950, 5, 3, 1, 2, 4, 7}})
	assert.Equal(t, a, b)
	assert.Equal(t, items.RarityRare, a.Rarity, "a draw of 950 falls in Rare's 920-984 band of 999")
	assert.Equal(t, items.QualityCrude, a.Quality, "a quality draw of 5 falls in Crude's first 10%")
}

func TestGenerateRarityAndQualityDistributionsFollowTheDesign(t *testing.T) {
	set := shippedSet(t)
	rng := seeded{rand.New(rand.NewSource(99))}
	rarity, quality := map[items.Rarity]int{}, map[items.Quality]int{}
	const n = 40000
	for i := 0; i < n; i++ {
		r, err := set.Generate(swordSpec(), Options{ILvl: 20}, rng)
		require.NoError(t, err)
		rarity[r.Rarity]++
		quality[r.Quality]++
	}
	near := func(got int, wantPct, tolPct float64) {
		pct := float64(got) * 100 / n
		assert.InDelta(t, wantPct, pct, tolPct)
	}
	near(rarity[items.RarityCommon], 70, 1.5)
	near(rarity[items.RarityUncommon], 22, 1.5)
	near(rarity[items.RarityRare], 6.5, 0.7)
	assert.Zero(t, rarity[items.RarityLegendary]+rarity[items.RaritySet], "authored rarities never roll by chance")
	near(quality[items.QualityStandard], 45, 1.5)
	near(quality[items.QualityCrude], 10, 1)
	near(quality[items.QualityFine], 17, 1)
	assert.Greater(t, quality[items.QualityExquisite], 0)
}

func TestGenerateRefusesNonEquipmentAndBadOptions(t *testing.T) {
	set := shippedSet(t)
	rng := seeded{rand.New(rand.NewSource(1))}
	_, err := set.Generate(items.ItemSpec{ItemId: 1, Name: "apple", Type: items.Food, Subtype: items.Edible}, Options{}, rng)
	assert.ErrorIs(t, err, ErrNotRollable)
	_, err = set.Generate(items.ItemSpec{ItemId: 2, Name: "satchel", Type: items.Pack, Subtype: items.Wearable}, Options{}, rng)
	assert.ErrorIs(t, err, ErrNotRollable, "packs never roll")
	_, err = set.Generate(swordSpec(), Options{Rarity: "mythic"}, rng)
	assert.Error(t, err)
	_, err = set.Generate(swordSpec(), Options{Quality: "shiny"}, rng)
	assert.Error(t, err)
}

func TestRollBuildsAnInstanceWhoseNumbersMatchTheRoll(t *testing.T) {
	spec := swordSpec()
	spec.StatMods = statmods.StatMods{}
	items.SetTestItemSpec(&spec)
	t.Cleanup(func() { items.RemoveTestItemSpec(spec.ItemId) })
	prev := Affixes()
	SetAffixes(shippedSet(t))
	t.Cleanup(func() { SetAffixes(prev) })

	itm, err := Roll(spec.ItemId, Options{ILvl: 40, Rarity: items.RarityEpic, Quality: items.QualityFine}, seeded{rand.New(rand.NewSource(5))})
	require.NoError(t, err)
	assert.False(t, itm.IsIdentified())
	assert.Equal(t, 35, itm.LevelRequirement())
	require.True(t, itm.Identify())

	want := map[string]int{}
	for _, a := range itm.Loot.Affixes {
		if len(a.Mechanic) > 8 && a.Mechanic[:8] == "statmod:" {
			want[a.Mechanic[8:]] += a.Value
		}
	}
	for stat, v := range want {
		assert.Equal(t, v, itm.GetSpec().StatMods.Get(stat), stat)
	}
	assert.Equal(t, 1, itm.GetSpec().Damage.BonusDamage, "a fine d6 sword gains 1 damage")
	assert.NotEqual(t, items.Item{}.UUID, itm.UUID)

	_, err = Roll(424242, Options{}, GameSource())
	assert.Error(t, err, "an unknown item")
}

func TestScribeRanksCoverRaritiesAsDesigned(t *testing.T) {
	cases := []struct {
		rank   int
		rarity items.Rarity
		covers bool
	}{
		{0, items.RarityRare, false},
		{1, items.RarityRare, true}, {1, items.RarityEpic, false},
		{2, items.RarityEpic, true}, {2, items.RarityLegendary, false},
		{3, items.RarityLegendary, true}, {3, items.RaritySet, true},
		{4, items.RarityRare, true}, {4, items.RaritySet, true},
		{4, items.RarityCommon, false}, {4, items.RarityUncommon, false},
	}
	for _, c := range cases {
		assert.Equal(t, c.covers, ScribeCovers(c.rank, c.rarity), "rank %d %s", c.rank, c.rarity)
	}
	assert.Equal(t, 8, ScribeManaCost(items.RarityRare))
	assert.Equal(t, 15, ScribeManaCost(items.RarityEpic))
	assert.Equal(t, 25, ScribeManaCost(items.RarityLegendary))
	assert.Equal(t, 25, ScribeManaCost(items.RaritySet))
	assert.Equal(t, 3, ScribeRankFor(items.RaritySet))
}

func TestLevelRequirementTable(t *testing.T) {
	assert.Equal(t, 0, LevelRequirement(items.RarityCommon, 30))
	assert.Equal(t, 20, LevelRequirement(items.RarityUncommon, 30))
	assert.Equal(t, 25, LevelRequirement(items.RarityRare, 30))
	assert.Equal(t, 25, LevelRequirement(items.RarityEpic, 30))
	assert.Equal(t, 0, LevelRequirement(items.RarityUncommon, 5), "never negative")
	assert.Equal(t, 0, LevelRequirement(items.RarityRare, 3))
}
