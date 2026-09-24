package standing_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/market"
	"github.com/GoMudEngine/GoMud/internal/standing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssessTierBoundaries(t *testing.T) {
	rules := standing.DefaultRules()
	for _, tc := range []struct {
		company, settlement int
		tier                standing.Tier
	}{
		{40, 40, standing.Welcome},
		{0, 40, standing.Welcome},
		{-1, 40, standing.Tolerated},
		{-40, 40, standing.Tolerated},
		{-41, 40, standing.Distrusted},
		{-90, 40, standing.Distrusted},
		{-91, 40, standing.Shunned},
		{100, -100, standing.Shunned},
		{500, 40, standing.Tolerated}, // clamped to 100
	} {
		s := standing.Assess(tc.company, tc.settlement, rules)
		assert.Equal(t, tc.tier, s.Tier, "company %d settlement %d", tc.company, tc.settlement)
	}
	assert.Equal(t, "welcome", standing.Welcome.String())
	assert.Equal(t, "shunned", standing.Shunned.String())
}

func TestPricesRoundAgainstPlayer(t *testing.T) {
	rules := standing.DefaultRules()
	welcome := standing.Assess(0, 0, rules)
	assert.Equal(t, 11, welcome.BuyPrice(11))
	assert.Equal(t, 11, welcome.SellPrice(11))
	assert.Equal(t, 10, welcome.InnPrice(10))

	distrusted := standing.Assess(-60, 40, rules)
	require.Equal(t, standing.Distrusted, distrusted.Tier)
	assert.Equal(t, 14, distrusted.BuyPrice(11), "13.2 rounds up")
	assert.Equal(t, 8, distrusted.SellPrice(11), "8.8 rounds down")
	assert.Equal(t, 1, distrusted.SellPrice(1), "a sale is never worth nothing")
	assert.Equal(t, 0, distrusted.SellPrice(0))
	assert.Equal(t, 15, distrusted.InnPrice(10))
}

func TestNoProfitableRoundTripUnderAnyTier(t *testing.T) {
	for _, good := range []market.Good{
		{ItemID: 1, BasePrice: 14, MinPrice: 1, MaxPrice: 30, MaxStock: 40, TargetStock: 20, StartStock: 4, DriftStep: 2},
		{ItemID: 2, BasePrice: 5, MinPrice: 2, MaxPrice: 12, MaxStock: 30, TargetStock: 15, StartStock: 15, DriftStep: 1},
		{ItemID: 3, BasePrice: 2, MinPrice: 1, MaxPrice: 3, MaxStock: 10, TargetStock: 5, StartStock: 5, DriftStep: 1}, // sells at the floor of 1
	} {
		assertNoProfitableRoundTrip(t, good)
	}
}

func assertNoProfitableRoundTrip(t *testing.T, good market.Good) {
	t.Helper()
	require.NoError(t, good.Validate())
	rules := standing.DefaultRules()
	rules.DistrustedMarkupPct = 90 // the configurable maximum
	for _, company := range []int{40, -40, -60, -100} {
		s := standing.Assess(company, 40, rules)
		for stock := 1; stock <= good.MaxStock; stock++ {
			buy, ok := good.AskForStock(stock)
			if !ok {
				continue
			}
			// Buying takes stock to stock-1; selling it back is priced there.
			sell, ok := good.BidForStock(stock-1, 20)
			if !ok {
				continue
			}
			assert.Less(t, s.SellPrice(sell), s.BuyPrice(buy), "tier %s stock %d", s.Tier, stock)
		}
	}
}

func TestRefusalsAndBlackMarket(t *testing.T) {
	rules := standing.DefaultRules()
	cases := map[standing.Tier]struct{ market, inn, black bool }{
		standing.Welcome:    {false, false, false},
		standing.Tolerated:  {false, false, false},
		standing.Distrusted: {false, false, true},
		standing.Shunned:    {true, true, true},
	}
	for _, company := range []int{40, -20, -60, -100} {
		s := standing.Assess(company, 40, rules)
		want := cases[s.Tier]
		assert.Equal(t, want.market, s.MarketRefused(), s.Tier.String())
		assert.Equal(t, want.inn, s.InnRefused(), s.Tier.String())
		assert.Equal(t, want.black, s.BlackMarketServes(), s.Tier.String())
	}
}

type stubProvider map[string]standing.Standing

func (p stubProvider) For(_ int, zone string) (standing.Standing, bool) {
	s, ok := p[zone]
	return s, ok
}

func TestProviderSeam(t *testing.T) {
	_, ok := standing.For(7, "Dunmar")
	assert.False(t, ok, "no provider, no standing")
	want := standing.Assess(-100, 40, standing.DefaultRules())
	standing.SetProvider(stubProvider{"Dunmar": want})
	t.Cleanup(func() { standing.SetProvider(nil) })
	got, ok := standing.For(7, "Dunmar")
	assert.True(t, ok)
	assert.Equal(t, want, got)
	_, ok = standing.For(7, "Nowhere")
	assert.False(t, ok)
}
