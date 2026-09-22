// Package encumbrance contains the GoMud-free durable company cargo
// container and the pure party-load calculation, matching the
// domain/module split of internal/expedition, internal/camping, and
// internal/weather.
//
// This is a separate, weight-based, party/expedition-level system. It is
// deliberately independent of GoMud's native, count-based
// Character.CarryCapacity() (internal/characters/character.go), which
// already throttles a single player's per-move action-point cost and must
// not be touched by this package.
package encumbrance

import "errors"

var (
	ErrInvalidCargo      = errors.New("encumbrance: invalid cargo")
	ErrInvalidAmount     = errors.New("encumbrance: amount must be positive")
	ErrInsufficientCargo = errors.New("encumbrance: not enough of that item in cargo")
)

// CargoStack is one item type's quantity in a company's shared cargo.
type CargoStack struct {
	ItemId int
	Count  int
}

// Cargo is the durable, leader-owned shared cargo container.
type Cargo struct {
	LeaderUserID int
	Stacks       []CargoStack
}

func (c Cargo) Validate() error {
	if c.LeaderUserID <= 0 {
		return ErrInvalidCargo
	}
	for _, s := range c.Stacks {
		if s.ItemId <= 0 || s.Count <= 0 {
			return ErrInvalidCargo
		}
	}
	return nil
}

// Established creates an empty cargo container for a leader.
func Established(leaderUserID int) (Cargo, error) {
	c := Cargo{LeaderUserID: leaderUserID}
	if err := c.Validate(); err != nil {
		return Cargo{}, err
	}
	return c, nil
}

// Deposit returns a copy with count more of itemId added, merging into an
// existing stack. Capacity is not this package's concern: a module checks
// prospective weight against capacity before calling Deposit.
func (c Cargo) Deposit(itemId, count int) (Cargo, error) {
	if err := c.Validate(); err != nil {
		return c, err
	}
	if itemId <= 0 || count <= 0 {
		return c, ErrInvalidAmount
	}
	stacks := append([]CargoStack(nil), c.Stacks...)
	for i, s := range stacks {
		if s.ItemId == itemId {
			stacks[i].Count += count
			c.Stacks = stacks
			return c, nil
		}
	}
	c.Stacks = append(stacks, CargoStack{ItemId: itemId, Count: count})
	return c, nil
}

// Withdraw returns a copy with count less of itemId, removing the stack
// entirely once its count reaches zero. It refuses to withdraw more than is
// stored.
func (c Cargo) Withdraw(itemId, count int) (Cargo, error) {
	if err := c.Validate(); err != nil {
		return c, err
	}
	if itemId <= 0 || count <= 0 {
		return c, ErrInvalidAmount
	}
	stacks := append([]CargoStack(nil), c.Stacks...)
	for i, s := range stacks {
		if s.ItemId != itemId {
			continue
		}
		if s.Count < count {
			return c, ErrInsufficientCargo
		}
		if s.Count == count {
			stacks = append(stacks[:i], stacks[i+1:]...)
		} else {
			stacks[i].Count -= count
		}
		c.Stacks = stacks
		return c, nil
	}
	return c, ErrInsufficientCargo
}

// TotalCount returns the total number of individual items across all
// stacks.
func (c Cargo) TotalCount() int {
	total := 0
	for _, s := range c.Stacks {
		total += s.Count
	}
	return total
}

// Load is a computed (never persisted) snapshot of a company's current
// weight, in grams, against its capacity.
type Load struct {
	PersonalGrams int
	CargoGrams    int
	CapacityGrams int
}

func (l Load) TotalGrams() int {
	return l.PersonalGrams + l.CargoGrams
}

// Ratio is TotalGrams/CapacityGrams, or 0 when capacity is non-positive
// (an unconfigured or misconfigured capacity never divides by zero or
// reports an infinite/negative ratio).
func (l Load) Ratio() float64 {
	if l.CapacityGrams <= 0 {
		return 0
	}
	return float64(l.TotalGrams()) / float64(l.CapacityGrams)
}

// LoadBand is one threshold in a module-configured, ratio-ordered table
// mapping load ratio to travel-duration/fatigue modifiers. 100 means
// unchanged for both.
type LoadBand struct {
	MinRatio          float64
	TravelDurationPct int
	FatiguePct        int
}

func (b LoadBand) Validate() error {
	if b.MinRatio < 0 || b.TravelDurationPct < 0 || b.FatiguePct < 0 {
		return ErrInvalidCargo
	}
	return nil
}

// ResolveBand returns the band with the highest MinRatio not exceeding
// ratio. bands must already be sorted ascending by MinRatio (the module
// validates this at config parse time); an empty table, or a ratio below
// every band's MinRatio, resolves to no modifier (100%/100%).
func ResolveBand(ratio float64, bands []LoadBand) LoadBand {
	resolved := LoadBand{TravelDurationPct: 100, FatiguePct: 100}
	for _, b := range bands {
		if ratio >= b.MinRatio {
			resolved = b
		}
	}
	return resolved
}
