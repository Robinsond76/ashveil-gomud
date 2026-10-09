package loot

import (
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"os"
)

// shippedCatalog reads the shipped catalog's equipment (weapons 10100s,
// armor 20100s, shields 20300s) straight from the item files.
func shippedCatalog(t *testing.T) []items.ItemSpec {
	t.Helper()
	root := "../../_datafiles/world/default/items"
	var out []items.ItemSpec
	require.NoError(t, filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".yaml" {
			return err
		}
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		spec := items.ItemSpec{}
		require.NoError(t, yaml.Unmarshal(data, &spec))
		require.NoError(t, spec.Validate(), path)
		if (spec.ItemId >= 10100 && spec.ItemId < 10200) || (spec.ItemId >= 20100 && spec.ItemId < 20400) || (spec.ItemId >= 21000 && spec.ItemId < 21200) {
			out = append(out, spec)
		}
		return nil
	}))
	require.NotEmpty(t, out)
	return out
}

// Phase 36b: every catalog weapon, armor piece and shield rolls at every
// rarity and quality through the real generator, keeps its catalog tier on
// the roll, and names itself from its material and quality.
func TestCatalogRollsAtEveryRarity(t *testing.T) {
	set := shippedSet(t)
	catalog := shippedCatalog(t)
	require.Greater(t, len(catalog), 90)
	rng := seeded{rand.New(rand.NewSource(36))}
	for _, spec := range catalog {
		require.True(t, Rollable(spec), "%s is equipment the generator rolls", spec.Name)
		for _, rarity := range []items.Rarity{items.RarityCommon, items.RarityUncommon, items.RarityRare, items.RarityEpic} {
			r, err := set.Generate(spec, Options{ILvl: 20, Rarity: rarity, Quality: items.QualityFine}, rng)
			require.NoError(t, err, "%s %s", rarity, spec.Name)
			assert.Equal(t, spec.Tier, r.Tier, "%s keeps its catalog tier", spec.Name)
			assert.Equal(t, rarity, r.Rarity)
		}
	}
}

// A fine tier 2 glaive is named for its quality and material, and a fine
// piece out-protects a standard one by the quality rule.
func TestCatalogNamesAndQuality(t *testing.T) {
	var glaive, jerkin *items.ItemSpec
	for _, spec := range shippedCatalog(t) {
		spec := spec
		switch spec.Name {
		case "steel glaive":
			glaive = &spec
		case "cured leather jerkin":
			jerkin = &spec
		}
	}
	require.NotNil(t, glaive)
	require.NotNil(t, jerkin)
	items.SetTestItemSpec(glaive)
	items.SetTestItemSpec(jerkin)
	t.Cleanup(func() { items.RemoveTestItemSpec(glaive.ItemId); items.RemoveTestItemSpec(jerkin.ItemId) })
	set := shippedSet(t)
	rng := seeded{rand.New(rand.NewSource(7))}

	r, err := set.Generate(*glaive, Options{ILvl: 12, Rarity: items.RarityCommon, Quality: items.QualityFine}, rng)
	require.NoError(t, err)
	itm := items.New(glaive.ItemId)
	itm.ApplyRoll(r)
	assert.Equal(t, "fine steel glaive", util.StripANSI(itm.DisplayName()))
	assert.Equal(t, 2, itm.Loot.Tier)
	assert.True(t, itm.GetSpec().Reach, "a rolled glaive keeps its reach")
	assert.Equal(t, 2, itm.GetSpec().Hands)

	std, err := set.Generate(*jerkin, Options{ILvl: 12, Rarity: items.RarityCommon, Quality: items.QualityStandard}, rng)
	require.NoError(t, err)
	fine, err := set.Generate(*jerkin, Options{ILvl: 12, Rarity: items.RarityCommon, Quality: items.QualityFine}, rng)
	require.NoError(t, err)
	a, b := items.New(jerkin.ItemId), items.New(jerkin.ItemId)
	a.ApplyRoll(std)
	b.ApplyRoll(fine)
	assert.Greater(t, b.GetSpec().DamageReduction, a.GetSpec().DamageReduction, "fine leather protects more than standard")
	assert.Equal(t, jerkin.DamageReduction, a.GetSpec().DamageReduction)
}
