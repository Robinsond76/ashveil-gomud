package encumbrance

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Phase 28: every shipped item weighs something (a service excepted),
// within a sane range for its type, so the company load means something.

type shippedItem struct {
	ItemId int    `yaml:"itemid"`
	Name   string `yaml:"name"`
	Type   string `yaml:"type"`
	Weight int    `yaml:"weight"`
}

func shippedItems(t *testing.T) map[int]shippedItem {
	t.Helper()
	return worldItems(t, "default")
}

func worldItems(t *testing.T, world string) map[int]shippedItem {
	t.Helper()
	out := map[int]shippedItem{}
	root := filepath.Join("..", "..", "_datafiles", "world", world, "items")
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".yaml" {
			return err
		}
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		var it shippedItem
		require.NoError(t, yaml.Unmarshal(data, &it), path)
		out[it.ItemId] = it
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, out)
	return out
}

// weightRange is the sane range, in grams, for an item type.
func weightRange(itemType string) (lo, hi int) {
	switch itemType {
	case "ring":
		return 5, 100
	case "neck", "head", "gloves", "belt":
		return 20, 3000
	case "feet", "legs":
		return 200, 5000
	case "body":
		return 150, 15000
	case "offhand":
		return 300, 10000
	case "weapon":
		return 50, 25000
	case "potion", "drink", "food":
		return 20, 2000
	case "botanical":
		return 5, 200
	case "readable", "key", "lockpicks":
		return 5, 2000
	}
	return 5, 10000
}

func TestShippedItemsWeighSomething(t *testing.T) {
	for _, world := range []string{"default", "empty"} {
		t.Run(world, func(t *testing.T) { weighSomething(t, worldItems(t, world)) })
	}
}

func weighSomething(t *testing.T, items map[int]shippedItem) {
	for id, it := range items {
		if it.Type == "service" {
			assert.Zero(t, it.Weight, "%d %s: a service isn't carried", id, it.Name)
			continue
		}
		lo, hi := weightRange(it.Type)
		assert.GreaterOrEqual(t, it.Weight, lo, "%d %s (%s)", id, it.Name, it.Type)
		assert.LessOrEqual(t, it.Weight, hi, "%d %s (%s)", id, it.Name, it.Type)
	}
}

func TestAuthoredWeightsKept(t *testing.T) {
	items := shippedItems(t)
	for id, grams := range map[int]int{30018: 20, 28: 700, 29: 300, 30: 300} {
		assert.Equal(t, grams, items[id].Weight, "item %d", id)
	}
}

// Each archetype's starter kit is a light pack: 1–12 kg, far below the
// first load band (75% of the company's capacity) on its own.
func TestStarterKitsAreLight(t *testing.T) {
	items := shippedItems(t)
	data, err := os.ReadFile(filepath.Join("..", "archetype", "files", "data-overlays", "config.yaml"))
	require.NoError(t, err)
	var cfg struct {
		Archetypes []struct {
			ArchetypeId string `yaml:"ArchetypeId"`
			Kit         []int  `yaml:"Kit"`
		} `yaml:"Archetypes"`
	}
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	require.NotEmpty(t, cfg.Archetypes)
	capacity, _ := parseConfig(200, nil)
	for _, a := range cfg.Archetypes {
		total := 0
		for _, id := range a.Kit {
			it, ok := items[id]
			require.True(t, ok, "%s kit item %d exists", a.ArchetypeId, id)
			total += it.Weight
		}
		assert.GreaterOrEqual(t, total, 1000, a.ArchetypeId)
		assert.LessOrEqual(t, total, 12000, a.ArchetypeId)
		assert.Less(t, float64(total)/float64(capacity), 0.1, "%s: well below the first band", a.ArchetypeId)
	}
}

// A fresh company of five (the leader's heaviest starter kit and four
// recruiters' candidates in their template gear) is still Light: well
// below the first band, 75% of capacity.
func TestFreshCompanyIsLight(t *testing.T) {
	items := shippedItems(t)
	heaviestKit := 0
	data, err := os.ReadFile(filepath.Join("..", "archetype", "files", "data-overlays", "config.yaml"))
	require.NoError(t, err)
	var arch struct {
		Archetypes []struct {
			Kit []int `yaml:"Kit"`
		} `yaml:"Archetypes"`
	}
	require.NoError(t, yaml.Unmarshal(data, &arch))
	for _, a := range arch.Archetypes {
		total := 0
		for _, id := range a.Kit {
			total += items[id].Weight
		}
		heaviestKit = max(heaviestKit, total)
	}
	companions := 0
	for _, template := range []int{61, 62, 63, 64} {
		found, err := filepath.Glob(filepath.Join("..", "..", "_datafiles", "world", "default", "mobs", "*", strconv.Itoa(template)+"-*.yaml"))
		require.NoError(t, err)
		require.Len(t, found, 1)
		raw, err := os.ReadFile(found[0])
		require.NoError(t, err)
		var mob struct {
			Character struct {
				Equipment map[string]struct {
					ItemId int `yaml:"itemid"`
				} `yaml:"equipment"`
				Items []struct {
					ItemId int `yaml:"itemid"`
				} `yaml:"items"`
			} `yaml:"character"`
		}
		require.NoError(t, yaml.Unmarshal(raw, &mob))
		for _, e := range mob.Character.Equipment {
			companions += items[e.ItemId].Weight
		}
		for _, it := range mob.Character.Items {
			companions += items[it.ItemId].Weight
		}
	}
	capacity, _ := parseConfig(200, nil)
	ratio := float64(heaviestKit+companions) / float64(capacity)
	assert.Less(t, ratio, 0.3, "a fresh company of five: %d g", heaviestKit+companions)
}
