package market

import "sort"

// Sighting is one good's stock at one market, as last heard (Phase 20
// trade rumours).
type Sighting struct {
	Place string
	Good  Good
	Stock int
}

// RumorKind is what a trade rumour says about a good at a market.
type RumorKind int

const (
	// RumorScarce: the market is short (stock level none or scarce).
	RumorScarce RumorKind = iota + 1
	// RumorGlut: the market is overstocked (plentiful or glutted).
	RumorGlut
	// RumorCheapest: of two or more markets trading the good, this one had
	// the unique lowest open buy price. A sold-out market isn't selling, so
	// it doesn't compete.
	RumorCheapest
	// RumorBestBuyer: of two or more markets trading the good, this one had
	// the unique highest open sell price. A full market isn't buying, so it
	// doesn't compete.
	RumorBestBuyer
)

// Rumor is one fuzzy trade hint: no prices, only a good, a market, and a
// kind.
type Rumor struct {
	Kind   RumorKind
	ItemID int
	Place  string
}

// Rumors derives every trade rumour from the sightings, ordered by item
// ID, then kind, then place. Sightings must hold valid goods; spreadPct is
// the markets' buy/sell spread (see Good.BidForStock).
func Rumors(sightings []Sighting, spreadPct int) []Rumor {
	out := []Rumor{}
	byItem := map[int][]Sighting{}
	for _, s := range sightings {
		switch s.Good.StockLevel(s.Stock) {
		case "none", "scarce":
			out = append(out, Rumor{Kind: RumorScarce, ItemID: s.Good.ItemID, Place: s.Place})
		case "plentiful", "glutted":
			out = append(out, Rumor{Kind: RumorGlut, ItemID: s.Good.ItemID, Place: s.Place})
		}
		byItem[s.Good.ItemID] = append(byItem[s.Good.ItemID], s)
	}
	for itemID, seen := range byItem {
		if len(seen) < 2 {
			continue
		}
		if place, ok := uniqueExtreme(seen, func(s Sighting) (int, bool) { return s.Good.AskForStock(s.Stock) }, false); ok {
			out = append(out, Rumor{Kind: RumorCheapest, ItemID: itemID, Place: place})
		}
		if place, ok := uniqueExtreme(seen, func(s Sighting) (int, bool) { return s.Good.BidForStock(s.Stock, spreadPct) }, true); ok {
			out = append(out, Rumor{Kind: RumorBestBuyer, ItemID: itemID, Place: place})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.ItemID != b.ItemID {
			return a.ItemID < b.ItemID
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.Place < b.Place
	})
	return out
}

// uniqueExtreme returns the place with the lowest (or highest) open price,
// when exactly one place holds it. Closed sides are skipped.
func uniqueExtreme(seen []Sighting, price func(Sighting) (int, bool), highest bool) (string, bool) {
	best, place, open, ties := 0, "", 0, 0
	for _, s := range seen {
		p, ok := price(s)
		if !ok {
			continue
		}
		open++
		switch {
		case open == 1, highest && p > best, !highest && p < best:
			best, place, ties = p, s.Place, 1
		case p == best:
			ties++
		}
	}
	if open == 0 || ties != 1 {
		return "", false
	}
	return place, true
}

// PickRumors returns up to n distinct rumours chosen at random with roll,
// leaving the input untouched.
func PickRumors(rumors []Rumor, n int, roll func() uint64) []Rumor {
	pool := append([]Rumor(nil), rumors...)
	n = min(max(n, 0), len(pool))
	for i := 0; i < n; i++ {
		j := i + int(roll()%uint64(len(pool)-i))
		pool[i], pool[j] = pool[j], pool[i]
	}
	return pool[:n]
}
