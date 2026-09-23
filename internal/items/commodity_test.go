package items

import "testing"

func TestCommodityListedAsItemType(t *testing.T) {
	for _, entry := range ItemTypes() {
		if entry.Type == string(Commodity) {
			return
		}
	}
	t.Fatal("commodity missing from item type registry")
}
