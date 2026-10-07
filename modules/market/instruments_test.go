package market

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Camp music: instruments are sold in Dunmar and never bought back, so a
// bought instrument is never a profit loop.
func TestShippedInstrumentsAreSupplyOnly(t *testing.T) {
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
	want := map[int]bool{3201: true, 3202: true, 3204: true, 3205: true, 3207: true, 3208: true}
	seen := 0
	for _, m := range overlay.Markets {
		for _, g := range m.Goods {
			if want[g.ItemId] {
				assert.True(t, g.SupplyOnly, "%s item %d", m.Zone, g.ItemId)
				seen++
			}
			assert.False(t, g.ItemId >= 3210 && g.ItemId <= 3212, "masterworks are looted, never stocked: %d", g.ItemId)
		}
	}
	assert.Equal(t, len(want), seen)
}
