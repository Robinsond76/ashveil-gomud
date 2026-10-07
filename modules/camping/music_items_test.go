package camping

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func shippedItem(t *testing.T, id int) *items.ItemSpec {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default", "items")
	var found *items.ItemSpec
	require.NoError(t, filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasPrefix(info.Name(), itoa(id)+"-") {
			return err
		}
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		found = &items.ItemSpec{}
		return yaml.Unmarshal(data, found)
	}))
	require.NotNil(t, found, "item %d is shipped", id)
	return found
}

func itoa(n int) string {
	s := ""
	for ; n > 0; n /= 10 {
		s = string(rune('0'+n%10)) + s
	}
	return s
}

// The Go instrument table and the item files must agree.
func TestInstrumentTableMatchesShippedItems(t *testing.T) {
	for _, in := range camping.Instruments {
		spec := shippedItem(t, in.ItemID)
		assert.Equal(t, in.Name, spec.Name, "item %d", in.ItemID)
		assert.Equal(t, string(in.Family), spec.Instrument, "item %d", in.ItemID)
		assert.Equal(t, in.Tier, spec.InstrumentTier, "item %d", in.ItemID)
	}
}

// Every crafting recipe makes a known instrument, and a fine one's recipe
// page teaches exactly that recipe.
func TestInstrumentRecipesMatchTheirPages(t *testing.T) {
	pages := map[int]int{3202: 3220, 3205: 3221, 3208: 3222}
	for _, r := range instrumentRecipes {
		in, ok := camping.InstrumentOf(r.Output)
		require.True(t, ok, "recipe output %d", r.Output)
		assert.Less(t, in.Tier, camping.InstrumentMasterwork, "masterworks are looted, never crafted")
		if page, fine := pages[r.Output]; fine {
			spec := shippedItem(t, page)
			assert.Equal(t, r.Output, spec.Recipe, "page %d teaches %s", page, in.Name)
		}
	}
}
