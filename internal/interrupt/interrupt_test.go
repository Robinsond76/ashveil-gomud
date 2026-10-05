package interrupt

import "testing"

func TestCanBreak(t *testing.T) {
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
		if got := CanBreak(c.hit, c.damage, c.chanting); got != c.want {
			t.Errorf("%s: CanBreak = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestBreakChance(t *testing.T) {
	cases := []struct {
		name          string
		damage, maxHP int
		heavy         bool
		difficulty    int
		want          int
	}{
		{"heavy force always breaks", 1, 100, true, 0, 100},
		{"a nick", 1, 100, false, 0, 42},
		{"a scratch on a giant", 1, 1000, false, 0, BreakChanceMin},
		{"a tenth of their health", 10, 100, false, 0, 60},
		{"a quarter of their health", 25, 100, false, 0, BreakChanceMax},
		{"more than a quarter is capped", 80, 100, false, 0, BreakChanceMax},
		{"no damage never breaks", 0, 100, false, 0, 0},
		{"no damage, even heavy", 0, 100, true, 0, 0},
		{"no max health counts as 1", 1, 0, false, 0, BreakChanceMax},
		// Phase 35b: a fifth of the spell's difficulty.
		{"Magic Missile adds 15", 1, 100, false, 75, 57},
		{"Withering Hex adds 8", 10, 100, false, 40, 68},
		{"difficulty under 5 adds nothing", 1, 1000, false, 4, BreakChanceMin},
		{"difficulty still caps at 90", 20, 100, false, 100, BreakChanceMax},
		{"difficulty never breaks a blow of no damage", 0, 100, false, 100, 0},
		{"heavy force stays 100 at any difficulty", 1, 100, true, 75, 100},
	}
	for _, c := range cases {
		if got := BreakChance(c.damage, c.maxHP, c.heavy, c.difficulty); got != c.want {
			t.Errorf("%s: BreakChance(%d, %d, %v, %d) = %d, want %d", c.name, c.damage, c.maxHP, c.heavy, c.difficulty, got, c.want)
		}
	}
}

func TestBreakChanceScalesWithDamage(t *testing.T) {
	prev := 0
	for damage := 1; damage <= 30; damage++ {
		got := BreakChance(damage, 100, false, 0)
		if got < prev || got < BreakChanceMin || got > BreakChanceMax {
			t.Fatalf("BreakChance(%d, 100) = %d after %d: want rising within %d-%d", damage, got, prev, BreakChanceMin, BreakChanceMax)
		}
		prev = got
	}
}

func TestRollBreak(t *testing.T) {
	never := func(int) int { t.Fatal("a certain outcome needs no roll"); return 0 }
	if !RollBreak(100, never) {
		t.Error("a chance of 100 breaks")
	}
	if RollBreak(0, never) {
		t.Error("a chance of 0 holds")
	}
	if !RollBreak(60, script(59)) {
		t.Error("a roll under the chance breaks")
	}
	if RollBreak(60, script(60)) {
		t.Error("a roll at the chance holds")
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
	// Phase 30g2: only a blocked strike reaches CanCounter (afterBlow
	// checks the block); these are the rest of the conditions.
	able := Counter{Melee: true, SameRoom: true, Shield: true, Able: true}
	if !CanCounter(able) {
		t.Fatal("an able bearer of a shield can counter")
	}
	cases := []struct {
		name string
		edit func(*Counter)
	}{
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
	// Phase 30g2: bash chance is now passed as parameter
	if _, ok := RollCounter(50, script(50)); ok {
		t.Error("a roll at the bash chance bashes, want none")
	}
	b, ok := RollCounter(50, script(49, 2, 25))
	if !ok || b.Damage != 3 || b.Stun {
		t.Errorf("bash = %+v %v, want 3 damage, no stun", b, ok)
	}
	b, ok = RollCounter(50, script(0, 0, 24))
	if !ok || b.Damage != 1 || !b.Stun {
		t.Errorf("bash = %+v %v, want 1 damage and a stun", b, ok)
	}
	b, _ = RollCounter(50, script(0, 3, 99))
	if b.Damage != BashDamage {
		t.Errorf("top damage = %d, want %d", b.Damage, BashDamage)
	}
}

func TestRollCounterChancesAdjustable(t *testing.T) {
	// Phase 30g2: bash chance is now passed as parameter
	defer func(s int) { StunChance = s }(StunChance)
	StunChance = 100
	b, ok := RollCounter(100, script(99, 0, 99))
	if !ok || !b.Stun {
		t.Errorf("certain chances: bash = %+v %v, want a stunning bash", b, ok)
	}
	if _, ok := RollCounter(0, script(0)); ok {
		t.Error("a bash chance of 0 never bashes")
	}
}

func TestBreaksWindUp(t *testing.T) {
	cases := []struct {
		name   string
		hit    bool
		damage int
		heavy  bool
		want   bool
	}{
		{"heavy force that drew blood", true, 3, true, true},
		{"an ordinary blow", true, 20, false, false},
		{"a miss", false, 0, true, false},
		{"heavy, but no damage", true, 0, true, false},
	}
	for _, c := range cases {
		if got := BreaksWindUp(c.hit, c.damage, c.heavy); got != c.want {
			t.Errorf("%s: BreaksWindUp = %v, want %v", c.name, got, c.want)
		}
	}
}
