package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/banter"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// opinionStub is a company module with a view on what the leader sells.
type opinionStub struct {
	choices []opinions.Choice
}

func (s *opinionStub) FormationFor(int) (company.Formation, bool) { return company.Formation{}, false }
func (s *opinionStub) InstanceFor(int, int) (int, bool)           { return 0, false }
func (s *opinionStub) LeaderAndKeyForInstance(int) (int, company.MemberKey, bool) {
	var k company.MemberKey
	return 0, k, false
}
func (s *opinionStub) CampBanter(int, string) []banter.Said { return nil }
func (s *opinionStub) LastBanter(int) []banter.Said         { return nil }
func (s *opinionStub) Opinion(_ int, c opinions.Choice) ([]string, error) {
	s.choices = append(s.choices, c)
	return []string{`Maren murmurs, "Those were someone's hallows."`}, nil
}

// Phase 64: selling a relic is a choice the company has an opinion on; an
// ordinary sale is not.
func TestSellingARelicLetsTheCompanyReact(t *testing.T) {
	stub := &opinionStub{}
	company.SetFormationProvider(stub)
	t.Cleanup(func() { company.SetFormationProvider(nil) })
	const relic = 50011
	seller, room, merchant := shopRoom(t, ironShortSword)
	useShippedItems(t, relic)
	merchant.Character.Shop = append(merchant.Character.Shop, characters.ShopItem{ItemId: relic, Quantity: 1, QuantityMax: 2})
	require.True(t, items.IsRelicItem(relic))

	require.True(t, seller.Character.StoreItem(items.New(ironShortSword)))
	run(t, Sell, "shortsword", seller, room)
	assert.Empty(t, stub.choices, "an ordinary sale is no choice")

	sold := items.New(relic)
	sold.Adjectives = []string{"glowing"} // DisplayName now carries colour tags
	require.True(t, seller.Character.StoreItem(sold))
	text := run(t, Sell, "helm", seller, room)
	require.Len(t, stub.choices, 1, text)
	assert.Equal(t, opinions.SellRelic, stub.choices[0].Kind)
	assert.NotContains(t, stub.choices[0].Subject, "<", "the remembered subject is the plain name, no colour tags (shown in the web tab)")
	assert.NotEmpty(t, stub.choices[0].Subject)
	assert.Contains(t, text, "Those were someone's hallows.")
}

// Phase 64 review: relics marked as junk and sold with `sell junk` are still
// relics sold; the company reacts once for the whole sale.
func TestSellingJunkRelicsLetsTheCompanyReactOnce(t *testing.T) {
	stub := &opinionStub{}
	company.SetFormationProvider(stub)
	t.Cleanup(func() { company.SetFormationProvider(nil) })
	const relic = 50011
	seller, room, merchant := shopRoom(t, ironShortSword)
	useShippedItems(t, relic)
	merchant.Character.Shop = append(merchant.Character.Shop, characters.ShopItem{ItemId: relic, Quantity: 1, QuantityMax: 5})

	for range 2 {
		it := items.New(relic)
		it.Junk = true
		require.True(t, seller.Character.StoreItem(it))
	}
	text := run(t, Sell, "junk", seller, room)
	require.Contains(t, text, "junk item(s)")
	require.Len(t, stub.choices, 1, text)
	assert.Equal(t, opinions.SellRelic, stub.choices[0].Kind)
	assert.Contains(t, text, "Those were someone's hallows.")
}
