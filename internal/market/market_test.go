package market

import (
	"errors"
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validGood() Good {
	return Good{
		ItemID:      28,
		BasePrice:   12,
		MinPrice:    6,
		MaxPrice:    30,
		MaxStock:    40,
		TargetStock: 20,
		StartStock:  4,
		DriftStep:   2,
	}
}

func TestGoodValidateAcceptsValidGood(t *testing.T) {
	require.NoError(t, validGood().Validate())
}

func TestGoodValidateRejectsNonPositivePrices(t *testing.T) {
	for name, mutate := range map[string]func(*Good){
		"zero min":     func(g *Good) { g.MinPrice = 0 },
		"negative min": func(g *Good) { g.MinPrice = -1 },
		"zero all":     func(g *Good) { g.MinPrice, g.BasePrice, g.MaxPrice = 0, 0, 0 },
	} {
		t.Run(name, func(t *testing.T) {
			g := validGood()
			mutate(&g)
			assert.True(t, errors.Is(g.Validate(), ErrInvalidGood))
		})
	}
}

func TestGoodValidateRejectsInvalidBounds(t *testing.T) {
	for name, mutate := range map[string]func(*Good){
		"zero item id":           func(g *Good) { g.ItemID = 0 },
		"negative item id":       func(g *Good) { g.ItemID = -3 },
		"base below min":         func(g *Good) { g.BasePrice = 5 },
		"base above max":         func(g *Good) { g.BasePrice = 31 },
		"zero target":            func(g *Good) { g.TargetStock = 0 },
		"target at max":          func(g *Good) { g.TargetStock = 40 },
		"target above max":       func(g *Good) { g.TargetStock = 41 },
		"zero max stock":         func(g *Good) { g.MaxStock, g.TargetStock, g.StartStock, g.DriftStep = 0, 0, 0, 0 },
		"negative start":         func(g *Good) { g.StartStock = -1 },
		"start above max":        func(g *Good) { g.StartStock = 41 },
		"zero drift step":        func(g *Good) { g.DriftStep = 0 },
		"drift step above max":   func(g *Good) { g.DriftStep = 41 },
		"negative drift step":    func(g *Good) { g.DriftStep = -2 },
		"one-unit stock ceiling": func(g *Good) { g.MaxStock, g.TargetStock, g.StartStock, g.DriftStep = 1, 1, 0, 1 },
	} {
		t.Run(name, func(t *testing.T) {
			g := validGood()
			mutate(&g)
			assert.True(t, errors.Is(g.Validate(), ErrInvalidGood))
		})
	}
}

func TestPriceForStockAtZeroReturnsMaxPrice(t *testing.T) {
	g := validGood()
	assert.Equal(t, 30, g.PriceForStock(0))
	assert.Equal(t, 30, g.PriceForStock(-50), "negative stock clamps to zero")
}

func TestPriceForStockAtTargetReturnsBasePrice(t *testing.T) {
	assert.Equal(t, 12, validGood().PriceForStock(20))
}

func TestPriceForStockAtOrAboveMaxStockReturnsMinPrice(t *testing.T) {
	g := validGood()
	assert.Equal(t, 6, g.PriceForStock(40))
	assert.Equal(t, 6, g.PriceForStock(1000))
}

func TestPriceForStockInterpolatesBothSegments(t *testing.T) {
	g := validGood()
	// Segment one: 30 - floor(18*10/20) = 21.
	assert.Equal(t, 21, g.PriceForStock(10))
	// Segment two: 12 - floor(6*10/20) = 9.
	assert.Equal(t, 9, g.PriceForStock(30))
}

func TestPriceForStockMonotonicallyDecreasing(t *testing.T) {
	goods := []Good{
		validGood(),
		{ItemID: 1, MinPrice: 1, BasePrice: 1, MaxPrice: 1, MaxStock: 2, TargetStock: 1, DriftStep: 1},
		{ItemID: 1, MinPrice: 1, BasePrice: 100, MaxPrice: 1000, MaxStock: 7, TargetStock: 3, DriftStep: 1},
		{ItemID: 1, MinPrice: 3, BasePrice: 4, MaxPrice: 5, MaxStock: 1000, TargetStock: 999, DriftStep: 10},
	}
	for _, g := range goods {
		require.NoError(t, g.Validate())
		prev := g.PriceForStock(-5)
		for stock := -4; stock <= g.MaxStock+5; stock++ {
			price := g.PriceForStock(stock)
			assert.LessOrEqual(t, price, prev, "stock %d", stock)
			assert.GreaterOrEqual(t, price, g.MinPrice)
			assert.LessOrEqual(t, price, g.MaxPrice)
			prev = price
		}
	}
}

func TestPriceForStockAvoidsOverflow(t *testing.T) {
	g := Good{
		ItemID:      1,
		MinPrice:    1,
		BasePrice:   math.MaxInt / 2,
		MaxPrice:    math.MaxInt,
		MaxStock:    math.MaxInt,
		TargetStock: math.MaxInt - 1,
		StartStock:  0,
		DriftStep:   math.MaxInt,
	}
	require.NoError(t, g.Validate())
	assert.Equal(t, math.MaxInt, g.PriceForStock(0))
	assert.Equal(t, math.MaxInt/2, g.PriceForStock(math.MaxInt-1))
	assert.Equal(t, 1, g.PriceForStock(math.MaxInt))

	mid := g.PriceForStock(math.MaxInt / 2)
	assert.Greater(t, mid, math.MaxInt/2, "halfway through segment one sits between base and max")
	assert.Less(t, mid, math.MaxInt)

	prev := math.MaxInt
	for _, stock := range []int{0, 1, math.MaxInt / 4, math.MaxInt / 2, math.MaxInt - 2, math.MaxInt - 1, math.MaxInt} {
		price := g.PriceForStock(stock)
		assert.LessOrEqual(t, price, prev)
		assert.Positive(t, price)
		prev = price
	}
}

func TestDriftStockStaysWithinBoundsAndConverges(t *testing.T) {
	rng := rand.New(rand.NewSource(19))
	goods := []Good{
		validGood(),
		{ItemID: 1, MinPrice: 1, BasePrice: 2, MaxPrice: 3, MaxStock: 2, TargetStock: 1, DriftStep: 2},
		{ItemID: 1, MinPrice: 1, BasePrice: 2, MaxPrice: 3, MaxStock: 500, TargetStock: 17, DriftStep: 13},
	}
	for _, g := range goods {
		require.NoError(t, g.Validate())
		starts := []int{-100, -1, 0, 1, g.TargetStock - 1, g.TargetStock, g.TargetStock + 1, g.MaxStock - 1, g.MaxStock, g.MaxStock + 1, math.MaxInt, math.MinInt}
		for _, start := range starts {
			stock := start
			converged := false
			for i := 0; i < 2*g.MaxStock+10; i++ {
				before := stock
				stock = g.DriftStock(stock, rng.Uint64())
				require.GreaterOrEqual(t, stock, 0)
				require.LessOrEqual(t, stock, g.MaxStock)
				clamped := min(max(before, 0), g.MaxStock)
				assert.LessOrEqual(t, abs(stock-g.TargetStock), abs(clamped-g.TargetStock), "never moves away from target")
				assert.LessOrEqual(t, abs(stock-clamped), g.DriftStep, "never moves more than one drift step")
				if stock == g.TargetStock {
					converged = true
					break
				}
			}
			require.True(t, converged, "start %d converges to target", start)
			for i := 0; i < 50; i++ {
				stock = g.DriftStock(stock, rng.Uint64())
				require.Equal(t, g.TargetStock, stock, "stays at target once there")
			}
		}
	}
}

func TestDriftStockIsDeterministicForFixedRoll(t *testing.T) {
	g := validGood() // DriftStep 2: roll%2 picks a step of 1 or 2
	assert.Equal(t, 5, g.DriftStock(4, 0))
	assert.Equal(t, 6, g.DriftStock(4, 1))
	assert.Equal(t, 6, g.DriftStock(4, 3))
	assert.Equal(t, 38, g.DriftStock(40, math.MaxUint64))
	assert.Equal(t, 20, g.DriftStock(19, 1), "never overshoots the target")
	assert.Equal(t, 20, g.DriftStock(20, 1))
}

func TestStockLevelDescribesBands(t *testing.T) {
	g := validGood() // target 20, max 40
	assert.Equal(t, "none", g.StockLevel(0))
	assert.Equal(t, "scarce", g.StockLevel(9))
	assert.Equal(t, "short", g.StockLevel(10))
	assert.Equal(t, "short", g.StockLevel(19))
	assert.Equal(t, "steady", g.StockLevel(20))
	assert.Equal(t, "steady", g.StockLevel(29))
	assert.Equal(t, "plentiful", g.StockLevel(30))
	assert.Equal(t, "glutted", g.StockLevel(40))
	assert.Equal(t, "glutted", g.StockLevel(99))
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
