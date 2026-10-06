package items

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// shippedSpecs reads every shipped item file directly (not through
// LoadDataFiles, which would replace the package's test specs) and
// validates it as the loader does.
func shippedSpecs(t *testing.T) map[int]*ItemSpec {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default", "items")
	specs := map[int]*ItemSpec{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return err
		}
		data, err := os.ReadFile(path)
		require.NoError(t, err, path)
		spec := &ItemSpec{}
		require.NoError(t, yaml.Unmarshal(data, spec), path)
		require.NoError(t, spec.Validate(), path)
		assert.Equal(t, spec.Filename(), filepath.Base(path), "file name follows the item id and name")
		assert.NotContains(t, specs, spec.ItemId, "%s duplicates item id %d", path, spec.ItemId)
		specs[spec.ItemId] = spec
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, specs)
	return specs
}

func avgDamage(d Damage) float64 {
	return float64(d.DiceCount)*(float64(d.SideCount)+1)/2 + float64(d.BonusDamage)
}

// Phase 36b audit: every shipped weapon and armor piece has a tier, and
// every item still loads and validates.
func TestShippedEquipmentIsAllTiered(t *testing.T) {
	for id, spec := range shippedSpecs(t) {
		equipment := spec.Type == Weapon || spec.IsArmor()
		if !equipment {
			assert.Zero(t, spec.Tier, "item %d (%s) is not equipment", id, spec.Name)
			continue
		}
		assert.GreaterOrEqual(t, spec.Tier, 1, "item %d (%s) has a tier", id, spec.Name)
		assert.LessOrEqual(t, spec.Tier, MaxTier, "item %d (%s)", id, spec.Name)
	}
}

// The catalog's weapons: every family at every tier, with the right hands,
// reach, subtype and class, rising damage by tier, and unique names.
func TestShippedWeaponCatalog(t *testing.T) {
	specs := shippedSpecs(t)
	type fam struct {
		sub, class string
		hands      int
		reach      bool
	}
	families := map[string]fam{
		"sword":     {"slashing", "sword", 1, false},
		"axe":       {"cleaving", "axe", 1, false},
		"mace":      {"bludgeoning", "mace", 1, false},
		"spear":     {"stabbing", "spear", 1, false},
		"war spear": {"stabbing", "spear", 2, true},
		"glaive":    {"slashing", "glaive", 2, true},
		"staff":     {"bludgeoning", "staff", 2, false},
		"bow":       {"shooting", "bow", 2, false},
		"crossbow":  {"shooting", "crossbow", 2, false},
	}
	byFamily := map[string]map[int]*ItemSpec{}
	names := map[string]int{}
	for id, spec := range specs {
		if id < 10100 || id >= 10200 {
			continue
		}
		require.Equal(t, Weapon, spec.Type, "item %d", id)
		f, ok := families[spec.Family]
		require.True(t, ok, "item %d (%s): unknown family %q", id, spec.Name, spec.Family)
		assert.Equal(t, ItemSubType(f.sub), spec.Subtype, spec.Name)
		assert.Equal(t, f.class, spec.WeaponClass, spec.Name)
		assert.Equal(t, f.hands, spec.Hands, "%s hands", spec.Name)
		assert.Equal(t, f.reach, spec.Reach, "%s reach", spec.Name)
		assert.Greater(t, spec.Value, 0, spec.Name)
		assert.Greater(t, spec.Weight, 0, spec.Name)
		assert.Contains(t, []int{1, 2, 3}, spec.Tier, "launch catalog is tiers 1 to 3: %s", spec.Name)
		assert.Equal(t, spec.Tier, id%10, "%s: the id's last digit is its tier", spec.Name)
		assert.NotContains(t, names, spec.Name, "unique name")
		names[spec.Name] = id
		if byFamily[spec.Family] == nil {
			byFamily[spec.Family] = map[int]*ItemSpec{}
		}
		byFamily[spec.Family][spec.Tier] = spec
	}
	for family := range families {
		tiers := byFamily[family]
		want := 3
		if family == "staff" {
			want = 2 // tier 1 is the shipped ash quarterstaff (10021)
		}
		require.Len(t, tiers, want, "family %s has its tiers", family)
		prev := 0.0
		for tier := 1; tier <= 3; tier++ {
			spec := tiers[tier]
			if spec == nil {
				continue
			}
			avg := avgDamage(spec.Damage)
			assert.Greater(t, avg, prev, "%s out-damages the tier below", spec.Name)
			prev = avg
		}
	}
	staff := specs[10021]
	require.NotNil(t, staff)
	assert.Equal(t, "staff", staff.WeaponClass)
	assert.Less(t, avgDamage(staff.Damage), avgDamage(byFamily["staff"][2].Damage))

	// Two-handed weapons out-damage one-handed ones of the tier, as the
	// price of the shield; reach and shooting are separate from that.
	for tier := 1; tier <= 3; tier++ {
		assert.Greater(t, avgDamage(byFamily["glaive"][tier].Damage), avgDamage(byFamily["sword"][tier].Damage), "tier %d", tier)
		assert.Greater(t, avgDamage(byFamily["war spear"][tier].Damage), avgDamage(byFamily["spear"][tier].Damage), "tier %d", tier)
	}
	// A crossbow is heavier than a bow of its tier.
	for tier := 1; tier <= 3; tier++ {
		assert.Greater(t, byFamily["crossbow"][tier].Weight, byFamily["bow"][tier].Weight, "tier %d", tier)
	}
	// The glaive ladder the equipment design names, with its reach.
	assert.Equal(t, "militia glaive", byFamily["glaive"][1].Name)
	assert.Equal(t, "steel glaive", byFamily["glaive"][2].Name)
	assert.Equal(t, "tempered war glaive", byFamily["glaive"][3].Name)
}

// The catalog's armor: four paths, five slots, three tiers, with protection
// that rises by tier and path and a bulk set by the path.
func TestShippedArmorCatalog(t *testing.T) {
	specs := shippedSpecs(t)
	bulk := map[string]string{"cloth": BulkLight, "leather": BulkLight, "medium": BulkMedium, "heavy": BulkHeavy}
	slots := []ItemType{Head, Body, Gloves, Legs, Feet}
	type key struct {
		path string
		slot ItemType
		tier int
	}
	got := map[key]*ItemSpec{}
	for id, spec := range specs {
		if id < 20100 || id >= 20300 {
			continue
		}
		assert.Equal(t, bulk[spec.Family], spec.Bulk, "%s bulk", spec.Name)
		assert.Contains(t, slots, spec.Type, spec.Name)
		assert.Greater(t, spec.DamageReduction, 0, spec.Name)
		assert.Empty(t, spec.StatMods, "%s: catalog armor has no stat mods (affixes add them)", spec.Name)
		got[key{spec.Family, spec.Type, spec.Tier}] = spec
	}
	require.Len(t, got, 4*5*3, "every path has every slot at every tier")

	set := func(path string, tier int) (dr, weight int) {
		for _, s := range slots {
			spec := got[key{path, s, tier}]
			dr += spec.DamageReduction
			weight += spec.Weight
		}
		return dr, weight
	}
	paths := []string{"cloth", "leather", "medium", "heavy"}
	for tier := 1; tier <= 3; tier++ {
		prevDR, prevWeight := 0, 0
		for _, path := range paths {
			dr, weight := set(path, tier)
			assert.Greater(t, dr, prevDR, "tier %d: %s protects more than the path before it", tier, path)
			assert.Greater(t, weight, prevWeight, "tier %d: %s weighs more than the path before it", tier, path)
			prevDR, prevWeight = dr, weight
		}
	}
	for _, path := range paths {
		for _, slot := range slots {
			prev := 0
			for tier := 1; tier <= 3; tier++ {
				dr := got[key{path, slot, tier}].DamageReduction
				assert.GreaterOrEqual(t, dr, prev, "%s %s tier %d", path, slot, tier)
				prev = dr
			}
		}
		// Higher tiers improve protection at about the same weight.
		_, w1 := set(path, 1)
		_, w3 := set(path, 3)
		assert.Equal(t, w1, w3, "%s weighs the same at every tier", path)
	}
	// Cloth does not gain plate's protection by tier: tier 3 cloth stays
	// below tier 1 medium.
	clothT3, _ := set("cloth", 3)
	mediumT1, _ := set("medium", 1)
	assert.Less(t, clothT3, mediumT1)
}

// The catalog's shields: sizes, bulk, and a ladder within each shield.
func TestShippedShieldCatalog(t *testing.T) {
	specs := shippedSpecs(t)
	byFamily := map[string]map[int]*ItemSpec{}
	for id, spec := range specs {
		if id < 20300 || id >= 20400 {
			continue
		}
		assert.True(t, spec.IsShield(), spec.Name)
		if byFamily[spec.Family] == nil {
			byFamily[spec.Family] = map[int]*ItemSpec{}
		}
		byFamily[spec.Family][spec.Tier] = spec
	}
	// The catalog fills in around the shipped wooden shield (round, 1), leather
	// buckler (1) and iron and tower shields.
	for _, id := range []int{20004, 20047} {
		spec := specs[id]
		byFamily[spec.Family][spec.Tier] = spec
	}
	for _, family := range []string{"buckler", "round shield", "kite shield"} {
		for _, tier := range []int{1, 2, 3} {
			require.NotNil(t, byFamily[family][tier], "%s at tier %d", family, tier)
		}
		for tier := 2; tier <= 3; tier++ {
			assert.Greater(t, byFamily[family][tier].DamageReduction, byFamily[family][tier-1].DamageReduction, "%s tier %d", family, tier)
		}
	}
	assert.Equal(t, ShieldBuckler, byFamily["buckler"][3].ShieldSize)
	assert.Equal(t, BulkLight, byFamily["buckler"][3].Bulk)
	assert.Greater(t, byFamily["kite shield"][1].Weight, byFamily["round shield"][1].Weight, "a kite shield is heavier than a round one")
	assert.Greater(t, byFamily["kite shield"][1].DamageReduction, byFamily["round shield"][1].DamageReduction)
	assert.Equal(t, 2, specs[20048].Tier, "the shipped tower shield is audited as tier 2")
}

// Goods: a category, a weight and a value density inside the category's
// band, never equipment.
func TestShippedGoods(t *testing.T) {
	specs := shippedSpecs(t)
	// Value per kg bands (gold per kilogram), inclusive.
	bands := map[string][2]float64{
		GoodsTrophy:    {4, 20},
		GoodsSalvage:   {1, 4},
		GoodsMaterial:  {5, 250},
		GoodsValuable:  {100, 2500},
		GoodsProvision: {5, 30},
		GoodsCurio:     {200, 1500},
	}
	perCategory := map[string]int{}
	var ids []int
	for id, spec := range specs {
		if spec.Goods != "" {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	for _, id := range ids {
		spec := specs[id]
		band, ok := bands[spec.Goods]
		require.True(t, ok, "item %d (%s): category %q", id, spec.Name, spec.Goods)
		perCategory[spec.Goods]++
		assert.True(t, spec.IsGoods())
		assert.NotEqual(t, Weapon, spec.Type)
		assert.Greater(t, spec.Weight, 0, spec.Name)
		assert.Greater(t, spec.Value, 0, spec.Name)
		perKg := spec.GoodsValuePerKg()
		assert.GreaterOrEqual(t, perKg, band[0], "%s (%s) is worth %.1f gold a kg", spec.Name, spec.Goods, perKg)
		assert.LessOrEqual(t, perKg, band[1], "%s (%s) is worth %.1f gold a kg", spec.Name, spec.Goods, perKg)
		assert.Empty(t, spec.QuestToken, spec.Name)
	}
	for _, c := range GoodsCategories() {
		assert.Positive(t, perCategory[c], "category %s has goods", c)
	}
	// Heavy goods pay less a kilogram than light ones.
	assert.Less(t, specs[210].GoodsValuePerKg(), specs[230].GoodsValuePerKg(), "rusted mail against a silver trinket")
	// The existing hide and meat are tagged.
	assert.Equal(t, GoodsTrophy, specs[28].Goods)
	assert.Equal(t, GoodsProvision, specs[29].Goods)
}

// The shops that stock the catalog name only items that exist, and Ivar's
// stock is tier 1 gear while the general trader stocks goods.
func TestShippedShopsStockCatalogItems(t *testing.T) {
	specs := shippedSpecs(t)
	_, thisFile, _, _ := runtime.Caller(0)
	mobDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default", "mobs", "frostfang")
	stock := func(file string) []int {
		data, err := os.ReadFile(filepath.Join(mobDir, file))
		require.NoError(t, err)
		var mob struct {
			Character struct {
				Shop []struct {
					ItemId int `yaml:"itemid"`
				} `yaml:"shop"`
			} `yaml:"character"`
		}
		require.NoError(t, yaml.Unmarshal(data, &mob))
		var ids []int
		for _, s := range mob.Character.Shop {
			require.Contains(t, specs, s.ItemId, "%s stocks item %d", file, s.ItemId)
			ids = append(ids, s.ItemId)
		}
		return ids
	}
	ivar := stock("10-ivar_froststeel.yaml")
	catalog := 0
	for _, id := range ivar {
		if id >= 10100 && id < 20400 && (id < 10200 || id >= 20100) {
			catalog++
			assert.Equal(t, 1, specs[id].Tier, "the armory stocks tier 1: %s", specs[id].Name)
		}
	}
	assert.GreaterOrEqual(t, catalog, 8, "Ivar stocks catalog gear")
	assert.LessOrEqual(t, len(ivar), 20, "a merchant's stock stays within the variety limit")
	goods := 0
	for _, id := range stock("11-brynja_snowdeal.yaml") {
		if specs[id].IsGoods() {
			goods++
		}
	}
	assert.GreaterOrEqual(t, goods, 3, "the general trader stocks goods, so she buys them")
}
