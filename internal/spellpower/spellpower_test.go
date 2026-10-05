package spellpower

import "testing"

var (
	mm   = Power{Base: 7, Dice: "1d6", LevelDiv: 10, MysticismDiv: 15}
	heal = Power{Base: 8, Dice: "2d4", LevelDiv: 6}
	all  = Power{Base: 8, Dice: "2d4", LevelDiv: 6, Percent: 55}
)

func TestRangeMatchesTheTable(t *testing.T) {
	cases := []struct {
		name          string
		p             Power
		level, myst   int
		wantLo, wantH int
	}{
		{"mm L1", mm, 1, 0, 8, 13},
		{"mm L10 myst 15", mm, 10, 15, 10, 15},
		{"mm L30 myst 30", mm, 30, 30, 13, 18},
		{"heal L1", heal, 1, 50, 10, 16},
		{"heal L10", heal, 10, 0, 11, 17},
		{"heal L30", heal, 30, 0, 15, 21},
		{"healall L1", all, 1, 0, 5, 8},
		{"healall L30", all, 30, 0, 8, 11},
	}
	for _, c := range cases {
		lo, hi := c.p.Range(c.level, c.myst)
		if lo != c.wantLo || hi != c.wantH {
			t.Errorf("%s: Range = %d-%d, want %d-%d", c.name, lo, hi, c.wantLo, c.wantH)
		}
	}
}

func TestRollStaysInRange(t *testing.T) {
	for _, p := range []Power{mm, heal, all} {
		lo, hi := p.Range(12, 20)
		low := p.Roll(12, 20, func(int) int { return 0 })
		high := p.Roll(12, 20, func(n int) int { return n - 1 })
		if low != lo || high != hi {
			t.Errorf("%+v: rolls %d-%d, range %d-%d", p, low, high, lo, hi)
		}
	}
	if got := (Power{Base: 1, Percent: 10}).Roll(1, 0, nil); got != 1 {
		t.Errorf("a tiny power rolls at least 1, got %d", got)
	}
}

func TestMean(t *testing.T) {
	if got := heal.Mean(0, 0); got != 13 {
		t.Errorf("heal mean at L0 = %v, want 13", got)
	}
	if got := all.Mean(0, 0); got != 13*0.55 {
		t.Errorf("healall mean at L0 = %v, want %v", got, 13*0.55)
	}
}

func TestValidate(t *testing.T) {
	bad := []Power{
		{Base: 1, Dice: "d6"},
		{Base: 1, Dice: "2x6"},
		{Base: 1, Dice: "0d6"},
		{Base: -1, Dice: "1d6"},
		{Base: 1, LevelDiv: -2},
		{},
	}
	for _, p := range bad {
		if p.Validate() == nil {
			t.Errorf("%+v should not validate", p)
		}
	}
	for _, p := range []Power{mm, heal, all, {Base: 3}} {
		if err := p.Validate(); err != nil {
			t.Errorf("%+v: %v", p, err)
		}
	}
}
