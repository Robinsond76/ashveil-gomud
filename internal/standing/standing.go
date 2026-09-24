// Package standing holds the Phase 21b settlement standing rules: how a
// settlement treats a company, judged by the gap between their alignments.
// Standing is derived, never stored.
package standing

import "sync"

// Tier is how a settlement treats a company.
type Tier int

const (
	Welcome Tier = iota
	Tolerated
	Distrusted
	Shunned
)

func (t Tier) String() string {
	switch t {
	case Welcome:
		return "welcome"
	case Tolerated:
		return "tolerated"
	case Distrusted:
		return "distrusted"
	case Shunned:
		return "shunned"
	}
	return "unknown"
}

// Rules are the standing knobs. Gaps are in engine alignment points
// (−100..100 scale); markups are percentages.
type Rules struct {
	WelcomeGap             int // gap at or below this is welcome
	ToleratedGap           int // ... tolerated
	DistrustedGap          int // ... distrusted; beyond is shunned
	DistrustedMarkupPct    int // market buy up, sell down, when distrusted
	DistrustedInnMarkupPct int // inn price up, when distrusted
}

// DefaultRules returns the design's defaults.
func DefaultRules() Rules {
	return Rules{WelcomeGap: 40, ToleratedGap: 80, DistrustedGap: 130, DistrustedMarkupPct: 20, DistrustedInnMarkupPct: 50}
}

// Standing is a company's standing in one settlement.
type Standing struct {
	Zone         string
	Tier         Tier
	Company      int // company alignment, engine scale
	Settlement   int // settlement alignment, engine scale
	MarkupPct    int
	InnMarkupPct int
}

func clampAlignment(a int) int { return min(max(a, -100), 100) }

// Assess judges a company against a settlement.
func Assess(company, settlement int, rules Rules) Standing {
	company, settlement = clampAlignment(company), clampAlignment(settlement)
	gap := company - settlement
	if gap < 0 {
		gap = -gap
	}
	s := Standing{Company: company, Settlement: settlement}
	switch {
	case gap <= rules.WelcomeGap:
		s.Tier = Welcome
	case gap <= rules.ToleratedGap:
		s.Tier = Tolerated
	case gap <= rules.DistrustedGap:
		s.Tier = Distrusted
		s.MarkupPct = rules.DistrustedMarkupPct
		s.InnMarkupPct = rules.DistrustedInnMarkupPct
	default:
		s.Tier = Shunned
	}
	return s
}

// BuyPrice is what the company pays for something priced p, rounded up.
func (s Standing) BuyPrice(p int) int {
	return (p*(100+s.MarkupPct) + 99) / 100
}

// SellPrice is what the company is paid for something priced p, rounded
// down but never below 1 for a sale worth something. Buy prices only rise
// and sell prices only fall, so a buy-then-sell round trip can't profit
// where it didn't before.
func (s Standing) SellPrice(p int) int {
	if p <= 0 {
		return 0
	}
	return max(p*(100-s.MarkupPct)/100, 1)
}

// InnPrice is what the company pays for a room priced p, rounded up.
func (s Standing) InnPrice(p int) int {
	return (p*(100+s.InnMarkupPct) + 99) / 100
}

// MarketRefused reports whether the settlement's market won't trade.
func (s Standing) MarketRefused() bool { return s.Tier == Shunned }

// InnRefused reports whether the settlement's inns won't give a room.
func (s Standing) InnRefused() bool { return s.Tier == Shunned }

// BlackMarketServes reports whether a black market deals with the company:
// only with those the settlement distrusts or shuns.
func (s Standing) BlackMarketServes() bool { return s.Tier >= Distrusted }

// Provider answers a leader's company standing in a zone; false when the
// zone isn't a settlement or the company's alignment is unknown.
type Provider interface {
	For(leaderUserID int, zone string) (Standing, bool)
}

var (
	providerMu sync.RWMutex
	provider   Provider
)

// SetProvider installs the standing provider (modules/standing).
func SetProvider(p Provider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	provider = p
}

// For is the leader's company standing in a zone, if any.
func For(leaderUserID int, zone string) (Standing, bool) {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	if p == nil {
		return Standing{}, false
	}
	return p.For(leaderUserID, zone)
}

// BlackMarketTagger is optionally implemented by the provider: the room tag
// marking black markets.
type BlackMarketTagger interface {
	BlackMarketTag() string
}

// BlackMarketTag is the black-market room tag, or "" when no provider
// supplies one (no black markets).
func BlackMarketTag() string {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	if t, ok := p.(BlackMarketTagger); ok {
		return t.BlackMarketTag()
	}
	return ""
}
