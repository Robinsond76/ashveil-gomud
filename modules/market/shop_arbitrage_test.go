package market

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/market"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Phase 36c review: a shopkeeper who sells a market good below what a
// market pays for it at its target stock is a risk-free loop (buy from the
// shop, walk to the market, sell, repeat at every restock). Every shipped
// shopkeeper's price for a traded good is at least the good's target-stock
// price in every market, which is above anything a market pays at or above
// target, haggle and standing included.
func TestShopkeepersNeverUndercutMarketsOnTradedGoods(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	world := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")

	var overlay struct {
		Markets []struct {
			Zone  string `yaml:"Zone"`
			Goods []struct {
				ItemId      int `yaml:"ItemId"`
				BasePrice   int `yaml:"BasePrice"`
				MinPrice    int `yaml:"MinPrice"`
				MaxPrice    int `yaml:"MaxPrice"`
				MaxStock    int `yaml:"MaxStock"`
				TargetStock int `yaml:"TargetStock"`
				StartStock  int `yaml:"StartStock"`
				DriftStep   int `yaml:"DriftStep"`
			} `yaml:"Goods"`
		} `yaml:"Markets"`
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "files", "data-overlays", "config.yaml"))
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(data, &overlay))
	floor := map[int]int{} // item id -> highest target-stock price in any market
	for _, m := range overlay.Markets {
		for _, w := range m.Goods {
			g := market.Good{ItemID: w.ItemId, BasePrice: w.BasePrice, MinPrice: w.MinPrice, MaxPrice: w.MaxPrice,
				MaxStock: w.MaxStock, TargetStock: w.TargetStock, StartStock: w.StartStock, DriftStep: w.DriftStep}
			require.NoError(t, g.Validate(), "%s %d", m.Zone, g.ItemID)
			floor[g.ItemID] = max(floor[g.ItemID], g.PriceForStock(g.TargetStock))
		}
	}
	require.NotEmpty(t, floor)

	values := map[int]int{}
	require.NoError(t, filepath.Walk(filepath.Join(world, "items"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".yaml" {
			return err
		}
		var spec struct {
			ItemId int `yaml:"itemid"`
			Value  int `yaml:"value"`
		}
		raw, err := os.ReadFile(path)
		if err == nil && yaml.Unmarshal(raw, &spec) == nil && floor[spec.ItemId] > 0 {
			values[spec.ItemId] = spec.Value
		}
		return nil
	}))

	checked := 0
	require.NoError(t, filepath.Walk(filepath.Join(world, "mobs"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".yaml" {
			return err
		}
		var mob struct {
			Character struct {
				Name string `yaml:"name"`
				Shop []struct {
					ItemId int `yaml:"itemid"`
					Price  int `yaml:"price"`
				} `yaml:"shop"`
			} `yaml:"character"`
		}
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		require.NoError(t, yaml.Unmarshal(raw, &mob), path)
		for _, si := range mob.Character.Shop {
			want, traded := floor[si.ItemId]
			if !traded {
				continue
			}
			price := si.Price
			if price == 0 {
				price = values[si.ItemId]
			}
			require.Positive(t, price, "%s: item %d has no price or value", path, si.ItemId)
			assert.GreaterOrEqual(t, price, want, "%s sells item %d for %d, under a market's %d", mob.Character.Name, si.ItemId, price, want)
			checked++
		}
		return nil
	}))
	assert.Positive(t, checked, "Brynja's goods are checked")
}

// Phase 40a2 review: firewood (free to gather) and fishing lines are sold
// in every shipped market but never bought back.
func TestShippedFirewoodAndLinesAreSupplyOnly(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	var overlay struct {
		Markets []struct {
			Zone  string `yaml:"Zone"`
			Goods []struct {
				ItemId     int  `yaml:"ItemId"`
				SupplyOnly bool `yaml:"SupplyOnly"`
			} `yaml:"Goods"`
		} `yaml:"Markets"`
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "files", "data-overlays", "config.yaml"))
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(data, &overlay))
	seen := 0
	for _, m := range overlay.Markets {
		for _, g := range m.Goods {
			if g.ItemId == 40 || g.ItemId == 42 {
				assert.True(t, g.SupplyOnly, "%s item %d", m.Zone, g.ItemId)
				seen++
			}
		}
	}
	assert.Equal(t, 4, seen, "firewood and lines in Dunmar and on the Old Kings Road")
}

// Phase 40a3: camp gear is sold in both shipped markets and never bought
// back, at the designed prices (Dunmar).
func TestShippedCampGearIsSupplyOnly(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	var overlay struct {
		Markets []struct {
			Zone  string `yaml:"Zone"`
			Goods []struct {
				ItemId     int  `yaml:"ItemId"`
				BasePrice  int  `yaml:"BasePrice"`
				SupplyOnly bool `yaml:"SupplyOnly"`
			} `yaml:"Goods"`
		} `yaml:"Markets"`
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "files", "data-overlays", "config.yaml"))
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(data, &overlay))
	price := map[int]int{45: 8, 46: 40, 47: 5, 48: 12, 49: 6, 50: 25}
	seen := map[string]int{}
	for _, m := range overlay.Markets {
		for _, g := range m.Goods {
			if want, ok := price[g.ItemId]; ok {
				assert.True(t, g.SupplyOnly, "%s item %d", m.Zone, g.ItemId)
				if m.Zone == "Dunmar" {
					assert.Equal(t, want, g.BasePrice, "item %d", g.ItemId)
				} else {
					assert.GreaterOrEqual(t, g.BasePrice, want, "the road is never cheaper: item %d", g.ItemId)
				}
				seen[m.Zone]++
			}
		}
	}
	assert.Equal(t, map[string]int{"Dunmar": 6, "Old Kings Road": 6}, seen)
}

// Phase 43b: weapon poison vials are sold in the markets (the common one on
// the road too) and never bought back, so they are never a profit loop.
func TestShippedPoisonVialsAreSupplyOnly(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	var overlay struct {
		Markets []struct {
			Zone  string `yaml:"Zone"`
			Goods []struct {
				ItemId     int  `yaml:"ItemId"`
				BasePrice  int  `yaml:"BasePrice"`
				SupplyOnly bool `yaml:"SupplyOnly"`
			} `yaml:"Goods"`
		} `yaml:"Markets"`
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "files", "data-overlays", "config.yaml"))
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(data, &overlay))
	seen := map[string][]int{}
	for _, m := range overlay.Markets {
		for _, g := range m.Goods {
			if g.ItemId >= 280 && g.ItemId <= 283 {
				assert.True(t, g.SupplyOnly, "%s item %d", m.Zone, g.ItemId)
				seen[m.Zone] = append(seen[m.Zone], g.ItemId)
				if g.ItemId == 280 {
					assert.LessOrEqual(t, g.BasePrice, 8, "bitterleaf costs about a bandage")
				}
			}
		}
	}
	assert.Equal(t, map[string][]int{"Dunmar": {280, 281, 282, 283}, "Old Kings Road": {280}}, seen)
}

// Phase 43a: camp supplies are sold in both shipped markets, never bought
// back, and the road is never cheaper than town.
func TestShippedCampSuppliesAreSupplyOnly(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	var overlay struct {
		Markets []struct {
			Zone  string `yaml:"Zone"`
			Goods []struct {
				ItemId     int  `yaml:"ItemId"`
				BasePrice  int  `yaml:"BasePrice"`
				SupplyOnly bool `yaml:"SupplyOnly"`
			} `yaml:"Goods"`
		} `yaml:"Markets"`
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "files", "data-overlays", "config.yaml"))
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(data, &overlay))
	town := map[int]int{}
	road := map[int]int{}
	for _, m := range overlay.Markets {
		for _, g := range m.Goods {
			if g.ItemId < 30040 || g.ItemId > 30043 {
				continue
			}
			assert.True(t, g.SupplyOnly, "%s item %d", m.Zone, g.ItemId)
			if m.Zone == "Dunmar" {
				town[g.ItemId] = g.BasePrice
			} else {
				road[g.ItemId] = g.BasePrice
			}
		}
	}
	require.Len(t, town, 4, "broth, draught, salve and incense in Dunmar")
	require.Len(t, road, 4, "and on the road")
	for id, price := range town {
		assert.GreaterOrEqual(t, road[id], price, "the road is never cheaper: item %d", id)
	}
}

// Phase 39d review: doll parts (a free starter kit item too) are sold in
// Dunmar and never bought back.
func TestShippedDollPartsAreSupplyOnly(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	var overlay struct {
		Markets []struct {
			Zone  string `yaml:"Zone"`
			Goods []struct {
				ItemId     int  `yaml:"ItemId"`
				SupplyOnly bool `yaml:"SupplyOnly"`
			} `yaml:"Goods"`
		} `yaml:"Markets"`
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "files", "data-overlays", "config.yaml"))
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(data, &overlay))
	seen := 0
	for _, m := range overlay.Markets {
		for _, g := range m.Goods {
			if g.ItemId == 70 {
				assert.True(t, g.SupplyOnly, "%s doll parts", m.Zone)
				seen++
			}
		}
	}
	assert.Equal(t, 1, seen, "doll parts are sold in Dunmar")
}
