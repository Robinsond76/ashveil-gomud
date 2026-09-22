package encumbrance

import (
	"errors"
	"testing"
)

func TestEstablished(t *testing.T) {
	c, err := Established(7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.LeaderUserID != 7 || len(c.Stacks) != 0 {
		t.Fatalf("unexpected cargo: %+v", c)
	}

	if _, err := Established(0); !errors.Is(err, ErrInvalidCargo) {
		t.Errorf("expected ErrInvalidCargo, got %v", err)
	}
}

func TestDeposit(t *testing.T) {
	c, _ := Established(7)

	c, err := c.Deposit(100, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.Stacks) != 1 || c.Stacks[0].Count != 3 {
		t.Fatalf("unexpected stacks: %+v", c.Stacks)
	}

	c, err = c.Deposit(100, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.Stacks) != 1 || c.Stacks[0].Count != 5 {
		t.Fatalf("expected merged stack of 5, got %+v", c.Stacks)
	}

	c, err = c.Deposit(200, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.Stacks) != 2 {
		t.Fatalf("expected a second stack, got %+v", c.Stacks)
	}

	if _, err := c.Deposit(0, 1); !errors.Is(err, ErrInvalidAmount) {
		t.Errorf("expected ErrInvalidAmount for bad itemId, got %v", err)
	}
	if _, err := c.Deposit(100, 0); !errors.Is(err, ErrInvalidAmount) {
		t.Errorf("expected ErrInvalidAmount for zero count, got %v", err)
	}
}

func TestWithdraw(t *testing.T) {
	c, _ := Established(7)
	c, _ = c.Deposit(100, 5)

	c, err := c.Withdraw(100, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.Stacks) != 1 || c.Stacks[0].Count != 3 {
		t.Fatalf("expected 3 remaining, got %+v", c.Stacks)
	}

	c, err = c.Withdraw(100, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.Stacks) != 0 {
		t.Fatalf("expected stack removed entirely, got %+v", c.Stacks)
	}

	if _, err := c.Withdraw(100, 1); !errors.Is(err, ErrInsufficientCargo) {
		t.Errorf("expected ErrInsufficientCargo for empty cargo, got %v", err)
	}

	c, _ = c.Deposit(100, 2)
	if _, err := c.Withdraw(100, 5); !errors.Is(err, ErrInsufficientCargo) {
		t.Errorf("expected ErrInsufficientCargo for over-withdraw, got %v", err)
	}
}

func TestTotalCount(t *testing.T) {
	c, _ := Established(7)
	c, _ = c.Deposit(100, 3)
	c, _ = c.Deposit(200, 4)
	if got := c.TotalCount(); got != 7 {
		t.Errorf("expected 7, got %d", got)
	}
}

func TestLoadRatio(t *testing.T) {
	l := Load{PersonalGrams: 3000, CargoGrams: 2000, CapacityGrams: 10000}
	if got := l.TotalGrams(); got != 5000 {
		t.Errorf("expected 5000, got %d", got)
	}
	if got := l.Ratio(); got != 0.5 {
		t.Errorf("expected 0.5, got %v", got)
	}

	zeroCapacity := Load{PersonalGrams: 100, CapacityGrams: 0}
	if got := zeroCapacity.Ratio(); got != 0 {
		t.Errorf("expected 0 ratio for non-positive capacity, got %v", got)
	}
}

func TestResolveBand(t *testing.T) {
	bands := []LoadBand{
		{MinRatio: 0, TravelDurationPct: 100, FatiguePct: 100},
		{MinRatio: 0.75, TravelDurationPct: 110, FatiguePct: 108},
		{MinRatio: 0.9, TravelDurationPct: 125, FatiguePct: 115},
	}

	cases := []struct {
		ratio      float64
		wantTravel int
	}{
		{0, 100},
		{0.5, 100},
		{0.75, 110},
		{0.89, 110},
		{0.9, 125},
		{2.0, 125},
	}
	for _, tc := range cases {
		got := ResolveBand(tc.ratio, bands)
		if got.TravelDurationPct != tc.wantTravel {
			t.Errorf("ratio %v: expected travel pct %d, got %d", tc.ratio, tc.wantTravel, got.TravelDurationPct)
		}
	}

	empty := ResolveBand(5.0, nil)
	if empty.TravelDurationPct != 100 || empty.FatiguePct != 100 {
		t.Errorf("expected no-modifier default for empty table, got %+v", empty)
	}
}
