package interrupt

import "testing"

func TestBreaks(t *testing.T) {
	cases := []struct {
		name     string
		hit      bool
		damage   int
		chanting bool
		want     bool
	}{
		{"a damaging hit on a chanter", true, 1, true, true},
		{"a heavy hit on a chanter", true, 9, true, true},
		{"a miss", false, 0, true, false},
		{"a hit for nothing", true, 0, true, false},
		{"not chanting", true, 5, false, false},
	}
	for _, c := range cases {
		if got := Breaks(c.hit, c.damage, c.chanting); got != c.want {
			t.Errorf("%s: Breaks = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestRefund(t *testing.T) {
	for cost, want := range map[int]int{0: 0, 1: 0, 3: 1, 8: 4, -2: 0} {
		if got := Refund(cost); got != want {
			t.Errorf("Refund(%d) = %d, want %d", cost, got, want)
		}
	}
}

func TestCanCounter(t *testing.T) {
	able := Counter{Missed: true, Melee: true, SameRoom: true, Shield: true, Able: true}
	if !CanCounter(able) {
		t.Fatal("an able bearer of a shield counters a missed melee blow")
	}
	cases := []struct {
		name string
		edit func(*Counter)
	}{
		{"the blow landed", func(c *Counter) { c.Missed = false }},
		{"a shot from a bow", func(c *Counter) { c.Melee = false }},
		{"from another room", func(c *Counter) { c.SameRoom = false }},
		{"no shield", func(c *Counter) { c.Shield = false }},
		{"down or stunned", func(c *Counter) { c.Able = false }},
		{"chanting", func(c *Counter) { c.Chanting = true }},
		{"already this round", func(c *Counter) { c.Countered = true }},
	}
	for _, tc := range cases {
		c := able
		tc.edit(&c)
		if CanCounter(c) {
			t.Errorf("%s: CanCounter = true, want false", tc.name)
		}
	}
}

// script returns each value in turn.
func script(values ...int) func(int) int {
	return func(int) int {
		v := values[0]
		values = values[1:]
		return v
	}
}

func TestRollCounter(t *testing.T) {
	// Rolls are 0..n-1: the bash, then the damage face, then the stun.
	if _, ok := RollCounter(script(BashChance)); ok {
		t.Error("a roll at the bash chance bashes, want none")
	}
	b, ok := RollCounter(script(BashChance-1, 2, StunChance))
	if !ok || b.Damage != 3 || b.Stun {
		t.Errorf("bash = %+v %v, want 3 damage, no stun", b, ok)
	}
	b, ok = RollCounter(script(0, 0, StunChance-1))
	if !ok || b.Damage != 1 || !b.Stun {
		t.Errorf("bash = %+v %v, want 1 damage and a stun", b, ok)
	}
	b, _ = RollCounter(script(0, BashDamage-1, 99))
	if b.Damage != BashDamage {
		t.Errorf("top damage = %d, want %d", b.Damage, BashDamage)
	}
}

func TestRollCounterChancesAdjustable(t *testing.T) {
	defer func(b, s int) { BashChance, StunChance = b, s }(BashChance, StunChance)
	BashChance, StunChance = 100, 100
	b, ok := RollCounter(script(99, 0, 99))
	if !ok || !b.Stun {
		t.Errorf("certain chances: bash = %+v %v, want a stunning bash", b, ok)
	}
	BashChance = 0
	if _, ok := RollCounter(script(0)); ok {
		t.Error("a bash chance of 0 never bashes")
	}
}
