package encumbrance

import (
	"errors"
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/items"
	"sync"
)

// Provider is implemented by modules/encumbrance. It is a read-only query
// seam, the same shape as weather.Provider and survival.CompanyService.
type Provider interface {
	CurrentLoad(leaderUserID int) (Load, bool)
}

var (
	providerMu sync.RWMutex
	provider   Provider
)

// SetProvider registers the active encumbrance provider. Passing nil clears
// it.
func SetProvider(p Provider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	provider = p
}

// CurrentLoad consults the registered provider for a leader's current party
// load. Without a provider, ok is false.
func CurrentLoad(leaderUserID int) (Load, bool) {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	if p == nil {
		return Load{}, false
	}
	return p.CurrentLoad(leaderUserID)
}

// BandProvider is optionally implemented by the registered Provider. It
// resolves the configured load band for a leader's current load (Phase 16).
type BandProvider interface {
	CurrentBand(leaderUserID int) (LoadBand, bool)
}

// CurrentBand resolves a leader's current load band through the registered
// provider. Without one, or for an untracked load, ok is false and the band
// is neutral (100%/100%).
func CurrentBand(leaderUserID int) (LoadBand, bool) {
	neutral := LoadBand{TravelDurationPct: 100, FatiguePct: 100}
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	bp, ok := p.(BandProvider)
	if !ok {
		return neutral, false
	}
	band, ok := bp.CurrentBand(leaderUserID)
	if !ok {
		return neutral, false
	}
	return band, true
}

// WouldExceed reports whether adding grams to a leader's company would put
// its load over capacity (Phase 32f), with the current load. An untracked
// load (no provider, no capacity configured) never refuses. Call it on the
// game loop: the load reads company state.
func WouldExceed(leaderUserID, addGrams int) (Load, bool) {
	if addGrams <= 0 {
		return Load{}, false
	}
	load, ok := CurrentLoad(leaderUserID)
	if !ok {
		return Load{}, false
	}
	return load, load.WouldExceed(addGrams)
}

// TooMuchToCarry is the refusal a player sees when adding grams would put
// their company over capacity (Phase 32f); ok is false when it fits.
func TooMuchToCarry(leaderUserID, addGrams int) (string, bool) {
	load, refuse := WouldExceed(leaderUserID, addGrams)
	if !refuse {
		return "", false
	}
	return fmt.Sprintf(`That would be too much for your company to carry: %.1f kg of %.1f kg already (<ansi fg="command">help cargo</ansi>).`,
		float64(load.TotalGrams())/1000, float64(load.CapacityGrams)/1000), true
}

// CargoProvider is optionally implemented by the registered Provider: it
// reads a company's cargo and uses it up (Phase 32f's company inventory
// and meals). Call it on the game loop.
type CargoProvider interface {
	CargoContents(leaderUserID int) []CargoStack
	ConsumeCargoUse(leaderUserID, itemId int) error
}

// ErrNoCargo is returned when no cargo provider is registered.
var ErrNoCargo = errors.New("encumbrance: no cargo")

func cargoProvider() (CargoProvider, bool) {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	cp, ok := p.(CargoProvider)
	return cp, ok
}

// CargoContents is a copy of a leader's cargo stacks; nil without a
// provider or cargo.
func CargoContents(leaderUserID int) []CargoStack {
	cp, ok := cargoProvider()
	if !ok {
		return nil
	}
	return cp.CargoContents(leaderUserID)
}

// ConsumeCargoUse takes one use from one of a leader's cargo items and
// saves, a partly used one first.
func ConsumeCargoUse(leaderUserID, itemId int) error {
	cp, ok := cargoProvider()
	if !ok {
		return ErrNoCargo
	}
	return cp.ConsumeCargoUse(leaderUserID, itemId)
}

// CargoKeeper is optionally implemented by the registered Provider (Phase
// 33f3): it adds to and takes from a company's cargo. Call it on the game
// loop.
type CargoKeeper interface {
	// DepositCargo adds full items to the leader's cargo and saves. A
	// non-empty op is applied once: a repeat of the same op deposits
	// nothing and returns nil.
	DepositCargo(leaderUserID int, op string, deposits []CargoStack) error
	// WithdrawCargo removes count of an item (partly used ones first) and
	// saves; ErrInsufficientCargo when there aren't that many.
	WithdrawCargo(leaderUserID, itemId, count int) error
}

func cargoKeeper() (CargoKeeper, bool) {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	ck, ok := p.(CargoKeeper)
	return ck, ok
}

// DepositCargo adds to a leader's cargo; ErrNoCargo without a keeper.
func DepositCargo(leaderUserID int, op string, deposits []CargoStack) error {
	ck, ok := cargoKeeper()
	if !ok {
		return ErrNoCargo
	}
	return ck.DepositCargo(leaderUserID, op, deposits)
}

// WithdrawCargo takes from a leader's cargo; ErrNoCargo without a keeper.
func WithdrawCargo(leaderUserID, itemId, count int) error {
	ck, ok := cargoKeeper()
	if !ok {
		return ErrNoCargo
	}
	return ck.WithdrawCargo(leaderUserID, itemId, count)
}

// SharedCargoProvider migrates the legacy container and activates shared cargo.
type SharedCargoProvider interface{ UnifyCargo(int) error }

func UnifyCargo(id int) error {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	if cp, ok := p.(SharedCargoProvider); ok {
		return cp.UnifyCargo(id)
	}
	return nil
}

func ConsumeCargoItemUse(id int, itm items.Item) error {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	if cp, ok := p.(interface{ ConsumeCargoItemUse(int, items.Item) error }); ok {
		return cp.ConsumeCargoItemUse(id, itm)
	}
	return ErrNoCargo
}

// AddedGrams accounts for the capacity a new pack creates in shared cargo.
func AddedGrams(id int, carried []items.Item, itm items.Item) int {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	if cp, ok := p.(interface {
		CargoAddedGrams(int, items.Item) (int, bool)
	}); ok {
		if grams, tracked := cp.CargoAddedGrams(id, itm); tracked {
			return grams
		}
	}
	largest := 0
	for _, old := range carried {
		largest = max(largest, old.CarryBonusGrams())
	}
	return itm.Weight() - max(0, itm.CarryBonusGrams()-largest)
}

// ExchangeGrams measures both weight and changed pack capacity for a trade-in.
func ExchangeGrams(id int, carried, removed, added []items.Item) int {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	if cp, ok := p.(interface {
		CargoExchangeGrams(int, []items.Item, []items.Item) (int, bool)
	}); ok {
		if grams, tracked := cp.CargoExchangeGrams(id, removed, added); tracked {
			return grams
		}
	}
	next := append([]items.Item(nil), carried...)
	before, after, grams := 0, 0, 0
	for _, itm := range carried {
		before = max(before, itm.CarryBonusGrams())
	}
	for _, itm := range removed {
		for i, old := range next {
			if old.Equals(itm) {
				next = append(next[:i], next[i+1:]...)
				grams -= old.Weight()
				break
			}
		}
	}
	for _, itm := range added {
		next = append(next, itm)
		grams += itm.Weight()
	}
	for _, itm := range next {
		after = max(after, itm.CarryBonusGrams())
	}
	return grams - (after - before)
}

// TransformCargo atomically replaces exact cargo instances, for recipes.
func TransformCargo(id int, inputs, outputs []items.Item) error {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	if cp, ok := p.(interface {
		TransformCargo(int, []items.Item, []items.Item) error
	}); ok {
		return cp.TransformCargo(id, inputs, outputs)
	}
	return ErrNoCargo
}
