// Package encumbrance contains the GoMud-free durable company cargo
// container and the pure party-load calculation, matching the
// domain/module split of internal/expedition, internal/camping, and
// internal/weather.
//
// This is a separate, weight-based, party/expedition-level system. Since
// Phase 32f it is the only carrying limit: GoMud's count-based
// Character.CarryCapacity() no longer throttles movement.
package encumbrance

import "errors"

var (
	ErrInvalidCargo      = errors.New("encumbrance: invalid cargo")
	ErrInvalidAmount     = errors.New("encumbrance: amount must be positive")
	ErrInsufficientCargo = errors.New("encumbrance: not enough of that item in cargo")
)

// CargoStack is one item type's quantity in a company's shared cargo.
// Uses is how many uses each item in the stack has left; 0 means full (or
// an item without uses). Partly used items stack apart from full ones, so
// a half-drunk waterskin comes back out half-drunk (Phase 32f).
type CargoStack struct {
	ItemId int
	Count  int
	Uses   int `yaml:"uses,omitempty"`
}

// Cargo is the durable, leader-owned shared cargo container.
type Cargo struct {
	LeaderUserID int
	Stacks       []CargoStack
	// Applied (Phase 33f3) lists the most recent operation IDs whose
	// deposits this cargo already holds, saved with the stacks, so a
	// retried deposit (a camp's forage after a restart) is never doubled.
	Applied []string `yaml:"applied,omitempty"`
}

// MaxAppliedOps bounds Cargo.Applied: older operations are forgotten.
const MaxAppliedOps = 32

// HasApplied reports whether an operation's deposit is already in the
// cargo.
func (c Cargo) HasApplied(op string) bool {
	for _, a := range c.Applied {
		if a == op {
			return true
		}
	}
	return false
}

// MarkApplied returns a copy remembering op, keeping the last
// MaxAppliedOps.
func (c Cargo) MarkApplied(op string) Cargo {
	applied := append(append([]string(nil), c.Applied...), op)
	if len(applied) > MaxAppliedOps {
		applied = applied[len(applied)-MaxAppliedOps:]
	}
	c.Applied = applied
	return c
}

func (c Cargo) Validate() error {
	if c.LeaderUserID <= 0 {
		return ErrInvalidCargo
	}
	for _, s := range c.Stacks {
		if s.ItemId <= 0 || s.Count <= 0 || s.Uses < 0 {
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

// Deposit returns a copy with count more full items of itemId added.
// Capacity is not this package's concern: a module checks prospective
// weight against capacity before calling Deposit.
func (c Cargo) Deposit(itemId, count int) (Cargo, error) {
	return c.DepositUses(itemId, 0, count)
}

// DepositUses returns a copy with count more of itemId, each with uses
// left (0 for full), merged into the stack with the same uses.
func (c Cargo) DepositUses(itemId, uses, count int) (Cargo, error) {
	if err := c.Validate(); err != nil {
		return c, err
	}
	if itemId <= 0 || count <= 0 || uses < 0 {
		return c, ErrInvalidAmount
	}
	stacks := append([]CargoStack(nil), c.Stacks...)
	for i, s := range stacks {
		if s.ItemId == itemId && s.Uses == uses {
			stacks[i].Count += count
			c.Stacks = stacks
			return c, nil
		}
	}
	c.Stacks = append(stacks, CargoStack{ItemId: itemId, Count: count, Uses: uses})
	return c, nil
}

// pick is the stack index to draw one itemId from: the partly used stack
// with the fewest uses left, else the full one; -1 when there is none.
func (c Cargo) pick(itemId int) int {
	best := -1
	for i, s := range c.Stacks {
		if s.ItemId != itemId {
			continue
		}
		if best < 0 {
			best = i
			continue
		}
		b := c.Stacks[best]
		if s.Uses > 0 && (b.Uses == 0 || s.Uses < b.Uses) {
			best = i
		}
	}
	return best
}

// WithdrawOne returns a copy with one itemId removed, preferring a partly
// used one, and the uses it had left (0 for full).
func (c Cargo) WithdrawOne(itemId int) (Cargo, int, error) {
	if err := c.Validate(); err != nil {
		return c, 0, err
	}
	if itemId <= 0 {
		return c, 0, ErrInvalidAmount
	}
	i := c.pick(itemId)
	if i < 0 {
		return c, 0, ErrInsufficientCargo
	}
	stacks := append([]CargoStack(nil), c.Stacks...)
	uses := stacks[i].Uses
	if stacks[i].Count == 1 {
		stacks = append(stacks[:i], stacks[i+1:]...)
	} else {
		stacks[i].Count--
	}
	c.Stacks = stacks
	return c, uses, nil
}

// Withdraw returns a copy with count less of itemId, partly used items
// first, removing a stack once its count reaches zero. It refuses to
// withdraw more than is stored.
func (c Cargo) Withdraw(itemId, count int) (Cargo, error) {
	if err := c.Validate(); err != nil {
		return c, err
	}
	if itemId <= 0 || count <= 0 {
		return c, ErrInvalidAmount
	}
	if c.CountOf(itemId) < count {
		return c, ErrInsufficientCargo
	}
	out := c
	for range count {
		var err error
		if out, _, err = out.WithdrawOne(itemId); err != nil {
			return c, err
		}
	}
	return out, nil
}

// ConsumeUse returns a copy with one use taken from one itemId, a partly
// used one first. fullUses is the item's uses when full; an item with one
// use or none is used up whole.
func (c Cargo) ConsumeUse(itemId, fullUses int) (Cargo, error) {
	out, uses, err := c.WithdrawOne(itemId)
	if err != nil {
		return c, err
	}
	if uses == 0 {
		uses = fullUses
	}
	if left := uses - 1; left > 0 {
		if out, err = out.DepositUses(itemId, left, 1); err != nil {
			return c, err
		}
	}
	return out, nil
}

// CountOf is how many of itemId the cargo holds, full or partly used.
func (c Cargo) CountOf(itemId int) int {
	n := 0
	for _, s := range c.Stacks {
		if s.ItemId == itemId {
			n += s.Count
		}
	}
	return n
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
	// CompanionGrams is the living companions' worn and carried gear
	// (Phase 28).
	CompanionGrams int
	CargoGrams     int
	CapacityGrams  int
	// MemberCapacityGrams and MountCapacityGrams split CapacityGrams
	// (Phase 32f): what the members carry, and what the horses do.
	MemberCapacityGrams int
	MountCapacityGrams  int
}

// WouldExceed reports whether adding grams would put the load over its
// capacity (Phase 32f). Adding nothing never does; reaching capacity
// exactly is allowed.
func (l Load) WouldExceed(addGrams int) bool {
	return addGrams > 0 && l.TotalGrams()+addGrams > l.CapacityGrams
}

// MemberCapacity is one member's share of the company's capacity (Phase
// 32f): the configured base, plus perStrength grams per point of Strength,
// plus their largest pack. Negative Strength adds nothing.
func MemberCapacity(baseGrams, perStrengthGrams, strength, packGrams int) int {
	if strength < 0 {
		strength = 0
	}
	if packGrams < 0 {
		packGrams = 0
	}
	return baseGrams + perStrengthGrams*strength + packGrams
}

func (l Load) TotalGrams() int {
	return l.PersonalGrams + l.CompanionGrams + l.CargoGrams
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
