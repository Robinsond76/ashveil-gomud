package blessings

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shipped(t *testing.T) *Data {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "_datafiles", "world", "default", "blessings.yaml"))
	require.NoError(t, err)
	require.NoError(t, LoadBytes(raw))
	t.Cleanup(func() { SetData(nil) })
	return Current()
}

func TestShippedBlessingsValidate(t *testing.T) {
	d := shipped(t)
	assert.GreaterOrEqual(t, len(d.Blessings), 7)
	itemsDir := filepath.Join("..", "..", "_datafiles", "world", "default", "items")
	for _, b := range d.Blessings {
		if b.Perk.Item == 0 {
			continue
		}
		matches, _ := filepath.Glob(filepath.Join(itemsDir, "*", strconv.Itoa(b.Perk.Item)+"-*.yaml"))
		assert.NotEmpty(t, matches, "%s: item %d exists", b.ID, b.Perk.Item)
	}
	assert.NotEmpty(t, d.Blessings[0].Condition())
	assert.NotEmpty(t, d.Blessings[0].PerkText())
}

func TestValidateRejectsBadBlessings(t *testing.T) {
	good := Blessing{ID: "a", Name: "A", Text: "t", Deed: "boss", Count: 1, Perk: Perk{Discount: 5}}
	tests := map[string]func(b *Blessing){
		"bad id":          func(b *Blessing) { b.ID = "Bad Id" },
		"unknown deed":    func(b *Blessing) { b.Deed = "dancing" },
		"no count":        func(b *Blessing) { b.Count = 0 },
		"gives nothing":   func(b *Blessing) { b.Perk = Perk{} },
		"huge discount":   func(b *Blessing) { b.Perk.Discount = MaxDiscount + 1 },
		"too many items":  func(b *Blessing) { b.Perk = Perk{Item: 30040, Qty: MaxItemsEach + 1} },
		"no name or text": func(b *Blessing) { b.Name = "" },
	}
	for name, mutate := range tests {
		b := good
		mutate(&b)
		assert.Error(t, (&Data{Blessings: []Blessing{b}}).Validate(), name)
	}
	assert.NoError(t, (&Data{Blessings: []Blessing{good}}).Validate())
	dup := &Data{Blessings: []Blessing{good, good}}
	assert.Error(t, dup.Validate(), "duplicate ids")
}

func TestEarnedReadsLifetimeDeedsAndIron(t *testing.T) {
	shipped(t)
	totals := map[chronicle.Kind]int{chronicle.Boss: 3, chronicle.Joined: 5, chronicle.Promoted: 1}
	total := func(k chronicle.Kind) int { return totals[k] }
	ids := func(iron bool) []string {
		var out []string
		for _, b := range Earned(total, iron) {
			out = append(out, b.ID)
		}
		return out
	}
	assert.ElementsMatch(t, []string{"road-tested", "company-keeper"}, ids(false), "a standard character earns no Iron blessing")
	assert.ElementsMatch(t, []string{"road-tested", "company-keeper", "iron-tested", "iron-oath"}, ids(true))
}

func TestDiscountIsCappedAndNeverFree(t *testing.T) {
	SetData(&Data{Blessings: []Blessing{
		{ID: "a", Perk: Perk{Discount: 8}},
		{ID: "b", Perk: Perk{Discount: 8}},
	}})
	t.Cleanup(func() { SetData(nil) })
	c := &characters.Character{Blessings: []string{"a", "b"}}
	assert.Equal(t, MaxDiscount, DiscountPercent(c))
	assert.Equal(t, 90, RecruitPrice(c, 100))
	assert.Equal(t, 1, RecruitPrice(c, 1), "a price is never rounded to free")
	assert.Equal(t, 0, RecruitPrice(c, 0))
	assert.Equal(t, 100, RecruitPrice(&characters.Character{}, 100), "no blessings, no discount")
	assert.Equal(t, 0, DiscountPercent(nil))
}

type fakeProvider struct{ ids []string }

func (f fakeProvider) Earned(int) []string { return f.ids }

func TestEarnedForUsesTheProvider(t *testing.T) {
	assert.Empty(t, EarnedFor(1))
	SetProvider(fakeProvider{ids: []string{"a"}})
	t.Cleanup(func() { SetProvider(nil) })
	assert.Equal(t, []string{"a"}, EarnedFor(1))
}
