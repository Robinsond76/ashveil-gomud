// Package market holds the pure Phase 19 settlement-market math: a tracked
// good's validated price and stock bounds, its stock-derived price, the
// bounded round-driven stock drift toward a baseline, and a coarse stock
// descriptor. It imports no engine packages; modules/market owns the
// durable zone ledger, the round listener, and the market command.
package market

import (
	"errors"
	"fmt"
	"math/bits"
)

// ErrInvalidGood reports a Good whose bounds are unusable.
var ErrInvalidGood = errors.New("market: invalid good")

// Good is one tracked commodity in a settlement's market.
type Good struct {
	ItemID      int `yaml:"itemid"`
	BasePrice   int `yaml:"baseprice"`   // price at TargetStock
	MinPrice    int `yaml:"minprice"`    // floor, reached at MaxStock
	MaxPrice    int `yaml:"maxprice"`    // ceiling, reached at stock 0
	MaxStock    int `yaml:"maxstock"`    // stock ceiling
	TargetStock int `yaml:"targetstock"` // baseline stock, strictly inside (0, MaxStock)
	StartStock  int `yaml:"startstock"`  // initial stock on first load
	DriftStep   int `yaml:"driftstep"`   // maximum stock movement per round
}

// Validate requires ItemID > 0, 0 < MinPrice <= BasePrice <= MaxPrice,
// 0 < TargetStock < MaxStock, 0 <= StartStock <= MaxStock, and
// 1 <= DriftStep <= MaxStock.
func (g Good) Validate() error {
	switch {
	case g.ItemID <= 0:
		return fmt.Errorf("%w: item id %d must be positive", ErrInvalidGood, g.ItemID)
	case g.MinPrice <= 0:
		return fmt.Errorf("%w: min price %d must be positive", ErrInvalidGood, g.MinPrice)
	case g.BasePrice < g.MinPrice || g.MaxPrice < g.BasePrice:
		return fmt.Errorf("%w: prices must satisfy min %d <= base %d <= max %d", ErrInvalidGood, g.MinPrice, g.BasePrice, g.MaxPrice)
	case g.TargetStock <= 0 || g.TargetStock >= g.MaxStock:
		return fmt.Errorf("%w: target stock %d must lie strictly between 0 and max stock %d", ErrInvalidGood, g.TargetStock, g.MaxStock)
	case g.StartStock < 0 || g.StartStock > g.MaxStock:
		return fmt.Errorf("%w: start stock %d must lie within 0..%d", ErrInvalidGood, g.StartStock, g.MaxStock)
	case g.DriftStep < 1 || g.DriftStep > g.MaxStock:
		return fmt.Errorf("%w: drift step %d must lie within 1..%d", ErrInvalidGood, g.DriftStep, g.MaxStock)
	}
	return nil
}

// ClampStock clamps stock to [0, MaxStock].
func (g Good) ClampStock(stock int) int {
	return min(max(stock, 0), max(g.MaxStock, 0))
}

// PriceForStock returns the price for a stock level. It clamps stock to
// [0, MaxStock] and interpolates two monotone linear segments,
// (0, MaxPrice) -> (TargetStock, BasePrice) and
// (TargetStock, BasePrice) -> (MaxStock, MinPrice), subtracting the floor
// of each segment's proportional decrease so the endpoints are exact.
// g must be valid.
func (g Good) PriceForStock(stock int) int {
	stock = g.ClampStock(stock)
	if stock <= g.TargetStock {
		return g.MaxPrice - scaledDrop(g.MaxPrice-g.BasePrice, stock, g.TargetStock)
	}
	return g.BasePrice - scaledDrop(g.BasePrice-g.MinPrice, stock-g.TargetStock, g.MaxStock-g.TargetStock)
}

// scaledDrop returns floor(diff*pos/span) for 0 <= pos <= span, span > 0,
// diff >= 0, using a 128-bit intermediate so extreme valid integers never
// overflow. The quotient is at most diff, so it always fits an int.
func scaledDrop(diff, pos, span int) int {
	if diff <= 0 || pos <= 0 || span <= 0 {
		return 0
	}
	if pos >= span {
		return diff
	}
	hi, lo := bits.Mul64(uint64(diff), uint64(pos))
	quo, _ := bits.Div64(hi, lo, uint64(span))
	return int(quo)
}

// DriftStock clamps currentStock to [0, MaxStock], then moves it toward
// TargetStock by min(distance, 1 + roll % DriftStep). At target it stays
// put. It changes only this good's stock, never the world clock.
func (g Good) DriftStock(currentStock int, roll uint64) int {
	stock := g.ClampStock(currentStock)
	if stock == g.TargetStock {
		return stock
	}
	step := 1
	if g.DriftStep > 1 {
		step += int(roll % uint64(g.DriftStep))
	}
	if stock < g.TargetStock {
		return stock + min(step, g.TargetStock-stock)
	}
	return stock - min(step, stock-g.TargetStock)
}

// StockLevel is a coarse, player-facing descriptor for a stock level:
// none (0), scarce (below half the target), short (below target),
// steady (target up to halfway to the ceiling), plentiful (beyond that),
// and glutted (at the ceiling).
func (g Good) StockLevel(stock int) string {
	stock = g.ClampStock(stock)
	switch {
	case stock <= 0:
		return "none"
	case stock >= g.MaxStock:
		return "glutted"
	case stock < g.TargetStock/2:
		return "scarce"
	case stock < g.TargetStock:
		return "short"
	case stock < g.TargetStock+(g.MaxStock-g.TargetStock)/2:
		return "steady"
	}
	return "plentiful"
}
